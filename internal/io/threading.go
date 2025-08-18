// Package io provides basic threading and concurrency primitives for the Orizon runtime.
// This implements the minimal threading functionality required for self-hosting,
// including threads, mutexes, channels, and basic synchronization primitives.
package io

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/allocator"
)

// ThreadID represents a unique thread identifier
type ThreadID uint64

// ThreadState represents the current state of a thread
type ThreadState int

const (
	ThreadStateCreated ThreadState = iota
	ThreadStateRunning
	ThreadStateBlocked
	ThreadStateFinished
	ThreadStateError
)

// String returns the string representation of ThreadState
func (s ThreadState) String() string {
	switch s {
	case ThreadStateCreated:
		return "created"
	case ThreadStateRunning:
		return "running"
	case ThreadStateBlocked:
		return "blocked"
	case ThreadStateFinished:
		return "finished"
	case ThreadStateError:
		return "error"
	default:
		return "unknown"
	}
}

// ThreadPriority represents thread priority levels
type ThreadPriority int

const (
	ThreadPriorityLow ThreadPriority = iota
	ThreadPriorityNormal
	ThreadPriorityHigh
)

// ThreadFunction represents a function that can be executed in a thread
type ThreadFunction func(data unsafe.Pointer) int

// Thread represents a thread handle
type Thread struct {
	id       ThreadID
	function ThreadFunction
	data     unsafe.Pointer
	state    ThreadState
	priority ThreadPriority
	started  time.Time
	finished time.Time
	result   int
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
}

// GetID returns the thread ID
func (t *Thread) GetID() ThreadID {
	return t.id
}

// GetState returns the current thread state
func (t *Thread) GetState() ThreadState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// SetState sets the thread state
func (t *Thread) setState(state ThreadState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = state
}

// GetResult returns the thread result (only valid after completion)
func (t *Thread) GetResult() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.result
}

// Cancel cancels the thread execution
func (t *Thread) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// IsFinished returns true if the thread has finished execution
func (t *Thread) IsFinished() bool {
	state := t.GetState()
	return state == ThreadStateFinished || state == ThreadStateError
}

// Mutex represents a mutual exclusion lock
type Mutex struct {
	mu     sync.Mutex
	locked int32
	owner  ThreadID
}

// NewMutex creates a new mutex
func NewMutex() *Mutex {
	return &Mutex{}
}

// Lock locks the mutex
func (m *Mutex) Lock() {
	m.mu.Lock()
	atomic.StoreInt32(&m.locked, 1)
	m.owner = GetCurrentThreadID()
}

// Unlock unlocks the mutex
func (m *Mutex) Unlock() {
	m.owner = 0
	atomic.StoreInt32(&m.locked, 0)
	m.mu.Unlock()
}

// TryLock attempts to lock the mutex without blocking
func (m *Mutex) TryLock() bool {
	acquired := atomic.CompareAndSwapInt32(&m.locked, 0, 1)
	if acquired {
		m.owner = GetCurrentThreadID()
		m.mu.Lock()
		return true
	}
	return false
}

// IsLocked returns true if the mutex is currently locked
func (m *Mutex) IsLocked() bool {
	return atomic.LoadInt32(&m.locked) == 1
}

// GetOwner returns the ID of the thread that owns the mutex
func (m *Mutex) GetOwner() ThreadID {
	return m.owner
}

// RWMutex represents a reader-writer mutex
type RWMutex struct {
	mu      sync.RWMutex
	readers int32
	writer  ThreadID
}

// NewRWMutex creates a new reader-writer mutex
func NewRWMutex() *RWMutex {
	return &RWMutex{}
}

// RLock locks for reading
func (rw *RWMutex) RLock() {
	rw.mu.RLock()
	atomic.AddInt32(&rw.readers, 1)
}

// RUnlock unlocks for reading
func (rw *RWMutex) RUnlock() {
	atomic.AddInt32(&rw.readers, -1)
	rw.mu.RUnlock()
}

// Lock locks for writing
func (rw *RWMutex) Lock() {
	rw.mu.Lock()
	rw.writer = GetCurrentThreadID()
}

// Unlock unlocks for writing
func (rw *RWMutex) Unlock() {
	rw.writer = 0
	rw.mu.Unlock()
}

// GetReaderCount returns the number of active readers
func (rw *RWMutex) GetReaderCount() int32 {
	return atomic.LoadInt32(&rw.readers)
}

// GetWriter returns the ID of the writing thread (0 if no writer)
func (rw *RWMutex) GetWriter() ThreadID {
	return rw.writer
}

// ConditionVariable represents a condition variable for thread synchronization
type ConditionVariable struct {
	cond *sync.Cond
	mu   *Mutex
}

// NewConditionVariable creates a new condition variable
func NewConditionVariable(mu *Mutex) *ConditionVariable {
	return &ConditionVariable{
		cond: sync.NewCond(&mu.mu),
		mu:   mu,
	}
}

// Wait waits for the condition to be signaled
func (cv *ConditionVariable) Wait() {
	cv.cond.Wait()
}

// Signal wakes up one waiting thread
func (cv *ConditionVariable) Signal() {
	cv.cond.Signal()
}

// Broadcast wakes up all waiting threads
func (cv *ConditionVariable) Broadcast() {
	cv.cond.Broadcast()
}

// Channel represents a communication channel between threads
type Channel struct {
	data     chan unsafe.Pointer
	capacity int
	closed   int32
	mu       sync.RWMutex
}

// NewChannel creates a new channel with the specified capacity
func NewChannel(capacity int) *Channel {
	if capacity < 0 {
		capacity = 0
	}

	return &Channel{
		data:     make(chan unsafe.Pointer, capacity),
		capacity: capacity,
	}
}

// Send sends data to the channel
func (ch *Channel) Send(data unsafe.Pointer) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if atomic.LoadInt32(&ch.closed) == 1 {
		return false
	}

	select {
	case ch.data <- data:
		return true
	default:
		return false // Channel full
	}
}

// Receive receives data from the channel
func (ch *Channel) Receive() (unsafe.Pointer, bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	select {
	case data := <-ch.data:
		return data, true
	default:
		return nil, false // Channel empty
	}
}

// SendBlocking sends data to the channel, blocking if necessary
func (ch *Channel) SendBlocking(data unsafe.Pointer) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if atomic.LoadInt32(&ch.closed) == 1 {
		return false
	}

	ch.data <- data
	return true
}

// ReceiveBlocking receives data from the channel, blocking if necessary
func (ch *Channel) ReceiveBlocking() (unsafe.Pointer, bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	data, ok := <-ch.data
	return data, ok
}

// Close closes the channel
func (ch *Channel) Close() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if atomic.CompareAndSwapInt32(&ch.closed, 0, 1) {
		close(ch.data)
	}
}

// IsClosed returns true if the channel is closed
func (ch *Channel) IsClosed() bool {
	return atomic.LoadInt32(&ch.closed) == 1
}

// Len returns the number of elements in the channel
func (ch *Channel) Len() int {
	return len(ch.data)
}

// Cap returns the capacity of the channel
func (ch *Channel) Cap() int {
	return ch.capacity
}

// ThreadManager manages thread creation and lifecycle
type ThreadManager struct {
	mu           sync.RWMutex
	nextThreadID uint64
	threads      map[ThreadID]*Thread
	allocator    allocator.Allocator
	maxThreads   int
	defaultStack int
	stats        ThreadStats
	shutdownChan chan struct{}
	shutdownOnce sync.Once
}

// ThreadStats provides threading statistics
type ThreadStats struct {
	ThreadsCreated  uint64
	ThreadsFinished uint64
	ThreadsActive   int32
	ThreadsBlocked  int32
	ContextSwitches uint64
	TotalRuntime    time.Duration
}

// GlobalThreadManager is the global thread manager instance
var GlobalThreadManager *ThreadManager

// InitializeThreading initializes the global thread manager
func InitializeThreading(allocator allocator.Allocator, options ...ThreadOption) error {
	if allocator == nil {
		return fmt.Errorf("allocator cannot be nil")
	}

	manager := &ThreadManager{
		threads:      make(map[ThreadID]*Thread),
		allocator:    allocator,
		maxThreads:   1024,
		defaultStack: 64 * 1024, // 64KB default stack
		shutdownChan: make(chan struct{}),
	}

	// Apply options
	for _, opt := range options {
		opt(manager)
	}

	GlobalThreadManager = manager
	return nil
}

// ThreadOption configures the thread manager
type ThreadOption func(*ThreadManager)

// WithMaxThreads sets the maximum number of threads
func WithMaxThreads(max int) ThreadOption {
	return func(tm *ThreadManager) { tm.maxThreads = max }
}

// WithDefaultStackSize sets the default stack size
func WithDefaultStackSize(size int) ThreadOption {
	return func(tm *ThreadManager) { tm.defaultStack = size }
}

// CreateThread creates a new thread
func (tm *ThreadManager) CreateThread(function ThreadFunction, data unsafe.Pointer, priority ThreadPriority) (*Thread, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check thread limit
	if len(tm.threads) >= tm.maxThreads {
		return nil, fmt.Errorf("maximum thread limit reached")
	}

	// Generate thread ID
	threadID := ThreadID(atomic.AddUint64(&tm.nextThreadID, 1))

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create thread
	thread := &Thread{
		id:       threadID,
		function: function,
		data:     data,
		state:    ThreadStateCreated,
		priority: priority,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Store thread
	tm.threads[threadID] = thread

	// Update statistics
	atomic.AddUint64(&tm.stats.ThreadsCreated, 1)
	atomic.AddInt32(&tm.stats.ThreadsActive, 1)

	return thread, nil
}

// StartThread starts execution of a thread
func (tm *ThreadManager) StartThread(thread *Thread) error {
	if thread == nil {
		return fmt.Errorf("thread cannot be nil")
	}

	thread.setState(ThreadStateRunning)
	thread.started = time.Now()

	// Start the goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				thread.setState(ThreadStateError)
				thread.result = -1
			}
			thread.finished = time.Now()
			atomic.AddUint64(&tm.stats.ThreadsFinished, 1)
			atomic.AddInt32(&tm.stats.ThreadsActive, -1)
		}()

		// Execute the thread function
		result := thread.function(thread.data)
		thread.result = result
		thread.setState(ThreadStateFinished)
	}()

	return nil
}

// JoinThread waits for a thread to complete
func (tm *ThreadManager) JoinThread(thread *Thread, timeoutMs int) (int, error) {
	if thread == nil {
		return -1, fmt.Errorf("thread cannot be nil")
	}

	if timeoutMs <= 0 {
		// Wait indefinitely
		for !thread.IsFinished() {
			time.Sleep(time.Millisecond)
		}
	} else {
		// Wait with timeout
		timeout := time.Duration(timeoutMs) * time.Millisecond
		deadline := time.Now().Add(timeout)

		for !thread.IsFinished() && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		if !thread.IsFinished() {
			return -1, fmt.Errorf("thread join timed out")
		}
	}

	return thread.GetResult(), nil
}

// DetachThread detaches a thread (allows it to run independently)
func (tm *ThreadManager) DetachThread(thread *Thread) error {
	if thread == nil {
		return fmt.Errorf("thread cannot be nil")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Remove from managed threads
	delete(tm.threads, thread.id)

	return nil
}

// GetThread returns a thread by ID
func (tm *ThreadManager) GetThread(id ThreadID) *Thread {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.threads[id]
}

// GetActiveThreads returns the number of active threads
func (tm *ThreadManager) GetActiveThreads() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.threads)
}

// GetStats returns threading statistics
func (tm *ThreadManager) GetStats() ThreadStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.stats
}

// Sleep pauses the current thread for the specified duration
func (tm *ThreadManager) Sleep(milliseconds int) {
	duration := time.Duration(milliseconds) * time.Millisecond
	time.Sleep(duration)
}

// Yield yields the current thread's time slice
func (tm *ThreadManager) Yield() {
	runtime.Gosched()
}

// GetCurrentThreadID returns the ID of the current thread (goroutine)
func GetCurrentThreadID() ThreadID {
	// Note: In Go, there's no direct equivalent to thread IDs
	// This is a simplified implementation
	return ThreadID(runtime.NumGoroutine())
}

// Shutdown shuts down the thread manager
func (tm *ThreadManager) Shutdown() error {
	var shutdownError error

	tm.shutdownOnce.Do(func() {
		tm.mu.Lock()
		defer tm.mu.Unlock()

		// Cancel all threads
		for _, thread := range tm.threads {
			thread.Cancel()
		}

		// Wait for threads to finish (with timeout)
		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for len(tm.threads) > 0 {
			select {
			case <-timeout:
				shutdownError = fmt.Errorf("thread shutdown timed out, %d threads still active", len(tm.threads))
				return
			case <-ticker.C:
				// Remove finished threads
				for id, thread := range tm.threads {
					if thread.IsFinished() {
						delete(tm.threads, id)
					}
				}
			}
		}

		close(tm.shutdownChan)
	})

	return shutdownError
}

// Global convenience functions

// CreateThread creates a thread using the global thread manager
func CreateThread(function ThreadFunction, data unsafe.Pointer, priority ThreadPriority) (*Thread, error) {
	if GlobalThreadManager == nil {
		return nil, fmt.Errorf("thread manager not initialized")
	}
	return GlobalThreadManager.CreateThread(function, data, priority)
}

// StartThread starts a thread using the global thread manager
func StartThread(thread *Thread) error {
	if GlobalThreadManager == nil {
		return fmt.Errorf("thread manager not initialized")
	}
	return GlobalThreadManager.StartThread(thread)
}

// JoinThread joins a thread using the global thread manager
func JoinThread(thread *Thread, timeoutMs int) (int, error) {
	if GlobalThreadManager == nil {
		return -1, fmt.Errorf("thread manager not initialized")
	}
	return GlobalThreadManager.JoinThread(thread, timeoutMs)
}

// Sleep pauses the current thread
func Sleep(milliseconds int) {
	if GlobalThreadManager == nil {
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)
		return
	}
	GlobalThreadManager.Sleep(milliseconds)
}

// Yield yields the current thread
func Yield() {
	if GlobalThreadManager == nil {
		runtime.Gosched()
		return
	}
	GlobalThreadManager.Yield()
}

// GetThreadStats returns global threading statistics
func GetThreadStats() ThreadStats {
	if GlobalThreadManager == nil {
		return ThreadStats{}
	}
	return GlobalThreadManager.GetStats()
}

// ShutdownThreading shuts down the global thread manager
func ShutdownThreading() error {
	if GlobalThreadManager == nil {
		return nil
	}
	return GlobalThreadManager.Shutdown()
}
