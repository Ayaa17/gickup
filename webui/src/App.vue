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
