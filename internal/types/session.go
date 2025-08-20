// Package types implements Phase 2.4.2 Session Type System for the Orizon compiler.
// This system provides session types for communication protocol verification and deadlock detection.
package types

import (
	"fmt"
	"strings"
	"sync"
)

// SessionTypeKind represents different kinds of session types.
type SessionTypeKind int

const (
	SessionKindEnd        SessionTypeKind = iota // Session termination
	SessionKindSend                              // Send operation: !T.S
	SessionKindReceive                           // Receive operation: ?T.S
	SessionKindChoice                            // Internal choice: +{l1:S1, l2:S2, ...}
	SessionKindBranch                            // External choice: &{l1:S1, l2:S2, ...}
	SessionKindRecursion                         // Recursive session: μX.S
	SessionKindVariable                          // Session variable: X
	SessionKindDual                              // Dual session: ~S
	SessionKindParallel                          // Parallel composition: S1 | S2
	SessionKindSequential                        // Sequential composition: S1; S2
)

// String returns a string representation of the session type kind.
func (stk SessionTypeKind) String() string {
	switch stk {
	case SessionKindEnd:
		return "end"
	case SessionKindSend:
		return "send"
	case SessionKindReceive:
		return "receive"
	case SessionKindChoice:
		return "choice"
	case SessionKindBranch:
		return "branch"
	case SessionKindRecursion:
		return "recursion"
	case SessionKindVariable:
		return "variable"
	case SessionKindDual:
		return "dual"
	case SessionKindParallel:
		return "parallel"
	case SessionKindSequential:
		return "sequential"
	default:
		return "unknown"
	}
}

// SessionType represents a session type for protocol specification.
type SessionType struct {
	PayloadType  *Type
	Branches     map[string]*SessionType
	Continuation *SessionType
	Body         *SessionType
	Left         *SessionType
	Right        *SessionType
	Label        string
	Variable     string
	UniqueId     string
	Kind         SessionTypeKind
}

// String returns a string representation of the session type.
func (st *SessionType) String() string {
	switch st.Kind {
	case SessionKindEnd:
		return "end"
	case SessionKindSend:
		cont := ""
		if st.Continuation != nil {
			cont = "." + st.Continuation.String()
		}

		return fmt.Sprintf("!%s%s", st.PayloadType.String(), cont)
	case SessionKindReceive:
		cont := ""
		if st.Continuation != nil {
			cont = "." + st.Continuation.String()
		}

		return fmt.Sprintf("?%s%s", st.PayloadType.String(), cont)
	case SessionKindChoice:
		branches := make([]string, 0, len(st.Branches))
		for label, session := range st.Branches {
			branches = append(branches, fmt.Sprintf("%s:%s", label, session.String()))
		}

		return fmt.Sprintf("+{%s}", strings.Join(branches, ", "))
	case SessionKindBranch:
		branches := make([]string, 0, len(st.Branches))
		for label, session := range st.Branches {
			branches = append(branches, fmt.Sprintf("%s:%s", label, session.String()))
		}

		return fmt.Sprintf("&{%s}", strings.Join(branches, ", "))
	case SessionKindRecursion:
		return fmt.Sprintf("μ%s.%s", st.Variable, st.Body.String())
	case SessionKindVariable:
		return st.Variable
	case SessionKindDual:
		return fmt.Sprintf("~%s", st.Continuation.String())
	case SessionKindParallel:
		return fmt.Sprintf("(%s | %s)", st.Left.String(), st.Right.String())
	case SessionKindSequential:
		return fmt.Sprintf("(%s; %s)", st.Left.String(), st.Right.String())
	default:
		return "unknown"
	}
}

// IsDual checks if two session types are dual to each other.
func (st *SessionType) IsDual(other *SessionType) bool {
	return st.computeDual().IsEquivalent(other)
}

// computeDual computes the dual of a session type.
func (st *SessionType) computeDual() *SessionType {
	switch st.Kind {
	case SessionKindEnd:
		return &SessionType{Kind: SessionKindEnd, UniqueId: generateSessionId()}
	case SessionKindSend:
		dualCont := &SessionType{Kind: SessionKindEnd, UniqueId: generateSessionId()}
		if st.Continuation != nil {
			dualCont = st.Continuation.computeDual()
		}

		return &SessionType{
			Kind:         SessionKindReceive,
			PayloadType:  st.PayloadType,
			Continuation: dualCont,
			UniqueId:     generateSessionId(),
		}
	case SessionKindReceive:
		dualCont := &SessionType{Kind: SessionKindEnd, UniqueId: generateSessionId()}
		if st.Continuation != nil {
			dualCont = st.Continuation.computeDual()
		}

		return &SessionType{
			Kind:         SessionKindSend,
			PayloadType:  st.PayloadType,
			Continuation: dualCont,
			UniqueId:     generateSessionId(),
		}
	case SessionKindChoice:
		dualBranches := make(map[string]*SessionType)
		for label, session := range st.Branches {
			dualBranches[label] = session.computeDual()
		}

		return &SessionType{
			Kind:     SessionKindBranch,
			Branches: dualBranches,
			UniqueId: generateSessionId(),
		}
	case SessionKindBranch:
		dualBranches := make(map[string]*SessionType)
		for label, session := range st.Branches {
			dualBranches[label] = session.computeDual()
		}

		return &SessionType{
			Kind:     SessionKindChoice,
			Branches: dualBranches,
			UniqueId: generateSessionId(),
		}
	case SessionKindRecursion:
		return &SessionType{
			Kind:     SessionKindRecursion,
			Variable: st.Variable,
			Body:     st.Body.computeDual(),
			UniqueId: generateSessionId(),
		}
	case SessionKindVariable:
		return &SessionType{
			Kind:     SessionKindVariable,
			Variable: st.Variable,
			UniqueId: generateSessionId(),
		}
	case SessionKindDual:
		return st.Continuation
	case SessionKindParallel:
		return &SessionType{
			Kind:     SessionKindParallel,
			Left:     st.Left.computeDual(),
			Right:    st.Right.computeDual(),
			UniqueId: generateSessionId(),
		}
	case SessionKindSequential:
		return &SessionType{
			Kind:     SessionKindSequential,
			Left:     st.Left.computeDual(),
			Right:    st.Right.computeDual(),
			UniqueId: generateSessionId(),
		}
	default:
		return &SessionType{Kind: SessionKindEnd, UniqueId: generateSessionId()}
	}
}

// IsEquivalent checks if two session types are equivalent.
func (st *SessionType) IsEquivalent(other *SessionType) bool {
	if st.Kind != other.Kind {
		return false
	}

	switch st.Kind {
	case SessionKindEnd:
		return true
	case SessionKindSend, SessionKindReceive:
		if !st.PayloadType.Equals(other.PayloadType) {
			return false
		}

		if st.Continuation == nil && other.Continuation == nil {
			return true
		}

		if st.Continuation != nil && other.Continuation != nil {
			return st.Continuation.IsEquivalent(other.Continuation)
		}

		return false
	case SessionKindChoice, SessionKindBranch:
		if len(st.Branches) != len(other.Branches) {
			return false
		}

		for label, session := range st.Branches {
			otherSession, exists := other.Branches[label]
			if !exists || !session.IsEquivalent(otherSession) {
				return false
			}
		}

		return true
	case SessionKindRecursion:
		return st.Variable == other.Variable && st.Body.IsEquivalent(other.Body)
	case SessionKindVariable:
		return st.Variable == other.Variable
	case SessionKindDual:
		return st.Continuation.IsEquivalent(other.Continuation)
	case SessionKindParallel, SessionKindSequential:
		return st.Left.IsEquivalent(other.Left) && st.Right.IsEquivalent(other.Right)
	default:
		return false
	}
}

// SessionChannel represents a communication channel with session type.
type SessionChannel struct {
	SessionType *SessionType
	Endpoint    *SessionEndpoint
	Partner     *SessionChannel
	Name        string
	Location    SourceLocation
	Direction   SessionChannelDirection
	State       ChannelState
}

// SessionChannelDirection represents the direction of communication.
type SessionChannelDirection int

const (
	SessionDirectionInput         SessionChannelDirection = iota // Input channel
	SessionDirectionOutput                                       // Output channel
	SessionDirectionBidirectional                                // Bidirectional channel
)

// String returns a string representation of the channel direction.
func (cd SessionChannelDirection) String() string {
	switch cd {
	case SessionDirectionInput:
		return "input"
	case SessionDirectionOutput:
		return "output"
	case SessionDirectionBidirectional:
		return "bidirectional"
	default:
		return "unknown"
	}
}

// ChannelState represents the current state of a session channel.
type ChannelState int

const (
	StateUnused    ChannelState = iota // Channel not yet used
	StateActive                        // Channel actively used
	StateCompleted                     // Session completed successfully
	StateClosed                        // Channel closed
	StateError                         // Channel in error state
)

// String returns a string representation of the channel state.
func (cs ChannelState) String() string {
	switch cs {
	case StateUnused:
		return "unused"
	case StateActive:
		return "active"
	case StateCompleted:
		return "completed"
	case StateClosed:
		return "closed"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// SessionEndpoint represents an endpoint in a session.
type SessionEndpoint struct {
	Id          string
	Process     string
	CurrentType *SessionType
	Operations  []SessionOperation
	IsCompleted bool
}

// SessionOperation represents an operation in a session.
type SessionOperation struct {
	Message   *Type
	Label     string
	Channel   string
	Location  SourceLocation
	Kind      SessionOperationKind
	Timestamp int64
}

// SessionOperationKind represents kinds of session operations.
type SessionOperationKind int

const (
	OpKindSend SessionOperationKind = iota
	OpKindReceive
	OpKindSelect // Select a branch in choice
	OpKindBranch // Branch on external choice
	OpKindClose  // Close session
)

// String returns a string representation of the operation kind.
func (sok SessionOperationKind) String() string {
	switch sok {
	case OpKindSend:
		return "send"
	case OpKindReceive:
		return "receive"
	case OpKindSelect:
		return "select"
	case OpKindBranch:
		return "branch"
	case OpKindClose:
		return "close"
	default:
		return "unknown"
	}
}

// SessionTypeChecker performs session type checking.
type SessionTypeChecker struct {
	channels    map[string]*SessionChannel
	processes   map[string]*SessionProcess
	constraints []SessionConstraint
	errors      []SessionTypeError
	deadlocks   []DeadlockError
	mutex       sync.RWMutex
}

// SessionProcess represents a process participating in sessions.
type SessionProcess struct {
	Channels   map[string]*SessionChannel
	Name       string
	Operations []SessionOperation
	Location   SourceLocation
	State      ProcessState
}

// ProcessState represents the state of a session process.
type ProcessState int

const (
	ProcessStateActive ProcessState = iota
	ProcessStateBlocked
	ProcessStateCompleted
	ProcessStateDeadlocked
)

// String returns a string representation of the process state.
func (ps ProcessState) String() string {
	switch ps {
	case ProcessStateActive:
		return "active"
	case ProcessStateBlocked:
		return "blocked"
	case ProcessStateCompleted:
		return "completed"
	case ProcessStateDeadlocked:
		return "deadlocked"
	default:
		return "unknown"
	}
}

// SessionConstraint represents a constraint in session type checking.
type SessionConstraint struct {
	Channel  string
	Process  string
	Message  string
	Location SourceLocation
	Kind     SessionConstraintKind
}

// SessionConstraintKind represents kinds of session constraints.
type SessionConstraintKind int

const (
	ConstraintKindProtocolCompliance SessionConstraintKind = iota
	ConstraintKindDuality
	ConstraintKindProgress
	ConstraintKindSafety
	ConstraintKindLiveness
)

// String returns a string representation of the constraint kind.
func (sck SessionConstraintKind) String() string {
	switch sck {
	case ConstraintKindProtocolCompliance:
		return "protocol-compliance"
	case ConstraintKindDuality:
		return "duality"
	case ConstraintKindProgress:
		return "progress"
	case ConstraintKindSafety:
		return "safety"
	case ConstraintKindLiveness:
		return "liveness"
	default:
		return "unknown"
	}
}

// SessionTypeError represents an error in session type checking.
type SessionTypeError struct {
	Expected *SessionType
	Actual   *SessionType
	Channel  string
	Process  string
	Message  string
	Location SourceLocation
	Kind     SessionErrorKind
}

// SessionErrorKind represents kinds of session type errors.
type SessionErrorKind int

const (
	ErrorKindProtocolViolation SessionErrorKind = iota
	ErrorKindDualityViolation
	ErrorKindTypeConflict
	ErrorKindUnexpectedOperation
	ErrorKindChannelMismatch
	ErrorKindSessionIncomplete
)

// String returns a string representation of the error kind.
func (sek SessionErrorKind) String() string {
	switch sek {
	case ErrorKindProtocolViolation:
		return "protocol-violation"
	case ErrorKindDualityViolation:
		return "duality-violation"
	case ErrorKindTypeConflict:
		return "type-conflict"
	case ErrorKindUnexpectedOperation:
		return "unexpected-operation"
	case ErrorKindChannelMismatch:
		return "channel-mismatch"
	case ErrorKindSessionIncomplete:
		return "session-incomplete"
	default:
		return "unknown"
	}
}

// Error implements the error interface.
func (ste SessionTypeError) Error() string {
	return fmt.Sprintf("Session type error (%s) at %s:%d:%d: %s",
		ste.Kind.String(), ste.Location.File, ste.Location.Line, ste.Location.Column, ste.Message)
}

// DeadlockError represents a deadlock detection error.
type DeadlockError struct {
	Message   string
	Processes []string
	Channels  []string
	Cycle     []string
	Location  SourceLocation
}

// Error implements the error interface.
func (de DeadlockError) Error() string {
	return fmt.Sprintf("Deadlock detected at %s:%d:%d: %s (processes: %v, channels: %v)",
		de.Location.File, de.Location.Line, de.Location.Column, de.Message,
		de.Processes, de.Channels)
}

// NewSessionTypeChecker creates a new session type checker.
func NewSessionTypeChecker() *SessionTypeChecker {
	return &SessionTypeChecker{
		channels:    make(map[string]*SessionChannel),
		processes:   make(map[string]*SessionProcess),
		constraints: make([]SessionConstraint, 0),
		errors:      make([]SessionTypeError, 0),
		deadlocks:   make([]DeadlockError, 0),
	}
}

// RegisterChannel registers a session channel.
func (stc *SessionTypeChecker) RegisterChannel(name string, sessionType *SessionType, direction SessionChannelDirection, location SourceLocation) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	if _, exists := stc.channels[name]; exists {
		return fmt.Errorf("channel %s already registered", name)
	}

	channel := &SessionChannel{
		Name:        name,
		SessionType: sessionType,
		Direction:   direction,
		State:       StateUnused,
		Location:    location,
	}

	stc.channels[name] = channel

	return nil
}

// RegisterProcess registers a session process.
func (stc *SessionTypeChecker) RegisterProcess(name string, location SourceLocation) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	if _, exists := stc.processes[name]; exists {
		return fmt.Errorf("process %s already registered", name)
	}

	process := &SessionProcess{
		Name:       name,
		Channels:   make(map[string]*SessionChannel),
		Operations: make([]SessionOperation, 0),
		State:      ProcessStateActive,
		Location:   location,
	}

	stc.processes[name] = process

	return nil
}

// ConnectChannels establishes a dual connection between two channels.
func (stc *SessionTypeChecker) ConnectChannels(channel1, channel2 string) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	ch1, exists1 := stc.channels[channel1]
	ch2, exists2 := stc.channels[channel2]

	if !exists1 {
		return fmt.Errorf("channel %s not found", channel1)
	}

	if !exists2 {
		return fmt.Errorf("channel %s not found", channel2)
	}

	// Check duality.
	if !ch1.SessionType.IsDual(ch2.SessionType) {
		return fmt.Errorf("channels %s and %s are not dual", channel1, channel2)
	}

	ch1.Partner = ch2
	ch2.Partner = ch1

	return nil
}

// CheckSendOperation checks a send operation.
func (stc *SessionTypeChecker) CheckSendOperation(process, channel string, messageType *Type, location SourceLocation) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	proc, procExists := stc.processes[process]
	if !procExists {
		return fmt.Errorf("process %s not found", process)
	}

	ch, chanExists := stc.channels[channel]
	if !chanExists {
		return fmt.Errorf("channel %s not found", channel)
	}

	// Check if the current session type allows sending.
	if ch.SessionType.Kind != SessionKindSend {
		return fmt.Errorf("channel %s does not expect send operation, current type: %s",
			channel, ch.SessionType.String())
	}

	// Check message type compatibility.
	if !ch.SessionType.PayloadType.Equals(messageType) {
		return fmt.Errorf("message type mismatch: expected %s, got %s",
			ch.SessionType.PayloadType.String(), messageType.String())
	}

	// Record the operation.
	operation := SessionOperation{
		Kind:     OpKindSend,
		Message:  messageType,
		Channel:  channel,
		Location: location,
	}

	proc.Operations = append(proc.Operations, operation)

	// Advance the session type.
	if ch.SessionType.Continuation != nil {
		ch.SessionType = ch.SessionType.Continuation
	} else {
		ch.SessionType = &SessionType{Kind: SessionKindEnd, UniqueId: generateSessionId()}
		ch.State = StateCompleted
	}

	return nil
}

// CheckReceiveOperation checks a receive operation.
func (stc *SessionTypeChecker) CheckReceiveOperation(process, channel string, messageType *Type, location SourceLocation) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	proc, procExists := stc.processes[process]
	if !procExists {
		return fmt.Errorf("process %s not found", process)
	}

	ch, chanExists := stc.channels[channel]
	if !chanExists {
		return fmt.Errorf("channel %s not found", channel)
	}

	// Check if the current session type allows receiving.
	if ch.SessionType.Kind != SessionKindReceive {
		return fmt.Errorf("channel %s does not expect receive operation, current type: %s",
			channel, ch.SessionType.String())
	}

	// Check message type compatibility.
	if !ch.SessionType.PayloadType.Equals(messageType) {
		return fmt.Errorf("message type mismatch: expected %s, got %s",
			ch.SessionType.PayloadType.String(), messageType.String())
	}

	// Record the operation.
	operation := SessionOperation{
		Kind:     OpKindReceive,
		Message:  messageType,
		Channel:  channel,
		Location: location,
	}

	proc.Operations = append(proc.Operations, operation)

	// Advance the session type.
	if ch.SessionType.Continuation != nil {
		ch.SessionType = ch.SessionType.Continuation
	} else {
		ch.SessionType = &SessionType{Kind: SessionKindEnd, UniqueId: generateSessionId()}
		ch.State = StateCompleted
	}

	return nil
}

// CheckChoiceOperation checks a choice (select) operation.
func (stc *SessionTypeChecker) CheckChoiceOperation(process, channel, label string, location SourceLocation) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	proc, procExists := stc.processes[process]
	if !procExists {
		return fmt.Errorf("process %s not found", process)
	}

	ch, chanExists := stc.channels[channel]
	if !chanExists {
		return fmt.Errorf("channel %s not found", channel)
	}

	// Check if the current session type is a choice.
	if ch.SessionType.Kind != SessionKindChoice {
		return fmt.Errorf("channel %s does not expect choice operation, current type: %s",
			channel, ch.SessionType.String())
	}

	// Check if the label exists.
	nextSession, labelExists := ch.SessionType.Branches[label]
	if !labelExists {
		return fmt.Errorf("label %s not found in choice for channel %s", label, channel)
	}

	// Record the operation.
	operation := SessionOperation{
		Kind:     OpKindSelect,
		Label:    label,
		Channel:  channel,
		Location: location,
	}

	proc.Operations = append(proc.Operations, operation)

	// Advance to the selected branch.
	ch.SessionType = nextSession

	return nil
}

// CheckBranchOperation checks a branch operation.
func (stc *SessionTypeChecker) CheckBranchOperation(process, channel string, branches map[string]func(), location SourceLocation) error {
	stc.mutex.Lock()
	defer stc.mutex.Unlock()

	proc, procExists := stc.processes[process]
	if !procExists {
		return fmt.Errorf("process %s not found", process)
	}

	ch, chanExists := stc.channels[channel]
	if !chanExists {
		return fmt.Errorf("channel %s not found", channel)
	}

	// Check if the current session type is a branch.
	if ch.SessionType.Kind != SessionKindBranch {
		return fmt.Errorf("channel %s does not expect branch operation, current type: %s",
			channel, ch.SessionType.String())
	}

	// Check if all expected branches are provided.
	for expectedLabel := range ch.SessionType.Branches {
		if _, provided := branches[expectedLabel]; !provided {
			return fmt.Errorf("missing branch handler for label %s in channel %s", expectedLabel, channel)
		}
	}

	// Check if no extra branches are provided.
	for providedLabel := range branches {
		if _, expected := ch.SessionType.Branches[providedLabel]; !expected {
			return fmt.Errorf("unexpected branch handler for label %s in channel %s", providedLabel, channel)
		}
	}

	// Record the operation.
	operation := SessionOperation{
		Kind:     OpKindBranch,
		Channel:  channel,
		Location: location,
	}

	proc.Operations = append(proc.Operations, operation)

	// Note: The actual branch selection will happen at runtime.
	// For static analysis, we assume all branches are possible.

	return nil
}

// DetectDeadlocks performs deadlock detection analysis.
func (stc *SessionTypeChecker) DetectDeadlocks() []DeadlockError {
	stc.mutex.RLock()
	defer stc.mutex.RUnlock()

	deadlocks := make([]DeadlockError, 0)

	// Build dependency graph.
	dependencies := stc.buildDependencyGraph()

	// Find strongly connected components (cycles).
	cycles := stc.findCycles(dependencies)

	for _, cycle := range cycles {
		if stc.isDeadlockCycle(cycle) {
			deadlock := DeadlockError{
				Processes: cycle,
				Channels:  stc.getChannelsInCycle(cycle),
				Cycle:     cycle,
				Message:   fmt.Sprintf("Circular dependency detected among processes: %v", cycle),
			}
			deadlocks = append(deadlocks, deadlock)
		}
	}

	return deadlocks
}

// buildDependencyGraph builds a dependency graph between processes.
func (stc *SessionTypeChecker) buildDependencyGraph() map[string][]string {
	dependencies := make(map[string][]string)

	for processName, process := range stc.processes {
		dependencies[processName] = make([]string, 0)

		for _, channel := range process.Channels {
			if channel.Partner != nil {
				// Find the process that owns the partner channel.
				for otherProcessName, otherProcess := range stc.processes {
					for _, otherChannel := range otherProcess.Channels {
						if otherChannel == channel.Partner {
							dependencies[processName] = append(dependencies[processName], otherProcessName)

							break
						}
					}
				}
			}
		}
	}

	return dependencies
}

// findCycles finds cycles in the dependency graph using Tarjan's algorithm.
func (stc *SessionTypeChecker) findCycles(dependencies map[string][]string) [][]string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	cycles := make([][]string, 0)

	var dfs func(string, []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		recStack[node] = true

		path = append(path, node)

		for _, neighbor := range dependencies[node] {
			if !visited[neighbor] {
				dfs(neighbor, path)
			} else if recStack[neighbor] {
				// Found a cycle.
				cycleStart := -1

				for i, p := range path {
					if p == neighbor {
						cycleStart = i

						break
					}
				}

				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
			}
		}

		recStack[node] = false
	}

	for node := range dependencies {
		if !visited[node] {
			dfs(node, make([]string, 0))
		}
	}

	return cycles
}

// isDeadlockCycle checks if a cycle represents a deadlock.
func (stc *SessionTypeChecker) isDeadlockCycle(cycle []string) bool {
	// A cycle is a deadlock if all processes in the cycle are waiting.
	for _, processName := range cycle {
		process, exists := stc.processes[processName]
		if !exists {
			continue
		}

		// Check if the process is blocked waiting for input.
		isBlocked := false

		for _, channel := range process.Channels {
			if channel.SessionType.Kind == SessionKindReceive {
				isBlocked = true

				break
			}
		}

		if !isBlocked {
			return false // If any process is not blocked, it's not a deadlock
		}
	}

	return true
}

// getChannelsInCycle gets all channels involved in a cycle.
func (stc *SessionTypeChecker) getChannelsInCycle(cycle []string) []string {
	channelSet := make(map[string]bool)

	for _, processName := range cycle {
		process, exists := stc.processes[processName]
		if !exists {
			continue
		}

		for channelName := range process.Channels {
			channelSet[channelName] = true
		}
	}

	channels := make([]string, 0, len(channelSet))
	for channel := range channelSet {
		channels = append(channels, channel)
	}

	return channels
}

// ValidateProtocol validates that all sessions follow their protocols.
func (stc *SessionTypeChecker) ValidateProtocol() []SessionTypeError {
	stc.mutex.RLock()
	defer stc.mutex.RUnlock()

	errors := make([]SessionTypeError, 0)

	// Check that all channels have completed their sessions.
	for channelName, channel := range stc.channels {
		if channel.SessionType.Kind != SessionKindEnd && channel.State != StateCompleted {
			error := SessionTypeError{
				Kind:    ErrorKindSessionIncomplete,
				Channel: channelName,
				Message: fmt.Sprintf("session for channel %s is incomplete, current type: %s",
					channelName, channel.SessionType.String()),
			}
			errors = append(errors, error)
		}
	}

	// Check duality constraints.
	for channelName, channel := range stc.channels {
		if channel.Partner != nil {
			if !channel.SessionType.IsDual(channel.Partner.SessionType) {
				error := SessionTypeError{
					Kind:     ErrorKindDualityViolation,
					Channel:  channelName,
					Expected: channel.SessionType.computeDual(),
					Actual:   channel.Partner.SessionType,
					Message: fmt.Sprintf("channel %s and its partner are not dual",
						channelName),
				}
				errors = append(errors, error)
			}
		}
	}

	return errors
}

// SessionTypeConstructors provides convenient constructors.

// NewEndSession creates an end session type.
func NewEndSession() *SessionType {
	return &SessionType{
		Kind:     SessionKindEnd,
		UniqueId: generateSessionId(),
	}
}

// NewSendSession creates a send session type.
func NewSendSession(payloadType *Type, continuation *SessionType) *SessionType {
	return &SessionType{
		Kind:         SessionKindSend,
		PayloadType:  payloadType,
		Continuation: continuation,
		UniqueId:     generateSessionId(),
	}
}

// NewReceiveSession creates a receive session type.
func NewReceiveSession(payloadType *Type, continuation *SessionType) *SessionType {
	return &SessionType{
		Kind:         SessionKindReceive,
		PayloadType:  payloadType,
		Continuation: continuation,
		UniqueId:     generateSessionId(),
	}
}

// NewChoiceSession creates a choice session type.
func NewChoiceSession(branches map[string]*SessionType) *SessionType {
	return &SessionType{
		Kind:     SessionKindChoice,
		Branches: branches,
		UniqueId: generateSessionId(),
	}
}

// NewBranchSession creates a branch session type.
func NewBranchSession(branches map[string]*SessionType) *SessionType {
	return &SessionType{
		Kind:     SessionKindBranch,
		Branches: branches,
		UniqueId: generateSessionId(),
	}
}

// NewRecursiveSession creates a recursive session type.
func NewRecursiveSession(variable string, body *SessionType) *SessionType {
	return &SessionType{
		Kind:     SessionKindRecursion,
		Variable: variable,
		Body:     body,
		UniqueId: generateSessionId(),
	}
}

// NewSessionVariable creates a session variable.
func NewSessionVariable(variable string) *SessionType {
	return &SessionType{
		Kind:     SessionKindVariable,
		Variable: variable,
		UniqueId: generateSessionId(),
	}
}

// NewDualSession creates a dual session type.
func NewDualSession(session *SessionType) *SessionType {
	return &SessionType{
		Kind:         SessionKindDual,
		Continuation: session,
		UniqueId:     generateSessionId(),
	}
}

// NewParallelSession creates a parallel session composition.
func NewParallelSession(left, right *SessionType) *SessionType {
	return &SessionType{
		Kind:     SessionKindParallel,
		Left:     left,
		Right:    right,
		UniqueId: generateSessionId(),
	}
}

// NewSequentialSession creates a sequential session composition.
func NewSequentialSession(left, right *SessionType) *SessionType {
	return &SessionType{
		Kind:     SessionKindSequential,
		Left:     left,
		Right:    right,
		UniqueId: generateSessionId(),
	}
}

// Protocol analysis and verification.

// ProtocolAnalyzer analyzes session protocols for properties.
type ProtocolAnalyzer struct {
	checker   *SessionTypeChecker
	protocols map[string]*Protocol
	safety    []SafetyProperty
	liveness  []LivenessProperty
}

// Protocol represents a communication protocol.
type Protocol struct {
	Name         string
	Participants []string
	Sessions     map[string]*SessionType
	Properties   []ProtocolProperty
}

// ProtocolProperty represents a property of a protocol.
type ProtocolProperty struct {
	Description string
	Formula     string
	Kind        PropertyKind
}

// PropertyKind represents kinds of protocol properties.
type PropertyKind int

const (
	PropertyKindSafety PropertyKind = iota
	PropertyKindLiveness
	PropertyKindFairness
	PropertyKindProgress
)

// SafetyProperty represents a safety property.
type SafetyProperty struct {
	Invariant   func(*SessionTypeChecker) bool
	Name        string
	Description string
}

// LivenessProperty represents a liveness property.
type LivenessProperty struct {
	Eventually  func(*SessionTypeChecker) bool
	Name        string
	Description string
}

// NewProtocolAnalyzer creates a new protocol analyzer.
func NewProtocolAnalyzer(checker *SessionTypeChecker) *ProtocolAnalyzer {
	return &ProtocolAnalyzer{
		checker:   checker,
		protocols: make(map[string]*Protocol),
		safety:    make([]SafetyProperty, 0),
		liveness:  make([]LivenessProperty, 0),
	}
}

// AddSafetyProperty adds a safety property to check.
func (pa *ProtocolAnalyzer) AddSafetyProperty(name, description string, invariant func(*SessionTypeChecker) bool) {
	property := SafetyProperty{
		Name:        name,
		Description: description,
		Invariant:   invariant,
	}
	pa.safety = append(pa.safety, property)
}

// AddLivenessProperty adds a liveness property to check.
func (pa *ProtocolAnalyzer) AddLivenessProperty(name, description string, eventually func(*SessionTypeChecker) bool) {
	property := LivenessProperty{
		Name:        name,
		Description: description,
		Eventually:  eventually,
	}
	pa.liveness = append(pa.liveness, property)
}

// VerifyProtocol verifies a protocol against its properties.
func (pa *ProtocolAnalyzer) VerifyProtocol(protocolName string) []ProtocolViolation {
	violations := make([]ProtocolViolation, 0)

	// Check safety properties.
	for _, safety := range pa.safety {
		if !safety.Invariant(pa.checker) {
			violation := ProtocolViolation{
				Kind:        ViolationKindSafety,
				Property:    safety.Name,
				Description: safety.Description,
				Message:     fmt.Sprintf("Safety property '%s' violated", safety.Name),
			}
			violations = append(violations, violation)
		}
	}

	// Check liveness properties.
	for _, liveness := range pa.liveness {
		if !liveness.Eventually(pa.checker) {
			violation := ProtocolViolation{
				Kind:        ViolationKindLiveness,
				Property:    liveness.Name,
				Description: liveness.Description,
				Message:     fmt.Sprintf("Liveness property '%s' violated", liveness.Name),
			}
			violations = append(violations, violation)
		}
	}

	return violations
}

// ProtocolViolation represents a violation of protocol properties.
type ProtocolViolation struct {
	Property    string
	Description string
	Message     string
	Kind        ViolationKind
}

// ViolationKind represents kinds of protocol violations.
type ViolationKind int

const (
	ViolationKindSafety ViolationKind = iota
	ViolationKindLiveness
	ViolationKindFairness
	ViolationKindProgress
)

// generateSessionId generates a unique identifier for sessions.
func generateSessionId() string {
	return fmt.Sprintf("session_%d", len("temp_id"))
}

// SessionTypeKind constant for type system integration.
const (
	TypeKindSession TypeKind = 201
)
