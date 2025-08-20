// Package numa tests - Comprehensive test suite for NUMA optimization Phase 3.1.3
package numa

import (
	"sync"
	"testing"
	"time"
)

func TestOptimizer_Creation(t *testing.T) {
	optimizer := NewOptimizer()

	if optimizer == nil {
		t.Fatal("Failed to create NUMA optimizer")
	}

	if !optimizer.enabled {
		t.Error("Optimizer should be enabled by default")
	}

	if optimizer.topology == nil {
		t.Error("Topology should be initialized")
	}

	if optimizer.scheduler == nil {
		t.Error("Scheduler should be initialized")
	}

	if optimizer.allocator == nil {
		t.Error("Allocator should be initialized")
	}

	if optimizer.monitor == nil {
		t.Error("Monitor should be initialized")
	}
}

func TestOptimizer_EnableDisable(t *testing.T) {
	optimizer := NewOptimizer()

	// Test disable.
	optimizer.Disable()

	if optimizer.enabled {
		t.Error("Optimizer should be disabled")
	}

	// Test enable.
	optimizer.Enable()

	if !optimizer.enabled {
		t.Error("Optimizer should be enabled")
	}
}

func TestTopology_Discovery(t *testing.T) {
	topology := NewTopology()

	if topology.nodeCount <= 0 {
		t.Error("Should discover at least one NUMA node")
	}

	if len(topology.nodes) != topology.nodeCount {
		t.Error("Node count mismatch")
	}

	// Check nodes are properly initialized.
	for i, node := range topology.nodes {
		if node.ID != i {
			t.Errorf("Node ID mismatch: expected %d, got %d", i, node.ID)
		}

		if len(node.CPUs) != topology.coresPerNode {
			t.Errorf("Node %d CPU count mismatch: expected %d, got %d",
				i, topology.coresPerNode, len(node.CPUs))
		}

		if node.Memory == nil {
			t.Errorf("Node %d memory not initialized", i)
		}

		if !node.IsOnline {
			t.Errorf("Node %d should be online", i)
		}
	}
}

func TestTopology_Distances(t *testing.T) {
	topology := NewTopology()

	if len(topology.distances) != topology.nodeCount {
		t.Error("Distance matrix size mismatch")
	}

	for i := 0; i < topology.nodeCount; i++ {
		if len(topology.distances[i]) != topology.nodeCount {
			t.Errorf("Distance matrix row %d size mismatch", i)
		}

		// Check diagonal (local access).
		if topology.distances[i][i] != 10 {
			t.Errorf("Local access cost should be 10, got %d", topology.distances[i][i])
		}

		// Check symmetry.
		for j := 0; j < topology.nodeCount; j++ {
			if topology.distances[i][j] != topology.distances[j][i] {
				t.Errorf("Distance matrix not symmetric at [%d][%d]", i, j)
			}
		}
	}
}

func TestNodeMemory_Initialization(t *testing.T) {
	memory := NewNodeMemory()

	if memory.Total == 0 {
		t.Error("Total memory should be initialized")
	}

	if memory.Available == 0 {
		t.Error("Available memory should be initialized")
	}

	if memory.Available > memory.Total {
		t.Error("Available memory cannot exceed total memory")
	}

	if memory.Used > memory.Total {
		t.Error("Used memory cannot exceed total memory")
	}
}

func TestAllocator_LocalAllocation(t *testing.T) {
	allocator := NewAllocator()

	// Test allocation on each node.
	for i := 0; i < len(allocator.pools); i++ {
		ptr := allocator.AllocateLocal(1024, i)
		if ptr == 0 {
			t.Errorf("Local allocation failed on node %d", i)
		}
	}
}

func TestAllocator_RemoteAllocation(t *testing.T) {
	allocator := NewAllocator()

	// Test remote allocation.
	ptr := allocator.AllocateRemote(1024, 0) // Exclude node 0
	if ptr == 0 && len(allocator.pools) > 1 {
		t.Error("Remote allocation should succeed when multiple nodes available")
	}
}

func TestMemoryPool_Allocation(t *testing.T) {
	pool := NewMemoryPool(0)

	// Test basic allocation.
	ptr1 := pool.Allocate(1024)
	if ptr1 == 0 {
		t.Error("Basic allocation should succeed")
	}

	ptr2 := pool.Allocate(2048)
	if ptr2 == 0 {
		t.Error("Second allocation should succeed")
	}

	if ptr1 == ptr2 {
		t.Error("Different allocations should return different pointers")
	}

	// Check used size tracking.
	expectedUsed := uint64(1024 + 2048)
	if pool.usedSize != expectedUsed {
		t.Errorf("Used size mismatch: expected %d, got %d", expectedUsed, pool.usedSize)
	}
}

func TestScheduler_Creation(t *testing.T) {
	scheduler := NewScheduler()

	if len(scheduler.queues) == 0 {
		t.Error("Scheduler should have at least one queue")
	}

	if scheduler.balancer == nil {
		t.Error("Load balancer should be initialized")
	}

	if scheduler.affinity == nil {
		t.Error("Affinity manager should be initialized")
	}

	if scheduler.isRunning {
		t.Error("Scheduler should not be running initially")
	}
}

func TestScheduler_NodeSelection(t *testing.T) {
	scheduler := NewScheduler()

	// Test task with affinity hint.
	task := &Task{
		ID:           1,
		NodeAffinity: 0,
		Priority:     1,
		Created:      time.Now(),
	}

	selectedNode := scheduler.SelectOptimalNode(task)
	if selectedNode != 0 {
		t.Errorf("Should select hinted node 0, got %d", selectedNode)
	}

	// Test task without affinity hint.
	task.NodeAffinity = -1

	selectedNode = scheduler.SelectOptimalNode(task)
	if selectedNode < 0 || selectedNode >= len(scheduler.queues) {
		t.Errorf("Selected node %d out of range", selectedNode)
	}
}

func TestScheduler_TaskEnqueue(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		ID:      1,
		Created: time.Now(),
		Result:  make(chan interface{}, 1),
	}

	err := scheduler.EnqueueTask(0, task)
	if err != nil {
		t.Errorf("Task enqueue failed: %v", err)
	}

	// Check queue length.
	if scheduler.queues[0].length != 1 {
		t.Error("Queue length should be 1 after enqueue")
	}

	// Test invalid node.
	err = scheduler.EnqueueTask(-1, task)
	if err == nil {
		t.Error("Should fail for invalid node ID")
	}
}

func TestLoadBalancer_Creation(t *testing.T) {
	balancer := NewLoadBalancer()

	if balancer.strategy != BalanceAdaptive {
		t.Error("Should default to adaptive balancing")
	}

	if balancer.threshold <= 0 || balancer.threshold >= 1 {
		t.Error("Threshold should be between 0 and 1")
	}

	if balancer.interval <= 0 {
		t.Error("Interval should be positive")
	}
}

func TestAffinityManager_Creation(t *testing.T) {
	affinity := NewAffinityManager()

	if affinity.cpuMasks == nil {
		t.Error("CPU masks should be initialized")
	}

	if affinity.memoryMasks == nil {
		t.Error("Memory masks should be initialized")
	}

	if affinity.policies == nil {
		t.Error("Policies should be initialized")
	}
}

func TestMonitor_Creation(t *testing.T) {
	monitor := NewMonitor()

	if len(monitor.samplers) == 0 {
		t.Error("Monitor should have at least one sampler")
	}

	if monitor.metrics == nil {
		t.Error("Metrics should be initialized")
	}

	if monitor.alerts == nil {
		t.Error("Alerts channel should be initialized")
	}

	if monitor.isRunning {
		t.Error("Monitor should not be running initially")
	}
}

func TestMetrics_Creation(t *testing.T) {
	metrics := NewMetrics()

	if metrics.nodes == nil {
		t.Error("Node metrics should be initialized")
	}

	if metrics.trends == nil {
		t.Error("Trends should be initialized")
	}

	if metrics.predictions == nil {
		t.Error("Predictions should be initialized")
	}
}

func TestSampler_MetricCollection(t *testing.T) {
	sampler := &Sampler{
		nodeID:     0,
		sampleRate: time.Millisecond,
	}

	sampler.collectMetrics()

	if sampler.cpu.Utilization < 0 || sampler.cpu.Utilization > 1 {
		t.Error("CPU utilization should be between 0 and 1")
	}

	if sampler.memory.Utilization < 0 || sampler.memory.Utilization > 1 {
		t.Error("Memory utilization should be between 0 and 1")
	}

	if sampler.network.Latency <= 0 {
		t.Error("Network latency should be positive")
	}
}

func TestOptimizer_Allocation(t *testing.T) {
	optimizer := NewOptimizer()

	// Test basic allocation.
	ptr := optimizer.Allocate(1024, 0)
	if ptr == 0 {
		t.Error("Allocation should succeed")
	}

	// Test allocation with different node hints.
	for i := 0; i < optimizer.topology.nodeCount; i++ {
		ptr := optimizer.Allocate(512, i)
		if ptr == 0 {
			t.Errorf("Allocation with node hint %d should succeed", i)
		}
	}
}

func TestOptimizer_TaskScheduling(t *testing.T) {
	optimizer := NewOptimizer()

	// Start the optimizer.
	err := optimizer.Start()
	if err != nil {
		t.Fatalf("Failed to start optimizer: %v", err)
	}
	defer optimizer.Stop()

	// Create and schedule a task.
	executed := false
	task := &Task{
		ID: 1,
		Function: func() interface{} {
			executed = true

			return "success"
		},
		NodeAffinity: 0,
		Created:      time.Now(),
		Result:       make(chan interface{}, 1),
	}

	err = optimizer.ScheduleTask(task)
	if err != nil {
		t.Errorf("Task scheduling failed: %v", err)
	}

	// Wait for execution.
	select {
	case result := <-task.Result:
		if result != "success" {
			t.Errorf("Unexpected task result: %v", result)
		}
	case <-time.After(time.Second):
		t.Error("Task execution timeout")
	}

	if !executed {
		t.Error("Task was not executed")
	}
}

func TestOptimizer_Statistics(t *testing.T) {
	optimizer := NewOptimizer()

	// Perform some operations.
	optimizer.Allocate(1024, 0)
	optimizer.Allocate(2048, 1)

	stats := optimizer.GetStatistics()

	// Verify required fields exist.
	requiredFields := []string{
		"enabled", "local_allocations", "remote_allocations",
		"migrations", "balance_operations", "topology_changes",
		"performance_gain", "node_count", "cores_per_node",
	}

	for _, field := range requiredFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Statistics should include field: %s", field)
		}
	}

	// Test string representation.
	str := optimizer.String()
	if len(str) == 0 {
		t.Error("String representation should not be empty")
	}
}

func TestOptimizer_ConcurrentAllocation(t *testing.T) {
	optimizer := NewOptimizer()

	const numGoroutines = 10

	const allocsPerGoroutine = 100

	var wg sync.WaitGroup

	successCount := int64(0)

	var mutex sync.Mutex

	// Launch concurrent allocations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func(goroutineID int) {
			defer wg.Done()

			localSuccess := 0

			for j := 0; j < allocsPerGoroutine; j++ {
				size := uintptr(64 + j*8)
				nodeHint := goroutineID % optimizer.topology.nodeCount

				ptr := optimizer.Allocate(size, nodeHint)
				if ptr != 0 {
					localSuccess++
				}
			}

			mutex.Lock()
			successCount += int64(localSuccess)
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	expectedTotal := int64(numGoroutines * allocsPerGoroutine)
	if successCount != expectedTotal {
		t.Errorf("Expected %d successful allocations, got %d", expectedTotal, successCount)
	}

	stats := optimizer.GetStatistics()

	totalAllocs := stats["local_allocations"].(int64) + stats["remote_allocations"].(int64)
	if totalAllocs != expectedTotal {
		t.Errorf("Statistics mismatch: expected %d, got %d", expectedTotal, totalAllocs)
	}
}

func TestOptimizer_StartStop(t *testing.T) {
	optimizer := NewOptimizer()

	// Test start.
	err := optimizer.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	// Test double start.
	err = optimizer.Start()
	if err == nil {
		t.Error("Second start should fail")
	}

	// Test stop.
	err = optimizer.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestOptimizer_DisabledMode(t *testing.T) {
	optimizer := NewOptimizer()
	optimizer.Disable()

	// Test allocation in disabled mode.
	ptr := optimizer.Allocate(1024, 0)
	if ptr == 0 {
		t.Error("Fallback allocation should succeed when disabled")
	}

	// Test start in disabled mode.
	err := optimizer.Start()
	if err == nil {
		t.Error("Start should fail when disabled")
	}

	// Test task scheduling in disabled mode.
	task := &Task{
		ID: 1,
		Function: func() interface{} {
			return "test"
		},
	}

	err = optimizer.ScheduleTask(task)
	if err == nil {
		t.Error("Task scheduling should fail when disabled")
	}
}

func TestLoadBalancer_Migration(t *testing.T) {
	scheduler := NewScheduler()

	// Create unbalanced queues.
	for i := 0; i < 5; i++ {
		task := &Task{
			ID:      uint64(i),
			Created: time.Now(),
		}
		scheduler.EnqueueTask(0, task) // All tasks on node 0
	}

	// Perform load balancing.
	scheduler.balancer.balance(scheduler)

	// Check if some tasks were migrated.
	queue0Length := scheduler.queues[0].length
	if queue0Length >= 5 {
		t.Error("Load balancer should have migrated some tasks")
	}
}

func TestAlert_Generation(t *testing.T) {
	alerts := make(chan *Alert, 10)

	sampler := &Sampler{
		nodeID: 0,
		cpu: CPUMetrics{
			Utilization: 0.95, // High utilization
		},
	}

	sampler.checkAlerts(alerts)

	select {
	case alert := <-alerts:
		if alert.Level != AlertWarning {
			t.Error("Should generate warning alert for high CPU")
		}

		if alert.NodeID != 0 {
			t.Error("Alert should be for node 0")
		}

		if alert.Metric != "cpu_utilization" {
			t.Error("Alert should be for CPU utilization")
		}
	case <-time.After(time.Millisecond * 100):
		t.Error("Should generate alert for high CPU utilization")
	}
}

func TestBalanceStrategy_Types(t *testing.T) {
	strategies := []BalanceStrategy{
		BalanceByLoad,
		BalanceByMemory,
		BalanceByLatency,
		BalanceAdaptive,
	}

	for _, strategy := range strategies {
		balancer := &LoadBalancer{
			strategy: strategy,
		}

		if balancer.strategy != strategy {
			t.Errorf("Strategy mismatch: expected %v, got %v", strategy, balancer.strategy)
		}
	}
}

func TestAffinityPolicy_Configuration(t *testing.T) {
	policies := []AffinityPolicy{
		{CPUStrict: true, MemoryStrict: true},
		{CPUStrict: false, MemoryStrict: true, Migration: true},
		{Interleaving: true},
	}

	manager := NewAffinityManager()

	for i, policy := range policies {
		manager.policies[i] = policy

		stored := manager.policies[i]
		if stored.CPUStrict != policy.CPUStrict {
			t.Error("CPU strict policy mismatch")
		}

		if stored.MemoryStrict != policy.MemoryStrict {
			t.Error("Memory strict policy mismatch")
		}

		if stored.Migration != policy.Migration {
			t.Error("Migration policy mismatch")
		}

		if stored.Interleaving != policy.Interleaving {
			t.Error("Interleaving policy mismatch")
		}
	}
}

func TestPerformanceOptimization(t *testing.T) {
	optimizer := NewOptimizer()

	// Test large allocation performance.
	const numAllocs = 1000

	start := time.Now()

	for i := 0; i < numAllocs; i++ {
		size := uintptr(1024 + i%1024)
		nodeHint := i % optimizer.topology.nodeCount

		ptr := optimizer.Allocate(size, nodeHint)
		if ptr == 0 {
			t.Errorf("Allocation %d failed", i)
		}
	}

	duration := time.Since(start)
	if duration > time.Second {
		t.Errorf("Performance test took too long: %v", duration)
	}

	avgTime := duration / numAllocs
	t.Logf("Average allocation time: %v", avgTime)
}
