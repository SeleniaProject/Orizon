// Package commands provides the fetch command implementation for package management.
// This handles downloading and caching of specific package versions.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	semver "github.com/Masterminds/semver/v3"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// FetchCommand handles package fetching operations.
// It downloads specific package versions and stores them in the local cache.
type FetchCommand struct {
	*BaseCommand
}

// NewFetchCommand creates a new fetch command handler.
func NewFetchCommand() *FetchCommand {
	return &FetchCommand{
		BaseCommand: NewBaseCommand(
			"Fetch a specific package version",
			"usage: orizon pkg fetch <name>@<constraint>",
		),
	}
}

// Execute implements the CommandHandler interface for fetch operations.
func (c *FetchCommand) Execute(ctx types.RegistryContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: orizon pkg fetch <name>@<constraint>")
	}

	// Parse package specification
	name, constraintStr := utils.SplitAt(args[0])

	// Parse version constraint
	constraint, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return fmt.Errorf("invalid constraint: %w", err)
	}

	// Find package matching constraint
	cid, manifest, err := ctx.Registry.Find(context.Background(), packagemanager.PackageID(name), constraint)
	if err != nil {
		return fmt.Errorf("failed to find package: %w", err)
	}

	// Fetch package data
	blob, err := ctx.Registry.Fetch(context.Background(), cid)
	if err != nil {
		return fmt.Errorf("failed to fetch package: %w", err)
	}

	// Prepare cache directory
	cachePath := filepath.Join(utils.DefaultCachePath, string(cid))
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write package data to cache
	if err := os.WriteFile(cachePath, blob.Data, 0644); err != nil {
		return fmt.Errorf("failed to write package to cache: %w", err)
	}

	fmt.Printf("fetched %s@%s -> %s\n", manifest.Name, manifest.Version, cachePath)
	return nil
}
