// Package commands provides the remove command implementation for package management.
// This handles removing dependencies from package manifests.
package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// RemoveCommand handles dependency removal operations.
// It removes specified dependencies from the manifest and optionally updates the lockfile.
type RemoveCommand struct {
	*BaseCommand
}

// NewRemoveCommand creates a new remove command handler.
func NewRemoveCommand() *RemoveCommand {
	return &RemoveCommand{
		BaseCommand: NewBaseCommand(
			"Remove dependencies from the package manifest",
			"usage: orizon pkg remove --dep <name> [--lock=true]",
		),
	}
}

// Execute implements the CommandHandler interface for remove operations.
func (c *RemoveCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	depName := fs.String("dep", "", "dependency name to remove")
	relock := fs.Bool("lock", true, "rewrite lockfile after removal")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse remove flags: %w", err)
	}

	if *depName == "" {
		return fmt.Errorf("usage: orizon pkg remove --dep <name> [--lock=true]")
	}

	// Read current manifest
	manifest, err := utils.ReadManifest()
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Remove dependency from manifest
	if _, exists := manifest.Dependencies[*depName]; !exists {
		return fmt.Errorf("dependency %s not found in manifest", *depName)
	}

	delete(manifest.Dependencies, *depName)

	// Write updated manifest
	if err := utils.WriteManifest(manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Optionally update lockfile
	if *relock {
		if err := utils.WriteLockFromManifest(context.Background(), ctx.Registry, manifest); err != nil {
			return fmt.Errorf("failed to update lockfile: %w", err)
		}
	}

	fmt.Printf("removed %s\n", *depName)
	return nil
}
