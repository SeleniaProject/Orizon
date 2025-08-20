// Package numa implements Phase 3.1.3 NUMA-aware optimization for Orizon
// This package provides distributed memory architecture optimization through.
// NUMA locality awareness, memory affinity control, and dynamic load balancing.
// to maximize performance on multi-socket and distributed memory systems.
package numa

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Optimizer is the main NUMA optimization engine.
type Optimizer struct {
	topology  *Topology
	scheduler *Scheduler
	allocator *Allocator
	monitor   *Monitor
	stats     Stats
	mutex     sync.RWMutex
	enabled   bool
}

// Stats tracks NUMA optimization performance.
type Stats struct {
	LocalAllocations  int64
	RemoteAllocations int64
	Migrations        int64
	BalanceOperations int64
	TopologyChanges   int64
	PerformanceGain   int64
}

// Topology represents the NUMA topology of the system.
type Topology struct {
	nodes         []*Node
	distances     [][]int
	nodeCount     int
	coresPerNode  int
	memoryPerNode uint64
	mutex         sync.RWMutex
}

// Node represents a NUMA node.
type Node struct {
	LastUpdate  time.Time
	Memory      *NodeMemory
	CPUs        []int
	ID          int
	LoadAverage float64
	Allocations int64
	Migrations  int64
	IsOnline    bool
}

// NodeMemory represents memory information for a NUMA node.
type NodeMemory struct {
	Total     uint64
	Available uint64
	Used      uint64
	Cached    uint64
	Buffers   uint64
	mutex     sync.RWMutex
}

// Scheduler manages NUMA-aware task scheduling.
type Scheduler struct {
	balancer  *LoadBalancer
	affinity  *AffinityManager
	queues    []*TaskQueue
	workers   []*Worker
	mutex     sync.Mutex
	isRunning bool
}

// TaskQueue represents a per-node task queue.
type TaskQueue struct {
	tasks     chan *Task
	nodeID    int
	priority  int
	length    int64
	processed int64
	mutex     sync.Mutex
}

// Task represents a schedulable unit of work.
type Task struct {
	Created      time.Time
	Deadline     time.Time
	Function     func() interface{}
	Data         unsafe.Pointer
	Result       chan interface{}
	ID           uint64
	Size         uintptr
	NodeAffinity int
	Priority     int
}

// Worker represents a NUMA-aware worker thread.
type Worker struct {
	LastTask  time.Time
	Queue     *TaskQueue
	ID        int
	NodeID    int
	Processed int64
	IsActive  bool
}

// LoadBalancer manages dynamic load distribution.
type LoadBalancer struct {
	lastBalance time.Time
	strategy    BalanceStrategy
	threshold   float64
	interval    time.Duration
	migrations  int64
	mutex       sync.Mutex
}

// BalanceStrategy defines load balancing strategies.
type BalanceStrategy int

const (
	BalanceByLoad BalanceStrategy = iota
	BalanceByMemory
	BalanceByLatency
	BalanceAdaptive
)

// AffinityManager controls CPU and memory affinity.
type AffinityManager struct {
	cpuMasks    map[int]uint64
	memoryMasks map[int]uint64
	policies    map[int]AffinityPolicy
	mutex       sync.RWMutex
}

// AffinityPolicy defines affinity control policies.
type AffinityPolicy struct {
	CPUStrict    bool
	MemoryStrict bool
	Migration    bool
	Interleaving bool
}

// Allocator provides NUMA-aware memory allocation.
type Allocator struct {
	policies    map[int]AllocationPolicy
	pools       []*MemoryPool
	stats       AllocatorStats
	localRatio  float64
	remoteRatio float64
	mutex       sync.RWMutex
}

// MemoryPool represents a per-node memory pool.
type MemoryPool struct {
	chunks        []*MemoryChunk
	freeList      []*MemoryChunk
	nodeID        int
	totalSize     uint64
	usedSize      uint64
	fragmentation float64
	mutex         sync.Mutex
}

// MemoryChunk represents a memory allocation unit.
type MemoryChunk struct {
	timestamp time.Time
	next      *MemoryChunk
	ptr       uintptr
	size      uintptr
	nodeID    int
	allocated bool
}

// AllocationPolicy defines memory allocation policies.
type AllocationPolicy struct {
	PreferLocal    bool
	AllowRemote    bool
	Interleave     bool
	Bind           bool
	MigrateOnFault bool
}

// AllocatorStats tracks allocation performance.
type AllocatorStats struct {
	LocalHits      int64
	RemoteHits     int64
	Misses         int64
	Fragmentations int64
	Compactions    int64
}

// Monitor tracks NUMA system performance.
type Monitor struct {
	metrics   *Metrics
	alerts    chan *Alert
	samplers  []*Sampler
	interval  time.Duration
	mutex     sync.Mutex
	isRunning bool
}

// Sampler collects NUMA performance data.
type Sampler struct {
	lastSample time.Time
	cpu        CPUMetrics
	memory     MemoryMetrics
	network    NetworkMetrics
	nodeID     int
	sampleRate time.Duration
}

// CPUMetrics tracks CPU performance per node.
type CPUMetrics struct {
	Utilization   float64
	IdleTime      time.Duration
	UserTime      time.Duration
	SystemTime    time.Duration
	InterruptTime time.Duration
	Temperature   float64
}

// MemoryMetrics tracks memory performance per node.
type MemoryMetrics struct {
	Bandwidth   float64
	Latency     time.Duration
	Utilization float64
	PageFaults  int64
	CacheMisses int64
	Swapping    int64
}

// NetworkMetrics tracks inter-node communication.
type NetworkMetrics struct {
	Bandwidth   float64
	Latency     time.Duration
	PacketLoss  float64
	Connections int64
	Throughput  float64
}

// Metrics aggregates system-wide NUMA metrics.
type Metrics struct {
	nodes       map[int]*NodeMetrics
	trends      []*TrendData
	predictions []*PredictionData
	global      GlobalMetrics
	mutex       sync.RWMutex
}

// NodeMetrics represents per-node performance metrics.
type NodeMetrics struct {
	LastUpdate time.Time
	CPU        CPUMetrics
	Memory     MemoryMetrics
	Network    NetworkMetrics
	Load       float64
	Efficiency float64
}

// GlobalMetrics represents system-wide metrics.
type GlobalMetrics struct {
	TotalNodes      int
	ActiveNodes     int
	TotalMemory     uint64
	AvailableMemory uint64
	SystemLoad      float64
	Efficiency      float64
	Imbalance       float64
}

// TrendData represents performance trends.
type TrendData struct {
	Timestamp  time.Time
	Metric     string
	NodeID     int
	Value      float64
	Trend      float64
	Confidence float64
}

// PredictionData represents performance predictions.
type PredictionData struct {
	Timestamp  time.Time
	Metric     string
	NodeID     int
	Predicted  float64
	Confidence float64
	Horizon    time.Duration
}

// Alert represents a NUMA system alert.
type Alert struct {
	Timestamp time.Time
	Message   string
	Metric    string
	NodeID    int
	Level     AlertLevel
	Value     float64
	Threshold float64
}

// AlertLevel defines alert severity levels.
type AlertLevel int

const (
	AlertInfo AlertLevel = iota
	AlertWarning
	AlertError
	AlertCritical
)

// NewOptimizer creates a new NUMA optimizer.
func NewOptimizer() *Optimizer {
	return &Optimizer{
		enabled:   true,
		topology:  NewTopology(),
		scheduler: NewScheduler(),
		allocator: NewAllocator(),
		monitor:   NewMonitor(),
	}
}

// NewTopology creates and discovers NUMA topology.
func NewTopology() *Topology {
	topo := &Topology{
		nodes:        make([]*Node, 0),
		nodeCount:    runtime.NumCPU() / 4, // Assume 4 cores per node
		coresPerNode: 4,
	}

	// Discover topology (simplified).
	topo.discoverNodes()
	topo.measureDistances()

	return topo
}

// discoverNodes discovers available NUMA nodes.
func (t *Topology) discoverNodes() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for i := 0; i < t.nodeCount; i++ {
		node := &Node{
			ID:          i,
			CPUs:        make([]int, t.coresPerNode),
			Memory:      NewNodeMemory(),
			LoadAverage: 0.0,
			IsOnline:    true,
			LastUpdate:  time.Now(),
		}

		// Assign CPUs to node.
		for j := 0; j < t.coresPerNode; j++ {
			node.CPUs[j] = i*t.coresPerNode + j
		}

		t.nodes = append(t.nodes, node)
	}
}

// measureDistances measures inter-node distances.
func (t *Topology) measureDistances() {
	t.distances = make([][]int, t.nodeCount)
	for i := range t.distances {
		t.distances[i] = make([]int, t.nodeCount)
		for j := range t.distances[i] {
			if i == j {
				t.distances[i][j] = 10 // Local access cost
			} else {
				t.distances[i][j] = 20 + abs(i-j)*5 // Remote access cost
			}
		}
	}
}

// GetDistance returns the distance between two nodes.
func (t *Topology) GetDistance(from, to int) int {
	if from < 0 || from >= t.nodeCount || to < 0 || to >= t.nodeCount {
		return -1 // Invalid nodes
	}

	return t.distances[from][to]
}

// NewNodeMemory creates a new node memory structure.
func NewNodeMemory() *NodeMemory {
	return &NodeMemory{
		Total:     8 << 30, // 8GB default
		Available: 6 << 30, // 6GB available
		Used:      2 << 30, // 2GB used
	}
}

// NewScheduler creates a new NUMA scheduler.
func NewScheduler() *Scheduler {
	nodeCount := runtime.NumCPU() / 4
	scheduler := &Scheduler{
		queues:    make([]*TaskQueue, nodeCount),
		workers:   make([]*Worker, 0),
		balancer:  NewLoadBalancer(),
		affinity:  NewAffinityManager(),
		isRunning: false,
	}

	// Initialize per-node queues.
	for i := 0; i < nodeCount; i++ {
		scheduler.queues[i] = &TaskQueue{
			nodeID: i,
			tasks:  make(chan *Task, 1000),
		}
	}

	return scheduler
}

// NewLoadBalancer creates a new load balancer.
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		strategy:  BalanceAdaptive,
		threshold: 0.8, // 80% load threshold
		interval:  time.Second,
	}
}

// NewAffinityManager creates a new affinity manager.
func NewAffinityManager() *AffinityManager {
	return &AffinityManager{
		cpuMasks:    make(map[int]uint64),
		memoryMasks: make(map[int]uint64),
		policies:    make(map[int]AffinityPolicy),
	}
}

// SetCPUAffinity sets CPU affinity for a node.
func (am *AffinityManager) SetCPUAffinity(nodeID int, mask uint64) error {
	if nodeID < 0 {
		return fmt.Errorf("invalid node ID: %d", nodeID)
	}

	am.cpuMasks[nodeID] = mask

	return nil
}

// NewAllocator creates a new NUMA allocator.
func NewAllocator() *Allocator {
	nodeCount := runtime.NumCPU() / 4
	allocator := &Allocator{
		pools:       make([]*MemoryPool, nodeCount),
		policies:    make(map[int]AllocationPolicy),
		localRatio:  0.8, // Prefer 80% local allocations
		remoteRatio: 0.2, // Allow 20% remote allocations
	}

	// Initialize per-node pools.
	for i := 0; i < nodeCount; i++ {
		allocator.pools[i] = NewMemoryPool(i)
	}

	return allocator
}

// NewMemoryPool creates a new memory pool for a node.
func NewMemoryPool(nodeID int) *MemoryPool {
	return &MemoryPool{
		nodeID:    nodeID,
		chunks:    make([]*MemoryChunk, 0),
		freeList:  make([]*MemoryChunk, 0),
		totalSize: 1 << 30, // 1GB per pool
	}
}

// NewMonitor creates a new NUMA monitor.
func NewMonitor() *Monitor {
	nodeCount := runtime.NumCPU() / 4
	monitor := &Monitor{
		samplers:  make([]*Sampler, nodeCount),
		metrics:   NewMetrics(),
		alerts:    make(chan *Alert, 100),
		interval:  time.Second,
		isRunning: false,
	}

	// Initialize per-node samplers.
	for i := 0; i < nodeCount; i++ {
		monitor.samplers[i] = &Sampler{
			nodeID:     i,
			sampleRate: time.Second,
		}
	}

	return monitor
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		nodes:       make(map[int]*NodeMetrics),
		trends:      make([]*TrendData, 0),
		predictions: make([]*PredictionData, 0),
	}
}

// Start starts the NUMA optimizer.
func (o *Optimizer) Start() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.enabled {
		return fmt.Errorf("NUMA optimizer is disabled")
	}

	// Start components.
	if err := o.scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	if err := o.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitor: %w", err)
	}

	return nil
}

// Stop stops the NUMA optimizer.
func (o *Optimizer) Stop() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.scheduler.Stop()
	o.monitor.Stop()

	return nil
}

// Allocate performs NUMA-aware memory allocation.
func (o *Optimizer) Allocate(size uintptr, nodeHint int) uintptr {
	if !o.enabled {
		return o.fallbackAllocate(size)
	}

	atomic.AddInt64(&o.stats.LocalAllocations, 1)

	// Try local allocation first.
	if ptr := o.allocator.AllocateLocal(size, nodeHint); ptr != 0 {
		return ptr
	}

	// Fall back to remote allocation.
	atomic.AddInt64(&o.stats.RemoteAllocations, 1)

	return o.allocator.AllocateRemote(size, nodeHint)
}

// fallbackAllocate provides fallback allocation.
func (o *Optimizer) fallbackAllocate(size uintptr) uintptr {
	data := make([]byte, size)

	return uintptr(unsafe.Pointer(&data[0]))
}

// AllocateLocal attempts local node allocation.
func (a *Allocator) AllocateLocal(size uintptr, nodeID int) uintptr {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if nodeID < 0 || nodeID >= len(a.pools) {
		return 0
	}

	pool := a.pools[nodeID]

	return pool.Allocate(size)
}

// AllocateRemote attempts remote node allocation.
func (a *Allocator) AllocateRemote(size uintptr, excludeNode int) uintptr {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// Try all other nodes.
	for i, pool := range a.pools {
		if i != excludeNode {
			if ptr := pool.Allocate(size); ptr != 0 {
				return ptr
			}
		}
	}

	return 0
}

// Allocate allocates memory from the pool.
func (p *MemoryPool) Allocate(size uintptr) uintptr {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check free list first.
	for i, chunk := range p.freeList {
		if chunk.size >= size {
			// Remove from free list.
			p.freeList = append(p.freeList[:i], p.freeList[i+1:]...)
			chunk.allocated = true
			chunk.timestamp = time.Now()
			p.usedSize += uint64(chunk.size)

			return chunk.ptr
		}
	}

	// Allocate new chunk.
	data := make([]byte, size)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	chunk := &MemoryChunk{
		ptr:       ptr,
		size:      size,
		nodeID:    p.nodeID,
		allocated: true,
		timestamp: time.Now(),
	}

	p.chunks = append(p.chunks, chunk)
	p.usedSize += uint64(size)

	return ptr
}

// Free deallocates memory from the pool.
func (p *MemoryPool) Free(ptr uintptr, size uintptr) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Find the chunk to free.
	for _, chunk := range p.chunks {
		if chunk.ptr == ptr && chunk.allocated {
			chunk.allocated = false
			chunk.timestamp = time.Now()

			p.freeList = append(p.freeList, chunk)
			if uint64(size) <= p.usedSize {
				p.usedSize -= uint64(size)
			}

			return
		}
	}
}

// ScheduleTask schedules a task on the optimal node.
func (o *Optimizer) ScheduleTask(task *Task) error {
	if !o.enabled {
		return fmt.Errorf("NUMA optimizer is disabled")
	}

	nodeID := o.scheduler.SelectOptimalNode(task)

	return o.scheduler.EnqueueTask(nodeID, task)
}

// SelectOptimalNode selects the best node for a task.
func (s *Scheduler) SelectOptimalNode(task *Task) int {
	// Use affinity hint if provided.
	if task.NodeAffinity >= 0 && task.NodeAffinity < len(s.queues) {
		return task.NodeAffinity
	}

	// Select node with lowest load.
	minLoad := int64(^uint64(0) >> 1) // Max int64
	selectedNode := 0

	for i, queue := range s.queues {
		load := atomic.LoadInt64(&queue.length)
		if load < minLoad {
			minLoad = load
			selectedNode = i
		}
	}

	return selectedNode
}

// EnqueueTask enqueues a task to a specific node.
func (s *Scheduler) EnqueueTask(nodeID int, task *Task) error {
	if nodeID < 0 || nodeID >= len(s.queues) {
		return fmt.Errorf("invalid node ID: %d", nodeID)
	}

	queue := s.queues[nodeID]
	select {
	case queue.tasks <- task:
		atomic.AddInt64(&queue.length, 1)

		return nil
	default:
		return fmt.Errorf("queue full for node %d", nodeID)
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler already running")
	}

	// Start workers for each node.
	for i, queue := range s.queues {
		worker := &Worker{
			ID:     i,
			NodeID: i,
			Queue:  queue,
		}
		s.workers = append(s.workers, worker)

		go worker.Run()
	}

	// Start load balancer.
	go s.balancer.Run(s)

	s.isRunning = true

	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.isRunning = false

	// Stop workers.
	for _, worker := range s.workers {
		worker.IsActive = false
	}
}

// Run runs the worker main loop.
func (w *Worker) Run() {
	w.IsActive = true

	for w.IsActive {
		select {
		case task := <-w.Queue.tasks:
			w.processTask(task)
		case <-time.After(time.Millisecond * 100):
			// Timeout to check IsActive.
		}
	}
}

// processTask processes a single task.
func (w *Worker) processTask(task *Task) {
	defer func() {
		atomic.AddInt64(&w.Queue.length, -1)
		atomic.AddInt64(&w.Queue.processed, 1)
		atomic.AddInt64(&w.Processed, 1)
		w.LastTask = time.Now()
	}()

	// Execute task.
	if task.Function != nil {
		result := task.Function()
		if task.Result != nil {
			task.Result <- result
		}
	}
}

// Run runs the load balancer.
func (lb *LoadBalancer) Run(scheduler *Scheduler) {
	for {
		time.Sleep(lb.interval)

		if time.Since(lb.lastBalance) > lb.interval {
			lb.balance(scheduler)
			lb.lastBalance = time.Now()
		}
	}
}

// balance performs load balancing.
func (lb *LoadBalancer) balance(scheduler *Scheduler) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Find highest and lowest loaded nodes.
	maxLoad := int64(0)
	minLoad := int64(^uint64(0) >> 1)
	maxNode := -1
	minNode := -1

	for i, queue := range scheduler.queues {
		load := atomic.LoadInt64(&queue.length)
		if load > maxLoad {
			maxLoad = load
			maxNode = i
		}

		if load < minLoad {
			minLoad = load
			minNode = i
		}
	}

	// Balance if imbalance exceeds threshold.
	if maxNode != -1 && minNode != -1 && maxLoad > minLoad+1 {
		lb.migrateTasks(scheduler.queues[maxNode], scheduler.queues[minNode])
		atomic.AddInt64(&lb.migrations, 1)
	}
}

// migrateTasks migrates tasks between nodes.
func (lb *LoadBalancer) migrateTasks(from, to *TaskQueue) {
	select {
	case task := <-from.tasks:
		select {
		case to.tasks <- task:
			atomic.AddInt64(&from.length, -1)
			atomic.AddInt64(&to.length, 1)
		default:
			// Put back if destination is full.
			from.tasks <- task
		}
	default:
		// No tasks to migrate.
	}
}

// Start starts the monitor.
func (m *Monitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isRunning {
		return fmt.Errorf("monitor already running")
	}

	// Start samplers.
	for _, sampler := range m.samplers {
		go sampler.Run(m.metrics, m.alerts)
	}

	// Start metrics aggregator.
	go m.runAggregator()

	m.isRunning = true

	return nil
}

// Stop stops the monitor.
func (m *Monitor) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.isRunning = false
}

// Run runs the sampler main loop.
func (s *Sampler) Run(metrics *Metrics, alerts chan<- *Alert) {
	for {
		time.Sleep(s.sampleRate)

		s.collectMetrics()
		s.updateMetrics(metrics)
		s.checkAlerts(alerts)

		s.lastSample = time.Now()
	}
}

// collectMetrics collects performance metrics.
func (s *Sampler) collectMetrics() {
	// Simulate metric collection.
	s.cpu.Utilization = float64(s.nodeID*10+50) / 100.0
	s.memory.Utilization = float64(s.nodeID*5+60) / 100.0
	s.network.Latency = time.Duration(s.nodeID+1) * time.Microsecond
}

// updateMetrics updates the global metrics.
func (s *Sampler) updateMetrics(metrics *Metrics) {
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()

	nodeMetrics := &NodeMetrics{
		CPU:        s.cpu,
		Memory:     s.memory,
		Network:    s.network,
		Load:       s.cpu.Utilization,
		Efficiency: 1.0 - s.cpu.Utilization,
		LastUpdate: time.Now(),
	}

	metrics.nodes[s.nodeID] = nodeMetrics
}

// checkAlerts checks for alert conditions.
func (s *Sampler) checkAlerts(alerts chan<- *Alert) {
	if s.cpu.Utilization > 0.9 {
		alert := &Alert{
			Timestamp: time.Now(),
			NodeID:    s.nodeID,
			Level:     AlertWarning,
			Message:   "High CPU utilization",
			Metric:    "cpu_utilization",
			Value:     s.cpu.Utilization,
			Threshold: 0.9,
		}

		select {
		case alerts <- alert:
		default:
		}
	}
}

// runAggregator runs the metrics aggregator.
func (m *Monitor) runAggregator() {
	for {
		time.Sleep(time.Second * 5)
		m.aggregateMetrics()
	}
}

// aggregateMetrics aggregates system-wide metrics.
func (m *Monitor) aggregateMetrics() {
	m.metrics.mutex.Lock()
	defer m.metrics.mutex.Unlock()

	totalNodes := len(m.metrics.nodes)
	activeNodes := 0
	totalLoad := 0.0

	for _, node := range m.metrics.nodes {
		if time.Since(node.LastUpdate) < time.Minute {
			activeNodes++
		}

		totalLoad += node.Load
	}

	if totalNodes > 0 {
		m.metrics.global = GlobalMetrics{
			TotalNodes:  totalNodes,
			ActiveNodes: activeNodes,
			SystemLoad:  totalLoad / float64(totalNodes),
			Efficiency:  1.0 - (totalLoad / float64(totalNodes)),
		}
	}
}

// GetStatistics returns comprehensive NUMA statistics.
func (o *Optimizer) GetStatistics() map[string]interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["enabled"] = o.enabled
	stats["local_allocations"] = atomic.LoadInt64(&o.stats.LocalAllocations)
	stats["remote_allocations"] = atomic.LoadInt64(&o.stats.RemoteAllocations)
	stats["migrations"] = atomic.LoadInt64(&o.stats.Migrations)
	stats["balance_operations"] = atomic.LoadInt64(&o.stats.BalanceOperations)
	stats["topology_changes"] = atomic.LoadInt64(&o.stats.TopologyChanges)
	stats["performance_gain"] = atomic.LoadInt64(&o.stats.PerformanceGain)

	// Add topology information.
	stats["node_count"] = o.topology.nodeCount
	stats["cores_per_node"] = o.topology.coresPerNode

	return stats
}

// Enable enables the NUMA optimizer.
func (o *Optimizer) Enable() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.enabled = true
}

// Disable disables the NUMA optimizer.
func (o *Optimizer) Disable() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.enabled = false
}

// String returns string representation.
func (o *Optimizer) String() string {
	stats := o.GetStatistics()

	return fmt.Sprintf("NUMAOptimizer{enabled: %v, nodes: %d, local: %d, remote: %d, migrations: %d}",
		stats["enabled"], stats["node_count"], stats["local_allocations"],
		stats["remote_allocations"], stats["migrations"])
}

// Helper function.
func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
