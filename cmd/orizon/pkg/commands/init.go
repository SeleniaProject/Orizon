// Package commands provides the init command implementation for package management.
// This handles initialization of new package manifests.
package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// InitCommand handles package initialization operations.
// It creates new package manifest files with sensible defaults.
type InitCommand struct {
	*BaseCommand
}

// NewInitCommand creates a new init command handler.
func NewInitCommand() *InitCommand {
	return &InitCommand{
		BaseCommand: NewBaseCommand(
			"Initialize a new package manifest",
			"usage: orizon pkg init",
		),
	}
}

// Execute implements the CommandHandler interface for init operations.
func (c *InitCommand) Execute(ctx types.RegistryContext, args []string) error {
	return c.initializeManifest()
}

// initializeManifest creates a new package manifest if one doesn't already exist.
func (c *InitCommand) initializeManifest() error {
	// Check if manifest already exists
	if _, err := os.Stat(utils.DefaultManifestPath); err == nil {
		fmt.Println("orizon.json exists")
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check manifest existence: %w", err)
	}

	// Create default manifest
	manifest := types.Manifest{
		Name:         "app",
		Version:      "0.1.0",
		Dependencies: make(map[string]string),
	}

	// Marshal and write manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(utils.DefaultManifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Println("created orizon.json")

	// Ensure .orizon directory exists
	if err := utils.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	return nil
}
