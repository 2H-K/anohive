import { useState, useEffect, useCallback, createContext, useContext } from 'react'
import { CheckCircleIcon, AlertIcon, DocumentIcon, InboxIcon } from './Icons'

const ToastContext = createContext(null)

export function useToast() {
  return useContext(ToastContext)
}

let toastId = 0

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])

  const addToast = useCallback((message, type = 'info', duration = 4000) => {
    const id = ++toastId
    setToasts(prev => [...prev, { id, message, type }])
    if (duration > 0) {
      setTimeout(() => removeToast(id), duration)
    }
    return id
  }, [])

  const removeToast = useCallback((id) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ addToast, removeToast }}>
      {children}
      <div className="toast-container" role="region" aria-label="Notifications" aria-live="polite">
        {toasts.map(toast => (
          <ToastItem key={toast.id} toast={toast} onRemove={removeToast} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

function ToastItem({ toast, onRemove }) {
  const [removing, setRemoving] = useState(false)

  const handleRemove = () => {
    setRemoving(true)
    setTimeout(() => onRemove(toast.id), 200)
  }

  const Icon = {
    success: CheckCircleIcon,
    error: AlertIcon,
    warning: AlertIcon,
    info: DocumentIcon,
  }[toast.type] || DocumentIcon

  return (
    <div className={`toast toast-${toast.type} ${removing ? 'removing' : ''}`} role="alert">
      <span className="toast-icon"><Icon size={18} /></span>
      <span>{toast.message}</span>
    </div>
  )
}
