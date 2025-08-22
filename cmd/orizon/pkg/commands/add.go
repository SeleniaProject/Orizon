// Package commands provides the add command implementation for package management.
// This handles adding new dependencies to package manifests.
package commands

import (
	"flag"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// AddCommand handles dependency addition operations.
// It adds new package dependencies to the manifest with version constraints.
type AddCommand struct {
	*BaseCommand
}

// NewAddCommand creates a new add command handler.
func NewAddCommand() *AddCommand {
	return &AddCommand{
		BaseCommand: NewBaseCommand(
			"Add a dependency to the package manifest",
			"usage: orizon pkg add --dep name@constraint",
		),
	}
}

// Execute implements the CommandHandler interface for add operations.
func (c *AddCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	dep := fs.String("dep", "", "dependency in form name@constraint (e.g., foo@^1.2.0)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse add flags: %w", err)
	}

	if *dep == "" {
		return fmt.Errorf("usage: orizon pkg add --dep name@constraint")
	}

	// Read current manifest
	manifest, err := utils.ReadManifest()
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Parse dependency specification
	name, constraint := utils.SplitAt(*dep)

	// Add dependency to manifest
	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}
	manifest.Dependencies[name] = constraint

	// Write updated manifest
	if err := utils.WriteManifest(manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Printf("added %s -> %s\n", name, constraint)
	return nil
}
