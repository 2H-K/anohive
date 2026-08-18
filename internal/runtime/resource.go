package runtime

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceMonitor monitors system resources and provides degradation signals
type ResourceMonitor struct {
	mu              sync.RWMutex
	enabled         bool
	maxMemoryMB     int64
	maxGoroutines   int
	maxCPUUsage     float64
	checkInterval   time.Duration
	currentLoad     int32
	degradationLevel int32 // 0=none, 1=light, 2=moderate, 3=severe
	stopChan        chan struct{}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		enabled:       true,
		maxMemoryMB:   400,  // Alert at 400MB
		maxGoroutines: 10000,
		maxCPUUsage:   80.0,
		checkInterval: 10 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

// Start begins monitoring resources
func (rm *ResourceMonitor) Start() {
	if !rm.enabled {
		return
	}

	ticker := time.NewTicker(rm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopChan:
			return
		case <-ticker.C:
			rm.checkResources()
		}
	}
}

// Stop stops the resource monitor
func (rm *ResourceMonitor) Stop() {
	close(rm.stopChan)
}

// checkResources checks current resource usage and updates degradation level
func (rm *ResourceMonitor) checkResources() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memMB := int64(memStats.Alloc / 1024 / 1024)
	numGoroutines := runtime.NumGoroutine()

	// Determine degradation level
	level := int32(0)

	// Memory pressure
	if memMB > rm.maxMemoryMB {
		level = 3 // Severe
	} else if memMB > rm.maxMemoryMB*3/4 {
		level = 2 // Moderate
	} else if memMB > rm.maxMemoryMB/2 {
		level = 1 // Light
	}

	// Goroutine pressure
	if numGoroutines > rm.maxGoroutines {
		level = max(level, 3)
	} else if numGoroutines > rm.maxGoroutines*3/4 {
		level = max(level, 2)
	} else if numGoroutines > rm.maxGoroutines/2 {
		level = max(level, 1)
	}

	atomic.StoreInt32(&rm.degradationLevel, level)
	atomic.StoreInt32(&rm.currentLoad, int32(memMB))

	// Force GC under severe pressure
	if level >= 3 {
		runtime.GC()
	}
}

// DegradationLevel returns the current degradation level
func (rm *ResourceMonitor) DegradationLevel() int {
	return int(atomic.LoadInt32(&rm.degradationLevel))
}

// ShouldSample returns true if log sampling should be applied
func (rm *ResourceMonitor) ShouldSample() bool {
	return rm.DegradationLevel() >= 2
}

// ShouldDropLogs returns true if non-critical logs should be dropped
func (rm *ResourceMonitor) ShouldDropLogs() bool {
	return rm.DegradationLevel() >= 3
}

// MemoryUsageMB returns current memory usage in MB
func (rm *ResourceMonitor) MemoryUsageMB() int64 {
	return int64(atomic.LoadInt32(&rm.currentLoad))
}

// Stats returns resource statistics
func (rm *ResourceMonitor) Stats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"memory_mb":      int64(memStats.Alloc / 1024 / 1024),
		"memory_sys_mb":  int64(memStats.Sys / 1024 / 1024),
		"goroutines":     runtime.NumGoroutine(),
		"gc_cycles":      memStats.NumGC,
		"degradation":    rm.DegradationLevel(),
	}
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu               sync.Mutex
	failureCount     int32
	successCount     int32
	state            int32 // 0=closed, 1=open, 2=half-open
	failureThreshold int32
	successThreshold int32
	timeout          time.Duration
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: int32(failureThreshold),
		successThreshold: int32(successThreshold),
		timeout:          timeout,
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn()
	cb.recordResult(err == nil)
	return err
}

// allowRequest checks if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch atomic.LoadInt32(&cb.state) {
	case 0: // Closed - allow
		return true
	case 1: // Open - check timeout
		if time.Since(cb.lastFailureTime) > cb.timeout {
			atomic.StoreInt32(&cb.state, 2) // Half-open
			return true
		}
		return false
	case 2: // Half-open - allow limited requests
		return true
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.successCount++
		if atomic.LoadInt32(&cb.state) == 2 && cb.successCount >= cb.successThreshold {
			atomic.StoreInt32(&cb.state, 0) // Close
			atomic.StoreInt32(&cb.failureCount, 0)
			atomic.StoreInt32(&cb.successCount, 0)
		}
	} else {
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		if cb.failureCount >= cb.failureThreshold {
			atomic.StoreInt32(&cb.state, 1) // Open
		}
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() string {
	switch atomic.LoadInt32(&cb.state) {
	case 0:
		return "closed"
	case 1:
		return "open"
	case 2:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = &CircuitOpenError{}

type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}

// LogSampler provides sampling functionality for high-load scenarios
type LogSampler struct {
	mu         sync.Mutex
	rate       float64 // 0.0 to 1.0
	counter    uint64
	sampleCount uint64
}

// NewLogSampler creates a new log sampler
func NewLogSampler(rate float64) *LogSampler {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	return &LogSampler{rate: rate}
}

// SetRate sets the sampling rate
func (ls *LogSampler) SetRate(rate float64) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.rate = rate
}

// ShouldLog returns true if the log should be kept (sampled)
func (ls *LogSampler) ShouldLog() bool {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.counter++

	if ls.rate >= 1.0 {
		ls.sampleCount++
		return true
	}

	if ls.rate <= 0.0 {
		return false
	}

	// Simple sampling: keep every Nth log
	interval := uint64(1.0 / ls.rate)
	if interval == 0 {
		interval = 1
	}

	if ls.counter%interval == 0 {
		ls.sampleCount++
		return true
	}

	return false
}

// Stats returns sampler statistics
func (ls *LogSampler) Stats() map[string]interface{} {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return map[string]interface{}{
		"rate":         ls.rate,
		"total":        ls.counter,
		"sampled":      ls.sampleCount,
		"dropped":      ls.counter - ls.sampleCount,
	}
}
