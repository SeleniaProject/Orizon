// Package commands provides the resolve command implementation for package management.
// This handles dependency resolution and version constraint solving.
package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// ResolveCommand handles dependency resolution operations.
// It resolves package dependencies against the registry and outputs the resolution plan.
type ResolveCommand struct {
	*BaseCommand
}

// NewResolveCommand creates a new resolve command handler.
func NewResolveCommand() *ResolveCommand {
	return &ResolveCommand{
		BaseCommand: NewBaseCommand(
			"Resolve current manifest dependencies against registry",
			"usage: orizon pkg resolve",
		),
	}
}

// Execute implements the CommandHandler interface for resolve operations.
func (c *ResolveCommand) Execute(ctx types.RegistryContext, args []string) error {
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

	// Convert to JSON output format
	result := make(map[string]struct {
		Version string `json:"version"`
		CID     string `json:"cid"`
	})

	for name, info := range resolved {
		result[string(name)] = struct {
			Version string `json:"version"`
			CID     string `json:"cid"`
		}{
			Version: string(info.Version),
			CID:     string(info.CID),
		}
	}

	// Output as JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resolution result: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
