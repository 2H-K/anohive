import { useState, useEffect, useCallback, useRef } from 'react'
import LogStream from './components/LogStream'
import StatsPanel from './components/StatsPanel'
import AnomalyList from './components/AnomalyList'
import SourceList from './components/SourceList'
import LanguageToggle from './components/LanguageToggle'
import { useWebSocket } from './hooks/useWebSocket'
import { useTheme } from './hooks/useTheme'
import { useLanguage } from './i18n'
import { ToastProvider, useToast } from './components/Toast'
import { fetchStats, fetchLogs, fetchAnomalies } from './services/api.js'
import { PulseIcon, SunIcon, MoonIcon, ListIcon, ShieldIcon, GridIcon } from './components/Icons'

function ThemeToggle() {
  const { theme, toggleTheme } = useTheme()
  const { t } = useLanguage()
  return (
    <button
      className="theme-toggle"
      onClick={toggleTheme}
      aria-label={theme === 'dark' ? t('app.theme.toLight') : t('app.theme.toDark')}
      title={theme === 'dark' ? t('app.theme.toLight') : t('app.theme.toDark')}
    >
      {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
    </button>
  )
}

function AppContent() {
  const [stats, setStats] = useState(null)
  const [logs, setLogs] = useState([])
  const [anomalies, setAnomalies] = useState([])
  const [activeTab, setActiveTab] = useState('logs')
  const [wsConnected, setWsConnected] = useState(false)
  const [statsLoading, setStatsLoading] = useState(true)
  const { addToast } = useToast()
  const { t } = useLanguage()
  const reconnectNotified = useRef(false)

  const loadInitialData = useCallback(async () => {
    try {
      const [statsData, logsData, anomaliesData] = await Promise.all([
        fetchStats(),
        fetchLogs({ limit: 200 }),
        fetchAnomalies({ limit: 50 })
      ])
      setStats(statsData)
      setLogs(logsData.logs || [])
      setAnomalies(anomaliesData.anomalies || [])
    } catch (err) {
      addToast(t('toast.loadFailed'), 'error')
    } finally {
      setStatsLoading(false)
    }
  }, [addToast, t])

  useEffect(() => {
    loadInitialData()
  }, [loadInitialData])

  const handleWebSocketMessage = useCallback((data) => {
    if (data && data.id) {
      if (data.level) {
        setLogs(prev => [data, ...prev].slice(0, 500))
      } else if (data.type) {
        setAnomalies(prev => [data, ...prev].slice(0, 100))
        addToast(t('toast.newAnomaly', { type: data.type }), 'warning', 5000)
      }
    }
  }, [addToast, t])

  const handleWebSocketOpen = useCallback(() => {
    setWsConnected(true)
    if (reconnectNotified.current) {
      addToast(t('app.connection.restored'), 'success', 3000)
      reconnectNotified.current = false
    }
  }, [addToast, t])

  const handleWebSocketClose = useCallback(() => {
    setWsConnected(false)
    reconnectNotified.current = true
  }, [])

  const handleWebSocketError = useCallback(() => {
    addToast(t('app.connection.error'), 'error', 5000)
  }, [addToast, t])

  useWebSocket(`ws://${window.location.host}/ws`, {
    onMessage: handleWebSocketMessage,
    onOpen: handleWebSocketOpen,
    onClose: handleWebSocketClose,
    onError: handleWebSocketError,
    reconnectInterval: 3000,
    maxReconnectAttempts: 10
  })

  const handleCopyLog = useCallback(() => {
    addToast(t('toast.logCopied'), 'success', 2000)
  }, [addToast, t])

  const tabs = [
    { id: 'logs', label: t('app.tabs.logs'), icon: ListIcon },
    { id: 'anomalies', label: t('app.tabs.anomalies'), icon: ShieldIcon },
    { id: 'sources', label: t('app.tabs.sources'), icon: GridIcon },
  ]

  return (
    <div className="app">
      <a href="#main-content" className="skip-link">{t('app.skipToContent')}</a>

      <header className="header">
        <div className="header-left">
          <div className="brand">
            <div className="brand-logo" aria-hidden="true">
              <PulseIcon size={20} />
            </div>
            <h1 className="brand-name"><span>{t('app.brand')}</span></h1>
          </div>
        </div>
        <div className="header-right">
          <div className="connection-status" role="status" aria-live="polite">
            <span className={`status-dot ${wsConnected ? 'online' : 'offline'}`} aria-hidden="true" />
            <span>{wsConnected ? t('app.connection.live') : t('app.connection.disconnected')}</span>
          </div>
          <LanguageToggle />
          <ThemeToggle />
        </div>
      </header>

      <StatsPanel stats={stats} loading={statsLoading} />

      <nav className="tabs" role="tablist" aria-label={t('app.mainNavigation')}>
        {tabs.map(tab => (
          <button
            key={tab.id}
            className={`tab-btn ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
            role="tab"
            aria-selected={activeTab === tab.id}
            aria-controls={`panel-${tab.id}`}
            id={`tab-${tab.id}`}
          >
            <tab.icon size={16} aria-hidden="true" />
            {tab.label}
            {tab.id === 'logs' && logs.length > 0 && (
              <span className="badge" aria-label={`${logs.length} ${t('app.tabs.logs')}`}>{logs.length}</span>
            )}
            {tab.id === 'anomalies' && anomalies.length > 0 && (
              <span className="badge badge-warn" aria-label={`${anomalies.length} ${t('app.tabs.anomalies')}`}>{anomalies.length}</span>
            )}
          </button>
        ))}
      </nav>

      <main id="main-content" className="content" role="tabpanel" aria-labelledby={`tab-${activeTab}`}>
        {activeTab === 'logs' && <LogStream logs={logs} onCopyLog={handleCopyLog} />}
        {activeTab === 'anomalies' && <AnomalyList anomalies={anomalies} />}
        {activeTab === 'sources' && <SourceList sourcesStats={stats?.sources_stats || []} />}
      </main>
    </div>
  )
}

export default function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  )
}
