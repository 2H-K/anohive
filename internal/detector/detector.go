package detector

import (
	"container/ring"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/2H-K/anohive/internal/models"
)

type Detector struct {
	mu              sync.RWMutex
	windowSize      int
	errorThreshold  float64
	rateThreshold   float64
	burstThreshold  int
	sources         map[string]*SourceState
	anomalyCallback func(*models.Anomaly)
}

type SourceState struct {
	Source          string
	LastAnomalyTime map[string]time.Time
	ErrorPatterns   map[string]int64
	LastLogTime     time.Time
	TotalLogs       int64
	ErrorLogs       int64
	RecentMessages  *ring.Ring
	RateWindow      []rateSample
	RateWindowMu    sync.Mutex
}

type rateSample struct {
	Timestamp time.Time
	Count     int
}

func NewDetector(windowSize int) *Detector {
	if windowSize <= 0 {
		windowSize = 60
	}

	return &Detector{
		windowSize:     windowSize,
		errorThreshold: 0.3,
		rateThreshold:  3.0,
		burstThreshold: 100,
		sources:        make(map[string]*SourceState),
	}
}

func (d *Detector) SetAnomalyCallback(cb func(*models.Anomaly)) {
	d.anomalyCallback = cb
}

func (d *Detector) SetThresholds(errorRate, rateMultiplier float64, burst int) {
	d.errorThreshold = errorRate
	d.rateThreshold = rateMultiplier
	d.burstThreshold = burst
}

func (d *Detector) Process(entry *models.LogEntry) []*models.Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()

	source := entry.Source
	if source == "" {
		source = "unknown"
	}

	state, exists := d.sources[source]
	if !exists {
		state = &SourceState{
			Source:          source,
			LastAnomalyTime: make(map[string]time.Time),
			ErrorPatterns:   make(map[string]int64),
			RecentMessages:  ring.New(100),
			RateWindow:      make([]rateSample, 0, 60),
		}
		d.sources[source] = state
	}

	state.LastLogTime = entry.Timestamp
	state.TotalLogs++

	isError := entry.Level == models.LogLevelError || entry.Level == models.LogLevelFatal
	if isError {
		state.ErrorLogs++
		normalized := normalizeMessage(entry.Message)
		state.ErrorPatterns[normalized]++
	}

	state.RecentMessages.Value = entry.Message
	state.RecentMessages = state.RecentMessages.Next()

	state.addRateSample(entry.Timestamp)

	anomalies := d.detectAnomalies(state, entry)

	for _, a := range anomalies {
		d.raiseAnomaly(a)
	}

	return anomalies
}

func (s *SourceState) addRateSample(ts time.Time) {
	s.RateWindowMu.Lock()
	defer s.RateWindowMu.Unlock()

	windowStart := ts.Add(-5 * time.Minute)
	cutoff := 0
	for i, sample := range s.RateWindow {
		if sample.Timestamp.After(windowStart) {
			cutoff = i
			break
		}
	}
	s.RateWindow = s.RateWindow[cutoff:]

	if len(s.RateWindow) > 0 {
		last := &s.RateWindow[len(s.RateWindow)-1]
		if ts.Sub(last.Timestamp) < time.Second {
			last.Count++
			return
		}
	}

	s.RateWindow = append(s.RateWindow, rateSample{Timestamp: ts, Count: 1})
}

func (s *SourceState) getRatePerMinute() float64 {
	s.RateWindowMu.Lock()
	defer s.RateWindowMu.Unlock()

	if len(s.RateWindow) < 2 {
		return 0
	}

	total := 0
	for _, s := range s.RateWindow {
		total += s.Count
	}

	duration := s.RateWindow[len(s.RateWindow)-1].Timestamp.Sub(s.RateWindow[0].Timestamp).Minutes()
	if duration <= 0 {
		return 0
	}

	return float64(total) / duration
}

func (s *SourceState) getBaselineRate() float64 {
	s.RateWindowMu.Lock()
	defer s.RateWindowMu.Unlock()

	if len(s.RateWindow) < 10 {
		return 0
	}

	mid := len(s.RateWindow) / 2
	firstHalf := s.RateWindow[:mid]
	secondHalf := s.RateWindow[mid:]

	if len(firstHalf) < 2 {
		return 0
	}

	total := 0
	for _, s := range firstHalf {
		total += s.Count
	}

	duration := firstHalf[len(firstHalf)-1].Timestamp.Sub(firstHalf[0].Timestamp).Minutes()
	if duration <= 0 {
		return 0
	}

	baseline := float64(total) / duration

	total = 0
	for _, s := range secondHalf {
		total += s.Count
	}

	duration = secondHalf[len(secondHalf)-1].Timestamp.Sub(secondHalf[0].Timestamp).Minutes()
	if duration <= 0 {
		return baseline
	}

	current := float64(total) / duration

	_ = current
	return baseline
}

func (d *Detector) detectAnomalies(state *SourceState, entry *models.LogEntry) []*models.Anomaly {
	var anomalies []*models.Anomaly
	now := entry.Timestamp

	if spike := d.detectErrorSpike(state, now); spike != nil {
		anomalies = append(anomalies, spike)
	}

	if burst := d.detectLogBurst(state, now); burst != nil {
		anomalies = append(anomalies, burst)
	}

	if newError := d.detectNewErrorPattern(state, entry, now); newError != nil {
		anomalies = append(anomalies, newError)
	}

	if rateChange := d.detectRateChange(state, now); rateChange != nil {
		anomalies = append(anomalies, rateChange)
	}

	return anomalies
}

func (d *Detector) detectErrorSpike(state *SourceState, now time.Time) *models.Anomaly {
	if state.TotalLogs < 10 {
		return nil
	}

	errorRate := float64(state.ErrorLogs) / float64(state.TotalLogs)
	if errorRate < d.errorThreshold {
		return nil
	}

	key := "error_spike"
	if last, ok := state.LastAnomalyTime[key]; ok && now.Sub(last) < 30*time.Second {
		return nil
	}
	state.LastAnomalyTime[key] = now

	severity := models.SeverityMedium
	if errorRate > 0.7 {
		severity = models.SeverityCritical
	} else if errorRate > 0.5 {
		severity = models.SeverityHigh
	}

	return &models.Anomaly{
		ID:          generateID(),
		Timestamp:   now,
		Type:        models.AnomalySpike,
		Severity:    severity,
		Description: fmt.Sprintf("Error rate spike detected: %.1f%% errors in source '%s' (total: %d, errors: %d)", errorRate*100, state.Source, state.TotalLogs, state.ErrorLogs),
		Source:      state.Source,
		Metadata: map[string]string{
			"error_rate": fmt.Sprintf("%.2f", errorRate),
			"total_logs": fmt.Sprintf("%d", state.TotalLogs),
			"error_logs": fmt.Sprintf("%d", state.ErrorLogs),
		},
	}
}

func (d *Detector) detectLogBurst(state *SourceState, now time.Time) *models.Anomaly {
	if now.Sub(state.LastLogTime) > 5*time.Minute {
		return nil
	}

	key := "log_burst"
	if last, ok := state.LastAnomalyTime[key]; ok && now.Sub(last) < 15*time.Second {
		return nil
	}

	count := 0
	state.RecentMessages.Do(func(v interface{}) {
		if v != nil {
			count++
		}
	})

	if count < d.burstThreshold/2 {
		return nil
	}

	state.LastAnomalyTime[key] = now

	return &models.Anomaly{
		ID:          generateID(),
		Timestamp:   now,
		Type:        models.AnomalyBurst,
		Severity:    models.SeverityMedium,
		Description: fmt.Sprintf("Log burst detected in source '%s': %d recent messages", state.Source, count),
		Source:      state.Source,
		Metadata: map[string]string{
			"recent_count": fmt.Sprintf("%d", count),
		},
	}
}

func (d *Detector) detectNewErrorPattern(state *SourceState, entry *models.LogEntry, now time.Time) *models.Anomaly {
	if entry.Level != models.LogLevelError && entry.Level != models.LogLevelFatal {
		return nil
	}

	normalized := normalizeMessage(entry.Message)
	count := state.ErrorPatterns[normalized]

	if count != 1 {
		return nil
	}

	key := "new_error_" + normalized
	if last, ok := state.LastAnomalyTime[key]; ok && now.Sub(last) < 5*time.Minute {
		return nil
	}
	state.LastAnomalyTime[key] = now

	return &models.Anomaly{
		ID:          generateID(),
		Timestamp:   now,
		Type:        models.AnomalyNewError,
		Severity:    models.SeverityLow,
		Description: fmt.Sprintf("New error pattern detected in source '%s': %s", state.Source, truncate(entry.Message, 120)),
		Source:      state.Source,
		Metadata: map[string]string{
			"pattern": normalized,
		},
	}
}

func (d *Detector) detectRateChange(state *SourceState, now time.Time) *models.Anomaly {
	if state.TotalLogs < 50 {
		return nil
	}

	currentRate := state.getRatePerMinute()
	baselineRate := state.getBaselineRate()

	if baselineRate < 1.0 {
		return nil
	}

	ratio := currentRate / baselineRate
	if ratio < d.rateThreshold {
		return nil
	}

	key := "rate_change"
	if last, ok := state.LastAnomalyTime[key]; ok && now.Sub(last) < 60*time.Second {
		return nil
	}
	state.LastAnomalyTime[key] = now

	return &models.Anomaly{
		ID:          generateID(),
		Timestamp:   now,
		Type:        models.AnomalyRateChange,
		Severity:    models.SeverityMedium,
		Description: fmt.Sprintf("Log rate change in source '%s': %.1fx increase (current: %.1f/min, baseline: %.1f/min)", state.Source, ratio, currentRate, baselineRate),
		Source:      state.Source,
		Metadata: map[string]string{
			"ratio":         fmt.Sprintf("%.1f", ratio),
			"current_rate":  fmt.Sprintf("%.1f", currentRate),
			"baseline_rate": fmt.Sprintf("%.1f", baselineRate),
		},
	}
}

func (d *Detector) raiseAnomaly(a *models.Anomaly) {
	if d.anomalyCallback != nil {
		d.anomalyCallback(a)
	}
}

func (d *Detector) GetSourceStates() map[string]*SourceState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[string]*SourceState)
	for k, v := range d.sources {
		result[k] = v
	}
	return result
}

func normalizeMessage(msg string) string {
	msg = strings.ToLower(msg)
	parts := strings.Fields(msg)
	if len(parts) > 0 {
		return parts[0]
	}
	return strings.TrimSpace(msg)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func generateID() string {
	return fmt.Sprintf("anm_%d", time.Now().UnixNano())
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}
