// Package commands provides the why command implementation for package management.
// This handles dependency path explanation and why packages are included.
package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// WhyCommand handles dependency explanation operations.
// It shows why a specific package is included in the dependency tree.
type WhyCommand struct {
	*BaseCommand
}

// NewWhyCommand creates a new why command handler.
func NewWhyCommand() *WhyCommand {
	return &WhyCommand{
		BaseCommand: NewBaseCommand(
			"Explain why a package is included",
			"usage: orizon pkg why [--verbose] [--cid] <name>",
		),
	}
}

// Execute implements the CommandHandler interface for why operations.
func (c *WhyCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("why", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "print versions along the path")
	showCID := fs.Bool("cid", false, "include CIDs when --verbose is set")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse why flags: %w", err)
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: orizon pkg why [--verbose] [--cid] <name>")
	}

	target := rest[0]

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

	// Build dependency graph
	graph, err := utils.BuildDependencyGraph(context.Background(), ctx.Registry, resolved)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Find dependency path
	roots := utils.GetRootDependencies(manifest)
	path := utils.FindDependencyPath(graph, roots, target)

	if len(path) == 0 {
		fmt.Printf("no path to %s\n", target)
		return nil
	}

	if *verbose {
		// Print detailed path with versions
		parts := make([]string, 0, len(path))
		for _, name := range path {
			if info, ok := resolved[packagemanager.PackageID(name)]; ok {
				if *showCID {
					parts = append(parts, fmt.Sprintf("%s@%s (%s)", name, info.Version, info.CID))
				} else {
					parts = append(parts, fmt.Sprintf("%s@%s", name, info.Version))
				}
			} else {
				parts = append(parts, name)
			}
		}
		fmt.Println(parts[0])
		for i := 1; i < len(parts); i++ {
			fmt.Printf("  -> %s\n", parts[i])
		}
	} else {
		// Print simple path
		fmt.Println(path[0])
		for i := 1; i < len(path); i++ {
			fmt.Printf("  -> %s\n", path[i])
		}
	}

	return nil
}
