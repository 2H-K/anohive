import { useState, useMemo, useCallback, useRef } from 'react'
import VirtualList from './VirtualList'
import { SearchIcon, CopyIcon, DownloadIcon, InboxIcon } from './Icons'
import { useLanguage } from '../i18n'

const levelColors = {
  DEBUG: '#6c757d',
  INFO: '#28a745',
  WARN: '#ffc107',
  WARNING: '#ffc107',
  ERROR: '#dc3545',
  FATAL: '#6f42c1',
  CRITICAL: '#6f42c1',
}

function formatTime(ts, locale) {
  if (!ts) return ''
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US')
  } catch {
    return ts
  }
}

function LogRow({ log, onCopy }) {
  const { t } = useLanguage()

  const handleCopy = useCallback(() => {
    const text = JSON.stringify(log, null, 2)
    navigator.clipboard.writeText(text).catch(() => {})
    onCopy?.()
  }, [log, onCopy])

  return (
    <div className={`log-row level-${log.level?.toLowerCase()}`}>
      <span className="log-time">{formatTime(log.timestamp)}</span>
      <span className="log-level" style={{ backgroundColor: levelColors[log.level] || '#6c757d' }}>
        {log.level}
      </span>
      <span className="log-source">{log.source}</span>
      <span className="log-message" title={log.message}>{log.message}</span>
      <button
        className="log-copy-btn"
        onClick={handleCopy}
        aria-label={t('logStream.copy')}
        title={t('logStream.copy')}
      >
        <CopyIcon size={12} />
      </button>
    </div>
  )
}

function useDebounce(value, delay) {
  const [debouncedValue, setDebouncedValue] = useState(value)
  const timeoutRef = useRef(null)

  const setValue = useCallback((newValue) => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    timeoutRef.current = setTimeout(() => setDebouncedValue(newValue), delay)
  }, [delay])

  return [debouncedValue, setValue]
}

export default function LogStream({ logs, onCopyLog }) {
  const [levelFilter, setLevelFilter] = useState('')
  const [sourceFilter, setSourceFilter] = useState('')
  const [searchText, setSearchText] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useDebounce('', 200)
  const { t } = useLanguage()

  const handleSearchChange = useCallback((e) => {
    setSearchText(e.target.value)
    setDebouncedSearch(e.target.value)
  }, [setDebouncedSearch])

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      if (levelFilter && log.level !== levelFilter) return false
      if (sourceFilter && log.source !== sourceFilter) return false
      if (debouncedSearch && !log.message.toLowerCase().includes(debouncedSearch.toLowerCase())) return false
      return true
    })
  }, [logs, levelFilter, sourceFilter, debouncedSearch])

  const sources = useMemo(() => {
    const sourceSet = new Set(logs.map(l => l.source).filter(Boolean))
    return Array.from(sourceSet).sort()
  }, [logs])

  const exportLogs = useCallback(() => {
    const data = JSON.stringify(filteredLogs, null, 2)
    const blob = new Blob([data], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `pulse-logs-${new Date().toISOString().slice(0, 19)}.json`
    a.click()
    URL.revokeObjectURL(url)
  }, [filteredLogs])

  if (!logs || logs.length === 0) {
    return (
      <div className="empty-state">
        <span className="empty-state-icon"><InboxIcon /></span>
        <p>{t('logStream.empty.title')}</p>
        <p className="hint">{t('logStream.empty.hint')}</p>
      </div>
    )
  }

  return (
    <div className="log-stream">
      <div className="log-filters" role="search" aria-label={t('logStream.filters.label')}>
        <select
          value={levelFilter}
          onChange={(e) => setLevelFilter(e.target.value)}
          className="filter-select"
          aria-label={t('logStream.filters.levelLabel')}
        >
          <option value="">{t('logStream.filters.allLevels')}</option>
          <option value="DEBUG">DEBUG</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
          <option value="FATAL">FATAL</option>
        </select>

        <select
          value={sourceFilter}
          onChange={(e) => setSourceFilter(e.target.value)}
          className="filter-select"
          aria-label={t('logStream.filters.sourceLabel')}
        >
          <option value="">{t('logStream.filters.allSources')}</option>
          {sources.map(s => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>

        <div style={{ flex: 1, position: 'relative', minWidth: 200 }}>
          <input
            type="text"
            placeholder={t('logStream.filters.searchPlaceholder')}
            value={searchText}
            onChange={handleSearchChange}
            className="filter-search"
            aria-label={t('logStream.filters.searchLabel')}
            style={{ width: '100%', paddingLeft: 32 }}
          />
          <span style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: 'var(--text-tertiary)' }}>
            <SearchIcon size={14} />
          </span>
        </div>

        <span className="filter-count" aria-live="polite">
          {t('logStream.filters.resultCount', { filtered: filteredLogs.length, total: logs.length })}
        </span>

        <button
          className="filter-export-btn"
          onClick={exportLogs}
          aria-label={t('logStream.export.label')}
          title={t('logStream.export')}
        >
          <DownloadIcon size={14} />
        </button>
      </div>

      <div className="log-header">
        <span>{t('logStream.headers.timestamp')}</span>
        <span>{t('logStream.headers.level')}</span>
        <span>{t('logStream.headers.source')}</span>
        <span>{t('logStream.headers.message')}</span>
      </div>

      {filteredLogs.length === 0 ? (
        <div className="empty-state">
          <p>{t('logStream.noMatch')}</p>
        </div>
      ) : (
        <VirtualList
          items={filteredLogs}
          itemHeight={36}
          height={500}
          renderItem={(log) => <LogRow log={log} onCopy={onCopyLog} />}
        />
      )}
    </div>
  )
}
