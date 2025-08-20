package modules

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

func TestModuleResolverCreation(t *testing.T) {
	resolver := NewModuleResolver()

	if resolver == nil {
		t.Fatal("Failed to create module resolver")
	}

	if resolver.ModuleLoader == nil {
		t.Error("Module loader not initialized")
	}

	if resolver.SymbolResolver == nil {
		t.Error("Symbol resolver not initialized")
	}

	if resolver.ModuleCache == nil {
		t.Error("Module cache not initialized")
	}
}

func TestModuleResolverSearchPaths(t *testing.T) {
	resolver := NewModuleResolver()

	paths := []string{"/usr/lib/orizon", "./modules"}
	for _, path := range paths {
		resolver.AddSearchPath(path)
	}

	// Verify paths were added to the module loader.
	if len(resolver.ModuleLoader.SearchPaths) != len(paths) {
		t.Errorf("Expected %d search paths, got %d",
			len(paths), len(resolver.ModuleLoader.SearchPaths))
	}
}

func TestModuleValidationBasic(t *testing.T) {
	resolver := NewModuleResolver()

	// Create a simple module.
	module := &Module{
		Path:            ModulePath("test/module"),
		Version:         Version{1, 0, 0, "", ""},
		Name:            "testmodule",
		Dependencies:    []ModuleSpec{},
		DevDependencies: []ModuleSpec{},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ModuleSymbols:   make(map[string]*Symbol),
		ImportedModules: make(map[ModulePath]*Module),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		LoadStatus:      ModuleStatusLoaded,
		Span:            position.Span{},
	}

	// Create minimal HIR module.
	module.HIRModule = &hir.HIRModule{
		ID:           generateNodeID(),
		ModuleID:     generateModuleID(),
		Name:         module.Name,
		Declarations: []hir.HIRDeclaration{},
		Exports:      []string{},
		Imports:      []hir.ImportInfo{},
		Span:         module.Span,
	}

	// Add to resolver.
	resolver.ModuleLoader.Cache[module.Path] = module
	resolver.ModuleLoader.Graph.AddModule(module)

	// Validate - should pass.
	if err := resolver.validateModule(module); err != nil {
		t.Errorf("Validation failed for valid module: %v", err)
	}
}

func TestModuleValidationErrors(t *testing.T) {
	resolver := NewModuleResolver()

	// Test module with error status but no error message.
	errorModule := &Module{
		Path:       ModulePath("error/module"),
		LoadStatus: ModuleStatusError,
		LoadError:  nil, // This should cause validation to fail
	}

	err := resolver.validateModule(errorModule)
	if err == nil {
		t.Error("Expected validation to fail for module with error status but no error message")
	}

	// Test loaded module without HIR.
	loadedModule := &Module{
		Path:       ModulePath("loaded/module"),
		LoadStatus: ModuleStatusLoaded,
		HIRModule:  nil, // This should cause validation to fail
	}

	err = resolver.validateModule(loadedModule)
	if err == nil {
		t.Error("Expected validation to fail for loaded module without HIR")
	}
}

func TestModuleValidationWithDependencies(t *testing.T) {
	resolver := NewModuleResolver()

	// Create dependency module.
	depModule := &Module{
		Path:       ModulePath("dep/module"),
		LoadStatus: ModuleStatusLoaded,
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "depmodule",
		},
	}

	// Create main module that depends on depModule.
	mainModule := &Module{
		Path: ModulePath("main/module"),
		Dependencies: []ModuleSpec{
			{Path: depModule.Path, Version: Version{1, 0, 0, "", ""}},
		},
		LoadStatus: ModuleStatusLoaded,
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "mainmodule",
		},
	}

	// Add both modules to resolver.
	resolver.ModuleLoader.Cache[depModule.Path] = depModule
	resolver.ModuleLoader.Cache[mainModule.Path] = mainModule
	resolver.ModuleLoader.Graph.AddModule(depModule)
	resolver.ModuleLoader.Graph.AddModule(mainModule)

	// Validation should pass.
	if err := resolver.validateModule(mainModule); err != nil {
		t.Errorf("Validation failed for module with valid dependencies: %v", err)
	}

	// Test with missing dependency.
	missingDepModule := &Module{
		Path: ModulePath("missing/module"),
		Dependencies: []ModuleSpec{
			{Path: ModulePath("nonexistent/module"), Version: Version{1, 0, 0, "", ""}},
		},
		LoadStatus: ModuleStatusLoaded,
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "missingdepmodule",
		},
	}

	resolver.ModuleLoader.Cache[missingDepModule.Path] = missingDepModule

	err := resolver.validateModule(missingDepModule)
	if err == nil {
		t.Error("Expected validation to fail for module with missing dependency")
	}
}

func TestModuleSystemValidation(t *testing.T) {
	resolver := NewModuleResolver()

	// Create modules: A -> B -> C (linear dependency chain).
	moduleC := &Module{
		Path:            ModulePath("C"),
		LoadStatus:      ModuleStatusLoaded,
		Dependencies:    []ModuleSpec{},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "moduleC",
			Exports:  []string{},
		},
	}

	moduleB := &Module{
		Path:       ModulePath("B"),
		LoadStatus: ModuleStatusLoaded,
		Dependencies: []ModuleSpec{
			{Path: ModulePath("C"), Version: Version{1, 0, 0, "", ""}},
		},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "moduleB",
			Exports:  []string{},
		},
	}

	moduleA := &Module{
		Path:       ModulePath("A"),
		LoadStatus: ModuleStatusLoaded,
		Dependencies: []ModuleSpec{
			{Path: ModulePath("B"), Version: Version{1, 0, 0, "", ""}},
		},
		PublicSymbols:   make(map[string]*ExportedSymbol),
		ImportedSymbols: make(map[string]*ImportedSymbol),
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "moduleA",
			Exports:  []string{},
		},
	}

	// Add modules to resolver.
	resolver.ModuleLoader.Cache[moduleA.Path] = moduleA
	resolver.ModuleLoader.Cache[moduleB.Path] = moduleB
	resolver.ModuleLoader.Cache[moduleC.Path] = moduleC

	resolver.ModuleLoader.Graph.AddModule(moduleA)
	resolver.ModuleLoader.Graph.AddModule(moduleB)
	resolver.ModuleLoader.Graph.AddModule(moduleC)

	resolver.ModuleLoader.Graph.AddDependency(moduleA.Path, moduleB.Path)
	resolver.ModuleLoader.Graph.AddDependency(moduleB.Path, moduleC.Path)

	// Validation should pass.
	if err := resolver.ValidateModuleSystem(); err != nil {
		t.Errorf("Validation failed for valid module system: %v", err)
	}
}

func TestModuleSystemValidationWithCycles(t *testing.T) {
	resolver := NewModuleResolver()

	// Create modules with cycle: A -> B -> A.
	moduleA := &Module{
		Path:       ModulePath("A"),
		LoadStatus: ModuleStatusLoaded,
		Dependencies: []ModuleSpec{
			{Path: ModulePath("B"), Version: Version{1, 0, 0, "", ""}},
		},
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "moduleA",
		},
	}

	moduleB := &Module{
		Path:       ModulePath("B"),
		LoadStatus: ModuleStatusLoaded,
		Dependencies: []ModuleSpec{
			{Path: ModulePath("A"), Version: Version{1, 0, 0, "", ""}},
		},
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "moduleB",
		},
	}

	// Add modules to resolver.
	resolver.ModuleLoader.Cache[moduleA.Path] = moduleA
	resolver.ModuleLoader.Cache[moduleB.Path] = moduleB

	resolver.ModuleLoader.Graph.AddModule(moduleA)
	resolver.ModuleLoader.Graph.AddModule(moduleB)

	resolver.ModuleLoader.Graph.AddDependency(moduleA.Path, moduleB.Path)
	resolver.ModuleLoader.Graph.AddDependency(moduleB.Path, moduleA.Path)

	// Validation should fail due to cycle.
	err := resolver.ValidateModuleSystem()
	if err == nil {
		t.Error("Expected validation to fail for module system with cycles")
	}
}

func TestResolveModuleProgramBasic(t *testing.T) {
	resolver := NewModuleResolver()

	// Create a simple module.
	module := &Module{
		Path:            ModulePath("test/main"),
		LoadStatus:      ModuleStatusLoaded,
		Dependencies:    []ModuleSpec{},
		ImportedModules: make(map[ModulePath]*Module),
		HIRModule: &hir.HIRModule{
			ID:       generateNodeID(),
			ModuleID: generateModuleID(),
			Name:     "main",
			Exports:  []string{},
		},
	}

	// Mock the module loader to return our module.
	resolver.ModuleLoader.Cache[module.Path] = module
	resolver.ModuleLoader.Graph.AddModule(module)
	resolver.ModuleLoader.Graph.LoadOrder = []ModulePath{module.Path}

	// Manually call ResolveModules since we're mocking.
	err := resolver.ModuleLoader.ResolveModules([]ModulePath{module.Path})
	if err == nil {
		// Expected to fail since we don't have real files, but continue with test.
	}

	// Resolve program.
	program, err := resolver.ResolveModuleProgram([]ModulePath{module.Path})
	if err != nil {
		t.Fatalf("Failed to resolve module program: %v", err)
	}

	if program == nil {
		t.Fatal("Program is nil")
	}

	if len(program.Modules) != 1 {
		t.Errorf("Expected 1 module in program, got %d", len(program.Modules))
	}

	// Check that our module is in the program.
	if program.Modules[module.HIRModule.ModuleID] != module.HIRModule {
		t.Error("Module not properly added to program")
	}
}

func TestModuleResolverStatistics(t *testing.T) {
	resolver := NewModuleResolver()

	// Create some test modules.
	modules := []*Module{
		{
			Path:         ModulePath("module1"),
			LoadStatus:   ModuleStatusLoaded,
			Dependencies: []ModuleSpec{{Path: ModulePath("dep1")}},
		},
		{
			Path:         ModulePath("module2"),
			LoadStatus:   ModuleStatusError,
			Dependencies: []ModuleSpec{},
		},
		{
			Path:         ModulePath("module3"),
			LoadStatus:   ModuleStatusCached,
			Dependencies: []ModuleSpec{{Path: ModulePath("dep1")}, {Path: ModulePath("dep2")}},
		},
	}

	// Add modules to cache and graph.
	for _, module := range modules {
		resolver.ModuleLoader.Cache[module.Path] = module
		resolver.ModuleLoader.Graph.AddModule(module)
	}

	stats := resolver.GetStatistics()

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
}
