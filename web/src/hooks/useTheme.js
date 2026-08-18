import { useState, useEffect, useCallback } from 'react'

const STORAGE_KEY = 'anohive-theme'

function getSystemTheme() {
  if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: light)').matches) {
    return 'light'
  }
  return 'dark'
}

function getStoredTheme() {
  try {
    return localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

export function useTheme() {
  const [theme, setTheme] = useState(() => getStoredTheme() || getSystemTheme())

  useEffect(() => {
    const root = document.documentElement
    // Always set data-theme explicitly so it overrides the media query
    root.setAttribute('data-theme', theme)
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      // ignore storage errors
    }
  }, [theme])

  const toggleTheme = useCallback(() => {
    setTheme(prev => prev === 'dark' ? 'light' : 'dark')
  }, [])

  return { theme, toggleTheme, setTheme }
}
