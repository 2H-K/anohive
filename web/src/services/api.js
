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
