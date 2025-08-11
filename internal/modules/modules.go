// Package modules provides the module system for the Orizon programming language.
//
// This package implements:
// - Module resolution algorithms for dependency management
// - Visibility control for exports and imports
// - Circular dependency detection and prevention
// - Module caching and optimization
//
// The module system integrates with the HIR (High-level Intermediate Representation)
// and symbol resolution system to provide a complete module management solution.
package modules

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

// Global counters for ID generation
var nextModuleID uint64 = 1
var nextNodeID uint64 = 1

// generateModuleID generates a unique module ID
func generateModuleID() hir.ModuleID {
	return hir.ModuleID(atomic.AddUint64(&nextModuleID, 1))
}

// generateNodeID generates a unique node ID
func generateNodeID() hir.NodeID {
	return hir.NodeID(atomic.AddUint64(&nextNodeID, 1))
}

// ModulePath represents a canonical module path
type ModulePath string

// Version represents a semantic version
type Version struct {
	Major         int
	Minor         int
	Patch         int
	PreRelease    string
	BuildMetadata string
}

// String returns the string representation of the version
func (v Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		base += "-" + v.PreRelease
	}
	if v.BuildMetadata != "" {
		base += "+" + v.BuildMetadata
	}
	return base
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	// For simplicity, ignore pre-release comparison for now
	return 0
}

// ModuleSpec specifies a module with version constraints
type ModuleSpec struct {
	Path              ModulePath
	Version           Version
	VersionConstraint string // e.g., ">=1.0.0,<2.0.0"
}

// Module represents a loaded module
type Module struct {
	ID       hir.ModuleID
	Path     ModulePath
	Version  Version
	Name     string
	FilePath string

	// Module metadata
	Description string
	Author      string
	License     string
	Keywords    []string

	// Dependencies
	Dependencies    []ModuleSpec
	DevDependencies []ModuleSpec

	// Module contents
	HIRModule   *hir.HIRModule
	SourceFiles []string

	// Export information
	PublicSymbols map[string]*ExportedSymbol
	ModuleSymbols map[string]*Symbol

	// Import information
	ImportedModules map[ModulePath]*Module
	ImportedSymbols map[string]*ImportedSymbol

	// Status information
	LoadStatus ModuleLoadStatus
	LoadError  error

	// Build information
	Checksum  string
	BuildTime int64
	BuildHash string

	// Span information
	Span position.Span
}

// ModuleLoadStatus represents the current load status of a module
type ModuleLoadStatus int

const (
	ModuleStatusUnloaded ModuleLoadStatus = iota
	ModuleStatusLoading
	ModuleStatusLoaded
	ModuleStatusError
	ModuleStatusCached
)

// String returns the string representation of the load status
func (s ModuleLoadStatus) String() string {
	switch s {
	case ModuleStatusUnloaded:
		return "unloaded"
	case ModuleStatusLoading:
		return "loading"
	case ModuleStatusLoaded:
		return "loaded"
	case ModuleStatusError:
		return "error"
	case ModuleStatusCached:
		return "cached"
	default:
		return "unknown"
	}
}

// ExportedSymbol represents a symbol exported by a module
type ExportedSymbol struct {
	Name          string
	Type          SymbolType
	Module        ModulePath
	Visibility    VisibilityLevel
	Signature     string
	Documentation string
	Span          position.Span
}

// Symbol represents any symbol in a module
type Symbol struct {
	Name       string
	Type       SymbolType
	Definition string
	Module     ModulePath
	IsExported bool
	Span       position.Span
}

// ImportedSymbol represents a symbol imported from another module
type ImportedSymbol struct {
	Name         string
	OriginalName string
	SourceModule ModulePath
	Type         SymbolType
	Alias        string
	Span         position.Span
}

// SymbolType represents the type of a symbol
type SymbolType int

const (
	SymbolTypeFunction SymbolType = iota
	SymbolTypeVariable
	SymbolTypeType
	SymbolTypeConstant
	SymbolTypeClass
	SymbolTypeInterface
	SymbolTypeEnum
	SymbolTypeModule
	SymbolTypeNamespace
)

// String returns the string representation of the symbol type
func (s SymbolType) String() string {
	switch s {
	case SymbolTypeFunction:
		return "function"
	case SymbolTypeVariable:
		return "variable"
	case SymbolTypeType:
		return "type"
	case SymbolTypeConstant:
		return "constant"
	case SymbolTypeClass:
		return "class"
	case SymbolTypeInterface:
		return "interface"
	case SymbolTypeEnum:
		return "enum"
	case SymbolTypeModule:
		return "module"
	case SymbolTypeNamespace:
		return "namespace"
	default:
		return "unknown"
	}
}

// VisibilityLevel represents the visibility level of a symbol
type VisibilityLevel int

const (
	VisibilityPrivate VisibilityLevel = iota
	VisibilityPackage
	VisibilityProtected
	VisibilityPublic
)

// String returns the string representation of the visibility level
func (v VisibilityLevel) String() string {
	switch v {
	case VisibilityPrivate:
		return "private"
	case VisibilityPackage:
		return "package"
	case VisibilityProtected:
		return "protected"
	case VisibilityPublic:
		return "public"
	default:
		return "unknown"
	}
}

// DependencyGraph represents the dependency relationships between modules
type DependencyGraph struct {
	Modules      map[ModulePath]*Module
	Dependencies map[ModulePath][]ModulePath
	Reverse      map[ModulePath][]ModulePath
	LoadOrder    []ModulePath
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Modules:      make(map[ModulePath]*Module),
		Dependencies: make(map[ModulePath][]ModulePath),
		Reverse:      make(map[ModulePath][]ModulePath),
		LoadOrder:    []ModulePath{},
	}
}

// AddModule adds a module to the dependency graph
func (dg *DependencyGraph) AddModule(module *Module) {
	dg.Modules[module.Path] = module

	// Initialize dependency lists
	if dg.Dependencies[module.Path] == nil {
		dg.Dependencies[module.Path] = []ModulePath{}
	}
	if dg.Reverse[module.Path] == nil {
		dg.Reverse[module.Path] = []ModulePath{}
	}
}

// AddDependency adds a dependency relationship
func (dg *DependencyGraph) AddDependency(from, to ModulePath) {
	if dg.Dependencies[from] == nil {
		dg.Dependencies[from] = []ModulePath{}
	}
	if dg.Reverse[to] == nil {
		dg.Reverse[to] = []ModulePath{}
	}

	// Add dependency
	dg.Dependencies[from] = append(dg.Dependencies[from], to)
	dg.Reverse[to] = append(dg.Reverse[to], from)
}

// DetectCycles detects circular dependencies in the graph
func (dg *DependencyGraph) DetectCycles() ([][]ModulePath, error) {
	var cycles [][]ModulePath
	visited := make(map[ModulePath]bool)
	recursionStack := make(map[ModulePath]bool)
	path := []ModulePath{}

	for modulePath := range dg.Modules {
		if !visited[modulePath] {
			if cycle := dg.detectCyclesDFS(modulePath, visited, recursionStack, path); cycle != nil {
				cycles = append(cycles, cycle)
			}
		}
	}

	if len(cycles) > 0 {
		return cycles, fmt.Errorf("circular dependencies detected: %d cycles found", len(cycles))
	}

	return nil, nil
}

// detectCyclesDFS performs DFS to detect cycles
func (dg *DependencyGraph) detectCyclesDFS(module ModulePath, visited, recursionStack map[ModulePath]bool, path []ModulePath) []ModulePath {
	visited[module] = true
	recursionStack[module] = true
	path = append(path, module)

	for _, dependency := range dg.Dependencies[module] {
		if !visited[dependency] {
			if cycle := dg.detectCyclesDFS(dependency, visited, recursionStack, path); cycle != nil {
				return cycle
			}
		} else if recursionStack[dependency] {
			// Found a cycle - return the cycle path
			cycleStart := -1
			for i, p := range path {
				if p == dependency {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]ModulePath, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				cycle = append(cycle, dependency) // Close the cycle
				return cycle
			}
		}
	}

	recursionStack[module] = false
	return nil
}

// TopologicalSort returns the modules in dependency order
func (dg *DependencyGraph) TopologicalSort() ([]ModulePath, error) {
	// First check for cycles
	cycles, err := dg.DetectCycles()
	if err != nil {
		return nil, fmt.Errorf("cannot perform topological sort: %v", err)
	}
	if len(cycles) > 0 {
		return nil, fmt.Errorf("circular dependencies prevent topological sorting")
	}

	// Kahn's algorithm for topological sorting
	inDegree := make(map[ModulePath]int)

	// Initialize in-degree counts
	for module := range dg.Modules {
		inDegree[module] = 0
	}

	// Calculate in-degrees
	for _, dependencies := range dg.Dependencies {
		for _, dep := range dependencies {
			inDegree[dep]++
		}
	}

	// Find modules with no incoming edges
	queue := []ModulePath{}
	for module, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, module)
		}
	}

	var result []ModulePath

	for len(queue) > 0 {
		// Dequeue a module
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each dependency of current module
		for _, dependency := range dg.Dependencies[current] {
			inDegree[dependency]--
			if inDegree[dependency] == 0 {
				queue = append(queue, dependency)
			}
		}
	}

	// Check if all modules were processed
	if len(result) != len(dg.Modules) {
		return nil, fmt.Errorf("topological sort failed: graph contains cycles")
	}

	// Reverse the result since we want dependencies to come first
	reversed := make([]ModulePath, len(result))
	for i, module := range result {
		reversed[len(result)-1-i] = module
	}

	dg.LoadOrder = reversed
	return reversed, nil
}

// GetDependencies returns the direct dependencies of a module
func (dg *DependencyGraph) GetDependencies(module ModulePath) []ModulePath {
	deps := dg.Dependencies[module]
	if deps == nil {
		return []ModulePath{}
	}
	result := make([]ModulePath, len(deps))
	copy(result, deps)
	return result
}

// GetDependents returns the modules that depend on the given module
func (dg *DependencyGraph) GetDependents(module ModulePath) []ModulePath {
	deps := dg.Reverse[module]
	if deps == nil {
		return []ModulePath{}
	}
	result := make([]ModulePath, len(deps))
	copy(result, deps)
	return result
}

// GetTransitiveDependencies returns all transitive dependencies of a module
func (dg *DependencyGraph) GetTransitiveDependencies(module ModulePath) ([]ModulePath, error) {
	visited := make(map[ModulePath]bool)
	var result []ModulePath

	var visit func(ModulePath) error
	visit = func(m ModulePath) error {
		if visited[m] {
			return nil
		}
		visited[m] = true

		for _, dep := range dg.Dependencies[m] {
			if err := visit(dep); err != nil {
				return err
			}
			if !contains(result, dep) {
				result = append(result, dep)
			}
		}
		return nil
	}

	if err := visit(module); err != nil {
		return nil, err
	}

	return result, nil
}

// contains checks if a slice contains a value
func contains(slice []ModulePath, value ModulePath) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// ModuleLoader handles loading and resolving modules
type ModuleLoader struct {
	Cache       map[ModulePath]*Module
	SearchPaths []string
	Registry    *ModuleRegistry
	Graph       *DependencyGraph
}

// NewModuleLoader creates a new module loader
func NewModuleLoader() *ModuleLoader {
	return &ModuleLoader{
		Cache:       make(map[ModulePath]*Module),
		SearchPaths: []string{},
		Registry:    NewModuleRegistry(),
		Graph:       NewDependencyGraph(),
	}
}

// AddSearchPath adds a search path for modules
func (ml *ModuleLoader) AddSearchPath(path string) {
	ml.SearchPaths = append(ml.SearchPaths, path)
}

// LoadModule loads a module and its dependencies
func (ml *ModuleLoader) LoadModule(path ModulePath) (*Module, error) {
	// Check cache first
	if module, exists := ml.Cache[path]; exists {
		if module.LoadStatus == ModuleStatusLoaded {
			return module, nil
		}
		if module.LoadStatus == ModuleStatusError {
			return nil, module.LoadError
		}
	}

	// Create or get module
	module := ml.getOrCreateModule(path)
	module.LoadStatus = ModuleStatusLoading

	// Find module file
	filePath, err := ml.findModuleFile(path)
	if err != nil {
		module.LoadStatus = ModuleStatusError
		module.LoadError = err
		return nil, err
	}

	module.FilePath = filePath

	// Load module content (simplified - in real implementation would parse source)
	if err := ml.loadModuleContent(module); err != nil {
		module.LoadStatus = ModuleStatusError
		module.LoadError = err
		return nil, err
	}

	// Load dependencies
	for _, dep := range module.Dependencies {
		depModule, err := ml.LoadModule(dep.Path)
		if err != nil {
			module.LoadStatus = ModuleStatusError
			module.LoadError = fmt.Errorf("failed to load dependency %s: %v", dep.Path, err)
			return nil, module.LoadError
		}

		module.ImportedModules[dep.Path] = depModule
		ml.Graph.AddDependency(path, dep.Path)
	}

	module.LoadStatus = ModuleStatusLoaded
	ml.Cache[path] = module
	ml.Graph.AddModule(module)

	return module, nil
}

// getOrCreateModule gets an existing module or creates a new one
func (ml *ModuleLoader) getOrCreateModule(path ModulePath) *Module {
	if module, exists := ml.Cache[path]; exists {
		return module
	}

	module := &Module{
		Path:            path,
		Name:            string(path),
		Dependencies:    []ModuleSpec{},
		DevDependencies: []ModuleSpec{},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ModuleSymbols:   make(map[string]*Symbol),
		ImportedModules: make(map[ModulePath]*Module),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		LoadStatus:      ModuleStatusUnloaded,
	}

	ml.Cache[path] = module
	return module
}

// findModuleFile finds the file for a module
func (ml *ModuleLoader) findModuleFile(path ModulePath) (string, error) {
	possibleNames := []string{
		string(path) + ".oriz",
		filepath.Join(string(path), "mod.oriz"),
		filepath.Join(string(path), "index.oriz"),
	}

	for _, searchPath := range ml.SearchPaths {
		for _, name := range possibleNames {
			fullPath := filepath.Join(searchPath, name)
			if fileExists(fullPath) {
				return fullPath, nil
			}
		}
	}

	return "", fmt.Errorf("module file not found for path: %s", path)
}

// loadModuleContent loads the content of a module (simplified)
func (ml *ModuleLoader) loadModuleContent(module *Module) error {
	// In a real implementation, this would:
	// 1. Parse the source file
	// 2. Extract dependencies from import statements
	// 3. Build HIR representation
	// 4. Extract exported symbols

	// For now, we'll create a minimal HIR module
	module.HIRModule = &hir.HIRModule{
		ID:           generateNodeID(),
		ModuleID:     generateModuleID(),
		Name:         module.Name,
		Declarations: []hir.HIRDeclaration{},
		Exports:      []string{},
		Imports:      []hir.ImportInfo{},
		Span:         module.Span,
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	// In a real implementation, this would check the filesystem
	// For now, we'll return false to avoid file system dependencies
	return false
}

// ResolveModules resolves all modules and their dependencies
func (ml *ModuleLoader) ResolveModules(entryPoints []ModulePath) error {
	// Load all entry point modules
	for _, entryPoint := range entryPoints {
		if _, err := ml.LoadModule(entryPoint); err != nil {
			return fmt.Errorf("failed to load entry point module %s: %v", entryPoint, err)
		}
	}

	// Check for circular dependencies
	cycles, err := ml.Graph.DetectCycles()
	if err != nil {
		return err
	}
	if len(cycles) > 0 {
		return ml.formatCycleError(cycles)
	}

	// Compute load order
	loadOrder, err := ml.Graph.TopologicalSort()
	if err != nil {
		return err
	}

	ml.Graph.LoadOrder = loadOrder
	return nil
}

// formatCycleError formats a circular dependency error
func (ml *ModuleLoader) formatCycleError(cycles [][]ModulePath) error {
	var parts []string
	parts = append(parts, fmt.Sprintf("Circular dependencies detected (%d cycles):", len(cycles)))

	for i, cycle := range cycles {
		parts = append(parts, fmt.Sprintf("  Cycle %d: %s", i+1, ml.formatCycle(cycle)))
	}

	return fmt.Errorf(strings.Join(parts, "\n"))
}

// formatCycle formats a single dependency cycle
func (ml *ModuleLoader) formatCycle(cycle []ModulePath) string {
	if len(cycle) == 0 {
		return ""
	}

	var parts []string
	for _, module := range cycle {
		parts = append(parts, string(module))
	}

	return strings.Join(parts, " -> ")
}

// GetLoadOrder returns the modules in dependency order
func (ml *ModuleLoader) GetLoadOrder() []ModulePath {
	return ml.Graph.LoadOrder
}

// GetModule returns a loaded module
func (ml *ModuleLoader) GetModule(path ModulePath) *Module {
	return ml.Cache[path]
}

// GetAllModules returns all loaded modules
func (ml *ModuleLoader) GetAllModules() map[ModulePath]*Module {
	result := make(map[ModulePath]*Module)
	for path, module := range ml.Cache {
		result[path] = module
	}
	return result
}

// ModuleRegistry manages module metadata and versions
type ModuleRegistry struct {
	Modules map[ModulePath]*ModuleMetadata
}

// ModuleMetadata contains metadata about a module
type ModuleMetadata struct {
	Path          ModulePath
	Name          string
	Description   string
	Author        string
	License       string
	Keywords      []string
	Versions      []Version
	LatestVersion Version
	Dependencies  map[Version][]ModuleSpec
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		Modules: make(map[ModulePath]*ModuleMetadata),
	}
}

// RegisterModule registers a module in the registry
func (mr *ModuleRegistry) RegisterModule(metadata *ModuleMetadata) {
	mr.Modules[metadata.Path] = metadata
}

// GetModuleMetadata returns metadata for a module
func (mr *ModuleRegistry) GetModuleMetadata(path ModulePath) (*ModuleMetadata, bool) {
	metadata, exists := mr.Modules[path]
	return metadata, exists
}

// GetLatestVersion returns the latest version of a module
func (mr *ModuleRegistry) GetLatestVersion(path ModulePath) (Version, error) {
	metadata, exists := mr.Modules[path]
	if !exists {
		return Version{}, fmt.Errorf("module not found: %s", path)
	}
	return metadata.LatestVersion, nil
}

// GetAvailableVersions returns all available versions of a module
func (mr *ModuleRegistry) GetAvailableVersions(path ModulePath) ([]Version, error) {
	metadata, exists := mr.Modules[path]
	if !exists {
		return nil, fmt.Errorf("module not found: %s", path)
	}

	versions := make([]Version, len(metadata.Versions))
	copy(versions, metadata.Versions)

	// Sort versions (descending)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Compare(versions[j]) > 0
	})

	return versions, nil
}

// Statistics provides statistics about the module system
type Statistics struct {
	TotalModules      int
	LoadedModules     int
	CachedModules     int
	FailedModules     int
	DependencyEdges   int
	CycleCount        int
	MaxDepthLevel     int
	AverageDepthLevel float64
}

// GetStatistics returns statistics about the module system
func (ml *ModuleLoader) GetStatistics() Statistics {
	stats := Statistics{
		TotalModules: len(ml.Cache),
	}

	var depthSum int
	maxDepth := 0

	for _, module := range ml.Cache {
		switch module.LoadStatus {
		case ModuleStatusLoaded:
			stats.LoadedModules++
		case ModuleStatusCached:
			stats.CachedModules++
		case ModuleStatusError:
			stats.FailedModules++
		}

		// Calculate depth (simplified)
		depth := len(module.Dependencies)
		depthSum += depth
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	stats.MaxDepthLevel = maxDepth
	if stats.TotalModules > 0 {
		stats.AverageDepthLevel = float64(depthSum) / float64(stats.TotalModules)
	}

	// Count dependency edges
	for _, deps := range ml.Graph.Dependencies {
		stats.DependencyEdges += len(deps)
	}

	// Count cycles
	cycles, _ := ml.Graph.DetectCycles()
	stats.CycleCount = len(cycles)

	return stats
}
