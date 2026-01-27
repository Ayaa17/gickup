<template>
  <div class="app-shell fade-in">
    <section class="hero">
      <div>
        <h1>Gickup Control Room</h1>
        <p>Local operator console for configs, runs, and metrics.</p>
      </div>
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

    <section class="panel">
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
          <h3>Destination: Local</h3>
          <div class="form-grid">
            <label>
              Path
              <input v-model="form.local.path" type="text" placeholder="D:\\gickup-backup" />
            </label>
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
            <button class="btn secondary" @click="resetForm">Reset</button>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <h2>Config Editor</h2>
      <div class="split">
        <div>
          <textarea v-model="configText" spellcheck="false"></textarea>
          <div class="controls" style="margin-top: 12px">
            <button class="btn" @click="saveConfig">Save Config</button>
            <button class="btn secondary" @click="validateConfig">Validate</button>
            <a class="btn secondary" href="/api/config/download">Download</a>
          </div>
        </div>
        <div>
          <h3>Validation Output</h3>
          <div class="log-box">
            <div v-if="validation.valid === true" class="badge">Valid</div>
            <div v-else-if="validation.valid === false" class="badge warn">Invalid</div>
            <pre>{{ validation.output || 'Run validation to see details.' }}</pre>
          </div>
        </div>
      </div>
    </section>

    <section class="section-grid">
      <div class="split">
        <div class="panel">
          <h2>Run History</h2>
          <div class="history">
            <div v-if="history.length === 0">No runs yet.</div>
            <div v-for="entry in history" :key="entry.id" class="history-item">
              <div>
                <strong>{{ formatDate(entry.start_time) }}</strong>
                <span v-if="entry.duration"> ¡P {{ entry.duration }}</span>
              </div>
              <div class="meta">
                Exit {{ entry.exit_code }} ¡P Stopped {{ entry.stopped ? 'yes' : 'no' }}
              </div>
              <div class="meta" v-if="entry.error">{{ entry.error }}</div>
            </div>
          </div>
        </div>
        <div class="panel">
          <h2>Recent Logs</h2>
          <div class="log-box">
            <pre>{{ logs.join('\n') }}</pre>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <h2>Prometheus Metrics</h2>
      <div class="metric-grid">
        <div class="metric-card">
          <h4>Sources</h4>
          <strong>{{ metrics.sources ?? '¡X' }}</strong>
        </div>
        <div class="metric-card">
          <h4>Destinations</h4>
          <strong>{{ metrics.destinations ?? '¡X' }}</strong>
        </div>
        <div class="metric-card">
          <h4>Jobs Started</h4>
          <strong>{{ metrics.jobsStarted ?? '¡X' }}</strong>
        </div>
        <div class="metric-card">
          <h4>Jobs Complete</h4>
          <strong>{{ metrics.jobsComplete ?? '¡X' }}</strong>
        </div>
        <div class="metric-card">
          <h4>Avg Duration</h4>
          <strong>{{ metrics.avgDuration ?? '¡X' }}</strong>
        </div>
        <div class="metric-card">
          <h4>Repos OK</h4>
          <strong>{{ metrics.repoOk ?? '¡X' }}</strong>
        </div>
      </div>
      <div class="log-box" style="margin-top: 16px">
        <pre>{{ metrics.raw || 'Metrics will appear once Prometheus is enabled and the backup is running.' }}</pre>
      </div>
    </section>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, reactive, ref } from 'vue'

const configText = ref('')
const validation = reactive({ valid: null, output: '' })
const status = reactive({ running: false })
const history = ref([])
const logs = ref([])
const metrics = reactive({
  sources: null,
  destinations: null,
  jobsStarted: null,
  jobsComplete: null,
  avgDuration: null,
  repoOk: null,
  raw: ''
})

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

let poller

const fetchJSON = async (url, options) => {
  const res = await fetch(url, options)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  return res.json()
}

const loadConfig = async () => {
  const res = await fetch('/api/config')
  if (res.ok) {
    configText.value = await res.text()
  }
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
  validation.output = data.output
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
}

const resetForm = () => {
  Object.assign(form, defaultForm())
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

const fetchLogs = async () => {
  const data = await fetchJSON('/api/logs?lines=200')
  logs.value = data.lines || []
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

