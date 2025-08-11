// Package types implements Phase 2.4.3 Resource Type System for the Orizon compiler.
// This system provides resource types for automatic resource management, cleanup, and leak detection.
package types

import (
	"fmt"
	"sync"
	"time"
)

// ResourceTypeKind represents different kinds of resources (avoiding conflict with linear.go)
type ResourceTypeKind int

const (
	ResourceTypeKindFile         ResourceTypeKind = iota // File handle
	ResourceTypeKindSocket                               // Network socket
	ResourceTypeKindMemoryAlloc                          // Memory allocation
	ResourceTypeKindMutexLock                            // Mutex/lock
	ResourceTypeKindDatabaseConn                         // Database connection
	ResourceTypeKindThread                               // Thread handle
	ResourceTypeKindSemaphore                            // Semaphore
	ResourceTypeKindPipe                                 // Named pipe
	ResourceTypeKindTimer                                // Timer resource
	ResourceTypeKindChannelComm                          // Communication channel
	ResourceTypeKindContext                              // Execution context
	ResourceTypeKindCustom                               // User-defined resource
)

// String returns a string representation of the resource kind
func (rk ResourceTypeKind) String() string {
	switch rk {
	case ResourceTypeKindFile:
		return "file"
	case ResourceTypeKindSocket:
		return "socket"
	case ResourceTypeKindMemoryAlloc:
		return "memory"
	case ResourceTypeKindMutexLock:
		return "mutex"
	case ResourceTypeKindDatabaseConn:
		return "database"
	case ResourceTypeKindThread:
		return "thread"
	case ResourceTypeKindSemaphore:
		return "semaphore"
	case ResourceTypeKindPipe:
		return "pipe"
	case ResourceTypeKindTimer:
		return "timer"
	case ResourceTypeKindChannelComm:
		return "channel"
	case ResourceTypeKindContext:
		return "context"
	case ResourceTypeKindCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// ResourceState represents the current state of a resource
type ResourceState int

const (
	ResourceStateUninitialized ResourceState = iota // Resource not yet acquired
	ResourceStateAcquired                           // Resource successfully acquired
	ResourceStateInUse                              // Resource currently in use
	ResourceStateReleased                           // Resource has been released
	ResourceStateError                              // Resource in error state
	ResourceStateLeaked                             // Resource leaked (not properly released)
)

// String returns a string representation of the resource state
func (rs ResourceState) String() string {
	switch rs {
	case ResourceStateUninitialized:
		return "uninitialized"
	case ResourceStateAcquired:
		return "acquired"
	case ResourceStateInUse:
		return "in-use"
	case ResourceStateReleased:
		return "released"
	case ResourceStateError:
		return "error"
	case ResourceStateLeaked:
		return "leaked"
	default:
		return "unknown"
	}
}

// ManagedResourceType represents a resource type with automatic management
type ManagedResourceType struct {
	BaseType      *Type
	Kind          ResourceTypeKind
	AcquireFunc   string        // Function to acquire the resource
	ReleaseFunc   string        // Function to release the resource
	Finalizer     string        // Optional finalizer function
	State         ResourceState // Current state
	Lifetime      *ResourceLifetime
	Constraints   []ResourceConstraint
	Dependencies  []*ManagedResourceType // Resources this depends on
	UsageCount    int
	MaxUsageCount int  // Maximum allowed concurrent usage
	IsShared      bool // Whether resource can be shared
	UniqueId      string
	Metadata      map[string]interface{}
}

// String returns a string representation of the resource type
func (rt *ManagedResourceType) String() string {
	shared := ""
	if rt.IsShared {
		shared = " (shared)"
	}

	usage := ""
	if rt.MaxUsageCount > 0 {
		usage = fmt.Sprintf(" [%d/%d]", rt.UsageCount, rt.MaxUsageCount)
	}

	return fmt.Sprintf("resource<%s:%s>%s%s%s",
		rt.Kind.String(), rt.BaseType.String(), shared, usage,
		rt.State.String())
}

// CanAcquire checks if the resource can be acquired
func (rt *ManagedResourceType) CanAcquire() bool {
	switch rt.State {
	case ResourceStateUninitialized:
		return true
	case ResourceStateReleased:
		return true
	case ResourceStateAcquired, ResourceStateInUse:
		return rt.IsShared && (rt.MaxUsageCount == 0 || rt.UsageCount < rt.MaxUsageCount)
	case ResourceStateError, ResourceStateLeaked:
		return false
	default:
		return false
	}
}

// Acquire marks the resource as acquired
func (rt *ManagedResourceType) Acquire() error {
	if !rt.CanAcquire() {
		return fmt.Errorf("cannot acquire resource %s in state %s",
			rt.Kind.String(), rt.State.String())
	}

	// Check constraints
	for _, constraint := range rt.Constraints {
		if err := constraint.Check(rt); err != nil {
			return fmt.Errorf("constraint violation: %v", err)
		}
	}

	// Check dependencies
	for _, dep := range rt.Dependencies {
		if dep.State != ResourceStateAcquired && dep.State != ResourceStateInUse {
			return fmt.Errorf("dependency %s not available", dep.Kind.String())
		}
	}

	rt.State = ResourceStateAcquired
	rt.UsageCount++

	if rt.Lifetime != nil {
		rt.Lifetime.StartTime = time.Now()
	}

	return nil
}

// Use marks the resource as in use
func (rt *ManagedResourceType) Use() error {
	if rt.State != ResourceStateAcquired {
		return fmt.Errorf("resource %s not acquired (state: %s)",
			rt.Kind.String(), rt.State.String())
	}

	rt.State = ResourceStateInUse
	return nil
}

// Release marks the resource as released
func (rt *ManagedResourceType) Release() error {
	if rt.State != ResourceStateAcquired && rt.State != ResourceStateInUse {
		return fmt.Errorf("resource %s not in releasable state (state: %s)",
			rt.Kind.String(), rt.State.String())
	}

	rt.State = ResourceStateReleased
	rt.UsageCount--

	if rt.Lifetime != nil {
		rt.Lifetime.EndTime = time.Now()
		rt.Lifetime.Duration = rt.Lifetime.EndTime.Sub(rt.Lifetime.StartTime)
	}

	return nil
}

// MarkLeaked marks the resource as leaked
func (rt *ManagedResourceType) MarkLeaked() {
	rt.State = ResourceStateLeaked
	if rt.Lifetime != nil {
		rt.Lifetime.IsLeaked = true
	}
}

// IsActive checks if the resource is currently active
func (rt *ManagedResourceType) IsActive() bool {
	return rt.State == ResourceStateAcquired || rt.State == ResourceStateInUse
}

// ResourceLifetime tracks the lifetime of a resource
type ResourceLifetime struct {
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	MaxLifetime     time.Duration // Maximum allowed lifetime
	IsTimedOut      bool
	IsLeaked        bool
	Location        SourceLocation // Where resource was acquired
	ReleaseLocation SourceLocation // Where resource was released
}

// IsExpired checks if the resource lifetime has expired
func (rl *ResourceLifetime) IsExpired() bool {
	if rl.MaxLifetime == 0 {
		return false
	}
	return time.Since(rl.StartTime) > rl.MaxLifetime
}

// CheckExpiration checks and marks expiration status
func (rl *ResourceLifetime) CheckExpiration() {
	rl.IsTimedOut = rl.IsExpired()
}

// ResourceConstraint represents a constraint on resource usage
type ResourceConstraint struct {
	Kind        ResourceConstraintKind
	Description string
	Check       func(*ManagedResourceType) error
	Parameters  map[string]interface{}
}

// ResourceConstraintKind represents kinds of resource constraints
type ResourceConstraintKind int

const (
	ConstraintKindExclusive  ResourceConstraintKind = iota // Exclusive access required
	ConstraintKindReadOnly                                 // Read-only access
	ConstraintKindWriteOnly                                // Write-only access
	ConstraintKindTimeLimit                                // Time-limited access
	ConstraintKindUsageLimit                               // Usage count limit
	ConstraintKindDependency                               // Dependency constraint
	ConstraintKindMutualEx                                 // Mutual exclusion
	ConstraintKindOrdering                                 // Ordering constraint
)

// String returns a string representation of the constraint kind
func (rck ResourceConstraintKind) String() string {
	switch rck {
	case ConstraintKindExclusive:
		return "exclusive"
	case ConstraintKindReadOnly:
		return "read-only"
	case ConstraintKindWriteOnly:
		return "write-only"
	case ConstraintKindTimeLimit:
		return "time-limit"
	case ConstraintKindUsageLimit:
		return "usage-limit"
	case ConstraintKindDependency:
		return "dependency"
	case ConstraintKindMutualEx:
		return "mutual-exclusion"
	case ConstraintKindOrdering:
		return "ordering"
	default:
		return "unknown"
	}
}

// ResourceManager manages resource allocation and lifecycle
type ResourceManager struct {
	resources    map[string]*ManagedResourceType
	allocations  map[string]*ResourceAllocation
	pools        map[string]*ResourcePool
	cleanupTasks []CleanupTask
	leakDetector *ResourceLeakDetector
	monitor      *ResourceMonitor
	mutex        sync.RWMutex
}

// ResourceAllocation represents an allocation of a resource
type ResourceAllocation struct {
	ResourceId string
	ProcessId  string
	ThreadId   string
	Timestamp  time.Time
	Location   SourceLocation
	IsActive   bool
	RefCount   int
	Finalizers []func()
}

// ResourcePool manages a pool of similar resources
type ResourcePool struct {
	Name      string
	Kind      ResourceTypeKind
	MaxSize   int
	MinSize   int
	Available []*ManagedResourceType
	InUse     []*ManagedResourceType
	Factory   func() (*ManagedResourceType, error)
	Validator func(*ManagedResourceType) bool
	ResetFunc func(*ManagedResourceType) error
	mutex     sync.Mutex
}

// Get retrieves a resource from the pool
func (rp *ResourcePool) Get() (*ManagedResourceType, error) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Try to get from available resources
	if len(rp.Available) > 0 {
		resource := rp.Available[len(rp.Available)-1]
		rp.Available = rp.Available[:len(rp.Available)-1]
		rp.InUse = append(rp.InUse, resource)

		// Validate and reset if needed
		if rp.Validator != nil && !rp.Validator(resource) {
			return nil, fmt.Errorf("resource validation failed")
		}

		if rp.ResetFunc != nil {
			if err := rp.ResetFunc(resource); err != nil {
				return nil, fmt.Errorf("resource reset failed: %v", err)
			}
		}

		return resource, nil
	}

	// Create new resource if under limit
	if len(rp.InUse) < rp.MaxSize {
		if rp.Factory != nil {
			resource, err := rp.Factory()
			if err != nil {
				return nil, fmt.Errorf("resource creation failed: %v", err)
			}
			rp.InUse = append(rp.InUse, resource)
			return resource, nil
		}
	}

	return nil, fmt.Errorf("resource pool exhausted")
}

// Put returns a resource to the pool
func (rp *ResourcePool) Put(resource *ManagedResourceType) error {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Remove from in-use list
	for i, r := range rp.InUse {
		if r.UniqueId == resource.UniqueId {
			rp.InUse = append(rp.InUse[:i], rp.InUse[i+1:]...)
			break
		}
	}

	// Add to available if under minimum
	if len(rp.Available) < rp.MinSize {
		resource.State = ResourceStateAcquired
		rp.Available = append(rp.Available, resource)
		return nil
	}

	// Otherwise release the resource
	return resource.Release()
}

// CleanupTask represents a cleanup task for resources
type CleanupTask struct {
	Id         string
	ResourceId string
	Priority   int
	Deadline   time.Time
	Cleanup    func() error
	OnFailure  func(error)
	IsExecuted bool
}

// Execute executes the cleanup task
func (ct *CleanupTask) Execute() error {
	if ct.IsExecuted {
		return nil
	}

	err := ct.Cleanup()
	if err != nil && ct.OnFailure != nil {
		ct.OnFailure(err)
	}

	ct.IsExecuted = true
	return err
}

// ResourceLeakDetector detects resource leaks
type ResourceLeakDetector struct {
	allocations  map[string]*ResourceAllocation
	scanInterval time.Duration
	threshold    time.Duration
	isRunning    bool
	stopChan     chan bool
	mutex        sync.RWMutex
}

// NewResourceLeakDetector creates a new leak detector
func NewResourceLeakDetector(scanInterval, threshold time.Duration) *ResourceLeakDetector {
	return &ResourceLeakDetector{
		allocations:  make(map[string]*ResourceAllocation),
		scanInterval: scanInterval,
		threshold:    threshold,
		stopChan:     make(chan bool),
	}
}

// Start starts the leak detection process
func (rld *ResourceLeakDetector) Start() {
	if rld.isRunning {
		return
	}

	rld.isRunning = true
	go rld.scanLoop()
}

// Stop stops the leak detection process
func (rld *ResourceLeakDetector) Stop() {
	if !rld.isRunning {
		return
	}

	rld.stopChan <- true
	rld.isRunning = false
}

// scanLoop performs periodic leak detection scans
func (rld *ResourceLeakDetector) scanLoop() {
	ticker := time.NewTicker(rld.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rld.scanForLeaks()
		case <-rld.stopChan:
			return
		}
	}
}

// scanForLeaks scans for potential resource leaks
func (rld *ResourceLeakDetector) scanForLeaks() {
	rld.mutex.RLock()
	defer rld.mutex.RUnlock()

	now := time.Now()
	for id, allocation := range rld.allocations {
		if allocation.IsActive && now.Sub(allocation.Timestamp) > rld.threshold {
			// Potential leak detected
			rld.reportLeak(id, allocation)
		}
	}
}

// reportLeak reports a detected resource leak
func (rld *ResourceLeakDetector) reportLeak(id string, allocation *ResourceAllocation) {
	fmt.Printf("Resource leak detected: %s allocated at %s:%d:%d, age: %v\n",
		id, allocation.Location.File, allocation.Location.Line, allocation.Location.Column,
		time.Since(allocation.Timestamp))
}

// AddAllocation tracks a new resource allocation
func (rld *ResourceLeakDetector) AddAllocation(id string, allocation *ResourceAllocation) {
	rld.mutex.Lock()
	defer rld.mutex.Unlock()
	rld.allocations[id] = allocation
}

// RemoveAllocation removes tracking for a resource allocation
func (rld *ResourceLeakDetector) RemoveAllocation(id string) {
	rld.mutex.Lock()
	defer rld.mutex.Unlock()
	delete(rld.allocations, id)
}

// ResourceMonitor monitors resource usage and performance
type ResourceMonitor struct {
	metrics    map[string]*ResourceMetrics
	collectors []MetricCollector
	isRunning  bool
	mutex      sync.RWMutex
}

// ResourceMetrics represents metrics for a resource
type ResourceMetrics struct {
	ResourceId       string
	TotalAllocations int64
	TotalReleases    int64
	CurrentUsage     int64
	PeakUsage        int64
	AverageLifetime  time.Duration
	TotalLeaks       int64
	LastAccess       time.Time
}

// MetricCollector interface for collecting resource metrics
type MetricCollector interface {
	Collect(resourceId string, metrics *ResourceMetrics)
	Name() string
}

// NewResourceManager creates a new resource manager
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		resources:    make(map[string]*ManagedResourceType),
		allocations:  make(map[string]*ResourceAllocation),
		pools:        make(map[string]*ResourcePool),
		cleanupTasks: make([]CleanupTask, 0),
		leakDetector: NewResourceLeakDetector(time.Minute, time.Hour),
		monitor: &ResourceMonitor{
			metrics:    make(map[string]*ResourceMetrics),
			collectors: make([]MetricCollector, 0),
		},
	}
}

// RegisterResource registers a new resource type
func (rm *ResourceManager) RegisterResource(id string, resourceType *ManagedResourceType) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.resources[id]; exists {
		return fmt.Errorf("resource %s already registered", id)
	}

	resourceType.UniqueId = id
	rm.resources[id] = resourceType
	return nil
}

// AllocateResource allocates a resource
func (rm *ResourceManager) AllocateResource(resourceId, processId string, location SourceLocation) (*ResourceAllocation, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	resource, exists := rm.resources[resourceId]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceId)
	}

	if err := resource.Acquire(); err != nil {
		return nil, fmt.Errorf("failed to acquire resource %s: %v", resourceId, err)
	}

	allocation := &ResourceAllocation{
		ResourceId: resourceId,
		ProcessId:  processId,
		Timestamp:  time.Now(),
		Location:   location,
		IsActive:   true,
		RefCount:   1,
		Finalizers: make([]func(), 0),
	}

	allocationId := fmt.Sprintf("%s_%s_%d", resourceId, processId, allocation.Timestamp.Unix())
	rm.allocations[allocationId] = allocation

	// Track allocation for leak detection
	rm.leakDetector.AddAllocation(allocationId, allocation)

	return allocation, nil
}

// ReleaseResource releases a resource allocation
func (rm *ResourceManager) ReleaseResource(allocationId string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	allocation, exists := rm.allocations[allocationId]
	if !exists {
		return fmt.Errorf("allocation %s not found", allocationId)
	}

	if !allocation.IsActive {
		return fmt.Errorf("allocation %s already released", allocationId)
	}

	resource, exists := rm.resources[allocation.ResourceId]
	if !exists {
		return fmt.Errorf("resource %s not found", allocation.ResourceId)
	}

	// Execute finalizers
	for _, finalizer := range allocation.Finalizers {
		finalizer()
	}

	if err := resource.Release(); err != nil {
		return fmt.Errorf("failed to release resource %s: %v", allocation.ResourceId, err)
	}

	allocation.IsActive = false
	rm.leakDetector.RemoveAllocation(allocationId)

	return nil
}

// CreateResourcePool creates a new resource pool
func (rm *ResourceManager) CreateResourcePool(name string, kind ResourceTypeKind, minSize, maxSize int,
	factory func() (*ManagedResourceType, error)) (*ResourcePool, error) {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.pools[name]; exists {
		return nil, fmt.Errorf("resource pool %s already exists", name)
	}

	pool := &ResourcePool{
		Name:      name,
		Kind:      kind,
		MinSize:   minSize,
		MaxSize:   maxSize,
		Available: make([]*ManagedResourceType, 0, minSize),
		InUse:     make([]*ManagedResourceType, 0),
		Factory:   factory,
	}

	// Pre-populate with minimum resources
	for i := 0; i < minSize; i++ {
		if factory != nil {
			resource, err := factory()
			if err != nil {
				return nil, fmt.Errorf("failed to create initial resource: %v", err)
			}
			pool.Available = append(pool.Available, resource)
		}
	}

	rm.pools[name] = pool
	return pool, nil
}

// ScheduleCleanup schedules a cleanup task
func (rm *ResourceManager) ScheduleCleanup(task CleanupTask) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.cleanupTasks = append(rm.cleanupTasks, task)
}

// ExecuteCleanup executes all pending cleanup tasks
func (rm *ResourceManager) ExecuteCleanup() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	now := time.Now()
	executed := make([]int, 0)

	for i, task := range rm.cleanupTasks {
		if !task.IsExecuted && now.After(task.Deadline) {
			if err := task.Execute(); err != nil {
				fmt.Printf("Cleanup task %s failed: %v\n", task.Id, err)
			}
			executed = append(executed, i)
		}
	}

	// Remove executed tasks (in reverse order to maintain indices)
	for i := len(executed) - 1; i >= 0; i-- {
		idx := executed[i]
		rm.cleanupTasks = append(rm.cleanupTasks[:idx], rm.cleanupTasks[idx+1:]...)
	}
}

// ResourceTypeChecker performs resource type checking
type ResourceTypeChecker struct {
	manager     *ResourceManager
	allocations map[string][]string // process -> allocation IDs
	errors      []ResourceTypeError
	warnings    []ResourceWarning
	mutex       sync.RWMutex
}

// ResourceTypeError represents an error in resource type checking
type ResourceTypeError struct {
	Kind     ResourceErrorKind
	Resource string
	Process  string
	Location SourceLocation
	Message  string
}

// ResourceErrorKind represents kinds of resource type errors
type ResourceErrorKind int

const (
	ErrorKindResourceNotFound ResourceErrorKind = iota
	ErrorKindResourceLeak
	ErrorKindDoubleRelease
	ErrorKindUseAfterRelease
	ErrorKindConstraintViolation
	ErrorKindDependencyMissing
	ErrorKindPoolExhaustion
	ErrorKindLifetimeExpired
)

// String returns a string representation of the error kind
func (rek ResourceErrorKind) String() string {
	switch rek {
	case ErrorKindResourceNotFound:
		return "resource-not-found"
	case ErrorKindResourceLeak:
		return "resource-leak"
	case ErrorKindDoubleRelease:
		return "double-release"
	case ErrorKindUseAfterRelease:
		return "use-after-release"
	case ErrorKindConstraintViolation:
		return "constraint-violation"
	case ErrorKindDependencyMissing:
		return "dependency-missing"
	case ErrorKindPoolExhaustion:
		return "pool-exhaustion"
	case ErrorKindLifetimeExpired:
		return "lifetime-expired"
	default:
		return "unknown"
	}
}

// Error implements the error interface
func (rte ResourceTypeError) Error() string {
	return fmt.Sprintf("Resource error (%s) at %s:%d:%d: %s",
		rte.Kind.String(), rte.Location.File, rte.Location.Line, rte.Location.Column, rte.Message)
}

// ResourceWarning represents a warning in resource usage
type ResourceWarning struct {
	Kind     ResourceWarningKind
	Resource string
	Process  string
	Location SourceLocation
	Message  string
}

// ResourceWarningKind represents kinds of resource warnings
type ResourceWarningKind int

const (
	WarningKindPotentialLeak ResourceWarningKind = iota
	WarningKindHighUsage
	WarningKindLongLifetime
	WarningKindUnusedResource
	WarningKindFrequentAllocation
)

// String returns a string representation of the warning kind
func (rwk ResourceWarningKind) String() string {
	switch rwk {
	case WarningKindPotentialLeak:
		return "potential-leak"
	case WarningKindHighUsage:
		return "high-usage"
	case WarningKindLongLifetime:
		return "long-lifetime"
	case WarningKindUnusedResource:
		return "unused-resource"
	case WarningKindFrequentAllocation:
		return "frequent-allocation"
	default:
		return "unknown"
	}
}

// NewResourceTypeChecker creates a new resource type checker
func NewResourceTypeChecker(manager *ResourceManager) *ResourceTypeChecker {
	return &ResourceTypeChecker{
		manager:     manager,
		allocations: make(map[string][]string),
		errors:      make([]ResourceTypeError, 0),
		warnings:    make([]ResourceWarning, 0),
	}
}

// CheckResourceUsage checks resource usage in a process
func (rtc *ResourceTypeChecker) CheckResourceUsage(processId string) []ResourceTypeError {
	rtc.mutex.Lock()
	defer rtc.mutex.Unlock()

	errors := make([]ResourceTypeError, 0)
	allocations, exists := rtc.allocations[processId]

	if !exists {
		return errors
	}

	for _, allocationId := range allocations {
		allocation, exists := rtc.manager.allocations[allocationId]
		if !exists {
			continue
		}

		// Check for leaks
		if allocation.IsActive && time.Since(allocation.Timestamp) > time.Hour {
			error := ResourceTypeError{
				Kind:     ErrorKindResourceLeak,
				Resource: allocation.ResourceId,
				Process:  processId,
				Location: allocation.Location,
				Message: fmt.Sprintf("potential resource leak: %s not released after %v",
					allocation.ResourceId, time.Since(allocation.Timestamp)),
			}
			errors = append(errors, error)
		}
	}

	return errors
}

// ValidateResourceConstraints validates resource constraints
func (rtc *ResourceTypeChecker) ValidateResourceConstraints() []ResourceTypeError {
	rtc.mutex.RLock()
	defer rtc.mutex.RUnlock()

	errors := make([]ResourceTypeError, 0)

	for id, resource := range rtc.manager.resources {
		for _, constraint := range resource.Constraints {
			if err := constraint.Check(resource); err != nil {
				error := ResourceTypeError{
					Kind:     ErrorKindConstraintViolation,
					Resource: id,
					Message:  fmt.Sprintf("constraint violation: %v", err),
				}
				errors = append(errors, error)
			}
		}
	}

	return errors
}

// Resource type constructors

// NewFileResource creates a new file resource type
func NewFileResource(baseType *Type, path string) *ManagedResourceType {
	return &ManagedResourceType{
		BaseType:    baseType,
		Kind:        ResourceTypeKindFile,
		AcquireFunc: "file_open",
		ReleaseFunc: "file_close",
		State:       ResourceStateUninitialized,
		Lifetime:    &ResourceLifetime{},
		Constraints: []ResourceConstraint{
			{
				Kind:        ConstraintKindExclusive,
				Description: "File requires exclusive access",
				Check: func(rt *ManagedResourceType) error {
					if rt.IsShared {
						return fmt.Errorf("file resources cannot be shared")
					}
					return nil
				},
			},
		},
		UniqueId: generateResourceId(),
		Metadata: map[string]interface{}{
			"path": path,
		},
	}
}

// NewSocketResource creates a new socket resource type
func NewSocketResource(baseType *Type, address string, port int) *ManagedResourceType {
	return &ManagedResourceType{
		BaseType:      baseType,
		Kind:          ResourceTypeKindSocket,
		AcquireFunc:   "socket_create",
		ReleaseFunc:   "socket_close",
		State:         ResourceStateUninitialized,
		Lifetime:      &ResourceLifetime{},
		MaxUsageCount: 1,
		UniqueId:      generateResourceId(),
		Metadata: map[string]interface{}{
			"address": address,
			"port":    port,
		},
	}
}

// NewMemoryResource creates a new memory resource type
func NewMemoryResource(baseType *Type, size int) *ManagedResourceType {
	return &ManagedResourceType{
		BaseType:    baseType,
		Kind:        ResourceTypeKindMemoryAlloc,
		AcquireFunc: "malloc",
		ReleaseFunc: "free",
		Finalizer:   "memory_finalizer",
		State:       ResourceStateUninitialized,
		Lifetime:    &ResourceLifetime{},
		Constraints: []ResourceConstraint{
			{
				Kind:        ConstraintKindUsageLimit,
				Description: "Memory usage limit",
				Check: func(rt *ManagedResourceType) error {
					if rt.UsageCount > 1 {
						return fmt.Errorf("memory resource already in use")
					}
					return nil
				},
			},
		},
		UniqueId: generateResourceId(),
		Metadata: map[string]interface{}{
			"size": size,
		},
	}
}

// NewMutexResource creates a new mutex resource type
func NewMutexResource(baseType *Type) *ManagedResourceType {
	return &ManagedResourceType{
		BaseType:    baseType,
		Kind:        ResourceTypeKindMutexLock,
		AcquireFunc: "mutex_lock",
		ReleaseFunc: "mutex_unlock",
		State:       ResourceStateUninitialized,
		Lifetime:    &ResourceLifetime{},
		Constraints: []ResourceConstraint{
			{
				Kind:        ConstraintKindExclusive,
				Description: "Mutex requires exclusive access",
				Check: func(rt *ManagedResourceType) error {
					if rt.UsageCount > 1 {
						return fmt.Errorf("mutex already locked")
					}
					return nil
				},
			},
		},
		UniqueId: generateResourceId(),
	}
}

// NewDatabaseResource creates a new database resource type
func NewDatabaseResource(baseType *Type, connectionString string) *ManagedResourceType {
	return &ManagedResourceType{
		BaseType:      baseType,
		Kind:          ResourceTypeKindDatabaseConn,
		AcquireFunc:   "db_connect",
		ReleaseFunc:   "db_disconnect",
		State:         ResourceStateUninitialized,
		Lifetime:      &ResourceLifetime{},
		IsShared:      true,
		MaxUsageCount: 100, // Connection pool limit
		UniqueId:      generateResourceId(),
		Metadata: map[string]interface{}{
			"connection_string": connectionString,
		},
	}
}

// generateResourceId generates a unique identifier for resources
func generateResourceId() string {
	return fmt.Sprintf("resource_%d", time.Now().UnixNano())
}

// ResourceTypeKind constant for type system integration
const (
	TypeKindResource TypeKind = 202
)

// RAII (Resource Acquisition Is Initialization) pattern support

// RAIIWrapper wraps a resource with RAII semantics
type RAIIWrapper struct {
	Resource   *ManagedResourceType
	Allocation *ResourceAllocation
	Manager    *ResourceManager
	IsOwner    bool
}

// NewRAIIWrapper creates a new RAII wrapper
func NewRAIIWrapper(manager *ResourceManager, resourceId, processId string, location SourceLocation) (*RAIIWrapper, error) {
	allocation, err := manager.AllocateResource(resourceId, processId, location)
	if err != nil {
		return nil, err
	}

	resource := manager.resources[resourceId]

	return &RAIIWrapper{
		Resource:   resource,
		Allocation: allocation,
		Manager:    manager,
		IsOwner:    true,
	}, nil
}

// Release releases the wrapped resource
func (rw *RAIIWrapper) Release() error {
	if !rw.IsOwner {
		return fmt.Errorf("not owner of resource")
	}

	allocationId := fmt.Sprintf("%s_%s_%d", rw.Allocation.ResourceId,
		rw.Allocation.ProcessId, rw.Allocation.Timestamp.Unix())

	err := rw.Manager.ReleaseResource(allocationId)
	if err != nil {
		return err
	}

	rw.IsOwner = false
	return nil
}

// Move transfers ownership of the resource
func (rw *RAIIWrapper) Move() *RAIIWrapper {
	if !rw.IsOwner {
		return nil
	}

	newWrapper := &RAIIWrapper{
		Resource:   rw.Resource,
		Allocation: rw.Allocation,
		Manager:    rw.Manager,
		IsOwner:    true,
	}

	rw.IsOwner = false
	return newWrapper
}

// Smart pointer-like functionality

// ResourcePtr represents a smart pointer to a resource
type ResourcePtr struct {
	Resource *ManagedResourceType
	RefCount *int
	Manager  *ResourceManager
	mutex    sync.Mutex
}

// NewResourcePtr creates a new resource pointer
func NewResourcePtr(manager *ResourceManager, resource *ManagedResourceType) *ResourcePtr {
	refCount := 1
	return &ResourcePtr{
		Resource: resource,
		RefCount: &refCount,
		Manager:  manager,
	}
}

// Clone creates a new reference to the same resource
func (rp *ResourcePtr) Clone() *ResourcePtr {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	*rp.RefCount++
	return &ResourcePtr{
		Resource: rp.Resource,
		RefCount: rp.RefCount,
		Manager:  rp.Manager,
	}
}

// Release decreases reference count and releases if zero
func (rp *ResourcePtr) Release() error {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	*rp.RefCount--
	if *rp.RefCount <= 0 {
		return rp.Resource.Release()
	}

	return nil
}
