// Phase 3.1.3: NUMA-Aware Memory Optimization
// This file implements NUMA topology discovery, memory affinity control,.
// and dynamic load balancing for optimal memory locality on multi-socket systems.

package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Type definitions.
type (
	NodeID          uint32  // NUMA node identifier
	ProcessorID     uint32  // Processor core identifier
	MemoryDomain    uint32  // Memory domain identifier
	AffinityMask    uint64  // CPU affinity mask
	BandwidthMetric float64 // Memory bandwidth in GB/s
	LatencyMetric   float64 // Memory latency in nanoseconds
	WorkloadID      uint64  // Workload identifier
)

// Allocation hint for NUMA-aware allocation.
type AllocationHint struct {
	AccessPattern string
	Priority      int
	PreferredNode NodeID
	Pinned        bool
	Interleaved   bool
}

// NUMA topology information.
type NUMANode struct {
	LastUpdate      time.Time
	Allocations     map[WorkloadID]*NUMAAllocation
	Distance        map[NodeID]uint32
	Processors      []ProcessorID
	MemorySize      uint64
	AvailableMemory uint64
	Bandwidth       BandwidthMetric
	Latency         LatencyMetric
	Load            float64
	mutex           sync.RWMutex
	ID              NodeID
}

// NUMA allocation tracking.
type NUMAAllocation struct {
	CreateTime    time.Time
	LastAccess    time.Time
	StartAddress  unsafe.Pointer
	EndAddress    unsafe.Pointer
	AccessPattern AccessPattern
	ID            WorkloadID
	Size          uint64
	ProcessorMask AffinityMask
	Priority      int
	OwnerNode     NodeID
	Pinned        bool
	Interleaved   bool
}

// NUMA optimizer main structure.
type NUMAOptimizer struct {
	statistics  NUMAStatistics
	nodes       map[NodeID]*NUMANode
	allocations map[WorkloadID]*NUMAAllocation
	topology    *NUMATopology
	balancer    *LoadBalancer
	monitor     *PerformanceMonitor
	stopChan    chan struct{}
	config      NUMAConfig
	mutex       sync.RWMutex
	enabled     bool
	running     bool
}

// NUMA system topology.
type NUMATopology struct {
	LastDiscovery   time.Time
	DistanceMatrix  [][]uint32
	BandwidthMatrix [][]BandwidthMetric
	LatencyMatrix   [][]LatencyMetric
	HotPathNodes    []NodeID
	ColdPathNodes   []NodeID
	TotalMemory     uint64
	NodeCount       uint32
	ProcessorCount  uint32
}

// Load balancer for NUMA-aware distribution.
type LoadBalancer struct {
	lastRebalance  time.Time
	policies       map[NodeID]BalancingPolicy
	rebalanceQueue chan *RebalanceRequest
	migrationCost  map[NodeID]map[NodeID]float64
	strategies     []NUMABalancingStrategy
	mutex          sync.RWMutex
	enabled        bool
}

// Performance monitoring for NUMA optimization.
type PerformanceMonitor struct {
	metrics         map[NodeID]*NodeMetrics
	samples         []PerformanceSample
	alerts          []PerformanceAlert
	collectors      []NUMAMetricsCollector
	thresholds      PerformanceThresholds
	samplingRate    time.Duration
	retentionPeriod time.Duration
	mutex           sync.RWMutex
	enabled         bool
}

// Per-node performance metrics.
type NodeMetrics struct {
	Timestamp      time.Time
	MemoryUsage    float64
	BandwidthUtil  float64
	LatencyAverage LatencyMetric
	CacheHitRate   float64
	RemoteAccesses uint64
	LocalAccesses  uint64
	Migrations     uint64
}

// Performance sample for analysis.
type PerformanceSample struct {
	Timestamp       time.Time
	MemoryBandwidth BandwidthMetric
	AccessLatency   LatencyMetric
	LocalityRatio   float64
	CacheEfficiency float64
	ThroughputMBps  float64
	NodeID          NodeID
}

// Balancing strategies enumeration.
type NUMABalancingStrategy int

const (
	NUMAFirstFit NUMABalancingStrategy = iota
	NUMABestFit
	NUMAWorstFit
	NUMANextFit
	LocalityAware
	LoadAware
	BandwidthOptimal
	LatencyOptimal
)

// Balancing policies.
type BalancingPolicy int

const (
	Static BalancingPolicy = iota
	Dynamic
	Adaptive
	Predictive
	Hybrid
)

// Migration strategies.
type MigrationStrategy int

const (
	Immediate MigrationStrategy = iota
	Deferred
	Lazy
	Batched
	Threshold
)

// NUMA configuration.
type NUMAConfig struct {
	EnableOptimization bool                  // Enable NUMA optimization
	EnableBalancing    bool                  // Enable load balancing
	EnableMigration    bool                  // Enable page migration
	EnableInterleaving bool                  // Enable memory interleaving
	BalancingInterval  time.Duration         // Balancing frequency
	MigrationThreshold float64               // Migration threshold
	LocalityThreshold  float64               // Locality threshold
	MaxMigrationCost   float64               // Maximum migration cost
	PreferredStrategy  NUMABalancingStrategy // Preferred strategy
	FallbackStrategy   NUMABalancingStrategy // Fallback strategy
	MonitoringInterval time.Duration         // Monitoring frequency
	RetentionPeriod    time.Duration         // Data retention
}

// Performance thresholds for alerts.
type PerformanceThresholds struct {
	MaxMemoryUsage   float64         // Maximum memory usage
	MaxLatency       LatencyMetric   // Maximum latency
	MinBandwidth     BandwidthMetric // Minimum bandwidth
	MaxRemoteRatio   float64         // Maximum remote access ratio
	MinLocalityRatio float64         // Minimum locality ratio
	MaxMigrationRate float64         // Maximum migration rate
}

// Performance alert.
type PerformanceAlert struct {
	Timestamp time.Time
	Message   string
	ID        uint64
	Type      AlertType
	Severity  NUMAAlertSeverity
	Threshold float64
	Current   float64
	NodeID    NodeID
	Resolved  bool
}

// Alert types.
type AlertType int

const (
	MemoryUsageAlert AlertType = iota
	LatencyAlert
	BandwidthAlert
	LocalityAlert
	MigrationAlert
	TopologyAlert
)

// Alert severity levels.
type NUMAAlertSeverity int

const (
	NUMAInfo NUMAAlertSeverity = iota
	NUMAWarning
	NUMAError
	NUMACritical
)

// Rebalance request.
type RebalanceRequest struct {
	Timestamp  time.Time
	Allocation *NUMAAllocation
	Reason     string
	ID         uint64
	Strategy   NUMABalancingStrategy
	Priority   int
	Cost       float64
	SourceNode NodeID
	TargetNode NodeID
}

// Metrics collector interface.
type NUMAMetricsCollector interface {
	CollectMetrics(nodeID NodeID) (*NodeMetrics, error)
	GetName() string
	GetFrequency() time.Duration
	IsEnabled() bool
	Configure(config interface{}) error
}

// NUMA statistics.
type NUMAStatistics struct {
	LastReset           time.Time
	TotalAllocations    uint64
	LocalAllocations    uint64
	RemoteAllocations   uint64
	Migrations          uint64
	RebalanceOperations uint64
	CacheHits           uint64
	CacheMisses         uint64
	AverageLatency      LatencyMetric
	AverageBandwidth    BandwidthMetric
	LocalityRatio       float64
}

// Default configurations.
var DefaultNUMAConfig = NUMAConfig{
	EnableOptimization: true,
	EnableBalancing:    true,
	EnableMigration:    true,
	EnableInterleaving: false,
	BalancingInterval:  time.Second * 5,
	MigrationThreshold: 0.8,
	LocalityThreshold:  0.9,
	MaxMigrationCost:   100.0,
	PreferredStrategy:  LocalityAware,
	FallbackStrategy:   NUMAFirstFit,
	MonitoringInterval: time.Millisecond * 100,
	RetentionPeriod:    time.Hour * 24,
}

var DefaultPerformanceThresholds = PerformanceThresholds{
	MaxMemoryUsage:   0.9,
	MaxLatency:       1000.0, // 1000ns
	MinBandwidth:     1.0,    // 1 GB/s
	MaxRemoteRatio:   0.3,
	MinLocalityRatio: 0.7,
	MaxMigrationRate: 0.1,
}

// Constructor functions.

// NewNUMAOptimizer creates a new NUMA optimizer.
func NewNUMAOptimizer(config NUMAConfig) (*NUMAOptimizer, error) {
	optimizer := &NUMAOptimizer{
		nodes:       make(map[NodeID]*NUMANode),
		allocations: make(map[WorkloadID]*NUMAAllocation),
		config:      config,
		enabled:     config.EnableOptimization,
		stopChan:    make(chan struct{}),
	}

	// Discover NUMA topology.
	topology, err := optimizer.discoverTopology()
	if err != nil {
		return nil, fmt.Errorf("failed to discover NUMA topology: %w", err)
	}

	optimizer.topology = topology

	// Initialize load balancer.
	if config.EnableBalancing {
		optimizer.balancer = NewLoadBalancer(config)
	}

	// Initialize performance monitor.
	optimizer.monitor = NewPerformanceMonitor(config, DefaultPerformanceThresholds)

	// Initialize NUMA nodes.
	for i := uint32(0); i < topology.NodeCount; i++ {
		nodeID := NodeID(i)
		node := &NUMANode{
			ID:          nodeID,
			Processors:  make([]ProcessorID, 0),
			Allocations: make(map[WorkloadID]*NUMAAllocation),
			Distance:    make(map[NodeID]uint32),
			LastUpdate:  time.Now(),
		}
		optimizer.nodes[nodeID] = node
	}

	// Start background tasks.
	if optimizer.enabled {
		go optimizer.runBackgroundTasks()
		optimizer.running = true
	}

	return optimizer, nil
}

// NewLoadBalancer creates a new load balancer.
func NewLoadBalancer(config NUMAConfig) *LoadBalancer {
	return &LoadBalancer{
		strategies:     []NUMABalancingStrategy{config.PreferredStrategy, config.FallbackStrategy},
		policies:       make(map[NodeID]BalancingPolicy),
		rebalanceQueue: make(chan *RebalanceRequest, 1000),
		migrationCost:  make(map[NodeID]map[NodeID]float64),
		enabled:        config.EnableBalancing,
	}
}

// NewPerformanceMonitor creates a new performance monitor.
func NewPerformanceMonitor(config NUMAConfig, thresholds PerformanceThresholds) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:         make(map[NodeID]*NodeMetrics),
		samples:         make([]PerformanceSample, 0, 10000),
		thresholds:      thresholds,
		alerts:          make([]PerformanceAlert, 0),
		collectors:      make([]NUMAMetricsCollector, 0),
		samplingRate:    config.MonitoringInterval,
		retentionPeriod: config.RetentionPeriod,
		enabled:         true,
	}
}

// Core allocation methods.

// AllocateNUMAAware allocates memory with NUMA awareness.
func (no *NUMAOptimizer) AllocateNUMAAware(size uint64, workloadID WorkloadID, hint AllocationHint) (*NUMAAllocation, error) {
	if !no.enabled {
		return nil, fmt.Errorf("NUMA optimizer is disabled")
	}

	no.mutex.Lock()
	defer no.mutex.Unlock()

	// Select optimal NUMA node.
	nodeID, err := no.selectOptimalNode(size, hint)
	if err != nil {
		return nil, fmt.Errorf("failed to select NUMA node: %w", err)
	}

	// Perform allocation.
	allocation := &NUMAAllocation{
		ID:          workloadID,
		Size:        size,
		OwnerNode:   nodeID,
		Priority:    hint.Priority,
		CreateTime:  time.Now(),
		LastAccess:  time.Now(),
		Pinned:      hint.Pinned,
		Interleaved: hint.Interleaved,
	}

	// Add to tracking.
	no.allocations[workloadID] = allocation
	no.nodes[nodeID].Allocations[workloadID] = allocation

	// Update statistics.
	atomic.AddUint64(&no.statistics.TotalAllocations, 1)

	if no.isLocalAllocation(allocation) {
		atomic.AddUint64(&no.statistics.LocalAllocations, 1)
	} else {
		atomic.AddUint64(&no.statistics.RemoteAllocations, 1)
	}

	return allocation, nil
}

// DeallocateNUMAAware deallocates NUMA-aware memory.
func (no *NUMAOptimizer) DeallocateNUMAAware(workloadID WorkloadID) error {
	if !no.enabled {
		return fmt.Errorf("NUMA optimizer is disabled")
	}

	no.mutex.Lock()
	defer no.mutex.Unlock()

	allocation, exists := no.allocations[workloadID]
	if !exists {
		return fmt.Errorf("allocation not found: %d", workloadID)
	}

	// Remove from tracking.
	delete(no.allocations, workloadID)
	delete(no.nodes[allocation.OwnerNode].Allocations, workloadID)

	return nil
}

// MigrateAllocation migrates an allocation between NUMA nodes.
func (no *NUMAOptimizer) MigrateAllocation(workloadID WorkloadID, targetNode NodeID, strategy MigrationStrategy) error {
	if !no.enabled || !no.config.EnableMigration {
		return fmt.Errorf("migration is disabled")
	}

	no.mutex.RLock()
	allocation, exists := no.allocations[workloadID]
	no.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("allocation not found: %d", workloadID)
	}

	// Calculate migration cost.
	cost := no.calculateMigrationCost(allocation.OwnerNode, targetNode, allocation.Size)
	if cost > no.config.MaxMigrationCost {
		return fmt.Errorf("migration cost too high: %.2f", cost)
	}

	// Perform migration.
	err := no.performMigration(allocation, targetNode, strategy)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Update statistics.
	atomic.AddUint64(&no.statistics.Migrations, 1)

	return nil
}

// Optimization methods.

// OptimizeMemoryPlacement optimizes memory placement based on access patterns.
func (no *NUMAOptimizer) OptimizeMemoryPlacement() error {
	if !no.enabled {
		return nil
	}

	no.mutex.RLock()

	allocations := make([]*NUMAAllocation, 0, len(no.allocations))
	for _, allocation := range no.allocations {
		allocations = append(allocations, allocation)
	}
	no.mutex.RUnlock()

	// Analyze access patterns and optimize placement.
	for _, allocation := range allocations {
		optimalNode := no.findOptimalNodeForAllocation(allocation)
		if optimalNode != allocation.OwnerNode {
			// Consider migration.
			if no.shouldMigrate(allocation, optimalNode) {
				err := no.MigrateAllocation(allocation.ID, optimalNode, Deferred)
				if err != nil {
					// Log migration failure but continue.
					continue
				}
			}
		}
	}

	return nil
}

// RebalanceLoad rebalances memory allocations across NUMA nodes.
func (no *NUMAOptimizer) RebalanceLoad() error {
	if !no.enabled || no.balancer == nil {
		return nil
	}

	return no.balancer.rebalanceNodes(no.nodes)
}

// Helper methods.

// discoverTopology discovers the NUMA topology of the system.
func (no *NUMAOptimizer) discoverTopology() (*NUMATopology, error) {
	// Platform-specific topology discovery would go here.
	// For now, create a simple 2-node topology.
	topology := &NUMATopology{
		NodeCount:       2,
		ProcessorCount:  8,
		TotalMemory:     16 * 1024 * 1024 * 1024, // 16GB
		DistanceMatrix:  make([][]uint32, 2),
		BandwidthMatrix: make([][]BandwidthMetric, 2),
		LatencyMatrix:   make([][]LatencyMetric, 2),
		HotPathNodes:    []NodeID{0},
		ColdPathNodes:   []NodeID{1},
		LastDiscovery:   time.Now(),
	}

	// Initialize matrices.
	for i := 0; i < 2; i++ {
		topology.DistanceMatrix[i] = make([]uint32, 2)
		topology.BandwidthMatrix[i] = make([]BandwidthMetric, 2)
		topology.LatencyMatrix[i] = make([]LatencyMetric, 2)

		for j := 0; j < 2; j++ {
			if i == j {
				topology.DistanceMatrix[i][j] = 10    // Local
				topology.BandwidthMatrix[i][j] = 12.8 // 12.8 GB/s
				topology.LatencyMatrix[i][j] = 100    // 100ns
			} else {
				topology.DistanceMatrix[i][j] = 20   // Remote
				topology.BandwidthMatrix[i][j] = 6.4 // 6.4 GB/s
				topology.LatencyMatrix[i][j] = 200   // 200ns
			}
		}
	}

	return topology, nil
}

// selectOptimalNode selects the optimal NUMA node for allocation.
func (no *NUMAOptimizer) selectOptimalNode(size uint64, hint AllocationHint) (NodeID, error) {
	bestNode := NodeID(0)
	bestScore := float64(-1)

	for nodeID, node := range no.nodes {
		score := no.calculateNodeScore(node, size, hint)
		if score > bestScore {
			bestScore = score
			bestNode = nodeID
		}
	}

	return bestNode, nil
}

// calculateNodeScore calculates a score for node suitability.
func (no *NUMAOptimizer) calculateNodeScore(node *NUMANode, size uint64, hint AllocationHint) float64 {
	// Base score from available memory.
	availableRatio := float64(node.AvailableMemory) / float64(node.MemorySize)
	score := availableRatio * 100

	// Adjust for load.
	loadPenalty := node.Load * 50
	score -= loadPenalty

	// Adjust for hint preferences.
	if hint.PreferredNode == node.ID {
		score += 50
	}

	// Locality bonus.
	localityBonus := (1.0 - no.calculateRemoteAccessRatio(node)) * 25
	score += localityBonus

	return score
}

// isLocalAllocation checks if allocation is local to its node.
func (no *NUMAOptimizer) isLocalAllocation(allocation *NUMAAllocation) bool {
	// In a real implementation, this would check processor affinity.
	return true // Simplified assumption
}

// calculateMigrationCost calculates the cost of migrating between nodes.
func (no *NUMAOptimizer) calculateMigrationCost(sourceNode, targetNode NodeID, size uint64) float64 {
	if sourceNode == targetNode {
		return 0.0
	}

	// Base cost from distance and size.
	distance := float64(no.topology.DistanceMatrix[sourceNode][targetNode])
	sizeCost := float64(size) / (1024 * 1024) // Cost per MB

	return distance * sizeCost * 0.1
}

// findOptimalNodeForAllocation finds the optimal node for an allocation.
func (no *NUMAOptimizer) findOptimalNodeForAllocation(allocation *NUMAAllocation) NodeID {
	// Analyze access pattern and find best node.
	// For now, return current node (no change).
	return allocation.OwnerNode
}

// shouldMigrate determines if an allocation should be migrated.
func (no *NUMAOptimizer) shouldMigrate(allocation *NUMAAllocation, targetNode NodeID) bool {
	if allocation.Pinned {
		return false
	}

	cost := no.calculateMigrationCost(allocation.OwnerNode, targetNode, allocation.Size)

	return cost <= no.config.MaxMigrationCost
}

// calculateRemoteAccessRatio calculates the ratio of remote memory accesses.
func (no *NUMAOptimizer) calculateRemoteAccessRatio(node *NUMANode) float64 {
	// This would be calculated from actual hardware counters.
	return 0.2 // 20% remote accesses (example)
}

// performMigration performs the actual memory migration.
func (no *NUMAOptimizer) performMigration(allocation *NUMAAllocation, targetNode NodeID, strategy MigrationStrategy) error {
	// Remove from source node.
	no.mutex.Lock()
	delete(no.nodes[allocation.OwnerNode].Allocations, allocation.ID)

	// Update allocation.
	allocation.OwnerNode = targetNode
	allocation.LastAccess = time.Now()

	// Add to target node.
	no.nodes[targetNode].Allocations[allocation.ID] = allocation
	no.mutex.Unlock()

	return nil
}

// Background tasks.

// runBackgroundTasks runs background optimization tasks.
func (no *NUMAOptimizer) runBackgroundTasks() {
	optimizeTicker := time.NewTicker(no.config.BalancingInterval)
	monitorTicker := time.NewTicker(no.config.MonitoringInterval)

	defer optimizeTicker.Stop()
	defer monitorTicker.Stop()

	for {
		select {
		case <-no.stopChan:
			return
		case <-optimizeTicker.C:
			no.OptimizeMemoryPlacement()
			no.RebalanceLoad()
		case <-monitorTicker.C:
			no.updateMetrics()
		}
	}
}

// updateMetrics updates performance metrics.
func (no *NUMAOptimizer) updateMetrics() {
	if no.monitor == nil {
		return
	}

	for nodeID, node := range no.nodes {
		metrics := &NodeMetrics{
			MemoryUsage:    float64(node.MemorySize-node.AvailableMemory) / float64(node.MemorySize),
			BandwidthUtil:  0.5, // Example value
			LatencyAverage: node.Latency,
			CacheHitRate:   0.9, // Example value
			Timestamp:      time.Now(),
		}

		no.monitor.updateNodeMetrics(nodeID, metrics)

		node.LastUpdate = time.Now()
	}
}

// Load balancer implementation.

// rebalanceNodes rebalances allocations across nodes.
func (lb *LoadBalancer) rebalanceNodes(nodes map[NodeID]*NUMANode) error {
	if !lb.enabled {
		return nil
	}

	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Calculate load imbalance.
	loads := make(map[NodeID]float64)
	totalLoad := 0.0

	for nodeID, node := range nodes {
		load := node.Load
		loads[nodeID] = load
		totalLoad += load
	}

	avgLoad := totalLoad / float64(len(nodes))

	// Find overloaded and underloaded nodes.
	overloaded := make([]NodeID, 0)
	underloaded := make([]NodeID, 0)

	for nodeID, load := range loads {
		if load > avgLoad*1.2 {
			overloaded = append(overloaded, nodeID)
		} else if load < avgLoad*0.8 {
			underloaded = append(underloaded, nodeID)
		}
	}

	// Create rebalance requests.
	for _, sourceNode := range overloaded {
		for _, targetNode := range underloaded {
			// Find suitable allocation to move.
			node := nodes[sourceNode]
			for _, allocation := range node.Allocations {
				if !allocation.Pinned {
					request := &RebalanceRequest{
						ID:         uint64(time.Now().UnixNano()),
						SourceNode: sourceNode,
						TargetNode: targetNode,
						Allocation: allocation,
						Strategy:   LoadAware,
						Priority:   1,
						Reason:     "Load balancing",
						Timestamp:  time.Now(),
					}

					select {
					case lb.rebalanceQueue <- request:
						// Request queued.
					default:
						// Queue full, skip.
					}

					break
				}
			}
		}
	}

	return nil
}

// Performance monitor implementation.

// updateNodeMetrics updates metrics for a specific node.
func (pm *PerformanceMonitor) updateNodeMetrics(nodeID NodeID, metrics *NodeMetrics) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.metrics[nodeID] = metrics

	// Add performance sample.
	sample := PerformanceSample{
		NodeID:          nodeID,
		Timestamp:       metrics.Timestamp,
		MemoryBandwidth: BandwidthMetric(10.0), // Example
		AccessLatency:   metrics.LatencyAverage,
		LocalityRatio:   0.8, // Example
		CacheEfficiency: metrics.CacheHitRate,
	}

	pm.samples = append(pm.samples, sample)

	// Check thresholds and generate alerts.
	pm.checkThresholds(nodeID, metrics)

	// Cleanup old samples.
	pm.cleanupOldSamples()
}

// checkThresholds checks performance thresholds and generates alerts.
func (pm *PerformanceMonitor) checkThresholds(nodeID NodeID, metrics *NodeMetrics) {
	// Memory usage alert.
	if metrics.MemoryUsage > pm.thresholds.MaxMemoryUsage {
		alert := PerformanceAlert{
			ID:        uint64(time.Now().UnixNano()),
			NodeID:    nodeID,
			Type:      MemoryUsageAlert,
			Severity:  NUMAWarning,
			Message:   fmt.Sprintf("High memory usage: %.2f%%", metrics.MemoryUsage*100),
			Threshold: pm.thresholds.MaxMemoryUsage,
			Current:   metrics.MemoryUsage,
			Timestamp: time.Now(),
		}
		pm.alerts = append(pm.alerts, alert)
	}

	// Latency alert.
	if metrics.LatencyAverage > pm.thresholds.MaxLatency {
		alert := PerformanceAlert{
			ID:        uint64(time.Now().UnixNano()),
			NodeID:    nodeID,
			Type:      LatencyAlert,
			Severity:  NUMAError,
			Message:   fmt.Sprintf("High latency: %.2fns", float64(metrics.LatencyAverage)),
			Threshold: float64(pm.thresholds.MaxLatency),
			Current:   float64(metrics.LatencyAverage),
			Timestamp: time.Now(),
		}
		pm.alerts = append(pm.alerts, alert)
	}
}

// cleanupOldSamples removes old performance samples.
func (pm *PerformanceMonitor) cleanupOldSamples() {
	cutoff := time.Now().Add(-pm.retentionPeriod)
	validSamples := make([]PerformanceSample, 0, len(pm.samples))

	for _, sample := range pm.samples {
		if sample.Timestamp.After(cutoff) {
			validSamples = append(validSamples, sample)
		}
	}

	pm.samples = validSamples
}

// Cleanup and shutdown.

// Stop stops the NUMA optimizer.
func (no *NUMAOptimizer) Stop() error {
	if !no.running {
		return nil
	}

	close(no.stopChan)
	no.running = false
	no.enabled = false

	return nil
}

// GetStatistics returns current NUMA statistics.
func (no *NUMAOptimizer) GetStatistics() NUMAStatistics {
	// Update locality ratio.
	localRatio := float64(atomic.LoadUint64(&no.statistics.LocalAllocations)) /
		float64(atomic.LoadUint64(&no.statistics.TotalAllocations))
	no.statistics.LocalityRatio = localRatio

	return no.statistics
}

// GetTopology returns the current NUMA topology.
func (no *NUMAOptimizer) GetTopology() *NUMATopology {
	return no.topology
}
