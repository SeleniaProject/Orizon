// Package commands provides the publish command implementation for package management.
// This handles publishing packages to the registry.
package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// PublishCommand handles package publishing operations.
// It uploads package data to the registry with proper metadata.
type PublishCommand struct {
	*BaseCommand
}

// NewPublishCommand creates a new publish command handler.
func NewPublishCommand() *PublishCommand {
	return &PublishCommand{
		BaseCommand: NewBaseCommand(
			"Publish a package to the registry",
			"usage: orizon pkg publish --name <id> --version <semver> --file <path>",
		),
	}
}

// Execute implements the CommandHandler interface for publish operations.
func (c *PublishCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	name := fs.String("name", "", "package name")
	version := fs.String("version", "", "package version (semver)")
	file := fs.String("file", "", "payload file to publish")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse publish flags: %w", err)
	}

	if *name == "" || *version == "" || *file == "" {
		return fmt.Errorf("usage: orizon pkg publish --name <id> --version <semver> --file <path>")
	}

	// Read package data
	data, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("failed to read package file: %w", err)
	}

	// Create package blob
	blob := packagemanager.PackageBlob{
		Manifest: packagemanager.PackageManifest{
			Name:    packagemanager.PackageID(*name),
			Version: packagemanager.Version(*version),
		},
		Data: data,
	}

	// Publish package
	cid, err := ctx.Registry.Publish(context.Background(), blob)
	if err != nil {
		return fmt.Errorf("failed to publish package: %w", err)
	}

	fmt.Printf("published %s@%s cid=%s\n", *name, *version, cid)
	return nil
}
