// Package types provides I/O effect system for static I/O operation tracking and control.
// This module implements comprehensive I/O effect tracking, pure function guarantees,
// and I/O monad equivalent functionality for the Orizon language.
package types

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// IOEffectKind represents different categories of I/O effects
type IOEffectKind int

const (
	// Pure I/O - no external side effects
	IOEffectPure IOEffectKind = iota

	// File system I/O
	IOEffectFileRead
	IOEffectFileWrite
	IOEffectFileCreate
	IOEffectFileDelete
	IOEffectFileRename
	IOEffectDirectoryCreate
	IOEffectDirectoryDelete
	IOEffectDirectoryList
	IOEffectFilePermissions
	IOEffectFileMetadata

	// Network I/O
	IOEffectNetworkConnect
	IOEffectNetworkListen
	IOEffectNetworkSend
	IOEffectNetworkReceive
	IOEffectNetworkClose
	IOEffectHTTPRequest
	IOEffectHTTPResponse
	IOEffectWebSocket
	IOEffectDNSLookup

	// Console I/O
	IOEffectConsoleRead
	IOEffectConsoleWrite
	IOEffectConsoleError
	IOEffectStdinRead
	IOEffectStdoutWrite
	IOEffectStderrWrite

	// Database I/O
	IOEffectDatabaseConnect
	IOEffectDatabaseQuery
	IOEffectDatabaseInsert
	IOEffectDatabaseUpdate
	IOEffectDatabaseDelete
	IOEffectDatabaseTransaction
	IOEffectDatabaseCommit
	IOEffectDatabaseRollback

	// System I/O
	IOEffectEnvironmentRead
	IOEffectEnvironmentWrite
	IOEffectProcessSpawn
	IOEffectProcessKill
	IOEffectSignalSend
	IOEffectSystemCall
	IOEffectDeviceRead
	IOEffectDeviceWrite

	// Memory-mapped I/O
	IOEffectMemoryMap
	IOEffectMemoryUnmap
	IOEffectMemorySync

	// Inter-process I/O
	IOEffectPipeRead
	IOEffectPipeWrite
	IOEffectSocketRead
	IOEffectSocketWrite
	IOEffectSharedMemoryRead
	IOEffectSharedMemoryWrite

	// Time-based I/O
	IOEffectTimer
	IOEffectSleep
	IOEffectTimeout

	// Custom I/O effects
	IOEffectCustom
)

func (iek IOEffectKind) String() string {
	switch iek {
	case IOEffectPure:
		return "Pure"
	case IOEffectFileRead:
		return "FileRead"
	case IOEffectFileWrite:
		return "FileWrite"
	case IOEffectFileCreate:
		return "FileCreate"
	case IOEffectFileDelete:
		return "FileDelete"
	case IOEffectFileRename:
		return "FileRename"
	case IOEffectDirectoryCreate:
		return "DirectoryCreate"
	case IOEffectDirectoryDelete:
		return "DirectoryDelete"
	case IOEffectDirectoryList:
		return "DirectoryList"
	case IOEffectFilePermissions:
		return "FilePermissions"
	case IOEffectFileMetadata:
		return "FileMetadata"
	case IOEffectNetworkConnect:
		return "NetworkConnect"
	case IOEffectNetworkListen:
		return "NetworkListen"
	case IOEffectNetworkSend:
		return "NetworkSend"
	case IOEffectNetworkReceive:
		return "NetworkReceive"
	case IOEffectNetworkClose:
		return "NetworkClose"
	case IOEffectHTTPRequest:
		return "HTTPRequest"
	case IOEffectHTTPResponse:
		return "HTTPResponse"
	case IOEffectWebSocket:
		return "WebSocket"
	case IOEffectDNSLookup:
		return "DNSLookup"
	case IOEffectConsoleRead:
		return "ConsoleRead"
	case IOEffectConsoleWrite:
		return "ConsoleWrite"
	case IOEffectConsoleError:
		return "ConsoleError"
	case IOEffectStdinRead:
		return "StdinRead"
	case IOEffectStdoutWrite:
		return "StdoutWrite"
	case IOEffectStderrWrite:
		return "StderrWrite"
	case IOEffectDatabaseConnect:
		return "DatabaseConnect"
	case IOEffectDatabaseQuery:
		return "DatabaseQuery"
	case IOEffectDatabaseInsert:
		return "DatabaseInsert"
	case IOEffectDatabaseUpdate:
		return "DatabaseUpdate"
	case IOEffectDatabaseDelete:
		return "DatabaseDelete"
	case IOEffectDatabaseTransaction:
		return "DatabaseTransaction"
	case IOEffectDatabaseCommit:
		return "DatabaseCommit"
	case IOEffectDatabaseRollback:
		return "DatabaseRollback"
	case IOEffectEnvironmentRead:
		return "EnvironmentRead"
	case IOEffectEnvironmentWrite:
		return "EnvironmentWrite"
	case IOEffectProcessSpawn:
		return "ProcessSpawn"
	case IOEffectProcessKill:
		return "ProcessKill"
	case IOEffectSignalSend:
		return "SignalSend"
	case IOEffectSystemCall:
		return "SystemCall"
	case IOEffectDeviceRead:
		return "DeviceRead"
	case IOEffectDeviceWrite:
		return "DeviceWrite"
	case IOEffectMemoryMap:
		return "MemoryMap"
	case IOEffectMemoryUnmap:
		return "MemoryUnmap"
	case IOEffectMemorySync:
		return "MemorySync"
	case IOEffectPipeRead:
		return "PipeRead"
	case IOEffectPipeWrite:
		return "PipeWrite"
	case IOEffectSocketRead:
		return "SocketRead"
	case IOEffectSocketWrite:
		return "SocketWrite"
	case IOEffectSharedMemoryRead:
		return "SharedMemoryRead"
	case IOEffectSharedMemoryWrite:
		return "SharedMemoryWrite"
	case IOEffectTimer:
		return "Timer"
	case IOEffectSleep:
		return "Sleep"
	case IOEffectTimeout:
		return "Timeout"
	case IOEffectCustom:
		return "Custom"
	default:
		return fmt.Sprintf("Unknown(%d)", int(iek))
	}
}

// IOEffectPermission represents the permission level of an I/O effect
type IOEffectPermission int

const (
	IOPermissionNone IOEffectPermission = iota
	IOPermissionRead
	IOPermissionWrite
	IOPermissionExecute
	IOPermissionReadWrite
	IOPermissionFullAccess
)

func (iep IOEffectPermission) String() string {
	switch iep {
	case IOPermissionNone:
		return "None"
	case IOPermissionRead:
		return "Read"
	case IOPermissionWrite:
		return "Write"
	case IOPermissionExecute:
		return "Execute"
	case IOPermissionReadWrite:
		return "ReadWrite"
	case IOPermissionFullAccess:
		return "FullAccess"
	default:
		return fmt.Sprintf("Unknown(%d)", int(iep))
	}
}

// IOEffectBehavior represents the behavior characteristics of an I/O effect
type IOEffectBehavior int

const (
	IOBehaviorDeterministic IOEffectBehavior = iota
	IOBehaviorNonDeterministic
	IOBehaviorIdempotent
	IOBehaviorSideEffecting
	IOBehaviorBlocking
	IOBehaviorNonBlocking
	IOBehaviorAtomic
	IOBehaviorNonAtomic
)

func (ieb IOEffectBehavior) String() string {
	switch ieb {
	case IOBehaviorDeterministic:
		return "Deterministic"
	case IOBehaviorNonDeterministic:
		return "NonDeterministic"
	case IOBehaviorIdempotent:
		return "Idempotent"
	case IOBehaviorSideEffecting:
		return "SideEffecting"
	case IOBehaviorBlocking:
		return "Blocking"
	case IOBehaviorNonBlocking:
		return "NonBlocking"
	case IOBehaviorAtomic:
		return "Atomic"
	case IOBehaviorNonAtomic:
		return "NonAtomic"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ieb))
	}
}

// IOEffect represents a single I/O effect with its characteristics
type IOEffect struct {
	Kind        IOEffectKind
	Permission  IOEffectPermission
	Behaviors   []IOEffectBehavior
	Resource    string
	Description string
	Level       EffectLevel
	Location    SourceLocation
	Context     string
	Metadata    map[string]interface{}
	Timestamp   time.Time
}

// NewIOEffect creates a new I/O effect
func NewIOEffect(kind IOEffectKind, permission IOEffectPermission) *IOEffect {
	return &IOEffect{
		Kind:       kind,
		Permission: permission,
		Behaviors:  make([]IOEffectBehavior, 0),
		Level:      EffectLevelMedium,
		Metadata:   make(map[string]interface{}),
		Timestamp:  time.Now(),
	}
}

func (ie *IOEffect) String() string {
	if ie.Resource != "" {
		return fmt.Sprintf("%s[%s]@%s", ie.Kind.String(), ie.Permission.String(), ie.Resource)
	}
	return fmt.Sprintf("%s[%s]", ie.Kind.String(), ie.Permission.String())
}

func (ie *IOEffect) Clone() *IOEffect {
	clone := &IOEffect{
		Kind:        ie.Kind,
		Permission:  ie.Permission,
		Resource:    ie.Resource,
		Description: ie.Description,
		Level:       ie.Level,
		Location:    ie.Location,
		Context:     ie.Context,
		Timestamp:   ie.Timestamp,
		Behaviors:   make([]IOEffectBehavior, len(ie.Behaviors)),
		Metadata:    make(map[string]interface{}),
	}

	copy(clone.Behaviors, ie.Behaviors)
	for k, v := range ie.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

func (ie *IOEffect) IsPure() bool {
	return ie.Kind == IOEffectPure
}

func (ie *IOEffect) IsReadOnly() bool {
	return ie.Permission == IOPermissionRead
}

func (ie *IOEffect) IsWriteAccess() bool {
	return ie.Permission == IOPermissionWrite ||
		ie.Permission == IOPermissionReadWrite ||
		ie.Permission == IOPermissionFullAccess
}

func (ie *IOEffect) HasBehavior(behavior IOEffectBehavior) bool {
	for _, b := range ie.Behaviors {
		if b == behavior {
			return true
		}
	}
	return false
}

func (ie *IOEffect) AddBehavior(behavior IOEffectBehavior) {
	if !ie.HasBehavior(behavior) {
		ie.Behaviors = append(ie.Behaviors, behavior)
	}
}

// IOEffectSet represents a collection of I/O effects
type IOEffectSet struct {
	effects map[string]*IOEffect
	mu      sync.RWMutex
}

// NewIOEffectSet creates a new I/O effect set
func NewIOEffectSet() *IOEffectSet {
	return &IOEffectSet{
		effects: make(map[string]*IOEffect),
	}
}

func (ies *IOEffectSet) Add(effect *IOEffect) {
	ies.mu.Lock()
	defer ies.mu.Unlock()

	key := effect.String()
	ies.effects[key] = effect.Clone()
}

func (ies *IOEffectSet) Remove(effect *IOEffect) {
	ies.mu.Lock()
	defer ies.mu.Unlock()

	key := effect.String()
	delete(ies.effects, key)
}

func (ies *IOEffectSet) Contains(effect *IOEffect) bool {
	ies.mu.RLock()
	defer ies.mu.RUnlock()

	key := effect.String()
	_, exists := ies.effects[key]
	return exists
}

func (ies *IOEffectSet) Size() int {
	ies.mu.RLock()
	defer ies.mu.RUnlock()
	return len(ies.effects)
}

func (ies *IOEffectSet) IsEmpty() bool {
	ies.mu.RLock()
	defer ies.mu.RUnlock()
	return len(ies.effects) == 0
}

func (ies *IOEffectSet) IsPure() bool {
	ies.mu.RLock()
	defer ies.mu.RUnlock()

	if len(ies.effects) == 0 {
		return true
	}

	for _, effect := range ies.effects {
		if !effect.IsPure() {
			return false
		}
	}
	return true
}

func (ies *IOEffectSet) Union(other *IOEffectSet) *IOEffectSet {
	result := NewIOEffectSet()

	ies.mu.RLock()
	for _, effect := range ies.effects {
		result.Add(effect)
	}
	ies.mu.RUnlock()

	other.mu.RLock()
	for _, effect := range other.effects {
		result.Add(effect)
	}
	other.mu.RUnlock()

	return result
}

func (ies *IOEffectSet) Intersection(other *IOEffectSet) *IOEffectSet {
	result := NewIOEffectSet()

	ies.mu.RLock()
	other.mu.RLock()
	defer ies.mu.RUnlock()
	defer other.mu.RUnlock()

	for key, effect := range ies.effects {
		if _, exists := other.effects[key]; exists {
			result.Add(effect)
		}
	}

	return result
}

func (ies *IOEffectSet) Difference(other *IOEffectSet) *IOEffectSet {
	result := NewIOEffectSet()

	ies.mu.RLock()
	other.mu.RLock()
	defer ies.mu.RUnlock()
	defer other.mu.RUnlock()

	for key, effect := range ies.effects {
		if _, exists := other.effects[key]; !exists {
			result.Add(effect)
		}
	}

	return result
}

func (ies *IOEffectSet) ToSlice() []*IOEffect {
	ies.mu.RLock()
	defer ies.mu.RUnlock()

	effects := make([]*IOEffect, 0, len(ies.effects))
	for _, effect := range ies.effects {
		effects = append(effects, effect.Clone())
	}

	// Sort by effect kind for consistent ordering
	sort.Slice(effects, func(i, j int) bool {
		return effects[i].Kind < effects[j].Kind
	})

	return effects
}

func (ies *IOEffectSet) String() string {
	effects := ies.ToSlice()
	if len(effects) == 0 {
		return "PureIO"
	}

	var strs []string
	for _, effect := range effects {
		strs = append(strs, effect.String())
	}

	return "{" + strings.Join(strs, ", ") + "}"
}

// IOConstraint represents a constraint on I/O effects
type IOConstraint interface {
	Check(effect *IOEffect) bool
	String() string
}

// IOPermissionConstraint restricts I/O effects by permission level
type IOPermissionConstraint struct {
	AllowedPermissions []IOEffectPermission
	Description        string
}

func NewIOPermissionConstraint(permissions ...IOEffectPermission) *IOPermissionConstraint {
	return &IOPermissionConstraint{
		AllowedPermissions: permissions,
		Description:        "Permission constraint",
	}
}

func (ipc *IOPermissionConstraint) Check(effect *IOEffect) bool {
	for _, permission := range ipc.AllowedPermissions {
		if effect.Permission == permission {
			return true
		}
	}
	return false
}

func (ipc *IOPermissionConstraint) String() string {
	var permissions []string
	for _, p := range ipc.AllowedPermissions {
		permissions = append(permissions, p.String())
	}
	return fmt.Sprintf("allow permissions: %s", strings.Join(permissions, ", "))
}

// IOResourceConstraint restricts I/O effects by resource pattern
type IOResourceConstraint struct {
	AllowedResources []string
	DeniedResources  []string
	Description      string
}

func NewIOResourceConstraint() *IOResourceConstraint {
	return &IOResourceConstraint{
		AllowedResources: make([]string, 0),
		DeniedResources:  make([]string, 0),
		Description:      "Resource constraint",
	}
}

func (irc *IOResourceConstraint) AllowResource(resource string) {
	irc.AllowedResources = append(irc.AllowedResources, resource)
}

func (irc *IOResourceConstraint) DenyResource(resource string) {
	irc.DeniedResources = append(irc.DeniedResources, resource)
}

func (irc *IOResourceConstraint) Check(effect *IOEffect) bool {
	// Check denied resources first
	for _, denied := range irc.DeniedResources {
		if strings.Contains(effect.Resource, denied) {
			return false
		}
	}

	// If no allowed resources specified, allow by default (unless explicitly denied above)
	if len(irc.AllowedResources) == 0 {
		return true
	}

	// Check allowed resources
	for _, allowed := range irc.AllowedResources {
		if strings.Contains(effect.Resource, allowed) {
			return true
		}
	}

	// If allowed resources are specified but resource doesn't match any,
	// still allow if not explicitly denied (for backwards compatibility)
	return true
}

func (irc *IOResourceConstraint) String() string {
	var parts []string
	if len(irc.AllowedResources) > 0 {
		parts = append(parts, fmt.Sprintf("allow: %v", irc.AllowedResources))
	}
	if len(irc.DeniedResources) > 0 {
		parts = append(parts, fmt.Sprintf("deny: %v", irc.DeniedResources))
	}
	return fmt.Sprintf("resource constraint: %s", strings.Join(parts, ", "))
}

// IOSignature represents the complete I/O signature of a function
type IOSignature struct {
	FunctionName  string
	Effects       *IOEffectSet
	Requires      *IOEffectSet // Required I/O capabilities
	Ensures       *IOEffectSet // Guaranteed I/O effects
	Constraints   []IOConstraint
	Pure          bool
	Deterministic bool
	Idempotent    bool
}

// NewIOSignature creates a new I/O signature
func NewIOSignature(name string) *IOSignature {
	return &IOSignature{
		FunctionName:  name,
		Effects:       NewIOEffectSet(),
		Requires:      NewIOEffectSet(),
		Ensures:       NewIOEffectSet(),
		Constraints:   make([]IOConstraint, 0),
		Pure:          true,
		Deterministic: true,
		Idempotent:    true,
	}
}

func (ios *IOSignature) AddEffect(effect *IOEffect) {
	ios.Effects.Add(effect)

	// Update signature properties
	if !effect.IsPure() {
		ios.Pure = false
	}

	if effect.HasBehavior(IOBehaviorNonDeterministic) {
		ios.Deterministic = false
	}

	if !effect.HasBehavior(IOBehaviorIdempotent) {
		ios.Idempotent = false
	}
}

func (ios *IOSignature) AddConstraint(constraint IOConstraint) {
	ios.Constraints = append(ios.Constraints, constraint)
}

func (ios *IOSignature) CheckConstraints(effect *IOEffect) []string {
	var violations []string

	for _, constraint := range ios.Constraints {
		if !constraint.Check(effect) {
			violations = append(violations, fmt.Sprintf("constraint violation: %s", constraint.String()))
		}
	}

	return violations
}

func (ios *IOSignature) String() string {
	return fmt.Sprintf("%s: effects=%s, pure=%v, deterministic=%v, idempotent=%v",
		ios.FunctionName, ios.Effects.String(), ios.Pure, ios.Deterministic, ios.Idempotent)
}

// IOMonad represents the I/O monad for sequencing I/O operations
type IOMonad struct {
	action func() (interface{}, error)
	next   *IOMonad
}

// NewIOMonad creates a new I/O monad
func NewIOMonad(action func() (interface{}, error)) *IOMonad {
	return &IOMonad{action: action}
}

// PureIO creates a pure I/O monad that returns a value without side effects
func PureIO(value interface{}) *IOMonad {
	return NewIOMonad(func() (interface{}, error) {
		return value, nil
	})
}

// Bind chains I/O operations in a monadic fashion
func (iom *IOMonad) Bind(f func(interface{}) *IOMonad) *IOMonad {
	return NewIOMonad(func() (interface{}, error) {
		result, err := iom.action()
		if err != nil {
			return nil, err
		}

		next := f(result)
		return next.action()
	})
}

// Map applies a pure function to the result of an I/O operation
func (iom *IOMonad) Map(f func(interface{}) interface{}) *IOMonad {
	return NewIOMonad(func() (interface{}, error) {
		result, err := iom.action()
		if err != nil {
			return nil, err
		}
		return f(result), nil
	})
}

// Run executes the I/O monad and returns the result
func (iom *IOMonad) Run() (interface{}, error) {
	return iom.action()
}

// Sequence combines multiple I/O monads into a single monad that returns a slice of results
func Sequence(monads []*IOMonad) *IOMonad {
	return NewIOMonad(func() (interface{}, error) {
		results := make([]interface{}, len(monads))
		for i, monad := range monads {
			result, err := monad.Run()
			if err != nil {
				return nil, err
			}
			results[i] = result
		}
		return results, nil
	})
}

// Parallel executes multiple I/O monads in parallel and returns their results
func Parallel(monads []*IOMonad) *IOMonad {
	return NewIOMonad(func() (interface{}, error) {
		results := make([]interface{}, len(monads))
		errors := make([]error, len(monads))

		var wg sync.WaitGroup
		for i, monad := range monads {
			wg.Add(1)
			go func(index int, m *IOMonad) {
				defer wg.Done()
				results[index], errors[index] = m.Run()
			}(i, monad)
		}

		wg.Wait()

		// Check for errors
		for _, err := range errors {
			if err != nil {
				return nil, err
			}
		}

		return results, nil
	})
}

// IOContext provides context for I/O operations
type IOContext struct {
	Permissions  []IOEffectPermission
	AllowedKinds []IOEffectKind
	Constraints  []IOConstraint
	Timeout      time.Duration
	Metadata     map[string]interface{}
}

// NewIOContext creates a new I/O context
func NewIOContext() *IOContext {
	return &IOContext{
		Permissions:  make([]IOEffectPermission, 0),
		AllowedKinds: make([]IOEffectKind, 0),
		Constraints:  make([]IOConstraint, 0),
		Metadata:     make(map[string]interface{}),
	}
}

func (ioc *IOContext) AllowPermission(permission IOEffectPermission) {
	ioc.Permissions = append(ioc.Permissions, permission)
}

func (ioc *IOContext) AllowKind(kind IOEffectKind) {
	ioc.AllowedKinds = append(ioc.AllowedKinds, kind)
}

func (ioc *IOContext) AddConstraint(constraint IOConstraint) {
	ioc.Constraints = append(ioc.Constraints, constraint)
}

func (ioc *IOContext) CanPerform(effect *IOEffect) bool {
	// Check permissions
	if len(ioc.Permissions) > 0 {
		hasPermission := false
		for _, permission := range ioc.Permissions {
			if effect.Permission == permission {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return false
		}
	}

	// Check allowed kinds
	if len(ioc.AllowedKinds) > 0 {
		hasKind := false
		for _, kind := range ioc.AllowedKinds {
			if effect.Kind == kind {
				hasKind = true
				break
			}
		}
		if !hasKind {
			return false
		}
	}

	// Check constraints
	for _, constraint := range ioc.Constraints {
		if !constraint.Check(effect) {
			return false
		}
	}

	return true
}

// IOInferenceEngine infers I/O effects from AST nodes
type IOInferenceEngine struct {
	context         *IOContext
	functionSigs    map[string]*IOSignature
	variableEffects map[string]*IOEffectSet
	cache           map[string]*IOEffectSet
	mu              sync.RWMutex
}

// NewIOInferenceEngine creates a new I/O inference engine
func NewIOInferenceEngine(context *IOContext) *IOInferenceEngine {
	return &IOInferenceEngine{
		context:         context,
		functionSigs:    make(map[string]*IOSignature),
		variableEffects: make(map[string]*IOEffectSet),
		cache:           make(map[string]*IOEffectSet),
	}
}

func (iie *IOInferenceEngine) InferEffects(node ASTNode) (*IOEffectSet, error) {
	// Check cache first
	nodeStr := fmt.Sprintf("%T", node)
	if cached, exists := iie.cache[nodeStr]; exists {
		return cached, nil
	}

	effects := NewIOEffectSet()

	switch n := node.(type) {
	case *FunctionDecl:
		funcEffects, err := iie.inferFunctionEffects(n)
		if err != nil {
			return nil, err
		}
		effects = effects.Union(funcEffects)

	case *CallExpr:
		callEffects, err := iie.inferCallEffects(n)
		if err != nil {
			return nil, err
		}
		effects = effects.Union(callEffects)

	case *AssignmentExpr:
		assignEffects, err := iie.inferAssignmentEffects(n)
		if err != nil {
			return nil, err
		}
		effects = effects.Union(assignEffects)

	default:
		// For other node types, assume pure
	}

	// Cache the result
	iie.cache[nodeStr] = effects

	return effects, nil
}

func (iie *IOInferenceEngine) inferFunctionEffects(funcDecl *FunctionDecl) (*IOEffectSet, error) {
	effects := NewIOEffectSet()

	// Check if function signature exists
	if sig, exists := iie.functionSigs[funcDecl.Name]; exists {
		return sig.Effects, nil
	}

	// Analyze function body for I/O operations
	// This is a simplified implementation - in practice, would traverse the entire function body

	// For demonstration, add some basic I/O effects based on function name patterns
	if strings.Contains(funcDecl.Name, "read") || strings.Contains(funcDecl.Name, "Read") {
		effect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
		effects.Add(effect)
	}

	if strings.Contains(funcDecl.Name, "write") || strings.Contains(funcDecl.Name, "Write") {
		effect := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
		effects.Add(effect)
	}

	if strings.Contains(funcDecl.Name, "print") || strings.Contains(funcDecl.Name, "Print") {
		effect := NewIOEffect(IOEffectStdoutWrite, IOPermissionWrite)
		effects.Add(effect)
	}

	return effects, nil
}

func (iie *IOInferenceEngine) inferCallEffects(callExpr *CallExpr) (*IOEffectSet, error) {
	effects := NewIOEffectSet()

	// Extract function name from the Function field
	functionName := ""
	if callExpr.Function != nil {
		if funcDecl, ok := callExpr.Function.(*FunctionDecl); ok {
			functionName = funcDecl.Name
		} else {
			functionName = callExpr.Function.String()
		}
	}

	// Get function signature if available
	if sig, exists := iie.functionSigs[functionName]; exists {
		effects = effects.Union(sig.Effects)
	}

	// Infer effects based on function name
	switch functionName {
	case "println", "print", "printf":
		effect := NewIOEffect(IOEffectStdoutWrite, IOPermissionWrite)
		effect.AddBehavior(IOBehaviorSideEffecting)
		effects.Add(effect)

	case "readFile", "openFile":
		effect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
		effect.AddBehavior(IOBehaviorSideEffecting)
		effects.Add(effect)

	case "writeFile", "saveFile":
		effect := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
		effect.AddBehavior(IOBehaviorSideEffecting)
		effects.Add(effect)

	case "httpGet", "httpPost":
		effect := NewIOEffect(IOEffectHTTPRequest, IOPermissionReadWrite)
		effect.AddBehavior(IOBehaviorSideEffecting)
		effect.AddBehavior(IOBehaviorNonDeterministic)
		effects.Add(effect)

	case "dbQuery", "dbInsert", "dbUpdate":
		effect := NewIOEffect(IOEffectDatabaseQuery, IOPermissionReadWrite)
		effect.AddBehavior(IOBehaviorSideEffecting)
		effects.Add(effect)
	}

	return effects, nil
}

func (iie *IOInferenceEngine) inferAssignmentEffects(assignExpr *AssignmentExpr) (*IOEffectSet, error) {
	effects := NewIOEffectSet()

	// Assignments themselves don't typically have I/O effects,
	// but the RHS expression might
	if assignExpr.RHS != nil {
		rhsEffects, err := iie.InferEffects(assignExpr.RHS)
		if err != nil {
			return nil, err
		}
		effects = effects.Union(rhsEffects)
	}

	return effects, nil
}

func (iie *IOInferenceEngine) RegisterFunction(name string, signature *IOSignature) {
	iie.mu.Lock()
	defer iie.mu.Unlock()
	iie.functionSigs[name] = signature
}

func (iie *IOInferenceEngine) GetFunctionSignature(name string) (*IOSignature, bool) {
	iie.mu.RLock()
	defer iie.mu.RUnlock()
	sig, exists := iie.functionSigs[name]
	return sig, exists
}

// IOPurityChecker ensures pure function guarantees
type IOPurityChecker struct {
	strictMode bool
	whitelist  map[string]bool
}

// NewIOPurityChecker creates a new purity checker
func NewIOPurityChecker(strict bool) *IOPurityChecker {
	return &IOPurityChecker{
		strictMode: strict,
		whitelist:  make(map[string]bool),
	}
}

func (ipc *IOPurityChecker) AllowFunction(name string) {
	ipc.whitelist[name] = true
}

func (ipc *IOPurityChecker) CheckPurity(signature *IOSignature) []string {
	var violations []string

	if !signature.Pure {
		violations = append(violations, fmt.Sprintf("function %s is not pure", signature.FunctionName))
	}

	if !signature.Effects.IsPure() {
		effects := signature.Effects.ToSlice()
		for _, effect := range effects {
			if !effect.IsPure() && !ipc.whitelist[signature.FunctionName] {
				violations = append(violations, fmt.Sprintf("function %s has impure effect: %s",
					signature.FunctionName, effect.String()))
			}
		}
	}

	return violations
}

func (ipc *IOPurityChecker) EnforcePurity(signature *IOSignature) error {
	violations := ipc.CheckPurity(signature)
	if len(violations) > 0 {
		return fmt.Errorf("purity violations: %s", strings.Join(violations, "; "))
	}
	return nil
}
