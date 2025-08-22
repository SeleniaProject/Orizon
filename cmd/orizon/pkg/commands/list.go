// Package commands provides the list command implementation for package management.
// This handles listing available packages in the registry.
package commands

import (
	"context"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
)

// ListCommand handles package listing operations.
// It lists all available packages in the registry with their versions.
type ListCommand struct {
	*BaseCommand
}

// NewListCommand creates a new list command handler.
func NewListCommand() *ListCommand {
	return &ListCommand{
		BaseCommand: NewBaseCommand(
			"List all known manifests in registry",
			"usage: orizon pkg list",
		),
	}
}

// Execute implements the CommandHandler interface for list operations.
func (c *ListCommand) Execute(ctx types.RegistryContext, args []string) error {
	// Get all manifests from registry
	manifests, err := ctx.Registry.All(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	// Print each package with version
	for _, manifest := range manifests {
		fmt.Printf("%s@%s\n", manifest.Name, manifest.Version)
	}

	return nil
}
