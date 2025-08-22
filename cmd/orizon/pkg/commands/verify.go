// Package commands provides the verify command implementation for package management.
// This handles lockfile verification and integrity checking.
package commands

import (
	"context"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// VerifyCommand handles lockfile verification operations.
// It verifies the integrity of lockfiles against the registry.
type VerifyCommand struct {
	*BaseCommand
}

// NewVerifyCommand creates a new verify command handler.
func NewVerifyCommand() *VerifyCommand {
	return &VerifyCommand{
		BaseCommand: NewBaseCommand(
			"Verify lockfile integrity",
			"usage: orizon pkg verify",
		),
	}
}

// Execute implements the CommandHandler interface for verify operations.
func (c *VerifyCommand) Execute(ctx types.RegistryContext, args []string) error {
	// Read lockfile
	data, err := utils.ReadLockfile()
	if err != nil {
		return fmt.Errorf("failed to read lockfile: %w", err)
	}

	// Convert to packagemanager lockfile
	lockfile := packagemanager.Lockfile{
		Entries: data.Entries,
	}

	// Verify lockfile
	if err := packagemanager.VerifyLockfile(context.Background(), ctx.Registry, lockfile); err != nil {
		return fmt.Errorf("lockfile verification failed: %w", err)
	}

	fmt.Println("lockfile verified")
	return nil
}
