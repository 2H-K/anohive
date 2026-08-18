export function StatSkeleton() {
  return (
    <div className="stat-card">
      <div className="skeleton" style={{ width: 44, height: 44, borderRadius: 'var(--radius-md)' }} />
      <div className="stat-info" style={{ flex: 1 }}>
        <div className="skeleton skeleton-text" style={{ width: '60%', height: 24, marginBottom: 6 }} />
        <div className="skeleton skeleton-text" style={{ width: '40%', height: 12 }} />
      </div>
    </div>
  )
}

export function LogRowSkeleton() {
  return (
    <div className="log-row">
      <div className="skeleton skeleton-text" style={{ width: 70, height: 12 }} />
      <div className="skeleton skeleton-text" style={{ width: 50, height: 12 }} />
      <div className="skeleton skeleton-text" style={{ width: 80, height: 12 }} />
      <div className="skeleton skeleton-text" style={{ width: '80%', height: 12 }} />
    </div>
  )
}

export function AnomalyCardSkeleton() {
  return (
    <div className="anomaly-card">
      <div style={{ display: 'flex', gap: 12, marginBottom: 8 }}>
        <div className="skeleton skeleton-text" style={{ width: 60, height: 16 }} />
        <div className="skeleton skeleton-text" style={{ width: 100, height: 16 }} />
      </div>
      <div className="skeleton skeleton-text" style={{ width: '90%', height: 14, marginBottom: 4 }} />
      <div className="skeleton skeleton-text" style={{ width: '60%', height: 14 }} />
    </div>
  )
}
