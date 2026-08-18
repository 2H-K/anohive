import { useState, useEffect, useCallback } from 'react'
import { fetchTrends } from '../services/api'
import { useLanguage } from '../i18n'

const LEVEL_COLORS = {
  DEBUG: '#6c757d',
  INFO: '#28a745',
  WARN: '#ffc107',
  ERROR: '#dc3545',
  FATAL: '#6f42c1'
}

export default function TrendChart({ source = '', height = 200 }) {
  const [trends, setTrends] = useState([])
  const [loading, setLoading] = useState(true)
  const [hours, setHours] = useState(24)
  const { t } = useLanguage()

  const loadTrends = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchTrends(hours, source)
      setTrends(data.trends || [])
    } catch {
      setTrends([])
    } finally {
      setLoading(false)
    }
  }, [hours, source])

  useEffect(() => {
    loadTrends()
  }, [loadTrends])

  const maxCount = Math.max(...trends.map(t => t.count), 1)

  return (
    <div className="chart-card">
      <div className="chart-header">
        <h3>{t('charts.logTrend')}</h3>
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
      ) : trends.length === 0 ? (
        <div className="chart-empty">{t('charts.noData')}</div>
      ) : (
        <svg
          viewBox={`0 0 ${trends.length * 40 + 40} ${height + 30}`}
          className="trend-chart"
          role="img"
          aria-label={t('charts.logTrend')}
        >
          {trends.map((point, i) => {
            const barHeight = (point.count / maxCount) * height
            const x = 40 + i * 40
            const y = height - barHeight + 10
            return (
              <g key={i}>
                <rect
                  x={x}
                  y={y}
                  width={30}
                  height={barHeight}
                  fill={LEVEL_COLORS.INFO}
                  rx={2}
                  opacity={0.85}
                >
                  <title>{`${point.time}: ${point.count} ${t('charts.logs')}`}</title>
                </rect>
                {trends.length <= 12 && (
                  <text
                    x={x + 15}
                    y={height + 24}
                    textAnchor="middle"
                    fontSize={10}
                    fill="var(--text-tertiary)"
                  >
                    {point.time.split(' ')[1]?.slice(0, 5) || ''}
                  </text>
                )}
              </g>
            )
          })}
          <line
            x1={40}
            y1={height + 10}
            x2={trends.length * 40 + 40}
            y2={height + 10}
            stroke="var(--border)"
            strokeWidth={1}
          />
        </svg>
      )}
    </div>
  )
}
