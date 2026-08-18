import { InboxIcon } from './Icons'
import { useLanguage } from '../i18n'

export default function SourceList({ sourcesStats }) {
  const { t, locale } = useLanguage()

  if (!sourcesStats || sourcesStats.length === 0) {
    return (
      <div className="empty-state">
        <span className="empty-state-icon"><InboxIcon /></span>
        <p>{t('sourceList.empty.title')}</p>
        <p className="hint">{t('sourceList.empty.hint')}</p>
      </div>
    )
  }

  return (
    <div className="source-list" role="table" aria-label={t('sourceList.label')}>
      <div className="source-header" role="row">
        <span role="columnheader">{t('sourceList.headers.source')}</span>
        <span role="columnheader">{t('sourceList.headers.totalLogs')}</span>
        <span role="columnheader">{t('sourceList.headers.errors')}</span>
        <span role="columnheader">{t('sourceList.headers.warnings')}</span>
        <span role="columnheader">{t('sourceList.headers.logsPerMin')}</span>
        <span role="columnheader">{t('sourceList.headers.lastLog')}</span>
      </div>
      {sourcesStats.map((src) => (
        <div key={src.source} className="source-row" role="row">
          <span role="cell" className="source-name">{src.source}</span>
          <span role="cell">{src.total_logs}</span>
          <span role="cell" className="error-count">{src.error_count}</span>
          <span role="cell" className="warn-count">{src.warn_count}</span>
          <span role="cell">{src.logs_per_min?.toFixed(1) || '0'}</span>
          <span role="cell">{src.last_log_time ? new Date(src.last_log_time).toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US') : t('sourceList.notAvailable')}</span>
        </div>
      ))}
    </div>
  )
}
