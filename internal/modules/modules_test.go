package modules

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/position"
)

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		v1       Version
		v2       Version
		expected int
	}{
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, 0},
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, Version{PreRelease: "", BuildMetadata: "", Major: 2, Minor: 0, Patch: 0}, -1},
		{Version{PreRelease: "", BuildMetadata: "", Major: 2, Minor: 0, Patch: 0}, Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, 1},
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 1, Patch: 0}, Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, 1},
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 1}, Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, 1},
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 2, Patch: 3}, Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 2, Patch: 4}, -1},
	}

	for i, test := range tests {
		result := test.v1.Compare(test.v2)
		if result != test.expected {
			t.Errorf("Test %d: expected %d, got %d for %s vs %s",
				i, test.expected, result, test.v1.String(), test.v2.String())
		}
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		version  Version
		expected string
	}{
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, "1.0.0"},
		{Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 2, Patch: 3}, "1.2.3"},
		{Version{PreRelease: "alpha", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}, "1.0.0-alpha"},
		{Version{PreRelease: "", BuildMetadata: "build123", Major: 1, Minor: 0, Patch: 0}, "1.0.0+build123"},
		{Version{PreRelease: "beta", BuildMetadata: "build456", Major: 1, Minor: 0, Patch: 0}, "1.0.0-beta+build456"},
	}

	for i, test := range tests {
		result := test.version.String()
		if result != test.expected {
			t.Errorf("Test %d: expected %s, got %s", i, test.expected, result)
		}
	}
}

func TestModuleCreation(t *testing.T) {
	path := ModulePath("test/module")
	version := Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0}

	module := &Module{
		Path:            path,
		Version:         version,
		Name:            "testmodule",
		Dependencies:    []ModuleSpec{},
		DevDependencies: []ModuleSpec{},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ModuleSymbols:   make(map[string]*Symbol),
		ImportedModules: make(map[ModulePath]*Module),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		LoadStatus:      ModuleStatusUnloaded,
	}

	if module.Path != path {
		t.Errorf("Expected path %s, got %s", path, module.Path)
	}

	if module.Version.Compare(version) != 0 {
		t.Errorf("Expected version %s, got %s", version.String(), module.Version.String())
	}

	if module.LoadStatus != ModuleStatusUnloaded {
		t.Errorf("Expected status %s, got %s",
			ModuleStatusUnloaded.String(), module.LoadStatus.String())
	}
}

func TestDependencyGraphCreation(t *testing.T) {
	graph := NewDependencyGraph()

	if graph == nil {
		t.Fatal("Failed to create dependency graph")
	}

	if len(graph.Modules) != 0 {
		t.Errorf("Expected empty modules map, got %d modules", len(graph.Modules))
	}

	if len(graph.Dependencies) != 0 {
		t.Errorf("Expected empty dependencies map, got %d entries", len(graph.Dependencies))
	}

	if len(graph.LoadOrder) != 0 {
		t.Errorf("Expected empty load order, got %d entries", len(graph.LoadOrder))
	}
}

func TestDependencyGraphAddModule(t *testing.T) {
	graph := NewDependencyGraph()

	module := &Module{
		Path:            ModulePath("test/module"),
		Version:         Version{PreRelease: "", BuildMetadata: "", Major: 1, Minor: 0, Patch: 0},
		Name:            "testmodule",
		Dependencies:    []ModuleSpec{},
		DevDependencies: []ModuleSpec{},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ModuleSymbols:   make(map[string]*Symbol),
		ImportedModules: make(map[ModulePath]*Module),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		LoadStatus:      ModuleStatusUnloaded,
	}

	graph.AddModule(module)

	if len(graph.Modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(graph.Modules))
	}

	if graph.Modules[module.Path] != module {
		t.Error("Module not properly added to graph")
	}

	// Check that dependency lists are initialized.
	if graph.Dependencies[module.Path] == nil {
		t.Error("Dependencies list not initialized")
	}

	if graph.Reverse[module.Path] == nil {
		t.Error("Reverse dependencies list not initialized")
	}
}

func TestDependencyGraphAddDependency(t *testing.T) {
	graph := NewDependencyGraph()

	moduleA := &Module{Path: ModulePath("test/moduleA")}
	moduleB := &Module{Path: ModulePath("test/moduleB")}

	graph.AddModule(moduleA)
	graph.AddModule(moduleB)

	// A depends on B.
	graph.AddDependency(moduleA.Path, moduleB.Path)

	// Check forward dependency.
	deps := graph.GetDependencies(moduleA.Path)
	if len(deps) != 1 || deps[0] != moduleB.Path {
		t.Errorf("Expected A to depend on B, got dependencies: %v", deps)
	}

	// Check reverse dependency.
	dependents := graph.GetDependents(moduleB.Path)
	if len(dependents) != 1 || dependents[0] != moduleA.Path {
		t.Errorf("Expected B to have A as dependent, got dependents: %v", dependents)
	}
}

func TestDependencyGraphSimpleTopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()

	// Create modules: A -> B -> C.
	moduleA := &Module{Path: ModulePath("A")}
	moduleB := &Module{Path: ModulePath("B")}
	moduleC := &Module{Path: ModulePath("C")}

	graph.AddModule(moduleA)
	graph.AddModule(moduleB)
	graph.AddModule(moduleC)

	// A depends on B, B depends on C.
	graph.AddDependency(moduleA.Path, moduleB.Path)
	graph.AddDependency(moduleB.Path, moduleC.Path)

	// Topological sort should give: C, B, A.
	order, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("Topological sort failed: %v", err)
	}

	expected := []ModulePath{ModulePath("C"), ModulePath("B"), ModulePath("A")}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d modules in order, got %d", len(expected), len(order))
	}

	for i, module := range expected {
		if order[i] != module {
			t.Errorf("Expected position %d to be %s, got %s", i, module, order[i])
		}
	}
}

func TestDependencyGraphCycleDetection(t *testing.T) {
	graph := NewDependencyGraph()

	// Create modules with cycle: A -> B -> C -> A.
	moduleA := &Module{Path: ModulePath("A")}
	moduleB := &Module{Path: ModulePath("B")}
	moduleC := &Module{Path: ModulePath("C")}

	graph.AddModule(moduleA)
	graph.AddModule(moduleB)
	graph.AddModule(moduleC)

	// Create cycle.
	graph.AddDependency(moduleA.Path, moduleB.Path)
	graph.AddDependency(moduleB.Path, moduleC.Path)
	graph.AddDependency(moduleC.Path, moduleA.Path)

	cycles, err := graph.DetectCycles()
	if err == nil {
		t.Error("Expected cycle detection to return error")
	}

	if len(cycles) == 0 {
		t.Error("Expected to detect at least one cycle")
	}

	// Topological sort should fail.
	_, err = graph.TopologicalSort()
	if err == nil {
		t.Error("Expected topological sort to fail with cycles")
	}
}

func TestDependencyGraphNoCycles(t *testing.T) {
	graph := NewDependencyGraph()

	// Create modules without cycles: A -> B, A -> C, B -> D.
	moduleA := &Module{Path: ModulePath("A")}
	moduleB := &Module{Path: ModulePath("B")}
	moduleC := &Module{Path: ModulePath("C")}
	moduleD := &Module{Path: ModulePath("D")}

	graph.AddModule(moduleA)
	graph.AddModule(moduleB)
	graph.AddModule(moduleC)
	graph.AddModule(moduleD)

	graph.AddDependency(moduleA.Path, moduleB.Path)
	graph.AddDependency(moduleA.Path, moduleC.Path)
	graph.AddDependency(moduleB.Path, moduleD.Path)

	cycles, err := graph.DetectCycles()
	if err != nil {
		t.Errorf("Unexpected error in cycle detection: %v", err)
	}

	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, found %d", len(cycles))
	}
}

func TestTransitiveDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	// Create modules: A -> B -> C -> D.
	moduleA := &Module{Path: ModulePath("A")}
	moduleB := &Module{Path: ModulePath("B")}
	moduleC := &Module{Path: ModulePath("C")}
	moduleD := &Module{Path: ModulePath("D")}

	graph.AddModule(moduleA)
	graph.AddModule(moduleB)
	graph.AddModule(moduleC)
	graph.AddModule(moduleD)

	graph.AddDependency(moduleA.Path, moduleB.Path)
	graph.AddDependency(moduleB.Path, moduleC.Path)
	graph.AddDependency(moduleC.Path, moduleD.Path)

	// Get transitive dependencies of A.
	transitive, err := graph.GetTransitiveDependencies(moduleA.Path)
	if err != nil {
		t.Fatalf("Failed to get transitive dependencies: %v", err)
	}

	expected := []ModulePath{ModulePath("B"), ModulePath("C"), ModulePath("D")}
	if len(transitive) != len(expected) {
		t.Fatalf("Expected %d transitive dependencies, got %d", len(expected), len(transitive))
	}

	// Check that all expected dependencies are present.
	for _, expectedDep := range expected {
		found := false

		for _, actualDep := range transitive {
			if actualDep == expectedDep {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Expected transitive dependency %s not found", expectedDep)
		}
	}
}

func TestModuleLoaderCreation(t *testing.T) {
	loader := NewModuleLoader()

	if loader == nil {
		t.Fatal("Failed to create module loader")
	}

	if len(loader.Cache) != 0 {
		t.Errorf("Expected empty cache, got %d modules", len(loader.Cache))
	}

	if len(loader.SearchPaths) != 0 {
		t.Errorf("Expected empty search paths, got %d paths", len(loader.SearchPaths))
	}

	if loader.Registry == nil {
		t.Error("Registry not initialized")
	}

	if loader.Graph == nil {
		t.Error("Dependency graph not initialized")
	}
}

func TestModuleLoaderSearchPaths(t *testing.T) {
	loader := NewModuleLoader()

	paths := []string{"/usr/lib/orizon", "/home/user/modules", "./local_modules"}

	for _, path := range paths {
		loader.AddSearchPath(path)
	}

	if len(loader.SearchPaths) != len(paths) {
		t.Errorf("Expected %d search paths, got %d", len(paths), len(loader.SearchPaths))
	}

	for i, expected := range paths {
		if loader.SearchPaths[i] != expected {
			t.Errorf("Expected search path %d to be %s, got %s",
				i, expected, loader.SearchPaths[i])
		}
	}
}

func TestSymbolTypes(t *testing.T) {
	tests := []struct {
		symbolType SymbolType
		expected   string
	}{
		{SymbolTypeFunction, "function"},
		{SymbolTypeVariable, "variable"},
		{SymbolTypeType, "type"},
		{SymbolTypeConstant, "constant"},
		{SymbolTypeClass, "class"},
		{SymbolTypeInterface, "interface"},
		{SymbolTypeEnum, "enum"},
		{SymbolTypeModule, "module"},
		{SymbolTypeNamespace, "namespace"},
	}

	for _, test := range tests {
		result := test.symbolType.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestVisibilityLevels(t *testing.T) {
	tests := []struct {
		visibility VisibilityLevel
		expected   string
	}{
		{VisibilityPrivate, "private"},
		{VisibilityPackage, "package"},
		{VisibilityProtected, "protected"},
		{VisibilityPublic, "public"},
	}

	for _, test := range tests {
		result := test.visibility.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestExportedSymbolCreation(t *testing.T) {
	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 10},
	}

	symbol := &ExportedSymbol{
		Name:          "testFunction",
		Type:          SymbolTypeFunction,
		Module:        ModulePath("test/module"),
		Visibility:    VisibilityPublic,
		Signature:     "func testFunction() -> void",
		Documentation: "A test function",
		Span:          span,
	}

	if symbol.Name != "testFunction" {
		t.Errorf("Expected name testFunction, got %s", symbol.Name)
	}

	if symbol.Type != SymbolTypeFunction {
		t.Errorf("Expected type function, got %s", symbol.Type.String())
	}

	if symbol.Visibility != VisibilityPublic {
		t.Errorf("Expected visibility public, got %s", symbol.Visibility.String())
	}
}

func TestModuleRegistryCreation(t *testing.T) {
	registry := NewModuleRegistry()

	if registry == nil {
		t.Fatal("Failed to create module registry")
	}

	if len(registry.Modules) != 0 {
		t.Errorf("Expected empty modules map, got %d modules", len(registry.Modules))
	}
}

func TestModuleRegistryRegisterModule(t *testing.T) {
	registry := NewModuleRegistry()

	metadata := &ModuleMetadata{
		Path:          ModulePath("test/module"),
		Name:          "Test Module",
		Description:   "A test module",
		Author:        "Test Author",
		License:       "MIT",
		Keywords:      []string{"test", "example"},
		Versions:      []Version{{Major: 1, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""}},
		LatestVersion: Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""},
		Dependencies:  make(map[Version][]ModuleSpec),
	}

	registry.RegisterModule(metadata)

	retrieved, exists := registry.GetModuleMetadata(metadata.Path)
	if !exists {
		t.Error("Module metadata not found after registration")
	}

	if retrieved.Name != metadata.Name {
		t.Errorf("Expected name %s, got %s", metadata.Name, retrieved.Name)
	}

	if retrieved.Author != metadata.Author {
		t.Errorf("Expected author %s, got %s", metadata.Author, retrieved.Author)
	}
}

func TestModuleRegistryGetLatestVersion(t *testing.T) {
	registry := NewModuleRegistry()

	metadata := &ModuleMetadata{
		Path:          ModulePath("test/module"),
		Name:          "Test Module",
		Versions:      []Version{{Major: 1, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""}, {Major: 1, Minor: 1, Patch: 0, PreRelease: "", BuildMetadata: ""}, {Major: 2, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""}},
		LatestVersion: Version{Major: 2, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""},
		Dependencies:  make(map[Version][]ModuleSpec),
	}

	registry.RegisterModule(metadata)

	latest, err := registry.GetLatestVersion(metadata.Path)
	if err != nil {
		t.Fatalf("Failed to get latest version: %v", err)
	}

	expected := Version{PreRelease: "", BuildMetadata: "", Major: 2, Minor: 0, Patch: 0}
	if latest.Compare(expected) != 0 {
		t.Errorf("Expected latest version %s, got %s", expected.String(), latest.String())
	}
}

func TestModuleRegistryGetAvailableVersions(t *testing.T) {
	registry := NewModuleRegistry()

	versions := []Version{
		{Major: 1, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""},
		{Major: 1, Minor: 1, Patch: 0, PreRelease: "", BuildMetadata: ""},
		{Major: 2, Minor: 0, Patch: 0, PreRelease: "", BuildMetadata: ""},
		{Major: 1, Minor: 0, Patch: 1, PreRelease: "", BuildMetadata: ""},
	}

	metadata := &ModuleMetadata{
		Path:          ModulePath("test/module"),
		Name:          "Test Module",
		Versions:      versions,
		LatestVersion: Version{PreRelease: "", BuildMetadata: "", Major: 2, Minor: 0, Patch: 0},
		Dependencies:  make(map[Version][]ModuleSpec),
	}

	registry.RegisterModule(metadata)

	available, err := registry.GetAvailableVersions(metadata.Path)
	if err != nil {
		t.Fatalf("Failed to get available versions: %v", err)
	}

	if len(available) != len(versions) {
		t.Errorf("Expected %d versions, got %d", len(versions), len(available))
	}

	// Check that versions are sorted (descending).
	for i := 1; i < len(available); i++ {
		if available[i-1].Compare(available[i]) < 0 {
			t.Errorf("Versions not properly sorted: %s should come before %s",
				available[i-1].String(), available[i].String())
		}
	}
}

func TestModuleLoaderStatistics(t *testing.T) {
	loader := NewModuleLoader()

	// Add some test modules.
	module1 := &Module{
		Path:         ModulePath("module1"),
		LoadStatus:   ModuleStatusLoaded,
		Dependencies: []ModuleSpec{{Path: ModulePath("dep1")}},
	}
	module2 := &Module{
		Path:         ModulePath("module2"),
		LoadStatus:   ModuleStatusError,
		Dependencies: []ModuleSpec{},
	}
	module3 := &Module{
		Path:         ModulePath("module3"),
		LoadStatus:   ModuleStatusCached,
		Dependencies: []ModuleSpec{{Path: ModulePath("dep1")}, {Path: ModulePath("dep2")}},
	}

	loader.Cache[module1.Path] = module1
	loader.Cache[module2.Path] = module2
	loader.Cache[module3.Path] = module3

	loader.Graph.AddModule(module1)
	loader.Graph.AddModule(module2)
	loader.Graph.AddModule(module3)

	stats := loader.GetStatistics()

	if stats.TotalModules != 3 {
		t.Errorf("Expected 3 total modules, got %d", stats.TotalModules)
	}

	if stats.LoadedModules != 1 {
		t.Errorf("Expected 1 loaded module, got %d", stats.LoadedModules)
	}

	if stats.CachedModules != 1 {
		t.Errorf("Expected 1 cached module, got %d", stats.CachedModules)
	}

	if stats.FailedModules != 1 {
		t.Errorf("Expected 1 failed module, got %d", stats.FailedModules)
	}

	expectedAvgDepth := float64(3) / float64(3) // (1+0+2)/3
	if stats.AverageDepthLevel != expectedAvgDepth {
		t.Errorf("Expected average depth %f, got %f", expectedAvgDepth, stats.AverageDepthLevel)
	}

	if stats.MaxDepthLevel != 2 {
		t.Errorf("Expected max depth 2, got %d", stats.MaxDepthLevel)
	}
}

func TestComplexDependencyGraphScenario(t *testing.T) {
	loader := NewModuleLoader()

	// Create a complex dependency scenario:.
	// App -> (UI, Core).
	// UI -> (Graphics, Utils).
	// Core -> (Utils, Database).
	// Graphics -> Utils.
	// Database -> Utils.
	// Utils -> (no dependencies).

	modules := map[string]*Module{
		"App":      {Path: ModulePath("App"), LoadStatus: ModuleStatusLoaded},
		"UI":       {Path: ModulePath("UI"), LoadStatus: ModuleStatusLoaded},
		"Core":     {Path: ModulePath("Core"), LoadStatus: ModuleStatusLoaded},
		"Graphics": {Path: ModulePath("Graphics"), LoadStatus: ModuleStatusLoaded},
		"Utils":    {Path: ModulePath("Utils"), LoadStatus: ModuleStatusLoaded},
		"Database": {Path: ModulePath("Database"), LoadStatus: ModuleStatusLoaded},
	}

	// Add modules to graph.
	for _, module := range modules {
		loader.Graph.AddModule(module)
	}

	// Add dependencies.
	loader.Graph.AddDependency(ModulePath("App"), ModulePath("UI"))
	loader.Graph.AddDependency(ModulePath("App"), ModulePath("Core"))
	loader.Graph.AddDependency(ModulePath("UI"), ModulePath("Graphics"))
	loader.Graph.AddDependency(ModulePath("UI"), ModulePath("Utils"))
	loader.Graph.AddDependency(ModulePath("Core"), ModulePath("Utils"))
	loader.Graph.AddDependency(ModulePath("Core"), ModulePath("Database"))
	loader.Graph.AddDependency(ModulePath("Graphics"), ModulePath("Utils"))
	loader.Graph.AddDependency(ModulePath("Database"), ModulePath("Utils"))

	// Test cycle detection - should be no cycles.
	cycles, err := loader.Graph.DetectCycles()
	if err != nil {
		t.Errorf("Unexpected error in cycle detection: %v", err)
	}

	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, found %d", len(cycles))
	}

	// Test topological sort.
	order, err := loader.Graph.TopologicalSort()
	if err != nil {
		t.Fatalf("Topological sort failed: %v", err)
	}

	// Utils should come first (no dependencies).
	if order[0] != ModulePath("Utils") {
		t.Errorf("Expected Utils to be first in load order, got %s", order[0])
	}

	// App should come last (depends on everything).
	if order[len(order)-1] != ModulePath("App") {
		t.Errorf("Expected App to be last in load order, got %s", order[len(order)-1])
	}

	// Test transitive dependencies of App.
	transitive, err := loader.Graph.GetTransitiveDependencies(ModulePath("App"))
	if err != nil {
		t.Fatalf("Failed to get transitive dependencies: %v", err)
	}

	// App should transitively depend on all other modules.
	expectedTransitive := []ModulePath{
		ModulePath("UI"), ModulePath("Core"), ModulePath("Graphics"),
		ModulePath("Utils"), ModulePath("Database"),
	}

	if len(transitive) != len(expectedTransitive) {
		t.Errorf("Expected %d transitive dependencies, got %d",
			len(expectedTransitive), len(transitive))
	}
}
