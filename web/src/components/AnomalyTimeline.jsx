import { useState, useEffect, useCallback } from 'react'
import { fetchAnomalyTimeline } from '../services/api'
import { useLanguage } from '../i18n'

const SEVERITY_COLORS = {
  LOW: '#17a2b8',
  MEDIUM: '#ffc107',
  HIGH: '#fd7e14',
  CRITICAL: '#dc3545'
}

export default function AnomalyTimeline({ source = '', height = 150 }) {
  const [timeline, setTimeline] = useState([])
  const [loading, setLoading] = useState(true)
  const [hours, setHours] = useState(24)
  const { t } = useLanguage()

  const loadTimeline = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchAnomalyTimeline(hours, source)
      setTimeline(data.timeline || [])
    } catch {
      setTimeline([])
    } finally {
      setLoading(false)
    }
  }, [hours, source])

  useEffect(() => {
    loadTimeline()
  }, [loadTimeline])

  const maxCount = Math.max(...timeline.map(t => t.count), 1)

  return (
    <div className="chart-card">
      <div className="chart-header">
        <h3>{t('charts.anomalyTimeline')}</h3>
        <div className="chart-controls">
          <select
            value={hours}
            onChange={e => setHours(Number(e.target.value))}
            className="chart-select"
            aria-label={t('charts.timeRange')}
          >
            <option value={1}>{t('charts.last1h')}</option>
            <option value={6}>{t('charts.last6h')}</option>
            <option value={24}>{t('charts.last24h')}</option>
            <option value={72}>{t('charts.last3d')}</option>
          </select>
        </div>
      </div>
      {loading ? (
        <div className="chart-loading">
          <div className="skeleton" style={{ height }} />
        </div>
      ) : timeline.length === 0 ? (
        <div className="chart-empty">{t('charts.noAnomalies')}</div>
      ) : (
        <div className="timeline-container">
          <svg
            viewBox={`0 0 ${timeline.length * 30 + 40} ${height + 20}`}
            className="timeline-chart"
            role="img"
            aria-label={t('charts.anomalyTimeline')}
          >
            {timeline.map((point, i) => {
              const barHeight = (point.count / maxCount) * height
              const x = 40 + i * 30
              const y = height - barHeight + 5
              return (
                <g key={i}>
                  <rect
                    x={x}
                    y={y}
                    width={20}
                    height={barHeight}
                    fill={SEVERITY_COLORS[point.severity] || '#999'}
                    rx={2}
                    opacity={0.85}
                  >
                    <title>{`${point.time} [${point.severity}]: ${point.count}`}</title>
                  </rect>
                </g>
              )
            })}
            <line
              x1={40}
              y1={height + 5}
              x2={timeline.length * 30 + 40}
              y2={height + 5}
              stroke="var(--border)"
              strokeWidth={1}
            />
          </svg>
          <div className="timeline-legend">
            {Object.entries(SEVERITY_COLORS).map(([severity, color]) => (
              <div key={severity} className="legend-item">
                <span className="legend-color" style={{ backgroundColor: color }} />
                <span className="legend-label">{t(`anomalyList.severity.${severity.toLowerCase()}`)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
