// Package runtime provides region statistics and performance monitoring.
// This module implements comprehensive metrics collection, analysis,
// and reporting for region-based memory management systems.
package runtime

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and manages runtime metrics
type MetricsCollector struct {
	mutex          sync.RWMutex
	regions        map[RegionID]*RegionMetrics // Per-region metrics
	global         *GlobalMetrics              // Global metrics
	history        *MetricsHistory             // Historical data
	alerts         *AlertManager               // Alert management
	config         MetricsConfig               // Configuration
	collectors     []Collector                 // Active collectors
	lastCollection time.Time                   // Last collection time
	enabled        bool                        // Collection enabled
}

// RegionMetrics contains detailed metrics for a single region
type RegionMetrics struct {
	ID         RegionID      // Region identifier
	CreatedAt  time.Time     // Creation timestamp
	LastUpdate time.Time     // Last update timestamp
	Lifetime   time.Duration // Region lifetime

	// Memory metrics
	TotalSize    uint64 // Total region size
	UsedMemory   uint64 // Currently used memory
	FreeMemory   uint64 // Currently free memory
	PeakUsage    uint64 // Peak memory usage
	LowWaterMark uint64 // Lowest memory usage

	// Allocation metrics
	AllocCount   uint64  // Total allocations
	FreeCount    uint64  // Total deallocations
	AllocRate    float64 // Allocations per second
	FreeRate     float64 // Deallocations per second
	NetAllocRate float64 // Net allocation rate

	// Size distribution
	SmallAllocs     uint64  // Allocations < 1KB
	MediumAllocs    uint64  // Allocations 1KB-64KB
	LargeAllocs     uint64  // Allocations > 64KB
	AvgAllocSize    float64 // Average allocation size
	MedianAllocSize uint64  // Median allocation size

	// Performance metrics
	AllocLatency LatencyMetrics // Allocation latency
	FreeLatency  LatencyMetrics // Deallocation latency
	Throughput   float64        // Operations per second
	Utilization  float64        // Memory utilization ratio

	// Fragmentation metrics
	InternalFrag   float64 // Internal fragmentation
	ExternalFrag   float64 // External fragmentation
	FreeBlockCount uint64  // Number of free blocks
	LargestFree    uint64  // Largest free block

	// Compaction metrics
	CompactionCount uint64        // Number of compactions
	CompactionTime  time.Duration // Total compaction time
	LastCompaction  time.Time     // Last compaction time

	// Error metrics
	FailedAllocs   uint64 // Failed allocations
	FailedFrees    uint64 // Failed deallocations
	Corruptions    uint64 // Memory corruptions
	LeakDetections uint64 // Memory leaks detected

	// Custom metrics
	CustomCounters map[string]uint64        // Custom counters
	CustomGauges   map[string]float64       // Custom gauges
	CustomTimers   map[string]*TimerMetrics // Custom timers
}

// GlobalMetrics contains system-wide metrics
type GlobalMetrics struct {
	// System metrics
	TotalRegions  uint64 // Total regions created
	ActiveRegions uint64 // Currently active regions
	SystemMemory  uint64 // Total system memory
	UsedMemory    uint64 // Total used memory
	FreeMemory    uint64 // Total free memory

	// Aggregate allocation metrics
	TotalAllocs     uint64  // Total allocations across all regions
	TotalFrees      uint64  // Total deallocations
	GlobalAllocRate float64 // Global allocation rate
	GlobalFreeRate  float64 // Global deallocation rate

	// Performance metrics
	OverallThroughput   float64 // Overall system throughput
	AverageUtilization  float64 // Average memory utilization
	SystemFragmentation float64 // System-wide fragmentation

	// Resource pressure
	MemoryPressure     float64 // Memory pressure indicator
	AllocationPressure float64 // Allocation pressure
	CompactionPressure float64 // Compaction pressure

	// Health metrics
	HealthScore       float64 // Overall health score
	ErrorRate         float64 // Error rate
	AvailabilityScore float64 // System availability
}

// LatencyMetrics tracks latency distribution
type LatencyMetrics struct {
	Count   uint64          // Number of samples
	Sum     time.Duration   // Sum of all latencies
	Min     time.Duration   // Minimum latency
	Max     time.Duration   // Maximum latency
	Mean    time.Duration   // Mean latency
	Median  time.Duration   // Median latency
	P95     time.Duration   // 95th percentile
	P99     time.Duration   // 99th percentile
	P999    time.Duration   // 99.9th percentile
	StdDev  time.Duration   // Standard deviation
	Samples []time.Duration // Sample buffer for percentile calculation
}

// TimerMetrics tracks timing metrics
type TimerMetrics struct {
	Count    uint64        // Number of measurements
	Total    time.Duration // Total time
	Average  time.Duration // Average time
	Min      time.Duration // Minimum time
	Max      time.Duration // Maximum time
	LastTime time.Duration // Last measurement
}

// MetricsHistory maintains historical metrics data
type MetricsHistory struct {
	mutex        sync.RWMutex
	snapshots    []*MetricsSnapshot // Historical snapshots
	maxSnapshots int                // Maximum snapshots to keep
	interval     time.Duration      // Snapshot interval
	lastSnapshot time.Time          // Last snapshot time
}

// MetricsSnapshot represents metrics at a point in time
type MetricsSnapshot struct {
	Timestamp time.Time                   // Snapshot timestamp
	Global    GlobalMetrics               // Global metrics snapshot
	Regions   map[RegionID]*RegionMetrics // Region metrics snapshot
}

// AlertManager manages metric alerts and thresholds
type AlertManager struct {
	mutex     sync.RWMutex
	rules     []*AlertRule    // Alert rules
	active    []*ActiveAlert  // Active alerts
	history   []*AlertHistory // Alert history
	callbacks []AlertCallback // Alert callbacks
	enabled   bool            // Alerting enabled
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string         // Rule identifier
	Name        string         // Human-readable name
	Description string         // Rule description
	Condition   AlertCondition // Alert condition
	Threshold   float64        // Threshold value
	Duration    time.Duration  // Duration threshold must be exceeded
	Severity    AlertSeverity  // Alert severity
	Actions     []AlertAction  // Actions to take
	Enabled     bool           // Rule enabled
}

// AlertCondition represents different alert conditions
type AlertCondition int

const (
	ConditionGreaterThan AlertCondition = iota
	ConditionLessThan
	ConditionEquals
	ConditionNotEquals
	ConditionChange
	ConditionTrend
)

// AlertSeverity represents alert severity levels
type AlertSeverity int

const (
	SeverityInfo AlertSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// AlertAction represents actions to take when alert triggers
type AlertAction int

const (
	ActionLog AlertAction = iota
	ActionCallback
	ActionCompact
	ActionShrink
	ActionGC
	ActionPanic
)

// ActiveAlert represents an active alert
type ActiveAlert struct {
	Rule         *AlertRule // Associated rule
	Triggered    time.Time  // When alert was triggered
	LastFired    time.Time  // Last time alert fired
	FireCount    uint64     // Number of times fired
	Value        float64    // Current metric value
	Acknowledged bool       // Alert acknowledged
}

// AlertHistory tracks alert history
type AlertHistory struct {
	Alert     *ActiveAlert  // Alert information
	Triggered time.Time     // Trigger time
	Resolved  time.Time     // Resolution time
	Duration  time.Duration // Alert duration
}

// AlertCallback handles alert notifications
type AlertCallback func(alert *ActiveAlert)

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	Enabled            bool          // Enable metrics collection
	CollectionInterval time.Duration // Collection interval
	HistorySize        int           // Number of snapshots to keep
	SampleSize         int           // Sample size for percentiles
	EnableLatency      bool          // Enable latency tracking
	EnableProfiling    bool          // Enable performance profiling
	AlertingEnabled    bool          // Enable alerting
	ExportEnabled      bool          // Enable metrics export
	ExportInterval     time.Duration // Export interval
}

// Collector interface for custom metrics collectors
type Collector interface {
	Name() string                             // Collector name
	Collect() (map[string]interface{}, error) // Collect metrics
	Reset()                                   // Reset collector state
}

// Standard metrics constants
const (
	DefaultCollectionInterval = time.Second * 5
	DefaultHistorySize        = 100
	DefaultSampleSize         = 1000
	MaxLatencySamples         = 10000

	// Size thresholds
	SmallAllocThreshold = 1024  // 1KB
	LargeAllocThreshold = 65536 // 64KB

	// Health score weights
	UtilizationWeight   = 0.3
	FragmentationWeight = 0.2
	ErrorRateWeight     = 0.3
	PerformanceWeight   = 0.2
)

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	return &MetricsCollector{
		regions:    make(map[RegionID]*RegionMetrics),
		global:     &GlobalMetrics{},
		history:    newMetricsHistory(config.HistorySize, config.CollectionInterval),
		alerts:     newAlertManager(),
		config:     config,
		collectors: make([]Collector, 0),
		enabled:    config.Enabled,
	}
}

// RegisterRegion registers a new region for metrics collection
func (mc *MetricsCollector) RegisterRegion(region *Region) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics := &RegionMetrics{
		ID:             region.Header.ID,
		CreatedAt:      time.Now(),
		LastUpdate:     time.Now(),
		TotalSize:      uint64(region.Header.Size),
		CustomCounters: make(map[string]uint64),
		CustomGauges:   make(map[string]float64),
		CustomTimers:   make(map[string]*TimerMetrics),
	}

	mc.regions[region.Header.ID] = metrics
	atomic.AddUint64(&mc.global.TotalRegions, 1)
	atomic.AddUint64(&mc.global.ActiveRegions, 1)
}

// UnregisterRegion removes a region from metrics collection
func (mc *MetricsCollector) UnregisterRegion(regionID RegionID) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.regions[regionID]; exists {
		delete(mc.regions, regionID)
		atomic.AddUint64(&mc.global.ActiveRegions, ^uint64(0)) // Subtract 1
	}
}

// RecordAllocation records an allocation event
func (mc *MetricsCollector) RecordAllocation(regionID RegionID, size uint64, latency time.Duration) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return
	}

	// Update allocation metrics
	metrics.AllocCount++
	metrics.LastUpdate = time.Now()

	// Update size distribution
	if size < SmallAllocThreshold {
		metrics.SmallAllocs++
	} else if size < LargeAllocThreshold {
		metrics.MediumAllocs++
	} else {
		metrics.LargeAllocs++
	}

	// Update average allocation size
	totalAllocs := float64(metrics.AllocCount)
	metrics.AvgAllocSize = (metrics.AvgAllocSize*(totalAllocs-1) + float64(size)) / totalAllocs

	// Update latency metrics
	if mc.config.EnableLatency {
		mc.updateLatencyMetrics(&metrics.AllocLatency, latency)
	}

	// Update global metrics
	atomic.AddUint64(&mc.global.TotalAllocs, 1)
}

// RecordDeallocation records a deallocation event
func (mc *MetricsCollector) RecordDeallocation(regionID RegionID, size uint64, latency time.Duration) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return
	}

	// Update deallocation metrics
	metrics.FreeCount++
	metrics.LastUpdate = time.Now()

	// Update latency metrics
	if mc.config.EnableLatency {
		mc.updateLatencyMetrics(&metrics.FreeLatency, latency)
	}

	// Update global metrics
	atomic.AddUint64(&mc.global.TotalFrees, 1)
}

// UpdateMemoryUsage updates memory usage metrics
func (mc *MetricsCollector) UpdateMemoryUsage(regionID RegionID, used, free uint64) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return
	}

	metrics.UsedMemory = used
	metrics.FreeMemory = free
	metrics.LastUpdate = time.Now()

	// Update peak usage
	if used > metrics.PeakUsage {
		metrics.PeakUsage = used
	}

	// Update utilization
	if metrics.TotalSize > 0 {
		metrics.Utilization = float64(used) / float64(metrics.TotalSize)
	}
}

// UpdateFragmentation updates fragmentation metrics
func (mc *MetricsCollector) UpdateFragmentation(regionID RegionID, internal, external float64, freeBlocks uint64, largestFree uint64) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return
	}

	metrics.InternalFrag = internal
	metrics.ExternalFrag = external
	metrics.FreeBlockCount = freeBlocks
	metrics.LargestFree = largestFree
	metrics.LastUpdate = time.Now()
}

// RecordCompaction records a compaction event
func (mc *MetricsCollector) RecordCompaction(regionID RegionID, duration time.Duration) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return
	}

	metrics.CompactionCount++
	metrics.CompactionTime += duration
	metrics.LastCompaction = time.Now()
	metrics.LastUpdate = time.Now()
}

// RecordError records an error event
func (mc *MetricsCollector) RecordError(regionID RegionID, errorType string) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return
	}

	switch errorType {
	case "allocation_failed":
		metrics.FailedAllocs++
	case "deallocation_failed":
		metrics.FailedFrees++
	case "corruption":
		metrics.Corruptions++
	case "leak":
		metrics.LeakDetections++
	}

	metrics.LastUpdate = time.Now()
}

// Collect performs a metrics collection cycle
func (mc *MetricsCollector) Collect() error {
	if !mc.enabled {
		return nil
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	now := time.Now()

	// Update rates for all regions
	for _, metrics := range mc.regions {
		mc.updateRates(metrics, now)
	}

	// Update global metrics
	mc.updateGlobalMetrics()

	// Run custom collectors
	for _, collector := range mc.collectors {
		data, err := collector.Collect()
		if err != nil {
			continue // Log error but continue
		}
		mc.processCustomMetrics(data)
	}

	// Create snapshot if needed
	if now.Sub(mc.history.lastSnapshot) >= mc.history.interval {
		mc.createSnapshot(now)
	}

	// Check alert conditions
	if mc.config.AlertingEnabled {
		mc.alerts.checkAlerts(mc.global, mc.regions)
	}

	mc.lastCollection = now
	return nil
}

// GetRegionMetrics returns metrics for a specific region
func (mc *MetricsCollector) GetRegionMetrics(regionID RegionID) (*RegionMetrics, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	metrics, exists := mc.regions[regionID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	copy := *metrics
	return &copy, true
}

// GetGlobalMetrics returns global metrics
func (mc *MetricsCollector) GetGlobalMetrics() *GlobalMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Return a copy
	copy := *mc.global
	return &copy
}

// GetHistory returns metrics history
func (mc *MetricsCollector) GetHistory() []*MetricsSnapshot {
	return mc.history.getSnapshots()
}

// AddCustomCollector adds a custom metrics collector
func (mc *MetricsCollector) AddCustomCollector(collector Collector) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.collectors = append(mc.collectors, collector)
}

// Helper methods

// updateLatencyMetrics updates latency distribution metrics
func (mc *MetricsCollector) updateLatencyMetrics(latency *LatencyMetrics, value time.Duration) {
	latency.Count++
	latency.Sum += value

	if latency.Count == 1 {
		latency.Min = value
		latency.Max = value
	} else {
		if value < latency.Min {
			latency.Min = value
		}
		if value > latency.Max {
			latency.Max = value
		}
	}

	latency.Mean = latency.Sum / time.Duration(latency.Count)

	// Add to samples for percentile calculation
	if len(latency.Samples) < MaxLatencySamples {
		latency.Samples = append(latency.Samples, value)
	} else {
		// Replace random sample to maintain reservoir
		idx := uint64(time.Now().UnixNano()) % MaxLatencySamples
		latency.Samples[idx] = value
	}

	// Calculate percentiles
	mc.calculatePercentiles(latency)
}

// calculatePercentiles calculates latency percentiles
func (mc *MetricsCollector) calculatePercentiles(latency *LatencyMetrics) {
	if len(latency.Samples) == 0 {
		return
	}

	// Sort samples
	samples := make([]time.Duration, len(latency.Samples))
	copy(samples, latency.Samples)
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})

	n := len(samples)
	latency.Median = samples[n/2]
	latency.P95 = samples[int(float64(n)*0.95)]
	latency.P99 = samples[int(float64(n)*0.99)]
	if n > 1000 {
		latency.P999 = samples[int(float64(n)*0.999)]
	}
}

// updateRates updates allocation and deallocation rates
func (mc *MetricsCollector) updateRates(metrics *RegionMetrics, now time.Time) {
	duration := now.Sub(metrics.LastUpdate).Seconds()
	if duration > 0 {
		metrics.AllocRate = float64(metrics.AllocCount) / duration
		metrics.FreeRate = float64(metrics.FreeCount) / duration
		metrics.NetAllocRate = metrics.AllocRate - metrics.FreeRate
		metrics.Throughput = metrics.AllocRate + metrics.FreeRate
	}

	metrics.Lifetime = now.Sub(metrics.CreatedAt)
}

// updateGlobalMetrics updates global metrics from region metrics
func (mc *MetricsCollector) updateGlobalMetrics() {
	var totalUsed, totalFree uint64
	var totalThroughput, totalUtilization float64
	var errorCount uint64

	regionCount := uint64(len(mc.regions))

	for _, metrics := range mc.regions {
		totalUsed += metrics.UsedMemory
		totalFree += metrics.FreeMemory
		totalThroughput += metrics.Throughput
		totalUtilization += metrics.Utilization
		errorCount += metrics.FailedAllocs + metrics.FailedFrees + metrics.Corruptions
	}

	mc.global.UsedMemory = totalUsed
	mc.global.FreeMemory = totalFree
	mc.global.SystemMemory = totalUsed + totalFree
	mc.global.OverallThroughput = totalThroughput

	if regionCount > 0 {
		mc.global.AverageUtilization = totalUtilization / float64(regionCount)
	}

	// Calculate health score
	mc.calculateHealthScore()
}

// calculateHealthScore calculates overall system health score
func (mc *MetricsCollector) calculateHealthScore() {
	score := 100.0

	// Factor in utilization (optimal around 70-80%)
	util := mc.global.AverageUtilization
	if util < 0.5 || util > 0.9 {
		score -= 20.0 * UtilizationWeight
	}

	// Factor in fragmentation
	if mc.global.SystemFragmentation > 0.3 {
		score -= 30.0 * FragmentationWeight
	}

	// Factor in error rate
	if mc.global.ErrorRate > 0.01 {
		score -= 40.0 * ErrorRateWeight
	}

	// Factor in performance
	if mc.global.OverallThroughput < 1000 {
		score -= 25.0 * PerformanceWeight
	}

	if score < 0 {
		score = 0
	}

	mc.global.HealthScore = score
}

// createSnapshot creates a metrics snapshot
func (mc *MetricsCollector) createSnapshot(timestamp time.Time) {
	snapshot := &MetricsSnapshot{
		Timestamp: timestamp,
		Global:    *mc.global,
		Regions:   make(map[RegionID]*RegionMetrics),
	}

	// Copy region metrics
	for id, metrics := range mc.regions {
		copy := *metrics
		snapshot.Regions[id] = &copy
	}

	mc.history.addSnapshot(snapshot)
}

// processCustomMetrics processes custom collector metrics
func (mc *MetricsCollector) processCustomMetrics(data map[string]interface{}) {
	// Process custom metrics data
	// This would integrate custom metrics into the overall metrics system
}

// newMetricsHistory creates a new metrics history tracker
func newMetricsHistory(maxSnapshots int, interval time.Duration) *MetricsHistory {
	return &MetricsHistory{
		snapshots:    make([]*MetricsSnapshot, 0, maxSnapshots),
		maxSnapshots: maxSnapshots,
		interval:     interval,
	}
}

// addSnapshot adds a new metrics snapshot
func (mh *MetricsHistory) addSnapshot(snapshot *MetricsSnapshot) {
	mh.mutex.Lock()
	defer mh.mutex.Unlock()

	mh.snapshots = append(mh.snapshots, snapshot)

	// Remove old snapshots if over limit
	if len(mh.snapshots) > mh.maxSnapshots {
		mh.snapshots = mh.snapshots[1:]
	}

	mh.lastSnapshot = snapshot.Timestamp
}

// getSnapshots returns a copy of all snapshots
func (mh *MetricsHistory) getSnapshots() []*MetricsSnapshot {
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()

	result := make([]*MetricsSnapshot, len(mh.snapshots))
	copy(result, mh.snapshots)
	return result
}

// newAlertManager creates a new alert manager
func newAlertManager() *AlertManager {
	return &AlertManager{
		rules:     make([]*AlertRule, 0),
		active:    make([]*ActiveAlert, 0),
		history:   make([]*AlertHistory, 0),
		callbacks: make([]AlertCallback, 0),
		enabled:   true,
	}
}

// checkAlerts checks all alert rules against current metrics
func (am *AlertManager) checkAlerts(global *GlobalMetrics, regions map[RegionID]*RegionMetrics) {
	if !am.enabled {
		return
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check rules against metrics
	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// Evaluate rule condition
		triggered := am.evaluateRule(rule, global, regions)

		// Handle alert state changes
		am.handleAlertState(rule, triggered)
	}
}

// evaluateRule evaluates an alert rule
func (am *AlertManager) evaluateRule(rule *AlertRule, global *GlobalMetrics, regions map[RegionID]*RegionMetrics) bool {
	// This would implement rule evaluation logic
	// For now, return false as placeholder
	return false
}

// handleAlertState handles alert state changes
func (am *AlertManager) handleAlertState(rule *AlertRule, triggered bool) {
	// Find existing active alert
	var existingAlert *ActiveAlert
	for _, alert := range am.active {
		if alert.Rule == rule {
			existingAlert = alert
			break
		}
	}

	if triggered && existingAlert == nil {
		// New alert
		alert := &ActiveAlert{
			Rule:      rule,
			Triggered: time.Now(),
			LastFired: time.Now(),
			FireCount: 1,
		}
		am.active = append(am.active, alert)

		// Execute callbacks
		for _, callback := range am.callbacks {
			callback(alert)
		}
	} else if !triggered && existingAlert != nil {
		// Alert resolved
		am.resolveAlert(existingAlert)
	}
}

// resolveAlert resolves an active alert
func (am *AlertManager) resolveAlert(alert *ActiveAlert) {
	// Remove from active alerts
	for i, active := range am.active {
		if active == alert {
			am.active[i] = am.active[len(am.active)-1]
			am.active = am.active[:len(am.active)-1]
			break
		}
	}

	// Add to history
	history := &AlertHistory{
		Alert:     alert,
		Triggered: alert.Triggered,
		Resolved:  time.Now(),
		Duration:  time.Since(alert.Triggered),
	}
	am.history = append(am.history, history)
}
