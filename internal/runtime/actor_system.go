// Phase 3.2.1: Actor System Concurrency Model
// This file implements a comprehensive actor-based concurrency system with mailboxes,
// message passing, supervision trees, and fault tolerance for Orizon runtime.

package runtime

import (
	"context"
	"fmt"
	"net"
	stdrt "runtime"
	"sync"
	"sync/atomic"
	"time"

	asyncio "github.com/orizon-lang/orizon/internal/runtime/asyncio"
)

// Type definitions for actor system
type (
	ActorID      uint64 // Actor identifier
	MessageID    uint64 // Message identifier
	SupervisorID uint64 // Supervisor identifier
	MailboxID    uint64 // Mailbox identifier
	ActorGroupID uint64 // Actor group identifier
	MessageType  uint32 // Message type identifier
)

// Reserved system message types
const (
	SystemTerminated MessageType = 0xFFFF0001
)

// I/O event message types for asyncio integration
const (
	IOReadable MessageType = 0x00010001
	IOWritable MessageType = 0x00010002
	IOErrorEvt MessageType = 0x00010003
)

// IOEvent carries I/O readiness information to actors
type IOEvent struct {
	Conn net.Conn
	Type asyncio.EventType
	Err  error
}

// Actor system main structure
type ActorSystem struct {
	actors      map[ActorID]*Actor           // Active actors
	supervisors map[SupervisorID]*Supervisor // Supervisors
	mailboxes   map[MailboxID]*Mailbox       // Mailboxes
	groups      map[ActorGroupID]*ActorGroup // Actor groups
	registry    *ActorRegistry               // Actor registry
	scheduler   *ActorScheduler              // Actor scheduler
	dispatcher  *MessageDispatcher           // Message dispatcher
	// Group management uses registry.groups (name -> groupID) and ActorSystem.groups (id -> group)
	config         ActorSystemConfig     // System configuration
	statistics     ActorSystemStatistics // System statistics
	running        bool                  // System running
	ctx            context.Context       // System context
	cancel         context.CancelFunc    // Cancel function
	rootSupervisor *Supervisor           // Root supervisor
	mutex          sync.RWMutex          // Synchronization
	// Remote enables distributed actor messaging integration when attached.
	Remote interface {
		Send(remoteAddrOrNode, receiverName string, msgType uint32, payload interface{}) error
	} // minimal interface to avoid import cycle in runtime package
	// ioPoller integrates asyncio events with the actor system when attached.
	ioPoller asyncio.Poller
}

// Actor represents an individual actor
type Actor struct {
	ID            ActorID            // Actor identifier
	Name          string             // Actor name
	Type          ActorType          // Actor type
	State         ActorState         // Current state
	Mailbox       *Mailbox           // Actor mailbox
	Supervisor    *Supervisor        // Supervisor
	Parent        *Actor             // Parent actor
	Children      map[ActorID]*Actor // Child actors
	Behavior      ActorBehavior      // Actor behavior
	Config        ActorConfig        // Actor configuration
	Statistics    ActorStatistics    // Actor statistics
	Context       *ActorContext      // Actor context
	LastHeartbeat time.Time          // Last heartbeat
	RestartCount  uint32             // Restart count
	CreateTime    time.Time          // Creation time
	mutex         sync.RWMutex       // Actor synchronization
}

// Mailbox for message storage and delivery
type Mailbox struct {
	ID               MailboxID             // Mailbox identifier
	Owner            ActorID               // Owning actor
	Type             MailboxType           // Mailbox type
	Capacity         uint32                // Maximum capacity
	Messages         []Message             // Stored messages
	PriorityQueue    *MessagePriorityQueue // Priority queue
	DeadLetters      []Message             // Dead letter queue
	Filters          []MessageFilter       // Message filters
	Statistics       MailboxStatistics     // Mailbox statistics
	OverflowPolicy   MailboxOverflowPolicy // Overflow handling
	LastActivity     time.Time             // Last activity
	BackPressureWait time.Duration         // Maximum wait time when applying back pressure
	mutex            sync.RWMutex          // Mailbox synchronization
	// notFull is an edge-triggered notification channel used to wake enqueuers
	// waiting for capacity under BackPressure policy.
	notFull chan struct{}
}

// Message represents communication between actors
type Message struct {
	ID            MessageID              // Message identifier
	Type          MessageType            // Message type
	Sender        ActorID                // Sender actor
	Receiver      ActorID                // Receiver actor
	Payload       interface{}            // Message payload
	Priority      MessagePriority        // Message priority
	Timestamp     time.Time              // Send timestamp
	TTL           time.Duration          // Time to live
	Deadline      time.Time              // Delivery deadline
	Headers       map[string]interface{} // Message headers
	ReplyTo       ActorID                // Reply destination
	CorrelationID string                 // Correlation ID
	Persistent    bool                   // Persistent message
	Delivered     bool                   // Delivery status
}

// Supervisor manages actor lifecycle and fault tolerance
type Supervisor struct {
	ID          SupervisorID         // Supervisor identifier
	Name        string               // Supervisor name
	Type        SupervisorType       // Supervisor type
	Strategy    SupervisionStrategy  // Supervision strategy
	Children    map[ActorID]*Actor   // Supervised actors
	childOrder  []ActorID            // Children creation order
	MaxRetries  uint32               // Maximum retries
	RetryPeriod time.Duration        // Retry period
	Escalations []EscalationRule     // Escalation rules
	Monitor     *SupervisorMonitor   // Monitor
	Statistics  SupervisorStatistics // Statistics
	Config      SupervisorConfig     // Configuration
	Parent      *Supervisor          // Parent supervisor
	CreateTime  time.Time            // Creation time
	mutex       sync.RWMutex         // Synchronization
	// restartTrack keeps per-child restart timestamps to enforce MaxRetries/RetryPeriod.
	restartTrack map[ActorID][]time.Time
}

// Actor registry for name resolution and discovery
type ActorRegistry struct {
	nameToID   map[string]ActorID      // Name to ID mapping
	idToActor  map[ActorID]*Actor      // ID to actor mapping
	groups     map[string]ActorGroupID // Group registry
	patterns   []RegistryPattern       // Pattern matching
	cache      map[string]ActorID      // Resolution cache
	statistics RegistryStatistics      // Registry statistics
	enabled    bool                    // Registry enabled
	mutex      sync.RWMutex            // Synchronization
}

// Actor scheduler for fair execution
type ActorScheduler struct {
	queues     map[ActorPriority]*ActorQueue // Priority queues
	roundRobin []ActorID                     // Round-robin actors
	workers    []*SchedulerWorker            // Worker threads
	strategies []SchedulingStrategy          // Scheduling strategies
	config     SchedulerConfig               // Configuration
	statistics SchedulerStatistics           // Statistics
	running    bool                          // Scheduler running
	ctx        context.Context               // Context
	// process is a callback invoked by workers to process one scheduled actor.
	// The callback must be set by the owning ActorSystem.
	process         func(ActorID)
	resolvePriority func(ActorID) ActorPriority
	mutex           sync.RWMutex // Synchronization
}

// Message dispatcher for routing
type MessageDispatcher struct {
	routes       map[MessageType][]DispatchRule // Routing rules
	interceptors []MessageInterceptor           // Message interceptors
	transformers []MessageTransformer           // Message transformers
	serializers  map[string]MessageSerializer   // Serializers
	statistics   DispatcherStatistics           // Statistics
	config       DispatcherConfig               // Configuration
	enabled      bool                           // Dispatcher enabled
	mutex        sync.RWMutex                   // Synchronization
}

// Actor group for collective operations
type ActorGroup struct {
	ID           ActorGroupID       // Group identifier
	Name         string             // Group name
	Type         ActorGroupType     // Group type
	Members      map[ActorID]*Actor // Group members
	Leader       ActorID            // Group leader
	Consensus    *ConsensusProtocol // Consensus protocol
	Broadcast    *BroadcastProtocol // Broadcast protocol
	LoadBalancer *GroupLoadBalancer // Load balancer
	Statistics   GroupStatistics    // Group statistics
	Config       GroupConfig        // Group configuration
	CreateTime   time.Time          // Creation time
	mutex        sync.RWMutex       // Synchronization
}

// Supporting data structures

// Actor context for execution environment
type ActorContext struct {
	ActorID  ActorID                // Current actor ID
	System   *ActorSystem           // Actor system
	Sender   ActorID                // Current sender
	Self     *Actor                 // Self reference
	Stash    []Message              // Stashed messages
	Timers   map[string]*ActorTimer // Active timers
	Watchers map[ActorID]bool       // Death watchers
	Watched  map[ActorID]bool       // Watched actors
	Props    map[string]interface{} // Properties
	Logger   ActorLogger            // Actor logger
}

// Actor behavior interface
type ActorBehavior interface {
	Receive(ctx *ActorContext, msg Message) error
	PreStart(ctx *ActorContext) error
	PostStop(ctx *ActorContext) error
	PreRestart(ctx *ActorContext, reason error, message *Message) error
	PostRestart(ctx *ActorContext, reason error) error
	GetBehaviorName() string
}

// Message priority queue
type MessagePriorityQueue struct {
	items    []PriorityMessage // Queue items
	size     int               // Current size
	capacity int               // Maximum capacity
	mutex    sync.RWMutex      // Queue synchronization
}

// Priority message wrapper
type PriorityMessage struct {
	Message    Message   // Wrapped message
	Priority   int       // Message priority
	InsertTime time.Time // Insert time
}

// Enumeration types

// Actor types
type ActorType int

const (
	UserActor ActorType = iota
	SystemActor
	ProxyActor
	RouterActor
	WorkerActor
	SupervisorActor
)

// Actor states
type ActorState int

const (
	ActorIdle ActorState = iota
	ActorBusy
	ActorWaiting
	ActorRestarting
	ActorStopping
	ActorStopped
	ActorFailed
)

// Mailbox types
type MailboxType int

const (
	StandardMailbox MailboxType = iota
	PriorityMailbox
	BoundedMailbox
	UnboundedMailbox
	StashingMailbox
)

// Message priorities
type MessagePriority int

const (
	LowPriority MessagePriority = iota
	NormalPriority
	HighPriority
	SystemPriority
	CriticalPriority
)

// Supervisor types
type SupervisorType int

const (
	OneForOne SupervisorType = iota
	OneForAll
	RestForOne
	SimpleOneForOne
)

// Supervision strategies
type SupervisionStrategy int

const (
	RestartStrategy SupervisionStrategy = iota
	ResumeStrategy
	StopStrategy
	EscalateStrategy
)

// Actor priorities
type ActorPriority int

const (
	LowActorPriority ActorPriority = iota
	NormalActorPriority
	HighActorPriority
	SystemActorPriority
)

// Mailbox overflow policies
type MailboxOverflowPolicy int

const (
	DropOldest MailboxOverflowPolicy = iota
	DropNewest
	DropLowPriority
	BackPressure
	DeadLetter
)

// Actor group types
type ActorGroupType int

const (
	StaticGroup ActorGroupType = iota
	DynamicGroup
	ReplicatedGroup
	PartitionedGroup
	HierarchicalGroup
)

// Scheduling strategies
type SchedulingStrategy int

const (
	FairScheduling SchedulingStrategy = iota
	PriorityScheduling
	RoundRobinScheduling
	WorkStealingScheduling
	LoadBasedScheduling
)

// Configuration types

// Actor system configuration
type ActorSystemConfig struct {
	MaxActors          uint32        // Maximum actors
	MaxSupervisors     uint32        // Maximum supervisors
	DefaultMailboxSize uint32        // Default mailbox size
	HeartbeatInterval  time.Duration // Heartbeat interval
	GCInterval         time.Duration // Garbage collection interval
	ShutdownTimeout    time.Duration // Shutdown timeout
	EnableMetrics      bool          // Enable metrics
	EnableTracing      bool          // Enable tracing
	EnableDeadLetters  bool          // Enable dead letters
	// DefaultIOWatchOptions provides system-wide defaults for I/O watcher registration.
	// Methods like WatchConnWithActorDefault will use these defaults.
	DefaultIOWatchOptions IOWatchOptions
}

// Actor configuration
type ActorConfig struct {
	MailboxType     MailboxType   // Mailbox type
	MailboxCapacity uint32        // Mailbox capacity
	Priority        ActorPriority // Actor priority
	RestartDelay    time.Duration // Restart delay
	MaxRestarts     uint32        // Maximum restarts
	RestartWindow   time.Duration // Restart window
	EnableStashing  bool          // Enable stashing
	EnableWatching  bool          // Enable death watching
	// CPUAffinityMask provides a scheduling hint for CPU/worker affinity.
	// Each scheduler worker is assigned a CPU bitmask. When set to non-zero,
	// the scheduler prefers workers whose mask overlaps with this mask while
	// still applying load-aware selection.
	CPUAffinityMask uint64
}

// Supervisor configuration
type SupervisorConfig struct {
	Strategy          SupervisionStrategy // Supervision strategy
	MaxRetries        uint32              // Maximum retries
	RetryPeriod       time.Duration       // Retry period
	EscalationTimeout time.Duration       // Escalation timeout
	EnableMonitoring  bool                // Enable monitoring
	EnableLogging     bool                // Enable logging
}

// Scheduler configuration
type SchedulerConfig struct {
	WorkerCount          int                // Number of workers
	Strategy             SchedulingStrategy // Scheduling strategy
	Fairness             float64            // Fairness factor
	PreemptionEnabled    bool               // Enable preemption
	WorkStealingEnabled  bool               // Enable work stealing
	LoadBalancingEnabled bool               // Enable load balancing
}

// Dispatcher configuration
type DispatcherConfig struct {
	BufferSize           uint32        // Buffer size
	EnableRouting        bool          // Enable routing
	EnableInterception   bool          // Enable interception
	EnableTransformation bool          // Enable transformation
	EnableSerialization  bool          // Enable serialization
	DefaultTimeout       time.Duration // Default timeout
}

// Group configuration
type GroupConfig struct {
	MaxMembers          uint32        // Maximum members
	LeaderElection      bool          // Enable leader election
	EnableConsensus     bool          // Enable consensus
	EnableBroadcast     bool          // Enable broadcast
	EnableLoadBalancing bool          // Enable load balancing
	SyncTimeout         time.Duration // Sync timeout
}

// Supporting interface types

// Message filter interface
type MessageFilter interface {
	Filter(msg Message) bool
	GetFilterName() string
}

// Message interceptor interface
type MessageInterceptor interface {
	Intercept(msg Message) (Message, error)
	GetInterceptorName() string
}

// Message transformer interface
type MessageTransformer interface {
	Transform(msg Message) (Message, error)
	GetTransformerName() string
}

// Message serializer interface
type MessageSerializer interface {
	Serialize(msg Message) ([]byte, error)
	Deserialize(data []byte) (Message, error)
	GetSerializerName() string
}

// Actor logger interface
type ActorLogger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Statistics types

// Actor system statistics
type ActorSystemStatistics struct {
	TotalActors       uint64    // Total actors created
	ActiveActors      uint64    // Currently active actors
	TotalMessages     uint64    // Total messages sent
	ProcessedMessages uint64    // Processed messages
	DroppedMessages   uint64    // Dropped messages
	DeadLetters       uint64    // Dead letter count
	Restarts          uint64    // Total restarts
	Failures          uint64    // Total failures
	LastReset         time.Time // Last statistics reset
}

// Actor statistics
type ActorStatistics struct {
	MessagesReceived      uint64        // Messages received
	MessagesProcessed     uint64        // Messages processed
	MessagesFailed        uint64        // Messages failed
	ProcessingTime        time.Duration // Total processing time
	AverageProcessingTime time.Duration // Average processing time
	Restarts              uint32        // Restart count
	LastActivity          time.Time     // Last activity
}

// Mailbox statistics
type MailboxStatistics struct {
	MessagesEnqueued uint64    // Messages enqueued
	MessagesDequeued uint64    // Messages dequeued
	MessagesDropped  uint64    // Messages dropped
	CurrentSize      uint32    // Current size
	MaxSize          uint32    // Maximum size reached
	OverflowCount    uint32    // Overflow occurrences
	LastEnqueue      time.Time // Last enqueue
	LastDequeue      time.Time // Last dequeue
}

// Supervisor statistics
type SupervisorStatistics struct {
	ChildrenCreated      uint64    // Children created
	ChildrenRestarted    uint64    // Children restarted
	ChildrenStopped      uint64    // Children stopped
	EscalationsTriggered uint64    // Escalations triggered
	StrategiesApplied    uint64    // Strategies applied
	LastAction           time.Time // Last action
}

// Additional supporting types
type (
	EscalationRule struct {
		Condition string
		Action    SupervisionStrategy
		Timeout   time.Duration
	}
	SupervisorMonitor struct {
		Enabled  bool
		Interval time.Duration
		Alerts   []string
	}
	RegistryPattern struct {
		Pattern string
		Handler func(string) ActorID
	}
	RegistryStatistics struct{ Lookups, Registrations, Evictions uint64 }
	ActorQueue         struct {
		items []ActorID
		mutex sync.Mutex
	}
	SchedulerWorker struct {
		ID      int
		Queue   chan ActorID
		Running bool
		// CPUMask is the worker's CPU affinity mask used for biased scheduling.
		CPUMask uint64
		// QueueLen tracks the current length of the worker queue for load-aware scheduling.
		QueueLen int64
	}
	SchedulerStatistics struct{ TasksScheduled, TasksCompleted, WorkerUtilization uint64 }
	DispatchRule        struct {
		Condition string
		Target    ActorID
		Priority  int
	}
	DispatcherStatistics struct{ MessagesRouted, InterceptionsApplied, TransformationsApplied uint64 }
	ConsensusProtocol    struct {
		Type         string
		Participants []ActorID
		State        string
	}
	BroadcastProtocol struct {
		Type        string
		Reliability string
		Ordering    string
	}
	GroupLoadBalancer struct {
		Strategy string
		Weights  map[ActorID]float64
	}
	GroupStatistics struct{ MessagesBroadcast, ConsensusReached, LeaderChanges uint64 }
	ActorTimer      struct {
		ID       string
		Interval time.Duration
		Callback func()
		timer    *time.Timer
	}
)

// StartTimer starts or restarts a named timer on the actor context.
func (ctx *ActorContext) StartTimer(id string, interval time.Duration, cb func()) {
	if ctx.Timers == nil {
		ctx.Timers = make(map[string]*ActorTimer)
	}
	// Stop existing
	if t, ok := ctx.Timers[id]; ok && t != nil && t.timer != nil {
		t.timer.Stop()
	}
	timer := time.AfterFunc(interval, cb)
	ctx.Timers[id] = &ActorTimer{ID: id, Interval: interval, Callback: cb, timer: timer}
}

// StopTimer stops and removes a named timer.
func (ctx *ActorContext) StopTimer(id string) {
	if ctx.Timers == nil {
		return
	}
	if t, ok := ctx.Timers[id]; ok && t != nil && t.timer != nil {
		t.timer.Stop()
	}
	delete(ctx.Timers, id)
}

// Default configurations
var DefaultActorSystemConfig = ActorSystemConfig{
	MaxActors:          10000,
	MaxSupervisors:     1000,
	DefaultMailboxSize: 1000,
	HeartbeatInterval:  time.Second * 30,
	GCInterval:         time.Minute * 5,
	ShutdownTimeout:    time.Second * 30,
	EnableMetrics:      true,
	EnableTracing:      false,
	EnableDeadLetters:  true,
	DefaultIOWatchOptions: IOWatchOptions{
		BackoffInitial:     time.Millisecond * 5,
		BackoffMax:         time.Millisecond * 100,
		HighWatermark:      0,
		LowWatermark:       0,
		MonitorInterval:    time.Millisecond * 10,
		ReadEventPriority:  NormalPriority,
		WriteEventPriority: NormalPriority,
		ErrorEventPriority: HighPriority,
	},
}

var DefaultActorConfig = ActorConfig{
	MailboxType:     StandardMailbox,
	MailboxCapacity: 1000,
	Priority:        NormalActorPriority,
	RestartDelay:    time.Second * 1,
	MaxRestarts:     5,
	RestartWindow:   time.Minute * 1,
	EnableStashing:  true,
	EnableWatching:  true,
	CPUAffinityMask: 0,
}

var DefaultSupervisorConfig = SupervisorConfig{
	Strategy:          RestartStrategy,
	MaxRetries:        3,
	RetryPeriod:       time.Second * 10,
	EscalationTimeout: time.Second * 30,
	EnableMonitoring:  true,
	EnableLogging:     true,
}

// Global counters
var (
	globalActorID      uint64
	globalMessageID    uint64
	globalSupervisorID uint64
	globalMailboxID    uint64
	globalGroupID      uint64
)

// Constructor functions

// NewActorSystem creates a new actor system
func NewActorSystem(config ActorSystemConfig) (*ActorSystem, error) {
	ctx, cancel := context.WithCancel(context.Background())

	system := &ActorSystem{
		actors:      make(map[ActorID]*Actor),
		supervisors: make(map[SupervisorID]*Supervisor),
		mailboxes:   make(map[MailboxID]*Mailbox),
		groups:      make(map[ActorGroupID]*ActorGroup),
		config:      config,
		running:     false,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize registry
	system.registry = NewActorRegistry()
	system.groups = make(map[ActorGroupID]*ActorGroup)

	// Initialize scheduler
	schedulerConfig := SchedulerConfig{
		WorkerCount:          4,
		Strategy:             FairScheduling,
		Fairness:             1.0,
		PreemptionEnabled:    true,
		WorkStealingEnabled:  true,
		LoadBalancingEnabled: true,
	}
	system.scheduler = NewActorScheduler(schedulerConfig)
	// Wire scheduler worker callback to process actor mailboxes
	system.scheduler.process = func(aid ActorID) {
		system.mutex.RLock()
		actor := system.actors[aid]
		system.mutex.RUnlock()
		if actor == nil {
			return
		}
		// Drain one message if available and process
		if msg, ok := actor.Mailbox.Dequeue(); ok {
			if err := actor.ProcessMessage(msg); err != nil {
				// Delegate to supervisor strategy
				system.handleFailure(actor, err)
			}
		}
	}

	// Initialize dispatcher
	dispatcherConfig := DispatcherConfig{
		BufferSize:           1000,
		EnableRouting:        true,
		EnableInterception:   false,
		EnableTransformation: false,
		EnableSerialization:  false,
		DefaultTimeout:       time.Second * 30,
	}
	system.dispatcher = NewMessageDispatcher(dispatcherConfig)

	// Initialize root supervisor (OneForOne)
	root, err := NewSupervisor("root", OneForOne, DefaultSupervisorConfig)
	if err == nil {
		system.rootSupervisor = root
	}
	system.scheduler.resolvePriority = func(aid ActorID) ActorPriority {
		system.mutex.RLock()
		act := system.actors[aid]
		system.mutex.RUnlock()
		if act == nil {
			return NormalActorPriority
		}
		return act.Config.Priority
	}

	return system, nil
}

// NewActor creates a new actor
func NewActor(name string, actorType ActorType, behavior ActorBehavior, config ActorConfig) (*Actor, error) {
	actorID := ActorID(atomic.AddUint64(&globalActorID, 1))

	// Create mailbox
	mailbox, err := NewMailbox(config.MailboxType, config.MailboxCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create mailbox: %v", err)
	}

	actor := &Actor{
		ID:            actorID,
		Name:          name,
		Type:          actorType,
		State:         ActorIdle,
		Mailbox:       mailbox,
		Children:      make(map[ActorID]*Actor),
		Behavior:      behavior,
		Config:        config,
		LastHeartbeat: time.Now(),
		RestartCount:  0,
		CreateTime:    time.Now(),
	}

	// Create actor context
	actor.Context = &ActorContext{
		ActorID:  actorID,
		Self:     actor,
		Stash:    make([]Message, 0),
		Timers:   make(map[string]*ActorTimer),
		Watchers: make(map[ActorID]bool),
		Watched:  make(map[ActorID]bool),
		Props:    make(map[string]interface{}),
	}

	// Set mailbox owner
	mailbox.Owner = actorID

	return actor, nil
}

// NewMailbox creates a new mailbox
func NewMailbox(mailboxType MailboxType, capacity uint32) (*Mailbox, error) {
	mailboxID := MailboxID(atomic.AddUint64(&globalMailboxID, 1))

	mailbox := &Mailbox{
		ID:               mailboxID,
		Type:             mailboxType,
		Capacity:         capacity,
		Messages:         make([]Message, 0, capacity),
		DeadLetters:      make([]Message, 0),
		Filters:          make([]MessageFilter, 0),
		OverflowPolicy:   DropOldest,
		LastActivity:     time.Now(),
		BackPressureWait: time.Millisecond * 100,
		notFull:          make(chan struct{}, 1),
	}

	// Initialize priority queue for priority mailboxes
	if mailboxType == PriorityMailbox {
		mailbox.PriorityQueue = NewMessagePriorityQueue(int(capacity))
	}

	return mailbox, nil
}

// NewSupervisor creates a new supervisor
func NewSupervisor(name string, supervisorType SupervisorType, config SupervisorConfig) (*Supervisor, error) {
	supervisorID := SupervisorID(atomic.AddUint64(&globalSupervisorID, 1))

	supervisor := &Supervisor{
		ID:           supervisorID,
		Name:         name,
		Type:         supervisorType,
		Strategy:     config.Strategy,
		Children:     make(map[ActorID]*Actor),
		childOrder:   make([]ActorID, 0),
		MaxRetries:   config.MaxRetries,
		RetryPeriod:  config.RetryPeriod,
		Escalations:  make([]EscalationRule, 0),
		Config:       config,
		CreateTime:   time.Now(),
		restartTrack: make(map[ActorID][]time.Time),
	}

	// Initialize monitor if enabled
	if config.EnableMonitoring {
		supervisor.Monitor = &SupervisorMonitor{
			Enabled:  true,
			Interval: time.Second * 10,
			Alerts:   make([]string, 0),
		}
	}

	return supervisor, nil
}

// Core actor system operations

// Start starts the actor system
func (as *ActorSystem) Start() error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if as.running {
		return fmt.Errorf("actor system is already running")
	}

	// Start scheduler
	if err := as.scheduler.Start(as.ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %v", err)
	}

	// Start dispatcher
	if err := as.dispatcher.Start(as.ctx); err != nil {
		return fmt.Errorf("failed to start dispatcher: %v", err)
	}

	// Start I/O poller if present
	if as.ioPoller != nil {
		if err := as.ioPoller.Start(as.ctx); err != nil {
			return fmt.Errorf("failed to start io poller: %v", err)
		}
	}

	as.running = true

	// Start system maintenance routines
	go as.runHeartbeatMonitor()
	go as.runGarbageCollector()

	return nil
}

// Stop stops the actor system
func (as *ActorSystem) Stop() error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.running {
		return nil
	}

	// Stop all actors
	for _, actor := range as.actors {
		as.stopActor(actor)
	}

	// Stop scheduler and dispatcher
	as.scheduler.Stop()
	as.dispatcher.Stop()

	// Stop I/O poller if attached
	if as.ioPoller != nil {
		_ = as.ioPoller.Stop()
	}

	// Cancel context
	as.cancel()

	as.running = false

	return nil
}

// SetIOPoller attaches an asyncio Poller to the actor system lifecycle.
func (as *ActorSystem) SetIOPoller(p asyncio.Poller) {
	as.mutex.Lock()
	as.ioPoller = p
	as.mutex.Unlock()
}

// WatchConnWithActor registers a net.Conn with the attached poller and routes events to target actor.
func (as *ActorSystem) WatchConnWithActor(conn net.Conn, kinds []asyncio.EventType, target ActorID) error {
	// Use system defaults when available
	as.mutex.RLock()
	def := as.config.DefaultIOWatchOptions
	as.mutex.RUnlock()
	return as.WatchConnWithActorOpts(conn, kinds, target, def)
}

// IOWatchOptions controls backpressure alignment when delivering I/O events to actors.
type IOWatchOptions struct {
	// DropOnOverflow drops events immediately when the target mailbox reports overflow.
	// If false, a small exponential backoff re-registration is attempted.
	DropOnOverflow bool
	// BackoffInitial is the initial delay for re-register when overflow occurs.
	BackoffInitial time.Duration
	// BackoffMax is the maximum backoff delay.
	BackoffMax time.Duration
	// HighWatermark indicates the mailbox length at or above which I/O events should be paused
	// by temporary deregistration from the poller. Set to 0 to disable watermark-based pausing.
	HighWatermark uint32
	// LowWatermark indicates the mailbox length at or below which I/O events may be resumed.
	// Must be less than or equal to HighWatermark. Ignored if HighWatermark == 0.
	LowWatermark uint32
	// ReadHighWatermark/ReadLowWatermark override per-read event watermarks (fallback to High/Low when zero).
	ReadHighWatermark uint32
	ReadLowWatermark  uint32
	// WriteHighWatermark/WriteLowWatermark override per-write event watermarks (fallback to High/Low when zero).
	WriteHighWatermark uint32
	WriteLowWatermark  uint32
	// MonitorInterval controls how often the watermark resume checker runs while paused.
	// Defaults to 10ms when zero.
	MonitorInterval time.Duration
	// ReadEventPriority maps IOReadable events to a message priority.
	ReadEventPriority MessagePriority
	// WriteEventPriority maps IOWritable events to a message priority.
	WriteEventPriority MessagePriority
	// ErrorEventPriority maps IOErrorEvt events to a message priority.
	ErrorEventPriority MessagePriority
}

// WatchConnWithActorOpts registers with options for backpressure alignment.
func (as *ActorSystem) WatchConnWithActorOpts(conn net.Conn, kinds []asyncio.EventType, target ActorID, opts IOWatchOptions) error {
	as.mutex.RLock()
	p := as.ioPoller
	as.mutex.RUnlock()
	if p == nil {
		return fmt.Errorf("no io poller attached")
	}
	if opts.BackoffInitial <= 0 {
		opts.BackoffInitial = time.Millisecond * 5
	}
	if opts.BackoffMax <= 0 {
		opts.BackoffMax = time.Millisecond * 100
	}
	if opts.MonitorInterval <= 0 {
		opts.MonitorInterval = time.Millisecond * 10
	}
	if opts.LowWatermark > opts.HighWatermark {
		opts.LowWatermark = opts.HighWatermark
	}
	// Normalize per-event watermarks: if not set, inherit global; fix low<=high
	if opts.ReadHighWatermark == 0 {
		opts.ReadHighWatermark = opts.HighWatermark
	}
	if opts.ReadLowWatermark == 0 {
		opts.ReadLowWatermark = opts.LowWatermark
	}
	if opts.ReadLowWatermark > opts.ReadHighWatermark {
		opts.ReadLowWatermark = opts.ReadHighWatermark
	}
	if opts.WriteHighWatermark == 0 {
		opts.WriteHighWatermark = opts.HighWatermark
	}
	if opts.WriteLowWatermark == 0 {
		opts.WriteLowWatermark = opts.LowWatermark
	}
	if opts.WriteLowWatermark > opts.WriteHighWatermark {
		opts.WriteLowWatermark = opts.WriteHighWatermark
	}
	// Default priorities
	if opts.ReadEventPriority == 0 {
		opts.ReadEventPriority = NormalPriority
	}
	if opts.WriteEventPriority == 0 {
		opts.WriteEventPriority = NormalPriority
	}
	if opts.ErrorEventPriority == 0 {
		opts.ErrorEventPriority = HighPriority
	}
	backoff := opts.BackoffInitial
	// paused flags for watermark-based pausing per event class
	var pausedRead int32  // 0 = false, 1 = true
	var pausedWrite int32 // 0 = false, 1 = true
	var handler func(ev asyncio.Event)
	maybeResumeRead := func() {
		if opts.ReadHighWatermark == 0 {
			return
		}
		if length, ok := as.GetMailboxLength(target); ok {
			if uint32(length) <= opts.ReadLowWatermark && atomic.LoadInt32(&pausedRead) == 1 {
				_ = p.Register(conn, kinds, handler)
				atomic.StoreInt32(&pausedRead, 0)
			}
		}
	}
	maybeResumeWrite := func() {
		if opts.WriteHighWatermark == 0 {
			return
		}
		if length, ok := as.GetMailboxLength(target); ok {
			if uint32(length) <= opts.WriteLowWatermark && atomic.LoadInt32(&pausedWrite) == 1 {
				_ = p.Register(conn, kinds, handler)
				atomic.StoreInt32(&pausedWrite, 0)
			}
		}
	}

	handler = func(ev asyncio.Event) {
		var mt MessageType
		var pr MessagePriority
		switch ev.Type {
		case asyncio.Readable:
			mt = IOReadable
			pr = opts.ReadEventPriority
		case asyncio.Writable:
			mt = IOWritable
			pr = opts.WriteEventPriority
		default:
			mt = IOErrorEvt
			pr = opts.ErrorEventPriority
		}
		// Watermark-based pausing per event class
		if ev.Type == asyncio.Readable && opts.ReadHighWatermark > 0 {
			if length, ok := as.GetMailboxLength(target); ok && uint32(length) >= opts.ReadHighWatermark && atomic.LoadInt32(&pausedRead) == 0 {
				_ = p.Deregister(conn)
				atomic.StoreInt32(&pausedRead, 1)
				go func() {
					ticker := time.NewTicker(opts.MonitorInterval)
					defer ticker.Stop()
					for {
						if atomic.LoadInt32(&pausedRead) == 0 {
							return
						}
						select {
						case <-ticker.C:
							maybeResumeRead()
						case <-as.ctx.Done():
							return
						}
					}
				}()
				return
			}
		}
		if ev.Type == asyncio.Writable && opts.WriteHighWatermark > 0 {
			if length, ok := as.GetMailboxLength(target); ok && uint32(length) >= opts.WriteHighWatermark && atomic.LoadInt32(&pausedWrite) == 0 {
				_ = p.Deregister(conn)
				atomic.StoreInt32(&pausedWrite, 1)
				go func() {
					ticker := time.NewTicker(opts.MonitorInterval)
					defer ticker.Stop()
					for {
						if atomic.LoadInt32(&pausedWrite) == 0 {
							return
						}
						select {
						case <-ticker.C:
							maybeResumeWrite()
						case <-as.ctx.Done():
							return
						}
					}
				}()
				return
			}
		}
		if err := as.SendMessageWithPriority(0, target, mt, IOEvent{Conn: ev.Conn, Type: ev.Type, Err: ev.Err}, pr); err != nil {
			// Mailbox overflow/backpressure: either drop or temporarily deregister and retry
			if opts.DropOnOverflow {
				return
			}
			_ = p.Deregister(conn)
			d := backoff
			if d > opts.BackoffMax {
				d = opts.BackoffMax
			}
			time.AfterFunc(d, func() {
				// re-register and reset/increase backoff
				_ = p.Register(conn, kinds, handler)
			})
			// Exponential growth for next time
			if backoff < opts.BackoffMax {
				backoff *= 2
				if backoff > opts.BackoffMax {
					backoff = opts.BackoffMax
				}
			}
		} else {
			// successful delivery resets backoff
			backoff = opts.BackoffInitial
			// Also attempt to resume if previously paused and capacity is available
			if ev.Type == asyncio.Readable && atomic.LoadInt32(&pausedRead) == 1 {
				maybeResumeRead()
			}
			if ev.Type == asyncio.Writable && atomic.LoadInt32(&pausedWrite) == 1 {
				maybeResumeWrite()
			}
		}
	}
	return p.Register(conn, kinds, handler)
}

// GetMailboxLength returns the current length of an actor's mailbox.
func (as *ActorSystem) GetMailboxLength(aid ActorID) (int, bool) {
	as.mutex.RLock()
	actor := as.actors[aid]
	as.mutex.RUnlock()
	if actor == nil || actor.Mailbox == nil {
		return 0, false
	}
	return actor.Mailbox.Len(), true
}

// SendMessageWithPriority sends a message with an explicit priority.
func (as *ActorSystem) SendMessageWithPriority(senderID, receiverID ActorID, messageType MessageType, payload interface{}, prio MessagePriority) error {
	if !as.running {
		return fmt.Errorf("actor system is not running")
	}
	message := Message{
		ID:         MessageID(atomic.AddUint64(&globalMessageID, 1)),
		Type:       messageType,
		Sender:     senderID,
		Receiver:   receiverID,
		Payload:    payload,
		Priority:   prio,
		Timestamp:  time.Now(),
		TTL:        time.Minute * 5,
		Headers:    make(map[string]interface{}),
		Persistent: false,
		Delivered:  false,
	}
	return as.deliverMessage(message)
}

// UnwatchConn deregisters a net.Conn from the attached poller.
func (as *ActorSystem) UnwatchConn(conn net.Conn) error {
	as.mutex.RLock()
	p := as.ioPoller
	as.mutex.RUnlock()
	if p == nil {
		return fmt.Errorf("no io poller attached")
	}
	return p.Deregister(conn)
}

// CreateActor creates a new actor in the system
func (as *ActorSystem) CreateActor(name string, actorType ActorType, behavior ActorBehavior, config ActorConfig) (*Actor, error) {
	if !as.running {
		return nil, fmt.Errorf("actor system is not running")
	}

	actor, err := NewActor(name, actorType, behavior, config)
	if err != nil {
		return nil, err
	}

	as.mutex.Lock()
	as.actors[actor.ID] = actor
	as.mailboxes[actor.Mailbox.ID] = actor.Mailbox
	// Attach to root supervisor by default
	if as.rootSupervisor != nil {
		actor.Supervisor = as.rootSupervisor
		as.rootSupervisor.Children[actor.ID] = actor
		as.rootSupervisor.childOrder = append(as.rootSupervisor.childOrder, actor.ID)
	}
	as.mutex.Unlock()

	// Register actor
	if err := as.registry.Register(name, actor.ID); err != nil {
		return nil, fmt.Errorf("failed to register actor: %v", err)
	}

	// Set system reference
	actor.Context.System = as

	// Call PreStart
	if err := actor.Behavior.PreStart(actor.Context); err != nil {
		return nil, fmt.Errorf("PreStart failed: %v", err)
	}

	// Schedule actor
	as.scheduler.Schedule(actor.ID)

	// Update statistics
	atomic.AddUint64(&as.statistics.TotalActors, 1)
	atomic.AddUint64(&as.statistics.ActiveActors, 1)

	return actor, nil
}

// CreateSupervisor creates a new supervisor under an optional parent (default: root) and returns it.
func (as *ActorSystem) CreateSupervisor(name string, supervisorType SupervisorType, cfg SupervisorConfig, parent *Supervisor) (*Supervisor, error) {
	if !as.running {
		return nil, fmt.Errorf("actor system is not running")
	}
	sup, err := NewSupervisor(name, supervisorType, cfg)
	if err != nil {
		return nil, err
	}
	if parent != nil {
		sup.Parent = parent
	} else {
		sup.Parent = as.rootSupervisor
	}
	as.mutex.Lock()
	as.supervisors[sup.ID] = sup
	as.mutex.Unlock()
	return sup, nil
}

// CreateActorUnder creates an actor supervised by the specified supervisor.
func (as *ActorSystem) CreateActorUnder(supervisor *Supervisor, name string, actorType ActorType, behavior ActorBehavior, config ActorConfig) (*Actor, error) {
	if !as.running {
		return nil, fmt.Errorf("actor system is not running")
	}
	actor, err := NewActor(name, actorType, behavior, config)
	if err != nil {
		return nil, err
	}
	as.mutex.Lock()
	as.actors[actor.ID] = actor
	as.mailboxes[actor.Mailbox.ID] = actor.Mailbox
	if supervisor != nil {
		actor.Supervisor = supervisor
		supervisor.Children[actor.ID] = actor
		supervisor.childOrder = append(supervisor.childOrder, actor.ID)
	} else if as.rootSupervisor != nil {
		actor.Supervisor = as.rootSupervisor
		as.rootSupervisor.Children[actor.ID] = actor
		as.rootSupervisor.childOrder = append(as.rootSupervisor.childOrder, actor.ID)
	}
	as.mutex.Unlock()

	if err := as.registry.Register(name, actor.ID); err != nil {
		return nil, fmt.Errorf("failed to register actor: %v", err)
	}
	actor.Context.System = as
	if err := actor.Behavior.PreStart(actor.Context); err != nil {
		return nil, fmt.Errorf("PreStart failed: %v", err)
	}
	as.scheduler.Schedule(actor.ID)
	atomic.AddUint64(&as.statistics.TotalActors, 1)
	atomic.AddUint64(&as.statistics.ActiveActors, 1)
	return actor, nil
}

// SendMessage sends a message to an actor
func (as *ActorSystem) SendMessage(senderID, receiverID ActorID, messageType MessageType, payload interface{}) error {
	if !as.running {
		return fmt.Errorf("actor system is not running")
	}

	message := Message{
		ID:         MessageID(atomic.AddUint64(&globalMessageID, 1)),
		Type:       messageType,
		Sender:     senderID,
		Receiver:   receiverID,
		Payload:    payload,
		Priority:   NormalPriority,
		Timestamp:  time.Now(),
		TTL:        time.Minute * 5,
		Headers:    make(map[string]interface{}),
		Persistent: false,
		Delivered:  false,
	}

	return as.deliverMessage(message)
}

// SendToName delivers a message to an actor by its registered name. If a Remote is attached and
// the name is qualified as node:name (e.g., "nodeA:svc"), it will attempt remote delivery.
func (as *ActorSystem) SendToName(senderID ActorID, qualifiedName string, messageType MessageType, payload interface{}) error {
	if !as.running {
		return fmt.Errorf("actor system is not running")
	}
	// Remote qualified route: node:name
	if idx := indexByte(qualifiedName, ':'); idx > 0 && idx < len(qualifiedName)-1 && as.Remote != nil {
		node := qualifiedName[:idx]
		name := qualifiedName[idx+1:]
		return as.Remote.Send(node, name, uint32(messageType), payload)
	}
	// Local lookup
	if id, ok := as.registry.Lookup(qualifiedName); ok {
		return as.SendMessage(senderID, id, messageType, payload)
	}
	return fmt.Errorf("actor not found: %s", qualifiedName)
}

// indexByte is a tiny helper to avoid extra import for strings.IndexByte in this file context.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// Actor operations

// ProcessMessage processes a message for an actor
func (a *Actor) ProcessMessage(msg Message) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.State == ActorStopped || a.State == ActorStopping {
		return fmt.Errorf("actor is stopped or stopping")
	}

	a.State = ActorBusy

	// Update context
	a.Context.Sender = msg.Sender

	// Update heartbeat before processing
	a.LastHeartbeat = time.Now()

	// Process message
	err := a.Behavior.Receive(a.Context, msg)

	// Update statistics
	a.Statistics.MessagesReceived++
	if err != nil {
		a.Statistics.MessagesFailed++
	} else {
		a.Statistics.MessagesProcessed++
	}

	a.Statistics.LastActivity = time.Now()
	a.State = ActorIdle

	return err
}

// Restart restarts an actor
func (a *Actor) Restart(reason error) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.State = ActorRestarting
	a.RestartCount++

	// Call PreRestart
	if err := a.Behavior.PreRestart(a.Context, reason, nil); err != nil {
		return fmt.Errorf("PreRestart failed: %v", err)
	}

	// Do not clear mailbox to preserve pending messages across restarts.

	// Call PostRestart
	if err := a.Behavior.PostRestart(a.Context, reason); err != nil {
		return fmt.Errorf("PostRestart failed: %v", err)
	}

	a.State = ActorIdle
	a.Statistics.Restarts++

	return nil
}

// handleFailure applies supervisor strategy for a failed actor
func (as *ActorSystem) handleFailure(failed *Actor, reason error) {
	sup := failed.Supervisor
	if sup == nil {
		_ = failed.Restart(reason)
		return
	}

	switch sup.Strategy {
	case RestartStrategy:
		// Apply restart according to supervisor type with restart limits
		switch sup.Type {
		case OneForOne:
			if !as.canRestart(sup, failed) {
				_ = as.stopActor(failed)
				return
			}
			if d := failed.Config.RestartDelay; d > 0 {
				// Schedule asynchronous restart to avoid blocking supervisor loop
				time.AfterFunc(d, func() { _ = failed.Restart(reason) })
			} else {
				_ = failed.Restart(reason)
			}
		case OneForAll:
			// Restart all children
			for _, child := range sup.Children {
				if !as.canRestart(sup, child) {
					_ = as.stopActor(child)
					continue
				}
				if d := child.Config.RestartDelay; d > 0 {
					time.AfterFunc(d, func(ch *Actor) func() {
						return func() { _ = ch.Restart(reason) }
					}(child))
				} else {
					_ = child.Restart(reason)
				}
			}
		case RestForOne:
			// Restart failed and all children created after it
			idx := -1
			for i, id := range sup.childOrder {
				if id == failed.ID {
					idx = i
					break
				}
			}
			if idx >= 0 {
				for i := idx; i < len(sup.childOrder); i++ {
					if c := sup.Children[sup.childOrder[i]]; c != nil {
						if !as.canRestart(sup, c) {
							_ = as.stopActor(c)
							continue
						}
						if d := c.Config.RestartDelay; d > 0 {
							time.AfterFunc(d, func(act *Actor) func() {
								return func() { _ = act.Restart(reason) }
							}(c))
						} else {
							_ = c.Restart(reason)
						}
					}
				}
			} else {
				if !as.canRestart(sup, failed) {
					_ = as.stopActor(failed)
				} else {
					if d := failed.Config.RestartDelay; d > 0 {
						time.AfterFunc(d, func() { _ = failed.Restart(reason) })
					} else {
						_ = failed.Restart(reason)
					}
				}
			}
		default:
			if !as.canRestart(sup, failed) {
				_ = as.stopActor(failed)
			} else {
				if d := failed.Config.RestartDelay; d > 0 {
					time.AfterFunc(d, func() { _ = failed.Restart(reason) })
				} else {
					_ = failed.Restart(reason)
				}
			}
		}
	case StopStrategy:
		// Stop according to supervisor type
		switch sup.Type {
		case OneForOne:
			_ = as.stopActor(failed)
		case OneForAll:
			for _, child := range sup.Children {
				_ = as.stopActor(child)
			}
		case RestForOne:
			idx := -1
			for i, id := range sup.childOrder {
				if id == failed.ID {
					idx = i
					break
				}
			}
			if idx >= 0 {
				for i := idx; i < len(sup.childOrder); i++ {
					if c := sup.Children[sup.childOrder[i]]; c != nil {
						_ = as.stopActor(c)
					}
				}
			} else {
				_ = as.stopActor(failed)
			}
		default:
			_ = as.stopActor(failed)
		}
	case ResumeStrategy:
		// No action; actor continues
		return
	case EscalateStrategy:
		// Bubble up to parent supervisor
		if failed.Supervisor != nil && failed.Supervisor.Parent != nil {
			// escalate towards parent supervisor (simple propagation)
			// Note: this simplified escalation reuses the same failed actor context.
			as.handleFailure(failed, fmt.Errorf("escalated: %v", reason))
		} else {
			_ = failed.Restart(reason)
		}
	default:
		_ = failed.Restart(reason)
	}
}

// canRestart checks supervisor's restart policy window for a child and records the attempt.
func (as *ActorSystem) canRestart(sup *Supervisor, child *Actor) bool {
	sup.mutex.Lock()
	defer sup.mutex.Unlock()
	if sup.MaxRetries == 0 || sup.RetryPeriod <= 0 {
		return true
	}
	hist := sup.restartTrack[child.ID]
	now := time.Now()
	cutoff := now.Add(-sup.RetryPeriod)
	filtered := hist[:0]
	for _, t := range hist {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	hist = filtered
	if uint32(len(hist)) >= sup.MaxRetries {
		sup.restartTrack[child.ID] = hist
		return false
	}
	hist = append(hist, now)
	sup.restartTrack[child.ID] = hist
	return true
}

// Mailbox operations

// Enqueue adds a message to the mailbox
func (m *Mailbox) Enqueue(msg Message) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check capacity
	if uint32(len(m.Messages)) >= m.Capacity {
		return m.handleOverflow(msg)
	}

	// Apply filters
	for _, filter := range m.Filters {
		if !filter.Filter(msg) {
			return fmt.Errorf("message filtered out")
		}
	}

	// Add message
	if m.Type == PriorityMailbox && m.PriorityQueue != nil {
		m.PriorityQueue.Push(PriorityMessage{
			Message:    msg,
			Priority:   int(msg.Priority),
			InsertTime: time.Now(),
		})
	} else {
		m.Messages = append(m.Messages, msg)
	}

	m.Statistics.MessagesEnqueued++
	m.LastActivity = time.Now()

	return nil
}

// Dequeue removes and returns a message from the mailbox
func (m *Mailbox) Dequeue() (Message, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.Type == PriorityMailbox && m.PriorityQueue != nil {
		if item, ok := m.PriorityQueue.Pop(); ok {
			m.Statistics.MessagesDequeued++
			m.LastActivity = time.Now()
			// Notify potential waiters that capacity may be available now
			if m.notFull != nil {
				select {
				case m.notFull <- struct{}{}:
				default:
				}
			}
			return item.Message, true
		}
	} else {
		if len(m.Messages) > 0 {
			msg := m.Messages[0]
			m.Messages = m.Messages[1:]
			m.Statistics.MessagesDequeued++
			m.LastActivity = time.Now()
			if m.notFull != nil {
				select {
				case m.notFull <- struct{}{}:
				default:
				}
			}
			return msg, true
		}
	}

	return Message{}, false
}

// Clear clears all messages from the mailbox
func (m *Mailbox) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Messages = m.Messages[:0]
	if m.PriorityQueue != nil {
		m.PriorityQueue.Clear()
	}
}

// Len returns the current number of queued messages in the mailbox.
func (m *Mailbox) Len() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.Type == PriorityMailbox && m.PriorityQueue != nil {
		return m.PriorityQueue.size
	}
	return len(m.Messages)
}

// Helper methods

// deliverMessage delivers a message to its destination
func (as *ActorSystem) deliverMessage(msg Message) error {
	// Interceptors / transformers and routing
	as.dispatcher.mutex.RLock()
	interceptors := append([]MessageInterceptor(nil), as.dispatcher.interceptors...)
	transformers := append([]MessageTransformer(nil), as.dispatcher.transformers...)
	routes := append([]DispatchRule(nil), as.dispatcher.routes[msg.Type]...)
	as.dispatcher.mutex.RUnlock()

	// Apply simple routing if configured
	if len(routes) > 0 {
		// Pick first route (simple strategy)
		msg.Receiver = routes[0].Target
	}

	as.mutex.RLock()
	receiver, exists := as.actors[msg.Receiver]
	as.mutex.RUnlock()

	if !exists {
		return as.sendToDeadLetters(msg)
	}

	// Apply interceptors
	for _, ic := range interceptors {
		m2, err := ic.Intercept(msg)
		if err != nil {
			return fmt.Errorf("interception failed: %w", err)
		}
		msg = m2
	}

	// Apply transformers
	for _, tf := range transformers {
		m2, err := tf.Transform(msg)
		if err != nil {
			return fmt.Errorf("transform failed: %w", err)
		}
		msg = m2
	}

	if err := receiver.Mailbox.Enqueue(msg); err != nil {
		return as.sendToDeadLetters(msg)
	}

	// Notify scheduler
	as.scheduler.Schedule(msg.Receiver)

	// Update statistics
	atomic.AddUint64(&as.statistics.TotalMessages, 1)

	return nil
}

// sendToDeadLetters sends a message to dead letters
func (as *ActorSystem) sendToDeadLetters(msg Message) error {
	if as.config.EnableDeadLetters {
		// Implementation would send to dead letter queue
		atomic.AddUint64(&as.statistics.DeadLetters, 1)
	}
	return fmt.Errorf("message sent to dead letters")
}

// handleOverflow handles mailbox overflow
func (m *Mailbox) handleOverflow(msg Message) error {
	switch m.OverflowPolicy {
	case DropOldest:
		if len(m.Messages) > 0 {
			m.Messages = m.Messages[1:]
			m.Messages = append(m.Messages, msg)
		}
	case DropNewest:
		// Drop the new message
		m.Statistics.MessagesDropped++
		return fmt.Errorf("message dropped due to overflow")
	case DropLowPriority:
		// Find and drop lowest priority message
		if m.dropLowestPriority() {
			m.Messages = append(m.Messages, msg)
		}
	case BackPressure:
		// Apply timed back pressure: wait for room up to BackPressureWait
		// This uses an edge-triggered notification channel to avoid busy-wait.
		deadline := time.Now().Add(m.BackPressureWait)
		for {
			if uint32(len(m.Messages)) < m.Capacity {
				m.Messages = append(m.Messages, msg)
				return nil
			}
			// Prepare to wait: capture notifier and unlock
			notifier := m.notFull
			if notifier == nil {
				// Safety fallback: initialize on demand
				notifier = make(chan struct{}, 1)
				m.notFull = notifier
			}
			m.mutex.Unlock()
			// Wait for either capacity notification or timeout tick
			now := time.Now()
			if !now.Before(deadline) {
				m.mutex.Lock()
				return fmt.Errorf("mailbox back pressure timeout")
			}
			timeout := time.NewTimer(deadline.Sub(now))
			select {
			case <-notifier:
				// Woken up, retry to check capacity
			case <-timeout.C:
				// Timed out
			}
			if !timeout.Stop() {
				// Drain if fired
				select {
				case <-timeout.C:
				default:
				}
			}
			m.mutex.Lock()
			if !time.Now().Before(deadline) {
				return fmt.Errorf("mailbox back pressure timeout")
			}
		}
	case DeadLetter:
		m.DeadLetters = append(m.DeadLetters, msg)
	}

	m.Statistics.OverflowCount++
	return nil
}

// dropLowestPriority finds and drops the lowest priority message
func (m *Mailbox) dropLowestPriority() bool {
	if len(m.Messages) == 0 {
		return false
	}

	minPriority := m.Messages[0].Priority
	minIndex := 0

	for i, msg := range m.Messages {
		if msg.Priority < minPriority {
			minPriority = msg.Priority
			minIndex = i
		}
	}

	// Remove the lowest priority message
	m.Messages = append(m.Messages[:minIndex], m.Messages[minIndex+1:]...)
	m.Statistics.MessagesDropped++

	return true
}

// stopActor stops an actor
func (as *ActorSystem) stopActor(actor *Actor) error {
	actor.mutex.Lock()
	defer actor.mutex.Unlock()

	if actor.State == ActorStopped || actor.State == ActorStopping {
		return nil
	}

	actor.State = ActorStopping

	// Call PostStop
	if err := actor.Behavior.PostStop(actor.Context); err != nil {
		// Log error but continue
	}

	actor.State = ActorStopped

	// Notify watchers with a system termination message
	if actor.Context != nil && len(actor.Context.Watchers) > 0 {
		for watcherID := range actor.Context.Watchers {
			_ = as.SendMessage(actor.ID, watcherID, SystemTerminated, actor.ID)
		}
	}

	// Update statistics
	atomic.AddUint64(&as.statistics.ActiveActors, ^uint64(0)) // Decrement

	return nil
}

// Watch registers the current actor as a watcher of the target actor. When the
// target terminates, a SystemTerminated message with payload=targetID is sent
// to the watcher.
func (ctx *ActorContext) Watch(target ActorID) {
	if ctx.Watched == nil {
		ctx.Watched = make(map[ActorID]bool)
	}
	ctx.Watched[target] = true
	if ctx.System != nil {
		ctx.System.mutex.RLock()
		tgt := ctx.System.actors[target]
		ctx.System.mutex.RUnlock()
		if tgt != nil {
			if tgt.Context == nil {
				tgt.Context = &ActorContext{Watchers: make(map[ActorID]bool)}
			}
			if tgt.Context.Watchers == nil {
				tgt.Context.Watchers = make(map[ActorID]bool)
			}
			tgt.Context.Watchers[ctx.ActorID] = true
		}
	}
}

// Unwatch unregisters a watcher from the target actor.
func (ctx *ActorContext) Unwatch(target ActorID) {
	if ctx.System == nil {
		return
	}
	ctx.System.mutex.RLock()
	tgt := ctx.System.actors[target]
	ctx.System.mutex.RUnlock()
	if tgt != nil && tgt.Context != nil && tgt.Context.Watchers != nil {
		delete(tgt.Context.Watchers, ctx.ActorID)
	}
	delete(ctx.Watched, target)
}

// System maintenance

// runHeartbeatMonitor monitors actor heartbeats
func (as *ActorSystem) runHeartbeatMonitor() {
	ticker := time.NewTicker(as.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.checkHeartbeats()
			// Emit warning alerts via dispatcher interceptors if needed (placeholder for future)
		}
	}
}

// runGarbageCollector performs system garbage collection
func (as *ActorSystem) runGarbageCollector() {
	ticker := time.NewTicker(as.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.performGC()
		}
	}
}

// checkHeartbeats checks actor heartbeats
func (as *ActorSystem) checkHeartbeats() {
	now := time.Now()
	timeout := as.config.HeartbeatInterval * 3

	as.mutex.RLock()
	defer as.mutex.RUnlock()

	for _, actor := range as.actors {
		if now.Sub(actor.LastHeartbeat) > timeout {
			// Actor may be dead, handle accordingly
			go as.handleDeadActor(actor)
		}
	}
}

// performGC performs garbage collection
func (as *ActorSystem) performGC() {
	// Implementation would clean up dead actors, expired messages, etc.
}

// handleDeadActor handles a potentially dead actor
func (as *ActorSystem) handleDeadActor(actor *Actor) {
	// Restart or escalate based on supervision strategy
	as.handleFailure(actor, fmt.Errorf("heartbeat timeout"))
}

// Statistics and monitoring

// GetStatistics returns system statistics
func (as *ActorSystem) GetStatistics() ActorSystemStatistics {
	return as.statistics
}

// Supporting constructor functions

func NewActorRegistry() *ActorRegistry {
	return &ActorRegistry{
		nameToID:  make(map[string]ActorID),
		idToActor: make(map[ActorID]*Actor),
		groups:    make(map[string]ActorGroupID),
		patterns:  make([]RegistryPattern, 0),
		cache:     make(map[string]ActorID),
		enabled:   true,
	}
}

// LookupActorID returns the actor ID for a registered name.
func (as *ActorSystem) LookupActorID(name string) (ActorID, bool) {
	if as == nil || as.registry == nil {
		return 0, false
	}
	return as.registry.Lookup(name)
}

// Group operations

// CreateGroup creates a new actor group and registers it by name
func (as *ActorSystem) CreateGroup(name string, groupType ActorGroupType, cfg GroupConfig) (*ActorGroup, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	gid := ActorGroupID(atomic.AddUint64(&globalGroupID, 1))
	grp := &ActorGroup{
		ID:         gid,
		Name:       name,
		Type:       groupType,
		Members:    make(map[ActorID]*Actor),
		Statistics: GroupStatistics{},
		Config:     cfg,
		CreateTime: time.Now(),
	}
	as.groups[gid] = grp
	as.registry.groups[name] = gid
	return grp, nil
}

// AddToGroup adds an actor to an existing group
func (as *ActorSystem) AddToGroup(groupID ActorGroupID, actorID ActorID) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	grp, ok := as.groups[groupID]
	if !ok {
		return fmt.Errorf("group not found")
	}
	act, ok := as.actors[actorID]
	if !ok {
		return fmt.Errorf("actor not found")
	}
	grp.Members[actorID] = act
	return nil
}

// Broadcast sends a message to all members of the group
func (as *ActorSystem) Broadcast(groupID ActorGroupID, messageType MessageType, payload interface{}) error {
	as.mutex.RLock()
	grp, ok := as.groups[groupID]
	as.mutex.RUnlock()
	if !ok {
		return fmt.Errorf("group not found")
	}
	for id := range grp.Members {
		if err := as.SendMessage(0, id, messageType, payload); err != nil {
			return err
		}
	}
	return nil
}

func NewActorScheduler(config SchedulerConfig) *ActorScheduler {
	return &ActorScheduler{
		queues:     make(map[ActorPriority]*ActorQueue),
		roundRobin: make([]ActorID, 0),
		workers:    make([]*SchedulerWorker, config.WorkerCount),
		strategies: []SchedulingStrategy{config.Strategy},
		config:     config,
		running:    false,
	}
}

func NewMessageDispatcher(config DispatcherConfig) *MessageDispatcher {
	return &MessageDispatcher{
		routes:       make(map[MessageType][]DispatchRule),
		interceptors: make([]MessageInterceptor, 0),
		transformers: make([]MessageTransformer, 0),
		serializers:  make(map[string]MessageSerializer),
		config:       config,
		enabled:      true,
	}
}

// AddRoute registers a simple route for a message type
func (md *MessageDispatcher) AddRoute(msgType MessageType, rule DispatchRule) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.routes[msgType] = append(md.routes[msgType], rule)
}

func NewMessagePriorityQueue(capacity int) *MessagePriorityQueue {
	return &MessagePriorityQueue{
		items:    make([]PriorityMessage, 0, capacity),
		size:     0,
		capacity: capacity,
	}
}

// Registry operations
func (ar *ActorRegistry) Register(name string, actorID ActorID) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	ar.nameToID[name] = actorID
	ar.statistics.Registrations++
	return nil
}

func (ar *ActorRegistry) Lookup(name string) (ActorID, bool) {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	actorID, exists := ar.nameToID[name]
	if exists {
		ar.statistics.Lookups++
	}
	return actorID, exists
}

// Scheduler operations
func (as *ActorScheduler) Start(ctx context.Context) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if as.running {
		return fmt.Errorf("scheduler already running")
	}

	as.ctx = ctx
	as.running = true

	// Start workers with CPU mask assignment (simple round-robin over logical CPUs)
	cpuCount := stdrt.NumCPU()
	var defaultMasks []uint64
	if cpuCount <= 64 {
		defaultMasks = make([]uint64, len(as.workers))
		for i := 0; i < len(as.workers); i++ {
			bit := uint64(1) << uint((i)%cpuCount)
			defaultMasks[i] = bit
		}
	} else {
		// If CPUs > 64, group multiple CPUs per worker; still provide a non-zero mask
		defaultMasks = make([]uint64, len(as.workers))
		for i := 0; i < len(as.workers); i++ {
			defaultMasks[i] = ^uint64(0) // all bits as a fallback
		}
	}

	for i := 0; i < len(as.workers); i++ {
		as.workers[i] = &SchedulerWorker{
			ID:       i,
			Queue:    make(chan ActorID, 100),
			Running:  true,
			CPUMask:  defaultMasks[i],
			QueueLen: 0,
		}
		go as.runWorker(as.workers[i])
	}

	return nil
}

func (as *ActorScheduler) Stop() {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.running {
		return
	}
	as.running = false
	for _, worker := range as.workers {
		if worker == nil {
			continue
		}
		worker.Running = false
		if worker.Queue != nil {
			close(worker.Queue)
			worker.Queue = nil
		}
	}
}

func (as *ActorScheduler) Schedule(actorID ActorID) {
	as.scheduleInternal(actorID, 0)
}

// ScheduleWithAffinity schedules with a CPU affinity mask hint.
func (as *ActorScheduler) ScheduleWithAffinity(actorID ActorID, cpuMask uint64) {
	as.scheduleInternal(actorID, cpuMask)
}

func (as *ActorScheduler) scheduleInternal(actorID ActorID, actorMask uint64) {
	if !as.running {
		return
	}

	// Choose candidate workers set: all workers if no mask, otherwise those overlapping.
	candidates := make([]*SchedulerWorker, 0, len(as.workers))
	if actorMask == 0 {
		candidates = as.workers
	} else {
		for _, w := range as.workers {
			if w.CPUMask&actorMask != 0 {
				candidates = append(candidates, w)
			}
		}
		if len(candidates) == 0 {
			candidates = as.workers
		}
	}

	// Pick least loaded candidate worker
	best := candidates[0]
	bestLen := atomic.LoadInt64(&best.QueueLen)
	for _, w := range candidates[1:] {
		if l := atomic.LoadInt64(&w.QueueLen); l < bestLen {
			best = w
			bestLen = l
		}
	}

	// Try enqueue to best, then fallback to a sibling worker if full
	idx := best.ID
	select {
	case as.workers[idx].Queue <- actorID:
		atomic.AddInt64(&as.workers[idx].QueueLen, 1)
		as.statistics.TasksScheduled++
		return
	default:
		// Fallback: probe another worker (least loaded among all)
	}

	// Global least-loaded fallback
	fallback := as.workers[0]
	fallbackLen := atomic.LoadInt64(&fallback.QueueLen)
	for _, w := range as.workers[1:] {
		if l := atomic.LoadInt64(&w.QueueLen); l < fallbackLen {
			fallback = w
			fallbackLen = l
		}
	}
	j := fallback.ID
	select {
	case as.workers[j].Queue <- actorID:
		atomic.AddInt64(&as.workers[j].QueueLen, 1)
		as.statistics.TasksScheduled++
	default:
		// All queues appear saturated; drop the scheduling hint (message stays in mailbox until next attempt)
	}
}

func (as *ActorScheduler) runWorker(worker *SchedulerWorker) {
	for worker.Running {
		select {
		case actorID := <-worker.Queue:
			// Process actor task
			atomic.AddInt64(&worker.QueueLen, -1)
			as.statistics.TasksCompleted++
			if as.process != nil {
				as.process(actorID)
			}
		case <-as.ctx.Done():
			return
		case <-time.After(time.Millisecond * 2):
			// Try to steal work from other workers if enabled
			if as.config.WorkStealingEnabled {
				if id, ok := as.trySteal(worker.ID); ok {
					as.statistics.TasksCompleted++
					if as.process != nil {
						as.process(id)
					}
				}
			}
		}
	}
}

// trySteal attempts to non-blockingly steal an actorID from other workers' queues
func (as *ActorScheduler) trySteal(selfID int) (ActorID, bool) {
	if len(as.workers) == 0 {
		return 0, false
	}
	start := (selfID + 1) % len(as.workers)
	for i := 0; i < len(as.workers)-1; i++ {
		w := as.workers[(start+i)%len(as.workers)]
		select {
		case id := <-w.Queue:
			// Decrement source queue length since we stole one
			atomic.AddInt64(&w.QueueLen, -1)
			return id, true
		default:
		}
	}
	return 0, false
}

// GetQueueLengths returns a snapshot of per-worker queue lengths. Intended for testing/monitoring.
func (as *ActorScheduler) GetQueueLengths() []int64 {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	out := make([]int64, len(as.workers))
	for i, w := range as.workers {
		if w != nil {
			out[i] = atomic.LoadInt64(&w.QueueLen)
		}
	}
	return out
}

// Dispatcher operations
func (md *MessageDispatcher) Start(ctx context.Context) error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	md.enabled = true
	return nil
}

func (md *MessageDispatcher) Stop() {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	md.enabled = false
}

// Priority queue operations
func (pq *MessagePriorityQueue) Push(item PriorityMessage) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if pq.size >= pq.capacity {
		return // Queue full
	}

	pq.items = append(pq.items, item)
	pq.size++
	pq.heapifyUp(pq.size - 1)
}

func (pq *MessagePriorityQueue) Pop() (PriorityMessage, bool) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if pq.size == 0 {
		return PriorityMessage{}, false
	}

	item := pq.items[0]
	pq.items[0] = pq.items[pq.size-1]
	pq.size--
	pq.items = pq.items[:pq.size]

	if pq.size > 0 {
		pq.heapifyDown(0)
	}

	return item, true
}

func (pq *MessagePriorityQueue) Clear() {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.items = pq.items[:0]
	pq.size = 0
}

func (pq *MessagePriorityQueue) heapifyUp(index int) {
	for index > 0 {
		parentIndex := (index - 1) / 2
		if pq.items[index].Priority <= pq.items[parentIndex].Priority {
			break
		}
		pq.items[index], pq.items[parentIndex] = pq.items[parentIndex], pq.items[index]
		index = parentIndex
	}
}

func (pq *MessagePriorityQueue) heapifyDown(index int) {
	for {
		leftChild := 2*index + 1
		rightChild := 2*index + 2
		largest := index

		if leftChild < pq.size && pq.items[leftChild].Priority > pq.items[largest].Priority {
			largest = leftChild
		}

		if rightChild < pq.size && pq.items[rightChild].Priority > pq.items[largest].Priority {
			largest = rightChild
		}

		if largest == index {
			break
		}

		pq.items[index], pq.items[largest] = pq.items[largest], pq.items[index]
		index = largest
	}
}
