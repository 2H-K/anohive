import { useState, useEffect, useCallback } from 'react'
import { fetchLevelDistribution } from '../services/api'
import { useLanguage } from '../i18n'

const LEVEL_COLORS = {
  DEBUG: '#6c757d',
  INFO: '#28a745',
  WARN: '#ffc107',
  ERROR: '#dc3545',
  FATAL: '#6f42c1'
}

export default function LevelDistribution({ source = '' }) {
  const [distribution, setDistribution] = useState({})
  const [loading, setLoading] = useState(true)
  const { t } = useLanguage()

  const loadDistribution = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchLevelDistribution(source)
      setDistribution(data.distribution || {})
    } catch {
      setDistribution({})
    } finally {
      setLoading(false)
    }
  }, [source])

  useEffect(() => {
    loadDistribution()
  }, [loadDistribution])

  const entries = Object.entries(distribution).filter(([, count]) => count > 0)
  const total = entries.reduce((sum, [, count]) => sum + count, 0)

  const radius = 70
  const cx = 100
  const cy = 100

  const polarToCartesian = (angle) => ({
    x: cx + radius * Math.cos((angle - 90) * Math.PI / 180),
    y: cy + radius * Math.sin((angle - 90) * Math.PI / 180)
  })

  const createArcPath = (startAngle, endAngle) => {
    const start = polarToCartesian(endAngle)
    const end = polarToCartesian(startAngle)
    const largeArc = endAngle - startAngle > 180 ? 1 : 0
    return `M ${cx} ${cy} L ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArc} 0 ${end.x} ${end.y} Z`
  }

  let currentAngle = 0
  const slices = entries.map(([level, count]) => {
    const angle = (count / total) * 360
    const slice = {
      level,
      count,
      path: createArcPath(currentAngle, currentAngle + angle),
      color: LEVEL_COLORS[level] || '#999'
    }
    currentAngle += angle
    return slice
  })

  return (
    <div className="chart-card">
      <div className="chart-header">
        <h3>{t('charts.levelDistribution')}</h3>
      </div>
      {loading ? (
        <div className="chart-loading">
          <div className="skeleton" style={{ height: 200 }} />
        </div>
      ) : entries.length === 0 ? (
        <div className="chart-empty">{t('charts.noData')}</div>
      ) : (
        <div className="pie-chart-container">
          <svg viewBox="0 0 200 200" className="pie-chart" role="img" aria-label={t('charts.levelDistribution')}>
            {slices.map((slice) => (
              <path
                key={slice.level}
                d={slice.path}
                fill={slice.color}
                opacity={0.85}
                stroke="var(--bg-card)"
                strokeWidth={1}
              >
                <title>{`${slice.level}: ${slice.count} (${((slice.count / total) * 100).toFixed(1)}%)`}</title>
              </path>
            ))}
          </svg>
          <div className="pie-legend">
            {slices.map((slice) => (
              <div key={slice.level} className="legend-item">
                <span className="legend-color" style={{ backgroundColor: slice.color }} />
                <span className="legend-label">{slice.level}</span>
                <span className="legend-count">{slice.count}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
