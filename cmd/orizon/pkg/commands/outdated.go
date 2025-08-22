// Package commands provides the outdated command implementation for package management.
// This handles checking for available updates to dependencies.
package commands

import (
	"context"
	"fmt"

	semver "github.com/Masterminds/semver/v3"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// OutdatedCommand handles dependency update checking operations.
// It compares current versions with available updates and latest versions.
type OutdatedCommand struct {
	*BaseCommand
}

// NewOutdatedCommand creates a new outdated command handler.
func NewOutdatedCommand() *OutdatedCommand {
	return &OutdatedCommand{
		BaseCommand: NewBaseCommand(
			"Check for outdated dependencies",
			"usage: orizon pkg outdated",
		),
	}
}

// Execute implements the CommandHandler interface for outdated operations.
func (c *OutdatedCommand) Execute(ctx types.RegistryContext, args []string) error {
	// Read current manifest
	manifest, err := utils.ReadManifest()
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Resolve current dependencies
	resolved, err := utils.ResolveCurrent(context.Background(), ctx.Registry, manifest)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Print header
	fmt.Println("name  current  allowed  latest")

	// Check each dependency
	for name, constraint := range manifest.Dependencies {
		current := string(resolved[packagemanager.PackageID(name)].Version)

		// Parse constraint
		constraintObj, err := semver.NewConstraint(constraint)
		if err != nil {
			fmt.Printf("%s  %s  error  error\n", name, current)
			continue
		}

		// List all versions of this package
		manifests, err := ctx.Registry.List(context.Background(), packagemanager.PackageID(name))
		if err != nil {
			fmt.Printf("%s  %s  error  error\n", name, current)
			continue
		}

		var bestAllowed, bestOverall string
		var bestAllowedVer, bestOverallVer *semver.Version

		// Find best versions
		for _, mf := range manifests {
			sv, err := semver.NewVersion(string(mf.Version))
			if err != nil {
				continue
			}

			// Track overall latest
			if bestOverallVer == nil || sv.GreaterThan(bestOverallVer) {
				bestOverallVer = sv
				bestOverall = sv.String()
			}

			// Track best allowed by constraint
			if constraintObj.Check(sv) {
				if bestAllowedVer == nil || sv.GreaterThan(bestAllowedVer) {
					bestAllowedVer = sv
					bestAllowed = sv.String()
				}
			}
		}

		// Set defaults for missing versions
		if bestAllowed == "" {
			bestAllowed = "-"
		}
		if bestOverall == "" {
			bestOverall = "-"
		}

		fmt.Printf("%s  %s  %s  %s\n", name, current, bestAllowed, bestOverall)
	}

	return nil
}
