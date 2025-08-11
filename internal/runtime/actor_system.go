// Phase 3.2.1: Actor System Concurrency Model
// This file implements a comprehensive actor-based concurrency system with mailboxes,
// message passing, supervision trees, and fault tolerance for Orizon runtime.

package runtime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
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
		ID:          supervisorID,
		Name:        name,
		Type:        supervisorType,
		Strategy:    config.Strategy,
		Children:    make(map[ActorID]*Actor),
		childOrder:  make([]ActorID, 0),
		MaxRetries:  config.MaxRetries,
		RetryPeriod: config.RetryPeriod,
		Escalations: make([]EscalationRule, 0),
		Config:      config,
		CreateTime:  time.Now(),
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

	// Cancel context
	as.cancel()

	as.running = false

	return nil
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
		// Apply restart according to supervisor type
		switch sup.Type {
		case OneForOne:
			_ = failed.Restart(reason)
		case OneForAll:
			// Restart all children
			for _, child := range sup.Children {
				_ = child.Restart(reason)
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
						_ = c.Restart(reason)
					}
				}
			} else {
				_ = failed.Restart(reason)
			}
		default:
			_ = failed.Restart(reason)
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
		if failed.Parent != nil {
			as.handleFailure(failed.Parent, reason)
		} else {
			_ = failed.Restart(reason)
		}
	default:
		_ = failed.Restart(reason)
	}
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
			return item.Message, true
		}
	} else {
		if len(m.Messages) > 0 {
			msg := m.Messages[0]
			m.Messages = m.Messages[1:]
			m.Statistics.MessagesDequeued++
			m.LastActivity = time.Now()
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

// Helper methods

// deliverMessage delivers a message to its destination
func (as *ActorSystem) deliverMessage(msg Message) error {
	as.mutex.RLock()
	receiver, exists := as.actors[msg.Receiver]
	as.mutex.RUnlock()

	if !exists {
		return as.sendToDeadLetters(msg)
	}

	// Interceptors
	as.dispatcher.mutex.RLock()
	interceptors := append([]MessageInterceptor(nil), as.dispatcher.interceptors...)
	transformers := append([]MessageTransformer(nil), as.dispatcher.transformers...)
	as.dispatcher.mutex.RUnlock()

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
		deadline := time.Now().Add(m.BackPressureWait)
		for time.Now().Before(deadline) {
			if uint32(len(m.Messages)) < m.Capacity {
				m.Messages = append(m.Messages, msg)
				return nil
			}
			m.mutex.Unlock()
			time.Sleep(time.Millisecond * 5)
			m.mutex.Lock()
		}
		return fmt.Errorf("mailbox back pressure timeout")
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

	// Update statistics
	atomic.AddUint64(&as.statistics.ActiveActors, ^uint64(0)) // Decrement

	return nil
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

	// Start workers
	for i := 0; i < len(as.workers); i++ {
		as.workers[i] = &SchedulerWorker{
			ID:      i,
			Queue:   make(chan ActorID, 100),
			Running: true,
		}
		go as.runWorker(as.workers[i])
	}

	return nil
}

func (as *ActorScheduler) Stop() {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	as.running = false
	for _, worker := range as.workers {
		worker.Running = false
		close(worker.Queue)
	}
}

func (as *ActorScheduler) Schedule(actorID ActorID) {
	if !as.running {
		return
	}

	// Simple round-robin scheduling
	workerIndex := int(actorID) % len(as.workers)
	select {
	case as.workers[workerIndex].Queue <- actorID:
		as.statistics.TasksScheduled++
	default:
		// Queue full, try next worker
		workerIndex = (workerIndex + 1) % len(as.workers)
		select {
		case as.workers[workerIndex].Queue <- actorID:
			as.statistics.TasksScheduled++
		default:
			// All queues full, drop task
		}
	}
}

func (as *ActorScheduler) runWorker(worker *SchedulerWorker) {
	for worker.Running {
		select {
		case actorID := <-worker.Queue:
			// Process actor task
			as.statistics.TasksCompleted++
			if as.process != nil {
				as.process(actorID)
			}
		case <-as.ctx.Done():
			return
		}
	}
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
