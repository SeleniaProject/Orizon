// Package commands provides the vendor command implementation for package management.
// This handles downloading dependencies into a local vendor directory.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// VendorCommand handles dependency vendoring operations.
// It downloads all lockfile entries into a local vendor directory for offline builds.
type VendorCommand struct {
	*BaseCommand
}

// NewVendorCommand creates a new vendor command handler.
func NewVendorCommand() *VendorCommand {
	return &VendorCommand{
		BaseCommand: NewBaseCommand(
			"Download dependencies into vendor directory",
			"usage: orizon pkg vendor",
		),
	}
}

// Execute implements the CommandHandler interface for vendor operations.
func (c *VendorCommand) Execute(ctx types.RegistryContext, args []string) error {
	// Read lockfile
	lockfile, err := utils.ReadLockfile()
	if err != nil {
		return fmt.Errorf("failed to read lockfile: %w", err)
	}

	// Prepare vendor directory
	vendorPath := filepath.Join(".orizon", "vendor")
	if err := os.MkdirAll(vendorPath, 0755); err != nil {
		return fmt.Errorf("failed to create vendor directory: %w", err)
	}

	// Download each package
	for _, entry := range lockfile.Entries {
		// Fetch package data
		blob, err := ctx.Registry.Fetch(context.Background(), entry.CID)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", entry.Name, err)
		}

		// Create output filename
		filename := fmt.Sprintf("%s-%s.blob", entry.Name, entry.Version)
		outputPath := filepath.Join(vendorPath, filename)

		// Write package data
		if err := os.WriteFile(outputPath, blob.Data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	fmt.Printf("vendored %d packages into %s\n", len(lockfile.Entries), vendorPath)
	return nil
}
