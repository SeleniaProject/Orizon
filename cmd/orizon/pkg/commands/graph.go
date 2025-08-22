// Package commands provides the graph command implementation for package management.
// This handles dependency graph visualization and analysis.
package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// GraphCommand handles dependency graph operations.
// It can display dependency relationships and explain why packages are included.
type GraphCommand struct {
	*BaseCommand
}

// NewGraphCommand creates a new graph command handler.
func NewGraphCommand() *GraphCommand {
	return &GraphCommand{
		BaseCommand: NewBaseCommand(
			"Display dependency graph relationships",
			"usage: orizon pkg graph [--dot] [--output <file>]",
		),
	}
}

// Execute implements the CommandHandler interface for graph operations.
func (c *GraphCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	dotFormat := fs.Bool("dot", false, "print Graphviz DOT instead of edges")
	outputPath := fs.String("output", "", "optional output file path")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse graph flags: %w", err)
	}

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

	var output strings.Builder

	if *dotFormat {
		// Generate Graphviz DOT format
		output.WriteString("digraph deps {\n")
		output.WriteString("  rankdir=LR;\n")

		roots := utils.GetRootDependencies(manifest)
		rootSet := make(map[string]bool)
		for _, root := range roots {
			rootSet[root] = true
		}

		for from, tos := range graph {
			name := from
			if i := strings.IndexByte(from, '@'); i >= 0 {
				name = from[:i]
			}

			if rootSet[name] {
				fmt.Fprintf(&output, "  \"%s\" [shape=box,style=bold];\n", from)
			} else if len(tos) == 0 {
				fmt.Fprintf(&output, "  \"%s\";\n", from)
			}

			for _, to := range tos {
				fmt.Fprintf(&output, "  \"%s\" -> \"%s\";\n", from, to)
			}
		}

		output.WriteString("}\n")
	} else {
		// Generate simple edge list
		for from, tos := range graph {
			if len(tos) == 0 {
				fmt.Fprintln(&output, from)
			} else {
				fmt.Fprintf(&output, "%s -> %s\n", from, strings.Join(tos, ", "))
			}
		}
	}

	result := output.String()

	if *outputPath != "" {
		// Write to file
		if err := os.WriteFile(*outputPath, []byte(result), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	} else {
		// Print to stdout
		fmt.Print(result)
	}

	return nil
}
