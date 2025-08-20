// Symbol resolution and scope management for the Orizon programming language.
// This package provides comprehensive name resolution capabilities for the HIR.

package resolver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

// SymbolKind represents the kind of symbol.
type SymbolKind int

const (
	SymbolKindVariable SymbolKind = iota
	SymbolKindFunction
	SymbolKindType
	SymbolKindModule
	SymbolKindConstant
	SymbolKindParameter
	SymbolKindField
	SymbolKindMethod
	SymbolKindNamespace
	SymbolKindGeneric
)

// String returns the string representation of SymbolKind.
func (sk SymbolKind) String() string {
	switch sk {
	case SymbolKindVariable:
		return "variable"
	case SymbolKindFunction:
		return "function"
	case SymbolKindType:
		return "type"
	case SymbolKindModule:
		return "module"
	case SymbolKindConstant:
		return "constant"
	case SymbolKindParameter:
		return "parameter"
	case SymbolKindField:
		return "field"
	case SymbolKindMethod:
		return "method"
	case SymbolKindNamespace:
		return "namespace"
	case SymbolKindGeneric:
		return "generic"
	default:
		return "unknown"
	}
}

// Visibility represents symbol visibility.
type Visibility int

const (
	VisibilityPrivate Visibility = iota
	VisibilityPublic
	VisibilityProtected
	VisibilityInternal
	VisibilityReadonly
)

// String returns the string representation of Visibility.
func (v Visibility) String() string {
	switch v {
	case VisibilityPrivate:
		return "private"
	case VisibilityPublic:
		return "public"
	case VisibilityProtected:
		return "protected"
	case VisibilityInternal:
		return "internal"
	case VisibilityReadonly:
		return "readonly"
	default:
		return "unknown"
	}
}

// Symbol represents a named entity in the program.
type Symbol struct {
	HIRNode        hir.HIRNode
	Type           hir.TypeInfo
	Documentation  string
	Name           string
	Dependencies   []string
	TypeParameters []GenericParameter
	DeclSpan       position.Span
	LastUsedSpan   position.Span
	ModuleID       hir.NodeID
	ScopeID        ScopeID
	Visibility     Visibility
	UsageCount     int
	Kind           SymbolKind
	IsMutable      bool
	IsGeneric      bool
	IsExported     bool
	IsDeprecated   bool
}

// GenericParameter represents a generic type parameter.
type GenericParameter struct {
	DefaultType *hir.TypeInfo
	Name        string
	Constraints []hir.TypeInfo
	Span        position.Span
	Variance    Variance
}

// Variance represents generic parameter variance.
type Variance int

const (
	VarianceInvariant Variance = iota
	VarianceCovariant
	VarianceContravariant
)

// ScopeID represents a unique scope identifier.
type ScopeID uint64

// Scope represents a lexical scope.
type Scope struct {
	Symbols         map[string]*Symbol
	ImportedSymbols map[string]*ImportedSymbol
	ParentID        *ScopeID
	Name            string
	Children        []ScopeID
	AccessRules     []AccessRule
	Span            position.Span
	LastAccessed    position.Span
	ID              ScopeID
	Kind            ScopeKind
	ModuleID        hir.NodeID
	Depth           int
	SymbolCount     int
	IsGeneric       bool
}

// ScopeKind represents the kind of scope.
type ScopeKind int

const (
	ScopeKindGlobal ScopeKind = iota
	ScopeKindModule
	ScopeKindFunction
	ScopeKindBlock
	ScopeKindStruct
	ScopeKindInterface
	ScopeKindEnum
	ScopeKindNamespace
	ScopeKindGeneric
	ScopeKindLoop
	ScopeKindConditional
)

// String returns the string representation of ScopeKind.
func (sk ScopeKind) String() string {
	switch sk {
	case ScopeKindGlobal:
		return "global"
	case ScopeKindModule:
		return "module"
	case ScopeKindFunction:
		return "function"
	case ScopeKindBlock:
		return "block"
	case ScopeKindStruct:
		return "struct"
	case ScopeKindInterface:
		return "interface"
	case ScopeKindEnum:
		return "enum"
	case ScopeKindNamespace:
		return "namespace"
	case ScopeKindGeneric:
		return "generic"
	case ScopeKindLoop:
		return "loop"
	case ScopeKindConditional:
		return "conditional"
	default:
		return "unknown"
	}
}

// ImportedSymbol represents a symbol imported from another module.
type ImportedSymbol struct {
	Symbol     *Symbol
	ImportPath string
	Alias      string
	ImportSpan position.Span
	IsWildcard bool
}

// AccessRule represents visibility access rules.
type AccessRule struct {
	Condition  AccessCondition
	Pattern    string
	Visibility Visibility
}

// AccessCondition represents conditions for access rules.
type AccessCondition struct {
	RequiredTrait string
	ModulePattern string
	ContextType   string
}

// SymbolTable manages symbol resolution and scopes.
type SymbolTable struct {
	importGraph    *ImportGraph
	moduleSymbols  map[hir.NodeID]map[string]*Symbol
	scopes         map[ScopeID]*Scope
	imports        map[string]*ImportInfo
	symbolCache    map[string][]*Symbol
	quickLookup    map[string]*Symbol
	errors         []ResolutionError
	warnings       []ResolutionWarning
	scopeCounter   ScopeID
	rootScopeID    ScopeID
	currentScope   ScopeID
	totalSymbols   int
	lookupCount    int
	cacheHitCount  int
	strictMode     bool
	allowShadowing bool
	caseSensitive  bool
}

// ImportInfo represents module import information.
type ImportInfo struct {
	ImportedSymbols map[string]string
	ModulePath      string
	Alias           string
	ImportSpan      position.Span
	ModuleID        hir.NodeID
	IsWildcard      bool
}

// ImportGraph represents the module dependency graph.
type ImportGraph struct {
	nodes    map[string]*ImportNode
	edges    map[string][]string
	hasCycle bool
}

// ImportNode represents a node in the import graph.
type ImportNode struct {
	ModulePath   string
	Dependencies []string
	Dependents   []string
	IsProcessed  bool
}

// ResolutionError represents a symbol resolution error.
type ResolutionError struct {
	Message string
	Symbol  string
	Related []RelatedInformation
	Span    position.Span
	Kind    ResolutionErrorKind
}

// ResolutionErrorKind represents the kind of resolution error.
type ResolutionErrorKind int

const (
	ErrorKindUndefinedSymbol ResolutionErrorKind = iota
	ErrorKindDuplicateSymbol
	ErrorKindCircularImport
	ErrorKindVisibilityViolation
	ErrorKindTypeConflict
	ErrorKindScopeViolation
	ErrorKindGenericConstraintViolation
	ErrorKindModuleNotFound
	ErrorKindAmbiguousSymbol
	ErrorKindInvalidImport
)

// ResolutionWarning represents a symbol resolution warning.
type ResolutionWarning struct {
	Message string
	Symbol  string
	Span    position.Span
	Kind    ResolutionWarningKind
}

// ResolutionWarningKind represents the kind of resolution warning.
type ResolutionWarningKind int

const (
	WarningKindUnusedSymbol ResolutionWarningKind = iota
	WarningKindShadowedSymbol
	WarningKindDeprecatedSymbol
	WarningKindRedundantImport
	WarningKindPotentiallyUnused
	WarningKindNamingConvention
)

// RelatedInformation provides additional context for errors.
type RelatedInformation struct {
	Message string
	Span    position.Span
}

// NewSymbolTable creates a new symbol table.
func NewSymbolTable() *SymbolTable {
	st := &SymbolTable{
		scopes:         make(map[ScopeID]*Scope),
		symbolCache:    make(map[string][]*Symbol),
		quickLookup:    make(map[string]*Symbol),
		moduleSymbols:  make(map[hir.NodeID]map[string]*Symbol),
		imports:        make(map[string]*ImportInfo),
		importGraph:    NewImportGraph(),
		errors:         []ResolutionError{},
		warnings:       []ResolutionWarning{},
		strictMode:     true,
		allowShadowing: false,
		caseSensitive:  true,
		scopeCounter:   0, // Start from 0 so first scope has ID 1
	}

	// Create global scope.
	st.rootScopeID = st.createScope(ScopeKindGlobal, "global", nil, position.Span{})
	st.currentScope = st.rootScopeID

	return st
}

// NewImportGraph creates a new import graph.
func NewImportGraph() *ImportGraph {
	return &ImportGraph{
		nodes: make(map[string]*ImportNode),
		edges: make(map[string][]string),
	}
}

// CreateScope creates a new scope.
func (st *SymbolTable) CreateScope(kind ScopeKind, name string, span position.Span) ScopeID {
	currentScopeID := st.currentScope

	return st.createScope(kind, name, &currentScopeID, span)
}

// createScope is the internal scope creation method.
func (st *SymbolTable) createScope(kind ScopeKind, name string, parentID *ScopeID, span position.Span) ScopeID {
	st.scopeCounter++
	scopeID := st.scopeCounter

	scope := &Scope{
		ID:              scopeID,
		Kind:            kind,
		Name:            name,
		ParentID:        parentID,
		Span:            span,
		Symbols:         make(map[string]*Symbol),
		Children:        []ScopeID{},
		ImportedSymbols: make(map[string]*ImportedSymbol),
		AccessRules:     []AccessRule{},
	}

	// Set depth based on parent.
	if parentID != nil {
		if parent, exists := st.scopes[*parentID]; exists {
			scope.Depth = parent.Depth + 1
			parent.Children = append(parent.Children, scopeID)
		}
	}

	st.scopes[scopeID] = scope

	return scopeID
}

// EnterScope changes the current scope.
func (st *SymbolTable) EnterScope(scopeID ScopeID) error {
	if _, exists := st.scopes[scopeID]; !exists {
		return fmt.Errorf("scope %d does not exist", scopeID)
	}

	st.currentScope = scopeID
	// Clear caches when entering new scope.
	st.quickLookup = make(map[string]*Symbol)

	return nil
}

// ExitScope returns to the parent scope.
func (st *SymbolTable) ExitScope() error {
	currentScope := st.scopes[st.currentScope]
	if currentScope.ParentID == nil {
		return fmt.Errorf("cannot exit root scope")
	}

	st.currentScope = *currentScope.ParentID
	// Clear caches when exiting scope.
	st.quickLookup = make(map[string]*Symbol)

	return nil
}

// DefineSymbol adds a new symbol to the current scope.
func (st *SymbolTable) DefineSymbol(symbol *Symbol) error {
	currentScope := st.scopes[st.currentScope]

	// Check for duplicate symbols.
	if existing, exists := currentScope.Symbols[symbol.Name]; exists {
		if !st.allowShadowing {
			return st.createDuplicateSymbolError(symbol, existing)
		}
	}

	// Set scope information.
	symbol.ScopeID = st.currentScope
	symbol.ModuleID = currentScope.ModuleID

	// Add to scope.
	currentScope.Symbols[symbol.Name] = symbol
	currentScope.SymbolCount++
	st.totalSymbols++

	// Update caches.
	st.invalidateCache(symbol.Name)

	// Add to module symbols.
	if st.moduleSymbols[symbol.ModuleID] == nil {
		st.moduleSymbols[symbol.ModuleID] = make(map[string]*Symbol)
	}

	st.moduleSymbols[symbol.ModuleID][symbol.Name] = symbol

	return nil
}

// LookupSymbol searches for a symbol by name.
func (st *SymbolTable) LookupSymbol(name string) (*Symbol, error) {
	st.lookupCount++

	// Check quick lookup cache first.
	if symbol, exists := st.quickLookup[name]; exists {
		st.cacheHitCount++

		return symbol, nil
	}

	// Search from current scope upward.
	scopeID := st.currentScope

	for {
		scope := st.scopes[scopeID]

		// Check local symbols.
		if symbol, exists := scope.Symbols[name]; exists {
			st.quickLookup[name] = symbol

			return symbol, nil
		}

		// Check imported symbols.
		if imported, exists := scope.ImportedSymbols[name]; exists {
			st.quickLookup[name] = imported.Symbol

			return imported.Symbol, nil
		}

		// Move to parent scope.
		if scope.ParentID == nil {
			break
		}

		scopeID = *scope.ParentID
	}

	return nil, st.createUndefinedSymbolError(name, st.getCurrentSpan())
}

// LookupSymbolInScope searches for a symbol in a specific scope.
func (st *SymbolTable) LookupSymbolInScope(name string, scopeID ScopeID) (*Symbol, error) {
	scope, exists := st.scopes[scopeID]
	if !exists {
		return nil, fmt.Errorf("scope %d does not exist", scopeID)
	}

	if symbol, exists := scope.Symbols[name]; exists {
		return symbol, nil
	}

	return nil, st.createUndefinedSymbolError(name, scope.Span)
}

// GetCurrentScope returns the current scope ID.
func (st *SymbolTable) GetCurrentScope() ScopeID {
	return st.currentScope
}

// GetScope returns a scope by ID.
func (st *SymbolTable) GetScope(scopeID ScopeID) (*Scope, error) {
	scope, exists := st.scopes[scopeID]
	if !exists {
		return nil, fmt.Errorf("scope %d does not exist", scopeID)
	}

	return scope, nil
}

// GetScopePath returns the path from root to the given scope.
func (st *SymbolTable) GetScopePath(scopeID ScopeID) ([]ScopeID, error) {
	path := []ScopeID{}
	current := scopeID

	for {
		scope, exists := st.scopes[current]
		if !exists {
			return nil, fmt.Errorf("scope %d does not exist", current)
		}

		path = append([]ScopeID{current}, path...)

		if scope.ParentID == nil {
			break
		}

		current = *scope.ParentID
	}

	return path, nil
}

// AddImport adds an import to the current scope.
func (st *SymbolTable) AddImport(importInfo *ImportInfo) error {
	currentScope := st.scopes[st.currentScope]

	// Add to import graph.
	st.importGraph.AddImport(importInfo.ModulePath, currentScope.Name)

	// Check for circular imports.
	if st.importGraph.HasCycle() {
		return st.createCircularImportError(importInfo)
	}

	// Store import info.
	st.imports[importInfo.ModulePath] = importInfo

	return nil
}

// ResolveHIRProgram performs symbol resolution on an HIR program.
func (st *SymbolTable) ResolveHIRProgram(program *hir.HIRProgram) error {
	resolver := NewResolver(st)

	return resolver.ResolveProgram(program)
}

// GetErrors returns all resolution errors.
func (st *SymbolTable) GetErrors() []ResolutionError {
	return st.errors
}

// GetWarnings returns all resolution warnings.
func (st *SymbolTable) GetWarnings() []ResolutionWarning {
	return st.warnings
}

// ClearErrors clears all errors and warnings.
func (st *SymbolTable) ClearErrors() {
	st.errors = []ResolutionError{}
	st.warnings = []ResolutionWarning{}
}

// GetStatistics returns symbol table statistics.
func (st *SymbolTable) GetStatistics() SymbolTableStatistics {
	return SymbolTableStatistics{
		TotalSymbols:  st.totalSymbols,
		TotalScopes:   len(st.scopes),
		LookupCount:   st.lookupCount,
		CacheHitCount: st.cacheHitCount,
		CacheHitRatio: float64(st.cacheHitCount) / float64(st.lookupCount),
		ErrorCount:    len(st.errors),
		WarningCount:  len(st.warnings),
	}
}

// SymbolTableStatistics contains symbol table performance statistics.
type SymbolTableStatistics struct {
	TotalSymbols  int
	TotalScopes   int
	LookupCount   int
	CacheHitCount int
	CacheHitRatio float64
	ErrorCount    int
	WarningCount  int
}

// Helper methods for error creation.
func (st *SymbolTable) createDuplicateSymbolError(new, existing *Symbol) error {
	err := ResolutionError{
		Kind:    ErrorKindDuplicateSymbol,
		Message: fmt.Sprintf("symbol '%s' is already defined", new.Name),
		Span:    new.DeclSpan,
		Symbol:  new.Name,
		Related: []RelatedInformation{
			{
				Span:    existing.DeclSpan,
				Message: "previous definition here",
			},
		},
	}
	st.errors = append(st.errors, err)

	return errors.New(err.Message)
}

func (st *SymbolTable) createUndefinedSymbolError(name string, span position.Span) error {
	err := ResolutionError{
		Kind:    ErrorKindUndefinedSymbol,
		Message: fmt.Sprintf("undefined symbol '%s'", name),
		Span:    span,
		Symbol:  name,
	}
	st.errors = append(st.errors, err)

	return errors.New(err.Message)
}

func (st *SymbolTable) createCircularImportError(importInfo *ImportInfo) error {
	err := ResolutionError{
		Kind:    ErrorKindCircularImport,
		Message: fmt.Sprintf("circular import detected: %s", importInfo.ModulePath),
		Span:    importInfo.ImportSpan,
		Symbol:  importInfo.ModulePath,
	}
	st.errors = append(st.errors, err)

	return errors.New(err.Message)
}

// Helper methods for cache management.
func (st *SymbolTable) invalidateCache(symbolName string) {
	delete(st.quickLookup, symbolName)
	delete(st.symbolCache, symbolName)
}

func (st *SymbolTable) getCurrentSpan() position.Span {
	if scope, exists := st.scopes[st.currentScope]; exists {
		return scope.LastAccessed
	}

	return position.Span{}
}

// Import graph methods.
func (ig *ImportGraph) AddImport(from, to string) {
	if ig.nodes[from] == nil {
		ig.nodes[from] = &ImportNode{
			ModulePath:   from,
			Dependencies: []string{},
			Dependents:   []string{},
		}
	}

	if ig.nodes[to] == nil {
		ig.nodes[to] = &ImportNode{
			ModulePath:   to,
			Dependencies: []string{},
			Dependents:   []string{},
		}
	}

	ig.edges[from] = append(ig.edges[from], to)
	ig.nodes[from].Dependencies = append(ig.nodes[from].Dependencies, to)
	ig.nodes[to].Dependents = append(ig.nodes[to].Dependents, from)
}

func (ig *ImportGraph) HasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range ig.nodes {
		if !visited[node] {
			if ig.detectCycleDFS(node, visited, recStack) {
				ig.hasCycle = true

				return true
			}
		}
	}

	ig.hasCycle = false

	return false
}

func (ig *ImportGraph) detectCycleDFS(node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range ig.edges[node] {
		if !visited[neighbor] {
			if ig.detectCycleDFS(neighbor, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false

	return false
}

// String methods for debugging.
func (s *Symbol) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Symbol{%s", s.Name))
	parts = append(parts, fmt.Sprintf("kind=%s", s.Kind))
	parts = append(parts, fmt.Sprintf("type=%s", s.Type.Name))
	parts = append(parts, fmt.Sprintf("visibility=%s", s.Visibility))

	if s.IsMutable {
		parts = append(parts, "mutable")
	}

	if s.IsGeneric {
		parts = append(parts, "generic")
	}

	if s.IsExported {
		parts = append(parts, "exported")
	}

	return strings.Join(parts, ", ") + "}"
}

func (s *Scope) String() string {
	return fmt.Sprintf("Scope{%s, kind=%s, symbols=%d, depth=%d}",
		s.Name, s.Kind, s.SymbolCount, s.Depth)
}
