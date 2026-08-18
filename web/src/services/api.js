const BASE_URL = import.meta.env.VITE_API_URL || ''

async function request(path, options = {}) {
  const url = `${BASE_URL}/api${path}`
  const response = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...options
  })

  if (!response.ok) {
    throw new Error(`API error: ${response.status}`)
  }

  return response.json()
}

export function fetchStats() {
  return request('/stats')
}

export function fetchLogs(query = {}) {
  const params = new URLSearchParams()
  if (query.source) params.set('source', query.source)
  if (query.level) params.set('level', query.level)
  if (query.search) params.set('search', query.search)
  if (query.limit) params.set('limit', query.limit)
  if (query.offset) params.set('offset', query.offset)
  const qs = params.toString()
  return request(`/logs${qs ? '?' + qs : ''}`)
}

export function fetchAnomalies(query = {}) {
  const params = new URLSearchParams()
  if (query.limit) params.set('limit', query.limit)
  if (query.severity) params.set('severity', query.severity)
  const qs = params.toString()
  return request(`/anomalies${qs ? '?' + qs : ''}`)
}

export function ingestLog(entry) {
  return request('/logs/ingest', {
    method: 'POST',
    body: JSON.stringify(entry)
  })
}

export function fetchSources() {
  return request('/sources')
}

export function fetchTrends(hours = 24, source = '') {
  const params = new URLSearchParams()
  params.set('hours', hours)
  if (source) params.set('source', source)
  return request(`/stats/trends?${params}`)
}

export function fetchLevelDistribution(source = '') {
  const params = new URLSearchParams()
  if (source) params.set('source', source)
  return request(`/stats/levels?${params}`)
}

export function fetchAnomalyTimeline(hours = 24, source = '') {
  const params = new URLSearchParams()
  params.set('hours', hours)
  if (source) params.set('source', source)
  return request(`/stats/anomalies/timeline?${params}`)
}

export function getExportUrl(query = {}) {
  const params = new URLSearchParams()
  if (query.source) params.set('source', query.source)
  if (query.level) params.set('level', query.level)
  if (query.search) params.set('search', query.search)
  if (query.start) params.set('start', query.start)
  if (query.end) params.set('end', query.end)
  return `${BASE_URL}/api/v1/logs/export?${params}`
}
