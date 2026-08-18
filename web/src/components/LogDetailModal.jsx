import { useLanguage } from '../i18n'

const LEVEL_COLORS = {
  DEBUG: '#6c757d',
  INFO: '#28a745',
  WARN: '#ffc107',
  ERROR: '#dc3545',
  FATAL: '#6f42c1'
}

export default function LogDetailModal({ log, onClose }) {
  const { t, locale } = useLanguage()

  if (!log) return null

  const handleCopyJSON = () => {
    navigator.clipboard.writeText(JSON.stringify(log, null, 2)).catch(() => {})
  }

  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget) onClose()
  }

  return (
    <div className="modal-backdrop" onClick={handleBackdropClick} role="dialog" aria-modal="true">
      <div className="modal-content log-detail-modal">
        <div className="modal-header">
          <h3>{t('logDetail.title')}</h3>
          <button className="modal-close" onClick={onClose} aria-label={t('common.close')}>
            &times;
          </button>
        </div>
        <div className="modal-body">
          <div className="log-detail-header">
            <span
              className="log-detail-level"
              style={{ backgroundColor: LEVEL_COLORS[log.level] || '#6c757d' }}
            >
              {log.level}
            </span>
            <span className="log-detail-time">
              {new Date(log.timestamp).toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US')}
            </span>
          </div>
          <div className="log-detail-message">{log.message}</div>
          <div className="log-detail-meta">
            {log.source && (
              <div className="meta-item">
                <span className="meta-label">{t('logDetail.source')}</span>
                <span className="meta-value">{log.source}</span>
              </div>
            )}
            {log.host && (
              <div className="meta-item">
                <span className="meta-label">{t('logDetail.host')}</span>
                <span className="meta-value">{log.host}</span>
              </div>
            )}
            {log.service && (
              <div className="meta-item">
                <span className="meta-label">{t('logDetail.service')}</span>
                <span className="meta-value">{log.service}</span>
              </div>
            )}
            {log.id && (
              <div className="meta-item">
                <span className="meta-label">{t('logDetail.id')}</span>
                <span className="meta-value meta-value-mono">{log.id}</span>
              </div>
            )}
          </div>
          {log.fields && Object.keys(log.fields).length > 0 && (
            <div className="log-detail-fields">
              <h4>{t('logDetail.fields')}</h4>
              <div className="fields-grid">
                {Object.entries(log.fields).map(([key, value]) => (
                  <div key={key} className="field-item">
                    <span className="field-key">{key}</span>
                    <span className="field-value">{value}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          {log.raw && (
            <div className="log-detail-raw">
              <h4>{t('logDetail.raw')}</h4>
              <pre className="raw-content">{log.raw}</pre>
            </div>
          )}
          <div className="log-detail-json">
            <div className="json-header">
              <h4>{t('logDetail.json')}</h4>
              <button className="btn-copy" onClick={handleCopyJSON}>
                {t('common.copy')}
              </button>
            </div>
            <pre className="json-content">{JSON.stringify(log, null, 2)}</pre>
          </div>
        </div>
      </div>
    </div>
  )
}
