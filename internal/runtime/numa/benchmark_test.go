// NUMA Optimizer Benchmarks - Performance validation for Phase 3.1.3
package numa

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

func BenchmarkOptimizer_Allocate(b *testing.B) {
	optimizer := NewOptimizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := uintptr(1024)
		nodeHint := i % optimizer.topology.nodeCount
		ptr := optimizer.Allocate(size, nodeHint)
		if ptr == 0 {
			b.Fatal("Allocation failed")
		}
	}
}

func BenchmarkOptimizer_AllocateParallel(b *testing.B) {
	optimizer := NewOptimizer()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := uintptr(1024 + i%1024)
			nodeHint := i % optimizer.topology.nodeCount
			ptr := optimizer.Allocate(size, nodeHint)
			if ptr == 0 {
				b.Fatal("Allocation failed")
			}
			i++
		}
	})
}

func BenchmarkOptimizer_AllocateSizes(b *testing.B) {
	optimizer := NewOptimizer()

	sizes := []uintptr{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				nodeHint := i % optimizer.topology.nodeCount
				ptr := optimizer.Allocate(size, nodeHint)
				if ptr == 0 {
					b.Fatal("Allocation failed")
				}
			}
		})
	}
}

func BenchmarkTopology_Discovery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		topology := NewTopology()
		if topology.nodeCount == 0 {
			b.Fatal("Topology discovery failed")
		}
	}
}

func BenchmarkScheduler_TaskSubmission(b *testing.B) {
	optimizer := NewOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := &Task{
			ID: uint64(i),
			Function: func() interface{} {
				return i
			},
			NodeAffinity: i % optimizer.topology.nodeCount,
			Created:      time.Now(),
			Result:       make(chan interface{}, 1),
		}

		err := optimizer.ScheduleTask(task)
		if err != nil {
			b.Fatal("Task scheduling failed:", err)
		}

		// Wait for completion
		<-task.Result
	}
}

func BenchmarkScheduler_TaskSubmissionParallel(b *testing.B) {
	optimizer := NewOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := &Task{
				ID: uint64(i),
				Function: func() interface{} {
					runtime.Gosched() // Simulate work
					return i
				},
				NodeAffinity: i % optimizer.topology.nodeCount,
				Created:      time.Now(),
				Result:       make(chan interface{}, 1),
			}

			err := optimizer.ScheduleTask(task)
			if err != nil {
				b.Fatal("Task scheduling failed:", err)
			}

			// Wait for completion
			<-task.Result
			i++
		}
	})
}

func BenchmarkLoadBalancer_Balance(b *testing.B) {
	scheduler := NewScheduler()

	// Create unbalanced workload
	for i := 0; i < 100; i++ {
		task := &Task{
			ID:      uint64(i),
			Created: time.Now(),
		}
		scheduler.EnqueueTask(0, task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.balancer.balance(scheduler)
	}
}

func BenchmarkMetrics_Collection(b *testing.B) {
	monitor := NewMonitor()
	sampler := monitor.samplers[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sampler.collectMetrics()
	}
}

func BenchmarkAffinityManager_SetCPUAffinity(b *testing.B) {
	affinity := NewAffinityManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mask := uint64(1 << (i % 8)) // Rotate through CPU cores
		err := affinity.SetCPUAffinity(i%len(affinity.cpuMasks), mask)
		if err != nil {
			b.Fatal("SetCPUAffinity failed:", err)
		}
	}
}

func BenchmarkMemoryPool_AllocateFree(b *testing.B) {
	pool := NewMemoryPool(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := pool.Allocate(1024)
		if ptr == 0 {
			b.Fatal("Allocation failed")
		}
		pool.Free(ptr, 1024)
	}
}

func BenchmarkOptimizer_ConcurrentWorkload(b *testing.B) {
	optimizer := NewOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	const numWorkers = 8
	const tasksPerWorker = 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for t := 0; t < tasksPerWorker; t++ {
					// Mixed allocation and task scheduling
					if t%2 == 0 {
						size := uintptr(1024 + t*64)
						nodeHint := workerID % optimizer.topology.nodeCount
						ptr := optimizer.Allocate(size, nodeHint)
						if ptr == 0 {
							b.Error("Allocation failed")
							return
						}
					} else {
						task := &Task{
							ID: uint64(workerID*tasksPerWorker + t),
							Function: func() interface{} {
								// Simulate CPU work
								sum := 0
								for i := 0; i < 1000; i++ {
									sum += i
								}
								return sum
							},
							NodeAffinity: workerID % optimizer.topology.nodeCount,
							Created:      time.Now(),
							Result:       make(chan interface{}, 1),
						}

						err := optimizer.ScheduleTask(task)
						if err != nil {
							b.Error("Task scheduling failed:", err)
							return
						}

						<-task.Result
					}
				}
			}(w)
		}

		wg.Wait()
	}
}

// Benchmark comparison with and without NUMA optimization
func BenchmarkComparison_WithNUMA(b *testing.B) {
	optimizer := NewOptimizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := uintptr(1024)
		nodeHint := i % optimizer.topology.nodeCount
		ptr := optimizer.Allocate(size, nodeHint)
		if ptr == 0 {
			b.Fatal("Allocation failed")
		}
	}
}

func BenchmarkComparison_WithoutNUMA(b *testing.B) {
	optimizer := NewOptimizer()
	optimizer.Disable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := uintptr(1024)
		nodeHint := i % optimizer.topology.nodeCount
		ptr := optimizer.Allocate(size, nodeHint)
		if ptr == 0 {
			b.Fatal("Allocation failed")
		}
	}
}

func BenchmarkScheduler_QueueOperations(b *testing.B) {
	scheduler := NewScheduler()

	// Pre-create tasks
	tasks := make([]*Task, b.N)
	for i := range tasks {
		tasks[i] = &Task{
			ID:      uint64(i),
			Created: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := i % len(scheduler.queues)
		err := scheduler.EnqueueTask(nodeID, tasks[i])
		if err != nil {
			b.Fatal("Enqueue failed:", err)
		}
	}
}

func BenchmarkTopology_DistanceCalculation(b *testing.B) {
	topology := NewTopology()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		from := i % topology.nodeCount
		to := (i + 1) % topology.nodeCount
		distance := topology.GetDistance(from, to)
		if distance <= 0 {
			b.Fatal("Invalid distance")
		}
	}
}

func BenchmarkNodeMemory_Update(b *testing.B) {
	memory := NewNodeMemory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate memory allocation/deallocation
		size := uint64(1024 + i%1024)
		memory.Used += size
		memory.Available -= size

		if memory.Available < size {
			memory.Available = memory.Total - memory.Used
		}
	}
}

func BenchmarkSampler_FullCycle(b *testing.B) {
	sampler := &Sampler{
		nodeID:     0,
		sampleRate: time.Microsecond,
	}

	alerts := make(chan *Alert, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sampler.collectMetrics()
		sampler.checkAlerts(alerts)
	}
}

// Memory usage benchmarks
func BenchmarkOptimizer_MemoryFootprint(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	optimizers := make([]*Optimizer, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizers[i] = NewOptimizer()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	avgMemPerOptimizer := (m2.Alloc - m1.Alloc) / uint64(b.N)
	b.Logf("Average memory per optimizer: %d bytes", avgMemPerOptimizer)

	// Keep reference to prevent GC
	_ = optimizers
}

// Latency distribution benchmark
func BenchmarkOptimizer_LatencyDistribution(b *testing.B) {
	optimizer := NewOptimizer()
	latencies := make([]time.Duration, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		size := uintptr(1024)
		nodeHint := i % optimizer.topology.nodeCount
		ptr := optimizer.Allocate(size, nodeHint)

		latencies[i] = time.Since(start)

		if ptr == 0 {
			b.Fatal("Allocation failed")
		}
	}

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := latencies[len(latencies)/2]
	p95 := latencies[len(latencies)*95/100]
	p99 := latencies[len(latencies)*99/100]

	b.Logf("Latency P50: %v, P95: %v, P99: %v", p50, p95, p99)
}

// Scalability benchmark
func BenchmarkOptimizer_Scalability(b *testing.B) {
	optimizer := NewOptimizer()

	workers := []int{1, 2, 4, 8, 16, 32}

	for _, numWorkers := range workers {
		b.Run(fmt.Sprintf("Workers%d", numWorkers), func(b *testing.B) {
			b.SetParallelism(numWorkers)

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					size := uintptr(1024)
					nodeHint := i % optimizer.topology.nodeCount
					ptr := optimizer.Allocate(size, nodeHint)
					if ptr == 0 {
						b.Fatal("Allocation failed")
					}
					i++
				}
			})
		})
	}
}
