package modules

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/resolver"
)

// ModuleResolver integrates the module system with the symbol resolver.
type ModuleResolver struct {
	ModuleLoader   *ModuleLoader
	SymbolResolver *resolver.Resolver
	ModuleCache    map[ModulePath]*resolver.SymbolTable
}

// NewModuleResolver creates a new module resolver.
func NewModuleResolver() *ModuleResolver {
	symbolTable := resolver.NewSymbolTable()

	return &ModuleResolver{
		ModuleLoader:   NewModuleLoader(),
		SymbolResolver: resolver.NewResolver(symbolTable),
		ModuleCache:    make(map[ModulePath]*resolver.SymbolTable),
	}
}

// ResolveModuleProgram resolves a complete program with all its modules.
func (mr *ModuleResolver) ResolveModuleProgram(entryPoints []ModulePath) (*hir.HIRProgram, error) {
	// Load all modules and their dependencies.
	if err := mr.ModuleLoader.ResolveModules(entryPoints); err != nil {
		return nil, fmt.Errorf("failed to resolve modules: %w", err)
	}

	// Get load order for proper resolution sequence.
	loadOrder := mr.ModuleLoader.GetLoadOrder()

	// Create HIR program.
	program := hir.NewHIRProgram()

	// Add modules in dependency order.
	for _, modulePath := range loadOrder {
		module := mr.ModuleLoader.GetModule(modulePath)
		if module == nil {
			continue
		}

		// Add module to program.
		if module.HIRModule != nil {
			program.Modules[module.HIRModule.ModuleID] = module.HIRModule
		}
	}

	return program, nil
}

// AddSearchPath adds a search path to the module loader.
func (mr *ModuleResolver) AddSearchPath(path string) {
	mr.ModuleLoader.AddSearchPath(path)
}

// GetModule returns a loaded module.
func (mr *ModuleResolver) GetModule(path ModulePath) *Module {
	return mr.ModuleLoader.GetModule(path)
}

// GetAllModules returns all loaded modules.
func (mr *ModuleResolver) GetAllModules() map[ModulePath]*Module {
	return mr.ModuleLoader.GetAllModules()
}

// GetDependencyGraph returns the dependency graph.
func (mr *ModuleResolver) GetDependencyGraph() *DependencyGraph {
	return mr.ModuleLoader.Graph
}

// GetStatistics returns statistics about the module system.
func (mr *ModuleResolver) GetStatistics() Statistics {
	return mr.ModuleLoader.GetStatistics()
}

// ValidateModuleSystem validates the entire module system for consistency.
func (mr *ModuleResolver) ValidateModuleSystem() error {
	// Check for circular dependencies.
	cycles, err := mr.ModuleLoader.Graph.DetectCycles()
	if err != nil {
		return err
	}

	if len(cycles) > 0 {
		return fmt.Errorf("circular dependencies detected: %d cycles found", len(cycles))
	}

	// Validate each module.
	for modulePath, module := range mr.ModuleLoader.GetAllModules() {
		if err := mr.validateModule(module); err != nil {
			return fmt.Errorf("validation failed for module %s: %w", modulePath, err)
		}
	}

	return nil
}

// validateModule validates a single module.
func (mr *ModuleResolver) validateModule(module *Module) error {
	// Check that load status is consistent.
	if module.LoadStatus == ModuleStatusError && module.LoadError == nil {
		return fmt.Errorf("module marked as error but no error recorded")
	}

	if module.LoadStatus == ModuleStatusLoaded && module.HIRModule == nil {
		return fmt.Errorf("module marked as loaded but no HIR representation")
	}

	// Validate dependencies.
	for _, dep := range module.Dependencies {
		depModule := mr.ModuleLoader.GetModule(dep.Path)
		if depModule == nil {
			return fmt.Errorf("dependency %s not loaded", dep.Path)
		}

		if depModule.LoadStatus != ModuleStatusLoaded &&
			depModule.LoadStatus != ModuleStatusCached {
			return fmt.Errorf("dependency %s not properly loaded (status: %s)",
				dep.Path, depModule.LoadStatus.String())
		}
	}

	return nil
}
