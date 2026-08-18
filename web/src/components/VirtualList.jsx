import { useState, useRef, useEffect, useCallback } from 'react'
import { useLanguage } from '../i18n'

export default function VirtualList({ items, itemHeight = 40, height = 600, renderItem, overscan = 5, ariaLabel }) {
  const [scrollTop, setScrollTop] = useState(0)
  const containerRef = useRef(null)
  const { t } = useLanguage()

  const totalHeight = items.length * itemHeight
  const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan)
  const visibleCount = Math.ceil(height / itemHeight) + 2 * overscan
  const endIndex = Math.min(items.length, startIndex + visibleCount)
  const offsetY = startIndex * itemHeight

  const handleScroll = useCallback((e) => {
    setScrollTop(e.target.scrollTop)
  }, [])

  useEffect(() => {
    const container = containerRef.current
    if (container) {
      container.addEventListener('scroll', handleScroll, { passive: true })
      return () => container.removeEventListener('scroll', handleScroll)
    }
  }, [handleScroll])

  const visibleItems = items.slice(startIndex, endIndex)

  const handleKeyDown = useCallback((e) => {
    const container = containerRef.current
    if (!container) return

    const scrollAmount = itemHeight * 5
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      container.scrollTop += scrollAmount
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      container.scrollTop -= scrollAmount
    } else if (e.key === 'Home') {
      e.preventDefault()
      container.scrollTop = 0
    } else if (e.key === 'End') {
      e.preventDefault()
      container.scrollTop = container.scrollHeight
    }
  }, [itemHeight])

  return (
    <div
      ref={containerRef}
      className="virtual-list-container"
      style={{ height, overflow: 'auto', position: 'relative' }}
      tabIndex={0}
      role="list"
      aria-label={ariaLabel || t('logStream.virtualList.label')}
      onKeyDown={handleKeyDown}
    >
      <div style={{ height: totalHeight, position: 'relative' }}>
        <div style={{ transform: `translateY(${offsetY}px)` }}>
          {visibleItems.map((item, index) => (
            <div key={startIndex + index} style={{ height: itemHeight }} role="listitem">
              {renderItem(item, startIndex + index)}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
