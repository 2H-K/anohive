import { CheckCircleIcon } from './Icons'
import { useLanguage } from '../i18n'

const severityColors = {
  LOW: '#17a2b8',
  MEDIUM: '#ffc107',
  HIGH: '#fd7e14',
  CRITICAL: '#dc3545',
}

function formatTime(ts, locale) {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US')
  } catch {
    return ts
  }
}

export default function AnomalyList({ anomalies }) {
  const { t, locale } = useLanguage()

  const getSeverityLabel = (severity) => {
    const key = `anomalyList.severity.${severity?.toLowerCase()}`
    return t(key)
  }

  if (!anomalies || anomalies.length === 0) {
    return (
      <div className="empty-state">
        <span className="empty-state-icon"><CheckCircleIcon /></span>
        <p>{t('anomalyList.empty.title')}</p>
        <p className="hint">{t('anomalyList.empty.hint')}</p>
      </div>
    )
  }

  return (
    <div className="anomaly-list" role="list" aria-label={t('anomalyList.label')}>
      {anomalies.map((anomaly) => {
        const color = severityColors[anomaly.severity] || severityColors.LOW
        return (
          <div
            key={anomaly.id}
            className={`anomaly-card severity-${anomaly.severity?.toLowerCase()}`}
            role="listitem"
          >
            <div className="anomaly-header">
              <span
                className={`anomaly-severity ${anomaly.severity?.toLowerCase()}`}
                style={{ backgroundColor: color }}
              >
                {getSeverityLabel(anomaly.severity)}
              </span>
              <span className="anomaly-type">{anomaly.type}</span>
              <span className="anomaly-time">{formatTime(anomaly.timestamp, locale)}</span>
            </div>
            <p className="anomaly-desc">{anomaly.description}</p>
            {anomaly.source && (
              <span className="anomaly-source">{t('anomalyList.source', { source: anomaly.source })}</span>
            )}
          </div>
        )
      })}
    </div>
  )
}
