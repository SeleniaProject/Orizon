// Package kernel provides advanced process scheduling and management
package kernel

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// Advanced Process Scheduler
// ============================================================================

// SchedulingPolicy represents different scheduling policies
type SchedulingPolicy int

const (
	SchedRoundRobin SchedulingPolicy = iota
	SchedCFS                         // Completely Fair Scheduler
	SchedRealTime                    // Real-time scheduling
	SchedIdle                        // Idle tasks
	SchedBatch                       // Batch processing
)

// ProcessPriority represents process priority levels
type ProcessPriority int

const (
	PriorityIdle ProcessPriority = iota
	PriorityLow
	PriorityNormal
	PriorityHigh
	PriorityRealTime
)

// AdvancedProcess extends the basic Process with scheduling information
type AdvancedProcess struct {
	*Process         // Embed basic process
	Policy           SchedulingPolicy
	Priority         ProcessPriority
	NicenessValue    int
	VRuntime         uint64 // Virtual runtime for CFS
	Weight           uint64 // Scheduling weight
	LoadWeight       uint64 // Load balancing weight
	TimeSlice        time.Duration
	LastRunTime      time.Time
	WakeupTime       time.Time
	SleepAverage     time.Duration
	InteractiveBonus int
	CPUAffinity      uint64 // Bitmask of allowed CPUs
	NumaNode         uint8  // NUMA node preference

	// Scheduling statistics
	TotalRunTime        time.Duration
	VoluntarySwitches   uint64
	InvoluntarySwitches uint64
	MinorFaults         uint64
	MajorFaults         uint64
}

// RunQueue represents a scheduler run queue
type RunQueue struct {
	processes   []*AdvancedProcess
	rbtree      *RedBlackTree // For CFS
	totalWeight uint64
	mutex       sync.RWMutex
}

// AdvancedScheduler implements advanced scheduling algorithms
type AdvancedScheduler struct {
	runQueues    []*RunQueue // Per-CPU run queues
	globalQueue  *RunQueue   // Global queue for load balancing
	currentCPU   int
	numCPUs      int
	loadBalancer *LoadBalancer
	mutex        sync.RWMutex

	// Scheduling parameters
	MinGranularity    time.Duration
	TargetLatency     time.Duration
	WakeupGranularity time.Duration
	MigrationCost     time.Duration

	// Statistics
	ContextSwitches uint64
	Migrations      uint64
	LoadBalanceRuns uint64
}

// GlobalAdvancedScheduler provides global access to advanced scheduling
var GlobalAdvancedScheduler *AdvancedScheduler

// InitializeAdvancedScheduler initializes the advanced scheduler
func InitializeAdvancedScheduler() error {
	if GlobalAdvancedScheduler != nil {
		return fmt.Errorf("advanced scheduler already initialized")
	}

	numCPUs := 1 // For now, assume single CPU
	runQueues := make([]*RunQueue, numCPUs)
	for i := 0; i < numCPUs; i++ {
		runQueues[i] = &RunQueue{
			processes: make([]*AdvancedProcess, 0),
			rbtree:    NewRedBlackTree(),
		}
	}

	GlobalAdvancedScheduler = &AdvancedScheduler{
		runQueues:         runQueues,
		globalQueue:       &RunQueue{processes: make([]*AdvancedProcess, 0)},
		numCPUs:           numCPUs,
		loadBalancer:      NewLoadBalancer(),
		MinGranularity:    time.Millisecond,
		TargetLatency:     20 * time.Millisecond,
		WakeupGranularity: time.Millisecond,
		MigrationCost:     500 * time.Microsecond,
	}

	return nil
}

// CreateAdvancedProcess creates a new advanced process
func (sched *AdvancedScheduler) CreateAdvancedProcess(name string, entryPoint uintptr, stackSize uintptr) (*AdvancedProcess, error) {
	// Create basic process first
	basicProcess, err := GlobalProcessManager.CreateProcess(name, entryPoint, stackSize)
	if err != nil {
		return nil, err
	}

	// Create advanced process
	advProcess := &AdvancedProcess{
		Process:       basicProcess,
		Policy:        SchedCFS,
		Priority:      PriorityNormal,
		NicenessValue: 0,
		VRuntime:      0,
		Weight:        1024, // Default weight
		LoadWeight:    1024,
		TimeSlice:     sched.calculateTimeSlice(PriorityNormal),
		CPUAffinity:   ^uint64(0), // All CPUs
		NumaNode:      0,
	}

	// Add to appropriate run queue
	cpu := sched.selectCPU(advProcess)
	sched.addToRunQueue(cpu, advProcess)

	return advProcess, nil
}

// calculateTimeSlice calculates time slice based on priority
func (sched *AdvancedScheduler) calculateTimeSlice(priority ProcessPriority) time.Duration {
	baseSlice := sched.TargetLatency / time.Duration(len(sched.runQueues[0].processes)+1)
	if baseSlice < sched.MinGranularity {
		baseSlice = sched.MinGranularity
	}

	switch priority {
	case PriorityRealTime:
		return baseSlice * 4
	case PriorityHigh:
		return baseSlice * 2
	case PriorityNormal:
		return baseSlice
	case PriorityLow:
		return baseSlice / 2
	case PriorityIdle:
		return baseSlice / 4
	default:
		return baseSlice
	}
}

// selectCPU selects the best CPU for a process
func (sched *AdvancedScheduler) selectCPU(proc *AdvancedProcess) int {
	// Simple selection: choose least loaded CPU
	minLoad := ^uint64(0)
	selectedCPU := 0

	for i, rq := range sched.runQueues {
		if (proc.CPUAffinity & (1 << uint(i))) != 0 {
			if rq.totalWeight < minLoad {
				minLoad = rq.totalWeight
				selectedCPU = i
			}
		}
	}

	return selectedCPU
}

// addToRunQueue adds a process to a run queue
func (sched *AdvancedScheduler) addToRunQueue(cpu int, proc *AdvancedProcess) {
	rq := sched.runQueues[cpu]
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	switch proc.Policy {
	case SchedCFS:
		rq.rbtree.Insert(proc.VRuntime, proc)
	default:
		rq.processes = append(rq.processes, proc)
	}

	rq.totalWeight += proc.Weight
	proc.State = ProcessReady
}

// ScheduleAdvanced performs advanced scheduling
func (sched *AdvancedScheduler) ScheduleAdvanced() *AdvancedProcess {
	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	cpu := sched.currentCPU
	rq := sched.runQueues[cpu]

	var nextProcess *AdvancedProcess

	rq.mutex.Lock()
	switch {
	case len(rq.processes) > 0:
		// Round-robin for non-CFS processes
		nextProcess = rq.processes[0]
		rq.processes = rq.processes[1:]

	case rq.rbtree.Size() > 0:
		// CFS scheduling
		node := rq.rbtree.GetMin()
		if node != nil {
			nextProcess = node.Value.(*AdvancedProcess)
			rq.rbtree.Delete(node.Key)
		}
	}
	rq.mutex.Unlock()

	if nextProcess == nil {
		// Try load balancing
		nextProcess = sched.loadBalance()
	}

	if nextProcess != nil {
		sched.contextSwitch(nextProcess)
		sched.ContextSwitches++
	}

	return nextProcess
}

// contextSwitch performs context switching with advanced features
func (sched *AdvancedScheduler) contextSwitch(proc *AdvancedProcess) {
	now := time.Now()

	// Update statistics
	if proc.State == ProcessRunning {
		proc.TotalRunTime += now.Sub(proc.LastRunTime)
	}

	proc.LastRunTime = now
	proc.State = ProcessRunning

	// Update virtual runtime for CFS
	if proc.Policy == SchedCFS {
		proc.VRuntime += uint64(proc.TimeSlice.Nanoseconds())
	}
}

// loadBalance performs load balancing between CPUs
func (sched *AdvancedScheduler) loadBalance() *AdvancedProcess {
	if sched.loadBalancer == nil {
		return nil
	}

	sched.LoadBalanceRuns++
	return sched.loadBalancer.Balance(sched.runQueues, sched.currentCPU)
}

// ============================================================================
// Load Balancer
// ============================================================================

// LoadBalancer manages load balancing between CPUs
type LoadBalancer struct {
	lastBalance time.Time
	interval    time.Duration
	mutex       sync.Mutex
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		interval: 100 * time.Millisecond,
	}
}

// Balance performs load balancing
func (lb *LoadBalancer) Balance(runQueues []*RunQueue, currentCPU int) *AdvancedProcess {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	now := time.Now()
	if now.Sub(lb.lastBalance) < lb.interval {
		return nil
	}
	lb.lastBalance = now

	// Find the most loaded CPU
	maxLoad := uint64(0)
	maxCPU := -1

	for i, rq := range runQueues {
		if i != currentCPU && rq.totalWeight > maxLoad {
			maxLoad = rq.totalWeight
			maxCPU = i
		}
	}

	if maxCPU == -1 || maxLoad <= runQueues[currentCPU].totalWeight*2 {
		return nil // No significant imbalance
	}

	// Migrate a process from the most loaded CPU
	sourceRQ := runQueues[maxCPU]
	sourceRQ.mutex.Lock()
	defer sourceRQ.mutex.Unlock()

	if len(sourceRQ.processes) == 0 {
		return nil
	}

	// Find a suitable process to migrate
	for i, proc := range sourceRQ.processes {
		if (proc.CPUAffinity & (1 << uint(currentCPU))) != 0 {
			// Remove from source queue
			sourceRQ.processes = append(sourceRQ.processes[:i], sourceRQ.processes[i+1:]...)
			sourceRQ.totalWeight -= proc.Weight

			// Add migration cost penalty
			proc.VRuntime += uint64(GlobalAdvancedScheduler.MigrationCost.Nanoseconds())

			GlobalAdvancedScheduler.Migrations++
			return proc
		}
	}

	return nil
}

// ============================================================================
// Red-Black Tree for CFS
// ============================================================================

// RBNode represents a red-black tree node
type RBNode struct {
	Key    uint64
	Value  interface{}
	Color  bool // true = red, false = black
	Left   *RBNode
	Right  *RBNode
	Parent *RBNode
}

// RedBlackTree implements a red-black tree
type RedBlackTree struct {
	root  *RBNode
	size  int
	mutex sync.RWMutex
}

// NewRedBlackTree creates a new red-black tree
func NewRedBlackTree() *RedBlackTree {
	return &RedBlackTree{}
}

// Insert inserts a key-value pair
func (rbt *RedBlackTree) Insert(key uint64, value interface{}) {
	rbt.mutex.Lock()
	defer rbt.mutex.Unlock()

	node := &RBNode{
		Key:   key,
		Value: value,
		Color: true, // New nodes are red
	}

	if rbt.root == nil {
		rbt.root = node
		node.Color = false // Root is black
		rbt.size++
		return
	}

	// Standard BST insertion
	current := rbt.root
	for {
		if key < current.Key {
			if current.Left == nil {
				current.Left = node
				node.Parent = current
				break
			}
			current = current.Left
		} else {
			if current.Right == nil {
				current.Right = node
				node.Parent = current
				break
			}
			current = current.Right
		}
	}

	rbt.fixInsert(node)
	rbt.size++
}

// fixInsert fixes the red-black tree properties after insertion
func (rbt *RedBlackTree) fixInsert(node *RBNode) {
	for node.Parent != nil && node.Parent.Color {
		if node.Parent == node.Parent.Parent.Left {
			uncle := node.Parent.Parent.Right
			if uncle != nil && uncle.Color {
				// Uncle is red
				node.Parent.Color = false
				uncle.Color = false
				node.Parent.Parent.Color = true
				node = node.Parent.Parent
			} else {
				if node == node.Parent.Right {
					node = node.Parent
					rbt.rotateLeft(node)
				}
				node.Parent.Color = false
				node.Parent.Parent.Color = true
				rbt.rotateRight(node.Parent.Parent)
			}
		} else {
			uncle := node.Parent.Parent.Left
			if uncle != nil && uncle.Color {
				node.Parent.Color = false
				uncle.Color = false
				node.Parent.Parent.Color = true
				node = node.Parent.Parent
			} else {
				if node == node.Parent.Left {
					node = node.Parent
					rbt.rotateRight(node)
				}
				node.Parent.Color = false
				node.Parent.Parent.Color = true
				rbt.rotateLeft(node.Parent.Parent)
			}
		}
	}
	rbt.root.Color = false
}

// rotateLeft performs left rotation
func (rbt *RedBlackTree) rotateLeft(x *RBNode) {
	y := x.Right
	x.Right = y.Left
	if y.Left != nil {
		y.Left.Parent = x
	}
	y.Parent = x.Parent
	if x.Parent == nil {
		rbt.root = y
	} else if x == x.Parent.Left {
		x.Parent.Left = y
	} else {
		x.Parent.Right = y
	}
	y.Left = x
	x.Parent = y
}

// rotateRight performs right rotation
func (rbt *RedBlackTree) rotateRight(x *RBNode) {
	y := x.Left
	x.Left = y.Right
	if y.Right != nil {
		y.Right.Parent = x
	}
	y.Parent = x.Parent
	if x.Parent == nil {
		rbt.root = y
	} else if x == x.Parent.Right {
		x.Parent.Right = y
	} else {
		x.Parent.Left = y
	}
	y.Right = x
	x.Parent = y
}

// GetMin returns the node with minimum key
func (rbt *RedBlackTree) GetMin() *RBNode {
	rbt.mutex.RLock()
	defer rbt.mutex.RUnlock()

	if rbt.root == nil {
		return nil
	}

	current := rbt.root
	for current.Left != nil {
		current = current.Left
	}
	return current
}

// Delete removes a node with the given key
func (rbt *RedBlackTree) Delete(key uint64) {
	rbt.mutex.Lock()
	defer rbt.mutex.Unlock()

	node := rbt.find(key)
	if node == nil {
		return
	}

	rbt.deleteNode(node)
	rbt.size--
}

// find finds a node with the given key
func (rbt *RedBlackTree) find(key uint64) *RBNode {
	current := rbt.root
	for current != nil {
		if key == current.Key {
			return current
		} else if key < current.Key {
			current = current.Left
		} else {
			current = current.Right
		}
	}
	return nil
}

// deleteNode deletes a specific node
func (rbt *RedBlackTree) deleteNode(node *RBNode) {
	// Simplified deletion - full implementation would be more complex
	if node.Left == nil && node.Right == nil {
		if node.Parent == nil {
			rbt.root = nil
		} else if node == node.Parent.Left {
			node.Parent.Left = nil
		} else {
			node.Parent.Right = nil
		}
	}
}

// Size returns the number of nodes in the tree
func (rbt *RedBlackTree) Size() int {
	rbt.mutex.RLock()
	defer rbt.mutex.RUnlock()
	return rbt.size
}

// ============================================================================
// Real-time scheduling support
// ============================================================================

// RTScheduler implements real-time scheduling
type RTScheduler struct {
	rtProcesses []*AdvancedProcess
	mutex       sync.RWMutex
}

// AddRTProcess adds a real-time process
func (rts *RTScheduler) AddRTProcess(proc *AdvancedProcess) {
	rts.mutex.Lock()
	defer rts.mutex.Unlock()

	proc.Priority = PriorityRealTime
	rts.rtProcesses = append(rts.rtProcesses, proc)

	// Sort by priority (highest first)
	sort.Slice(rts.rtProcesses, func(i, j int) bool {
		return rts.rtProcesses[i].Priority > rts.rtProcesses[j].Priority
	})
}

// GetNextRTProcess gets the next real-time process to run
func (rts *RTScheduler) GetNextRTProcess() *AdvancedProcess {
	rts.mutex.Lock()
	defer rts.mutex.Unlock()

	for i, proc := range rts.rtProcesses {
		if proc.State == ProcessReady {
			// Move to end of same priority group
			rts.rtProcesses = append(rts.rtProcesses[:i], rts.rtProcesses[i+1:]...)
			rts.rtProcesses = append(rts.rtProcesses, proc)
			return proc
		}
	}

	return nil
}

// ============================================================================
// Kernel API functions for advanced scheduling
// ============================================================================

// KernelCreateAdvancedProcess creates an advanced process
func KernelCreateAdvancedProcess(name string, entryPoint uintptr, stackSize uintptr, priority ProcessPriority) uint32 {
	if GlobalAdvancedScheduler == nil {
		return 0
	}

	proc, err := GlobalAdvancedScheduler.CreateAdvancedProcess(name, entryPoint, stackSize)
	if err != nil {
		return 0
	}

	proc.Priority = priority
	return proc.PID
}

// KernelSetProcessPriority sets process priority
func KernelSetProcessPriority(pid uint32, priority ProcessPriority) bool {
	if GlobalAdvancedScheduler == nil {
		return false
	}

	// Find process and update priority
	for _, rq := range GlobalAdvancedScheduler.runQueues {
		rq.mutex.Lock()
		for _, proc := range rq.processes {
			if proc.PID == pid {
				proc.Priority = priority
				proc.TimeSlice = GlobalAdvancedScheduler.calculateTimeSlice(priority)
				rq.mutex.Unlock()
				return true
			}
		}
		rq.mutex.Unlock()
	}

	return false
}

// KernelSetCPUAffinity sets CPU affinity for a process
func KernelSetCPUAffinity(pid uint32, cpuMask uint64) bool {
	if GlobalAdvancedScheduler == nil {
		return false
	}

	for _, rq := range GlobalAdvancedScheduler.runQueues {
		rq.mutex.Lock()
		for _, proc := range rq.processes {
			if proc.PID == pid {
				proc.CPUAffinity = cpuMask
				rq.mutex.Unlock()
				return true
			}
		}
		rq.mutex.Unlock()
	}

	return false
}

// KernelGetSchedulerStats returns scheduler statistics
func KernelGetSchedulerStats() (contextSwitches, migrations, loadBalanceRuns uint64) {
	if GlobalAdvancedScheduler == nil {
		return 0, 0, 0
	}

	GlobalAdvancedScheduler.mutex.RLock()
	defer GlobalAdvancedScheduler.mutex.RUnlock()

	return GlobalAdvancedScheduler.ContextSwitches,
		GlobalAdvancedScheduler.Migrations,
		GlobalAdvancedScheduler.LoadBalanceRuns
}
