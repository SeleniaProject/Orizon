// Package commands provides the update command implementation for package management.
// This handles updating dependencies to newer versions while respecting constraints.
package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// UpdateCommand handles dependency update operations.
// It updates packages to newer versions while maintaining compatibility constraints.
type UpdateCommand struct {
	*BaseCommand
}

// NewUpdateCommand creates a new update command handler.
func NewUpdateCommand() *UpdateCommand {
	return &UpdateCommand{
		BaseCommand: NewBaseCommand(
			"Update dependencies to newer versions",
			"usage: orizon pkg update [--dep <names>]",
		),
	}
}

// Execute implements the CommandHandler interface for update operations.
func (c *UpdateCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	onlyDeps := fs.String("dep", "", "comma-separated dependency names to update")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse update flags: %w", err)
	}

	// Read current manifest
	manifest, err := utils.ReadManifest()
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Parse dependency filter
	var depsToUpdate []string
	if strings.TrimSpace(*onlyDeps) != "" {
		for _, dep := range strings.Split(*onlyDeps, ",") {
			dep = strings.TrimSpace(dep)
			if dep != "" {
				depsToUpdate = append(depsToUpdate, dep)
			}
		}
	}

	if len(depsToUpdate) == 0 {
		// Update all dependencies
		return c.updateAllDependencies(context.Background(), ctx.Registry, manifest)
	}

	// Update specific dependencies
	return c.updateSpecificDependencies(context.Background(), ctx.Registry, manifest, depsToUpdate)
}

// updateAllDependencies updates all dependencies in the manifest.
func (c *UpdateCommand) updateAllDependencies(ctx context.Context, reg packagemanager.Registry, manifest types.Manifest) error {
	if err := utils.WriteLockFromManifest(ctx, reg, manifest); err != nil {
		return fmt.Errorf("failed to update dependencies: %w", err)
	}

	fmt.Println("dependencies updated and lockfile rewritten")
	return nil
}

// updateSpecificDependencies updates only the specified dependencies.
func (c *UpdateCommand) updateSpecificDependencies(ctx context.Context, reg packagemanager.Registry, manifest types.Manifest, deps []string) error {
	// Load existing lockfile for pinned versions
	locked := make(map[string]string)
	if data, err := os.ReadFile(utils.DefaultLockfilePath); err == nil {
		var lf packagemanager.Lockfile
		if json.Unmarshal(data, &lf) == nil {
			for _, entry := range lf.Entries {
				locked[string(entry.Name)] = string(entry.Version)
			}
		}
	}

	// If no lockfile, resolve current state
	if len(locked) == 0 {
		current, err := utils.ResolveCurrent(ctx, reg, manifest)
		if err != nil {
			return fmt.Errorf("failed to resolve current state: %w", err)
		}

		for name, info := range current {
			locked[string(name)] = string(info.Version)
		}
	}

	// Build requirements with selective updates
	reqs := make([]packagemanager.Requirement, 0, len(manifest.Dependencies))
	updateSet := make(map[string]bool)
	for _, dep := range deps {
		updateSet[dep] = true
	}

	for name := range manifest.Dependencies {
		if updateSet[name] {
			// Use manifest constraint for packages being updated
			reqs = append(reqs, packagemanager.Requirement{
				Name:      packagemanager.PackageID(name),
				Constraint: manifest.Dependencies[name],
			})
		} else if version, ok := locked[name]; ok {
			// Pin to current version for packages not being updated
			reqs = append(reqs, packagemanager.Requirement{
				Name:      packagemanager.PackageID(name),
				Constraint: "=" + version,
			})
		} else {
			// Fallback to manifest constraint
			reqs = append(reqs, packagemanager.Requirement{
				Name:      packagemanager.PackageID(name),
				Constraint: manifest.Dependencies[name],
			})
		}
	}

	// Resolve with new requirements
	manager := packagemanager.NewManager(reg)
	out, err := manager.ResolveAndFetch(ctx, reqs, true)
	if err != nil {
		return fmt.Errorf("failed to resolve updated dependencies: %w", err)
	}

	// Generate and write new lockfile
	resolution := make(packagemanager.Resolution)
	for name, info := range out {
		resolution[name] = info.Version
	}

	_, data, err := packagemanager.GenerateLockfile(ctx, reg, resolution)
	if err != nil {
		return fmt.Errorf("failed to generate updated lockfile: %w", err)
	}

	if err := utils.WriteLockfile(data); err != nil {
		return fmt.Errorf("failed to write updated lockfile: %w", err)
	}

	fmt.Printf("updated %s and rewrote lockfile\n", deps)
	return nil
}
