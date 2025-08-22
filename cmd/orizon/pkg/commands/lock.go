// Package commands provides the lock command implementation for package management.
// This handles lockfile generation from resolved dependencies.
package commands

import (
	"context"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// LockCommand handles lockfile generation operations.
// It creates lockfiles from resolved dependency trees for reproducible builds.
type LockCommand struct {
	*BaseCommand
}

// NewLockCommand creates a new lock command handler.
func NewLockCommand() *LockCommand {
	return &LockCommand{
		BaseCommand: NewBaseCommand(
			"Generate lockfile from current resolved state",
			"usage: orizon pkg lock",
		),
	}
}

// Execute implements the CommandHandler interface for lock operations.
func (c *LockCommand) Execute(ctx types.RegistryContext, args []string) error {
	// Read current manifest
	manifest, err := utils.ReadManifest()
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Resolve dependencies
	resolved, err := utils.ResolveCurrent(context.Background(), ctx.Registry, manifest)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Create resolution map
	resolution := make(packagemanager.Resolution)
	for name, info := range resolved {
		resolution[name] = info.Version
	}

	// Generate lockfile
	lock, data, err := packagemanager.GenerateLockfile(context.Background(), ctx.Registry, resolution)
	if err != nil {
		return fmt.Errorf("failed to generate lockfile: %w", err)
	}

	// Write lockfile
	if err := utils.WriteLockfile(data); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	fmt.Printf("lockfile written (%d entries)\n", len(lock.Entries))
	return nil
}
