import { createContext, useContext, useState, useCallback, useEffect } from 'react';

const translations = {
  en: {
    // App
    'app.brand': 'AnoHive',
    'app.skipToContent': 'Skip to main content',
    'app.mainNavigation': 'Main navigation',
    'app.connection.live': 'Live',
    'app.connection.disconnected': 'Disconnected',
    'app.connection.restored': 'Connection restored',
    'app.connection.error': 'WebSocket connection error',
    'app.tabs.logs': 'Log Stream',
    'app.tabs.analytics': 'Analytics',
    'app.tabs.anomalies': 'Anomalies',
    'app.tabs.sources': 'Sources',
    'app.theme.toLight': 'Switch to light mode',
    'app.theme.toDark': 'Switch to dark mode',
    'app.language.switch': 'Switch to Chinese',

    // Toast
    'toast.loadFailed': 'Failed to load initial data',
    'toast.newAnomaly': 'New anomaly: {type}',
    'toast.logCopied': 'Log copied to clipboard',

    // Log Stream
    'logStream.empty.title': 'No log entries yet',
    'logStream.empty.hint': 'Send logs to POST /api/logs/ingest',
    'logStream.noMatch': 'No logs match the current filters',
    'logStream.headers.timestamp': 'Timestamp',
    'logStream.headers.level': 'Level',
    'logStream.headers.source': 'Source',
    'logStream.headers.message': 'Message',
    'logStream.filters.allLevels': 'All Levels',
    'logStream.filters.allSources': 'All Sources',
    'logStream.filters.searchPlaceholder': 'Search messages...',
    'logStream.filters.resultCount': '{filtered} of {total}',
    'logStream.filters.label': 'Log filters',
    'logStream.filters.levelLabel': 'Filter by level',
    'logStream.filters.sourceLabel': 'Filter by source',
    'logStream.filters.searchLabel': 'Search log messages',
    'logStream.export.label': 'Export logs as JSON',
    'logStream.copy': 'Copy',
    'logStream.export': 'Export',
    'logStream.virtualList.label': 'Log entries',

    // Source List
    'sourceList.empty.title': 'No sources registered',
    'sourceList.empty.hint': 'Send logs to start seeing sources',
    'sourceList.headers.source': 'Source',
    'sourceList.headers.totalLogs': 'Total Logs',
    'sourceList.headers.errors': 'Errors',
    'sourceList.headers.warnings': 'Warnings',
    'sourceList.headers.logsPerMin': 'Logs/min',
    'sourceList.headers.lastLog': 'Last Log',
    'sourceList.notAvailable': 'N/A',
    'sourceList.label': 'Log sources',

    // Stats Panel
    'stats.totalLogs': 'Total Logs',
    'stats.sources': 'Sources',
    'stats.anomalies': 'Anomalies',
    'stats.uptime': 'Uptime',
    'stats.noData': '—',
    'stats.label': 'Statistics overview',

    // Anomaly List
    'anomalyList.empty.title': 'No anomalies detected',
    'anomalyList.empty.hint': 'All systems operating normally',
    'anomalyList.severity.low': 'Low',
    'anomalyList.severity.medium': 'Medium',
    'anomalyList.severity.high': 'High',
    'anomalyList.severity.critical': 'Critical',
    'anomalyList.source': 'Source: {source}',
    'anomalyList.label': 'Detected anomalies',

    // Common
    'common.copy': 'Copy',
    'common.export': 'Export',
    'common.error': 'Error',
    'common.warning': 'Warning',
    'common.info': 'Info',
    'common.debug': 'Debug',
    'common.fatal': 'Fatal',
    'common.loading': 'Loading...',
    'common.retry': 'Retry',
    'common.close': 'Close',

    // API
    'api.error': 'API error: {status}',
    'api.wsError': 'WebSocket error',

    // Charts
    'charts.logTrend': 'Log Trend',
    'charts.levelDistribution': 'Level Distribution',
    'charts.anomalyTimeline': 'Anomaly Timeline',
    'charts.timeRange': 'Time range',
    'charts.last1h': 'Last 1 hour',
    'charts.last6h': 'Last 6 hours',
    'charts.last24h': 'Last 24 hours',
    'charts.last3d': 'Last 3 days',
    'charts.logs': 'logs',
    'charts.noData': 'No trend data available',
    'charts.noAnomalies': 'No anomalies in period',

    // Log Detail
    'logDetail.title': 'Log Detail',
    'logDetail.source': 'Source',
    'logDetail.host': 'Host',
    'logDetail.service': 'Service',
    'logDetail.id': 'ID',
    'logDetail.fields': 'Fields',
    'logDetail.raw': 'Raw Log',
    'logDetail.json': 'Full JSON',
  },
  zh: {
    // App
    'app.brand': 'AnoHive',
    'app.skipToContent': '跳转到主内容',
    'app.mainNavigation': '主导航',
    'app.connection.live': '在线',
    'app.connection.disconnected': '已断开',
    'app.connection.restored': '连接已恢复',
    'app.connection.error': 'WebSocket 连接错误',
    'app.tabs.logs': '日志流',
    'app.tabs.analytics': '分析',
    'app.tabs.anomalies': '异常',
    'app.tabs.sources': '数据源',
    'app.theme.toLight': '切换到浅色模式',
    'app.theme.toDark': '切换到深色模式',
    'app.language.switch': '切换到英文',

    // Toast
    'toast.loadFailed': '加载初始数据失败',
    'toast.newAnomaly': '新异常: {type}',
    'toast.logCopied': '日志已复制到剪贴板',

    // Log Stream
    'logStream.empty.title': '暂无日志条目',
    'logStream.empty.hint': '发送日志到 POST /api/logs/ingest',
    'logStream.noMatch': '没有符合当前过滤条件的日志',
    'logStream.headers.timestamp': '时间戳',
    'logStream.headers.level': '级别',
    'logStream.headers.source': '来源',
    'logStream.headers.message': '消息',
    'logStream.filters.allLevels': '所有级别',
    'logStream.filters.allSources': '所有来源',
    'logStream.filters.searchPlaceholder': '搜索消息...',
    'logStream.filters.resultCount': '{filtered} / {total}',
    'logStream.filters.label': '日志过滤',
    'logStream.filters.levelLabel': '按级别过滤',
    'logStream.filters.sourceLabel': '按来源过滤',
    'logStream.filters.searchLabel': '搜索日志消息',
    'logStream.export.label': '导出日志为 JSON',
    'logStream.copy': '复制',
    'logStream.export': '导出',
    'logStream.virtualList.label': '日志条目',

    // Source List
    'sourceList.empty.title': '暂无数据源',
    'sourceList.empty.hint': '发送日志后开始查看数据源',
    'sourceList.headers.source': '来源',
    'sourceList.headers.totalLogs': '总日志数',
    'sourceList.headers.errors': '错误数',
    'sourceList.headers.warnings': '警告数',
    'sourceList.headers.logsPerMin': '日志/分钟',
    'sourceList.headers.lastLog': '最后日志',
    'sourceList.notAvailable': '无',
    'sourceList.label': '日志来源',

    // Stats Panel
    'stats.totalLogs': '总日志数',
    'stats.sources': '数据源数',
    'stats.anomalies': '异常数',
    'stats.uptime': '运行时间',
    'stats.noData': '—',
    'stats.label': '统计概览',

    // Anomaly List
    'anomalyList.empty.title': '未检测到异常',
    'anomalyList.empty.hint': '所有系统运行正常',
    'anomalyList.severity.low': '低',
    'anomalyList.severity.medium': '中',
    'anomalyList.severity.high': '高',
    'anomalyList.severity.critical': '严重',
    'anomalyList.source': '来源: {source}',
    'anomalyList.label': '检测到的异常',

    // Common
    'common.copy': '复制',
    'common.export': '导出',
    'common.error': '错误',
    'common.warning': '警告',
    'common.info': '信息',
    'common.debug': '调试',
    'common.fatal': '致命',
    'common.loading': '加载中...',
    'common.retry': '重试',
    'common.close': '关闭',

    // API
    'api.error': 'API 错误: {status}',
    'api.wsError': 'WebSocket 错误',

    // Charts
    'charts.logTrend': '日志趋势',
    'charts.levelDistribution': '级别分布',
    'charts.anomalyTimeline': '异常时间线',
    'charts.timeRange': '时间范围',
    'charts.last1h': '最近 1 小时',
    'charts.last6h': '最近 6 小时',
    'charts.last24h': '最近 24 小时',
    'charts.last3d': '最近 3 天',
    'charts.logs': '条日志',
    'charts.noData': '暂无趋势数据',
    'charts.noAnomalies': '该时段无异常',

    // Log Detail
    'logDetail.title': '日志详情',
    'logDetail.source': '来源',
    'logDetail.host': '主机',
    'logDetail.service': '服务',
    'logDetail.id': 'ID',
    'logDetail.fields': '扩展字段',
    'logDetail.raw': '原始日志',
    'logDetail.json': '完整 JSON',
  },
};

const STORAGE_KEY = 'anohive_language';

const LanguageContext = createContext(null);

export function LanguageProvider({ children }) {
  const [locale, setLocale] = useState(() => {
    // Try localStorage first
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && translations[stored]) return stored;
    // Try browser language
    const browserLang = navigator.language || navigator.userLanguage;
    if (browserLang && browserLang.startsWith('zh')) return 'zh';
    return 'en';
  });

  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    setIsLoaded(true);
  }, []);

  const t = useCallback((key, params = {}) => {
    const template = translations[locale]?.[key] || translations['en']?.[key] || key;

    // Replace placeholders like {name}, {type}, etc.
    return template.replace(/\{(\w+)\}/g, (match, paramKey) => {
      return params[paramKey] !== undefined ? params[paramKey] : match;
    });
  }, [locale]);

  const switchLanguage = useCallback(() => {
    const newLocale = locale === 'en' ? 'zh' : 'en';
    setLocale(newLocale);
    localStorage.setItem(STORAGE_KEY, newLocale);
  }, [locale]);

  const setLanguage = useCallback((newLocale) => {
    if (translations[newLocale]) {
      setLocale(newLocale);
      localStorage.setItem(STORAGE_KEY, newLocale);
    }
  }, []);

  const value = {
    locale,
    t,
    switchLanguage,
    setLanguage,
    isLoaded,
  };

  return (
    <LanguageContext.Provider value={value}>
      {children}
    </LanguageContext.Provider>
  );
}

export function useLanguage() {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return context;
}

export default LanguageContext;
