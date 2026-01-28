<template>
  <div class="app-shell fade-in">
    <header class="topbar">
      <div>
        <h1>Gickup Control Room</h1>
        <p>Local operator console for configs, runs, and metrics.</p>
      </div>
      <div class="tabs">
        <button
          class="tab"
          :class="{ active: activeTab === 'config' }"
          @click="setTab('config')"
        >
          Config
        </button>
        <button
          class="tab"
          :class="{ active: activeTab === 'dashboard' }"
          @click="setTab('dashboard')"
        >
          Dashboard
        </button>
      </div>
    </header>

    <section v-if="activeTab === 'dashboard'" class="hero">
      <div class="status-bar">
        <div class="status-pill">
          <span class="status-dot" :class="{ running: status.running }"></span>
          <span>{{ status.running ? 'Backup Running' : 'Idle' }}</span>
          <span v-if="status.running">PID {{ status.pid }}</span>
        </div>
        <div class="controls">
          <button class="btn" :disabled="status.running" @click="startBackup">Start Backup</button>
          <button class="btn secondary" :disabled="!status.running" @click="stopBackup">Stop Backup</button>
          <button class="btn secondary" @click="refreshAll">Refresh</button>
        </div>
      </div>
      <div class="status-bar">
        <div class="badge" v-if="status.running && status.start_time">
          Started {{ new Date(status.start_time).toLocaleString() }}
        </div>
        <div class="badge warn" v-if="status.last_error">
          {{ status.last_error }}
        </div>
        <div class="badge" v-if="status.last_exit_code !== undefined && !status.running">
          Last exit code {{ status.last_exit_code }}
        </div>
      </div>
    </section>

    <section v-if="activeTab === 'config'" class="panel">
      <h2>Config Builder (GitHub Source)</h2>
      <div class="split">
        <div class="panel inset">
          <h3>Source: GitHub</h3>
          <div class="form-grid">
            <label>
              Token
              <input v-model="form.github.token" type="text" placeholder="ghp_..." />
            </label>
            <label>
              Token File
              <input v-model="form.github.tokenFile" type="text" placeholder="token.txt" />
            </label>
            <div v-if="tokenConflict" class="hint warn">
              Use either Token or Token File (not both).
            </div>
            <label>
              User (target owner)
              <input v-model="form.github.user" type="text" placeholder="some-user" />
            </label>
            <label>
              Username (clone auth)
              <input v-model="form.github.username" type="text" placeholder="your-user" />
            </label>
            <label>
              Password (clone auth)
              <input v-model="form.github.password" type="password" placeholder="password or token" />
            </label>
            <label class="checkbox">
              <input v-model="form.github.ssh" type="checkbox" />
              Use SSH
            </label>
            <label>
              SSH Key
              <input v-model="form.github.sshkey" type="text" placeholder="C:\\path\\to\\id_rsa" />
            </label>
          </div>
        </div>

        <div class="panel inset">
          <h3>Destination: Local</h3>
          <div class="form-grid">
            <label>
              Path
              <input v-model="form.local.path" type="text" placeholder="D:\\gickup-backup" />
            </label>
          </div>

          <h3 style="margin-top: 16px">Schedule & Metrics</h3>
          <div class="form-grid">
            <label>
              Cron
              <input v-model="form.cron" type="text" placeholder="0 22 * * *" />
            </label>
            <label>
              Prometheus Listen
              <input v-model="form.prometheus.listen" type="text" placeholder=":6178" />
            </label>
            <label>
              Prometheus Endpoint
              <input v-model="form.prometheus.endpoint" type="text" placeholder="/metrics" />
            </label>
          </div>
          <div class="controls" style="margin-top: 12px">
            <button class="btn" @click="applyForm">Generate YAML</button>
            <button class="btn secondary" @click="fillFormFromYaml">Load From YAML</button>
            <button class="btn secondary" @click="resetForm">Reset</button>
            <button
              v-if="showGoDashboard"
              class="btn secondary"
              @click="setTab('dashboard')"
            >
              Go to Dashboard
            </button>
          </div>
        </div>
      </div>

      <div class="extend">
        <button class="extend-toggle" @click="showExtend = !showExtend">
          Extend Settings
          <span>{{ showExtend ? 'Hide' : 'Show' }}</span>
        </button>
        <div v-if="showExtend" class="extend-body">
          <div class="split">
            <div class="panel inset">
              <h3>GitHub Options</h3>
              <div class="form-grid">
                <label>
                  Include Repos (comma separated)
                  <input v-model="form.github.include" type="text" placeholder="repo-a, repo-b" />
                </label>
                <label>
                  Exclude Repos (comma separated)
                  <input v-model="form.github.exclude" type="text" placeholder="repo-x, repo-y" />
                </label>
                <label>
                  Include Orgs (comma separated)
                  <input v-model="form.github.includeOrgs" type="text" placeholder="org-a, org-b" />
                </label>
                <label>
                  Exclude Orgs (comma separated)
                  <input v-model="form.github.excludeOrgs" type="text" placeholder="org-x, org-y" />
                </label>
                <label class="checkbox">
                  <input v-model="form.github.wiki" type="checkbox" />
                  Include Wiki
                </label>
                <label class="checkbox">
                  <input v-model="form.github.issues" type="checkbox" />
                  Include Issues (local only)
                </label>
                <label class="checkbox">
                  <input v-model="form.github.starred" type="checkbox" />
                  Include Starred
                </label>
                <label class="checkbox">
                  <input v-model="form.github.gists" type="checkbox" />
                  Include Gists
                </label>
              </div>

              <h3 style="margin-top: 16px">Filter</h3>
              <div class="form-grid">
                <label>
                  Stars (min)
                  <input v-model.number="form.github.filterStars" type="number" min="0" />
                </label>
                <label>
                  Last Activity (e.g. 1y, 6M, 30d)
                  <input v-model="form.github.filterLastActivity" type="text" placeholder="1y" />
                </label>
                <label>
                  Languages (comma separated)
                  <input v-model="form.github.filterLanguages" type="text" placeholder="go, java" />
                </label>
                <label class="checkbox">
                  <input v-model="form.github.filterExcludeArchived" type="checkbox" />
                  Exclude Archived
                </label>
                <label class="checkbox">
                  <input v-model="form.github.filterExcludeForks" type="checkbox" />
                  Exclude Forks
                </label>
              </div>
            </div>

            <div class="panel inset">
              <h3>Local Options</h3>
              <div class="form-grid">
                <label class="checkbox">
                  <input v-model="form.local.bare" type="checkbox" />
                  Bare
                </label>
                <label class="checkbox">
                  <input v-model="form.local.mirror" type="checkbox" />
                  Mirror
                </label>
                <label class="checkbox">
                  <input v-model="form.local.structured" type="checkbox" />
                  Structured
                </label>
                <label class="checkbox">
                  <input v-model="form.local.zip" type="checkbox" />
                  Zip
                </label>
                <label>
                  Keep (days)
                  <input v-model.number="form.local.keep" type="number" min="0" />
                </label>
                <label class="checkbox">
                  <input v-model="form.local.lfs" type="checkbox" />
                  LFS
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section v-if="activeTab === 'config'" class="panel">
      <h2>Config Editor</h2>
      <div class="split">
        <div>
          <textarea v-model="configText" spellcheck="false"></textarea>
          <div class="controls" style="margin-top: 12px">
            <button class="btn" @click="saveConfig">Save Config</button>
            <button class="btn secondary" @click="validateConfig">Validate</button>
            <button class="btn secondary" @click="clearValidation">Clear Output</button>
            <a class="btn secondary" href="/api/config/download">Download</a>
          </div>
        </div>
        <div>
          <h3>Validation Output</h3>
          <div class="log-box">
            <div v-if="validation.valid === true" class="badge">Valid</div>
            <div v-else-if="validation.valid === false" class="badge warn">Invalid</div>
            <pre v-html="validation.output || 'Run validation to see details.'"></pre>
          </div>
        </div>
      </div>
    </section>

    <section v-if="activeTab === 'dashboard'" class="section-grid">
      <div class="split">
        <div class="panel">
          <div class="panel-title">
            <h2>Run History</h2>
            <button class="btn secondary" @click="clearHistory">Clear History</button>
          </div>
          <div class="history">
            <div v-if="history.length === 0">No runs yet.</div>
            <div v-for="entry in history" :key="entry.id" class="history-item">
              <div>
                <strong>{{ formatDate(entry.start_time) }}</strong>
                <span v-if="entry.duration"> {{ historySeparator }}{{ entry.duration }}</span>
              </div>
              <div class="meta">
                Exit {{ entry.exit_code }}{{ historySeparator }}Stopped {{ entry.stopped ? 'yes' : 'no' }}
              </div>
              <div class="meta" v-if="entry.error">{{ entry.error }}</div>
            </div>
          </div>
        </div>
        <div class="panel">
          <div class="panel-title">
            <h2>Recent Logs</h2>
            <button class="btn secondary" @click="clearLogs">Clear Logs</button>
          </div>
          <div class="log-box">
            <div class="ansi-output" v-html="logsHtml || ''"></div>
          </div>
        </div>
      </div>
    </section>

    <section v-if="activeTab === 'dashboard'" class="panel">
      <h2>Prometheus Metrics</h2>
      <div class="metric-grid">
        <div class="metric-card">
          <h4>Sources</h4>
          <strong>{{ metrics.sources ?? placeholder }}</strong>
        </div>
        <div class="metric-card">
          <h4>Destinations</h4>
          <strong>{{ metrics.destinations ?? placeholder }}</strong>
        </div>
        <div class="metric-card">
          <h4>Jobs Started</h4>
          <strong>{{ metrics.jobsStarted ?? placeholder }}</strong>
        </div>
        <div class="metric-card">
          <h4>Jobs Complete</h4>
          <strong>{{ metrics.jobsComplete ?? placeholder }}</strong>
        </div>
        <div class="metric-card">
          <h4>Avg Duration</h4>
          <strong>{{ metrics.avgDuration ?? placeholder }}</strong>
        </div>
        <div class="metric-card">
          <h4>Repos OK</h4>
          <strong>{{ metrics.repoOk ?? placeholder }}</strong>
        </div>
      </div>
      <div class="log-box metrics" style="margin-top: 16px">
        <pre>{{ metrics.raw || 'Metrics will appear once Prometheus is enabled and the backup is running.' }}</pre>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, reactive, ref } from 'vue'
import yaml from 'js-yaml'
import AnsiToHtml from 'ansi-to-html'

const configText = ref('')
const validation = reactive({ valid: null, output: '' })
const status = reactive({ running: false })
const history = ref([])
const logs = ref([])
const logsHtml = ref('')
const historySeparator = ' \u00b7 '
const placeholder = '\u2014'
const metrics = reactive({
  sources: null,
  destinations: null,
  jobsStarted: null,
  jobsComplete: null,
  avgDuration: null,
  repoOk: null,
  raw: ''
})

const activeTab = ref('config')
const showExtend = ref(false)
const showGoDashboard = ref(false)

const ansiConverter = new AnsiToHtml({ fg: '#e2e8f0', bg: '#0f172a', newline: true })

const renderAnsi = (value) => ansiConverter.toHtml(value || '')

const defaultForm = () => ({
  github: {
    token: '',
    tokenFile: '',
    user: '',
    username: '',
    password: '',
    ssh: false,
    sshkey: '',
    include: '',
    exclude: '',
    includeOrgs: '',
    excludeOrgs: '',
    wiki: false,
    issues: false,
    starred: false,
    gists: false,
    filterStars: 0,
    filterLastActivity: '',
    filterLanguages: '',
    filterExcludeArchived: false,
    filterExcludeForks: false
  },
  local: {
    path: '',
    bare: false,
    mirror: false,
    structured: false,
    zip: false,
    keep: 0,
    lfs: false
  },
  cron: '',
  prometheus: {
    listen: '',
    endpoint: ''
  }
})

const form = reactive(defaultForm())

const tokenConflict = computed(
  () => form.github.token.trim() !== '' && form.github.tokenFile.trim() !== ''
)

let poller

const fetchJSON = async (url, options) => {
  const res = await fetch(url, options)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  return res.json()
}

const setTab = (tab) => {
  activeTab.value = tab
  if (tab === 'dashboard') {
    refreshAll()
  }
}

const loadConfig = async () => {
  const res = await fetch('/api/config')
  if (res.ok) {
    configText.value = await res.text()
  } else {
    configText.value = ''
  }

  if (configText.value.trim().length === 0) {
    activeTab.value = 'config'
  } else {
    activeTab.value = 'dashboard'
  }

  fillFormFromYaml()
}

const saveConfig = async () => {
  await fetchJSON('/api/config', {
    method: 'POST',
    headers: { 'Content-Type': 'text/yaml' },
    body: configText.value
  })
}

const validateConfig = async () => {
  const data = await fetchJSON('/api/validate', {
    method: 'POST',
    headers: { 'Content-Type': 'text/yaml' },
    body: configText.value
  })
  validation.valid = data.valid
  validation.output = renderAnsi(data.output || '')
}

const clearValidation = () => {
  validation.valid = null
  validation.output = ''
}

const splitList = (value) =>
  value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)

const yamlLine = (key, value, indent) => {
  const pad = ' '.repeat(indent)
  return `${pad}${key}: ${value}`
}

const yamlList = (key, values, indent) => {
  const pad = ' '.repeat(indent)
  const listPad = ' '.repeat(indent + 2)
  if (!values || values.length === 0) return []
  return [`${pad}${key}:`, ...values.map((v) => `${listPad}- ${v}`)]
}

const applyForm = () => {
  const lines = []
  lines.push('source:')
  lines.push('  github:')
  lines.push('    -')

  const gh = form.github
  const addField = (key, value) => {
    if (value === '' || value === null || value === undefined) return
    lines.push(yamlLine(key, value, 6))
  }
  const addBool = (key, value) => {
    if (value === true) {
      lines.push(yamlLine(key, 'true', 6))
    }
  }

  addField('token', gh.token)
  addField('token_file', gh.tokenFile)
  addField('user', gh.user)
  addField('username', gh.username)
  addField('password', gh.password)
  addBool('ssh', gh.ssh)
  addField('sshkey', gh.sshkey)

  yamlList('include', splitList(gh.include), 6).forEach((l) => lines.push(l))
  yamlList('exclude', splitList(gh.exclude), 6).forEach((l) => lines.push(l))
  yamlList('includeorgs', splitList(gh.includeOrgs), 6).forEach((l) => lines.push(l))
  yamlList('excludeorgs', splitList(gh.excludeOrgs), 6).forEach((l) => lines.push(l))

  addBool('wiki', gh.wiki)
  addBool('issues', gh.issues)
  addBool('starred', gh.starred)
  addBool('gists', gh.gists)

  const filterLines = []
  if (gh.filterStars && gh.filterStars > 0) {
    filterLines.push(yamlLine('stars', gh.filterStars, 8))
  }
  if (gh.filterLastActivity) {
    filterLines.push(yamlLine('lastactivity', gh.filterLastActivity, 8))
  }
  const languages = splitList(gh.filterLanguages)
  if (languages.length > 0) {
    filterLines.push('        languages:')
    languages.forEach((lang) => {
      filterLines.push(`          - ${lang}`)
    })
  }
  if (gh.filterExcludeArchived) {
    filterLines.push(yamlLine('excludearchived', 'true', 8))
  }
  if (gh.filterExcludeForks) {
    filterLines.push(yamlLine('excludeforks', 'true', 8))
  }
  if (filterLines.length > 0) {
    lines.push('      filter:')
    filterLines.forEach((l) => lines.push(l))
  }

  lines.push('destination:')
  lines.push('  local:')
  lines.push('    -')
  const local = form.local
  const addLocalField = (key, value) => {
    if (value === '' || value === null || value === undefined) return
    lines.push(yamlLine(key, value, 6))
  }
  const addLocalBool = (key, value) => {
    if (value === true) {
      lines.push(yamlLine(key, 'true', 6))
    }
  }
  addLocalField('path', local.path)
  addLocalBool('bare', local.bare)
  addLocalBool('mirror', local.mirror)
  addLocalBool('structured', local.structured)
  addLocalBool('zip', local.zip)
  if (local.keep && local.keep > 0) {
    lines.push(yamlLine('keep', local.keep, 6))
  }
  addLocalBool('lfs', local.lfs)

  if (form.cron) {
    lines.push(`cron: "${form.cron}"`)
  }

  if (form.prometheus.listen || form.prometheus.endpoint) {
    lines.push('metrics:')
    lines.push('  prometheus:')
    if (form.prometheus.endpoint) {
      lines.push(`    endpoint: ${form.prometheus.endpoint}`)
    }
    if (form.prometheus.listen) {
      lines.push(`    listen_addr: "${form.prometheus.listen}"`)
    }
  }

  configText.value = `${lines.join('\n')}\n`
  showGoDashboard.value = true
}

const resetForm = () => {
  Object.assign(form, defaultForm())
}

const fillFormFromYaml = () => {
  if (!configText.value.trim()) {
    resetForm()
    return
  }

  let parsed
  try {
    parsed = yaml.load(configText.value)
  } catch (err) {
    return
  }
  if (!parsed || typeof parsed !== 'object') {
    return
  }

  const next = defaultForm()

  const github = parsed?.source?.github?.[0] || {}
  next.github.token = github.token || ''
  next.github.tokenFile = github.token_file || ''
  next.github.user = github.user || ''
  next.github.username = github.username || ''
  next.github.password = github.password || ''
  next.github.ssh = !!github.ssh
  next.github.sshkey = github.sshkey || ''
  next.github.include = Array.isArray(github.include) ? github.include.join(', ') : ''
  next.github.exclude = Array.isArray(github.exclude) ? github.exclude.join(', ') : ''
  next.github.includeOrgs = Array.isArray(github.includeorgs) ? github.includeorgs.join(', ') : ''
  next.github.excludeOrgs = Array.isArray(github.excludeorgs) ? github.excludeorgs.join(', ') : ''
  next.github.wiki = !!github.wiki
  next.github.issues = !!github.issues
  next.github.starred = !!github.starred
  next.github.gists = !!github.gists

  const filter = github.filter || {}
  next.github.filterStars = filter.stars || 0
  next.github.filterLastActivity = filter.lastactivity || ''
  next.github.filterLanguages = Array.isArray(filter.languages) ? filter.languages.join(', ') : ''
  next.github.filterExcludeArchived = !!filter.excludearchived
  next.github.filterExcludeForks = !!filter.excludeforks

  const local = parsed?.destination?.local?.[0] || {}
  next.local.path = local.path || ''
  next.local.bare = !!local.bare
  next.local.mirror = !!local.mirror
  next.local.structured = !!local.structured
  next.local.zip = !!local.zip
  next.local.keep = local.keep || 0
  next.local.lfs = !!local.lfs

  next.cron = parsed?.cron || ''
  next.prometheus.listen = parsed?.metrics?.prometheus?.listen_addr || ''
  next.prometheus.endpoint = parsed?.metrics?.prometheus?.endpoint || ''

  Object.assign(form, next)
}

const startBackup = async () => {
  await fetchJSON('/api/start', { method: 'POST' })
  await refreshAll()
}

const stopBackup = async () => {
  await fetchJSON('/api/stop', { method: 'POST' })
  await refreshAll()
}

const fetchStatus = async () => {
  const data = await fetchJSON('/api/status')
  Object.assign(status, data)
}

const fetchHistory = async () => {
  history.value = await fetchJSON('/api/history')
}

const clearHistory = async () => {
  await fetchJSON('/api/history/clear', { method: 'POST' })
  history.value = []
}

const fetchLogs = async () => {
  const data = await fetchJSON('/api/logs?lines=200')
  const lines = data.lines || []
  logs.value = lines
  const rawLogs = lines.join('\n')
  const normalized = rawLogs.replace(/\\n/g, '\n')
  logsHtml.value = renderAnsi(normalized)
}

const clearLogs = async () => {
  await fetchJSON('/api/logs/clear', { method: 'POST' })
  logs.value = []
  logsHtml.value = ''
}

const parseMetrics = (raw) => {
  const summary = {
    sources: null,
    destinations: null,
    jobsStarted: null,
    jobsComplete: null,
    avgDuration: null,
    repoOk: null
  }

  let durationSum = null
  let durationCount = null
  let repoOk = 0

  raw.split('\n').forEach((line) => {
    if (!line || line.startsWith('#')) return
    const parts = line.trim().split(/\s+/)
    if (parts.length < 2) return
    const name = parts[0]
    const value = Number(parts[parts.length - 1])
    if (Number.isNaN(value)) return

    if (name === 'gickup_sources') summary.sources = value
    if (name === 'gickup_destinations') summary.destinations = value
    if (name === 'gickup_jobs_started') summary.jobsStarted = value
    if (name === 'gickup_jobs_complete') summary.jobsComplete = value
    if (name === 'gickup_job_duration_sum') durationSum = value
    if (name === 'gickup_job_duration_count') durationCount = value
    if (name.startsWith('gickup_repo_success')) {
      if (value === 1) repoOk += 1
    }
  })

  if (durationSum !== null && durationCount) {
    summary.avgDuration = `${(durationSum / durationCount).toFixed(2)}s`
  }
  summary.repoOk = repoOk || null

  return summary
}

const fetchMetrics = async () => {
  const res = await fetch('/api/metrics')
  if (!res.ok) {
    metrics.raw = await res.text()
    return
  }
  const raw = await res.text()
  metrics.raw = raw
  const parsed = parseMetrics(raw)
  Object.assign(metrics, parsed)
}

const refreshAll = async () => {
  await Promise.all([fetchStatus(), fetchHistory(), fetchLogs(), fetchMetrics()])
}

const formatDate = (value) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

onMounted(async () => {
  await loadConfig()
  await refreshAll()
  poller = setInterval(refreshAll, 5000)
})

onBeforeUnmount(() => {
  clearInterval(poller)
})
</script>







