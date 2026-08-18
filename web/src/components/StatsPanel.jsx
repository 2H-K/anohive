import { DocumentIcon, RadioIcon, AlertIcon, ClockIcon } from './Icons'
import { StatSkeleton } from './Skeleton'
import { useLanguage } from '../i18n'

function formatNumber(n, locale) {
  if (n === undefined || n === null) return null
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

function formatUptime(uptime, locale) {
  if (!uptime) return null
  if (typeof uptime === 'string') return uptime
  const seconds = Math.floor(uptime / 1000000000)
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 24) return `${Math.floor(hours / 24)}d ${hours % 24}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

export default function StatsPanel({ stats, loading }) {
  const { t } = useLanguage()

  if (loading && !stats) {
    return (
      <div className="stats-panel">
        {Array.from({ length: 4 }).map((_, i) => (
          <StatSkeleton key={i} />
        ))}
      </div>
    )
  }

  const cards = [
    { labelKey: 'stats.totalLogs', value: formatNumber(stats?.total_logs), icon: DocumentIcon, className: 'logs' },
    { labelKey: 'stats.sources', value: stats?.total_sources ?? 0, icon: RadioIcon, className: 'sources' },
    { labelKey: 'stats.anomalies', value: formatNumber(stats?.total_anomalies), icon: AlertIcon, className: 'anomalies' },
    { labelKey: 'stats.uptime', value: formatUptime(stats?.uptime), icon: ClockIcon, className: 'uptime' },
  ]

  return (
    <div className="stats-panel" role="region" aria-label={t('stats.label')}>
      {cards.map((card) => (
        <div key={card.labelKey} className="stat-card">
          <span className={`stat-icon ${card.className}`} aria-hidden="true">
            <card.icon size={20} />
          </span>
          <div className="stat-info">
            <span className="stat-value">{card.value ?? t('stats.noData')}</span>
            <span className="stat-label">{t(card.labelKey)}</span>
          </div>
        </div>
      ))}
    </div>
  )
}
