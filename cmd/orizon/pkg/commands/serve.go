// Package commands provides the serve command implementation for package management.
// This handles starting HTTP registry servers for package distribution.
package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// ServeCommand handles HTTP registry server operations.
// It starts a local HTTP server to serve packages from a file registry.
type ServeCommand struct {
	*BaseCommand
}

// NewServeCommand creates a new serve command handler.
func NewServeCommand() *ServeCommand {
	return &ServeCommand{
		BaseCommand: NewBaseCommand(
			"Start HTTP registry server",
			"usage: orizon pkg serve [--addr <addr>] [--token <token>] [--tls-cert <cert>] [--tls-key <key>]",
		),
	}
}

// Execute implements the CommandHandler interface for serve operations.
func (c *ServeCommand) Execute(ctx types.RegistryContext, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":9321", "listen address")
	token := fs.String("token", "", "optional bearer token")
	tlsCert := fs.String("tls-cert", "", "path to TLS certificate (PEM)")
	tlsKey := fs.String("tls-key", "", "path to TLS private key (PEM)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse serve flags: %w", err)
	}

	// Determine registry path
	regPath := utils.GetRegistryPath()
	if regPath == "" || strings.HasPrefix(strings.ToLower(regPath), "http") {
		regPath = utils.DefaultRegistryPath
	}

	// Create file registry
	fileReg, err := packagemanager.NewFileRegistry(regPath)
	if err != nil {
		return fmt.Errorf("failed to create file registry: %w", err)
	}

	// Set token environment variable if provided
	if *token != "" {
		if err := os.Setenv("ORIZON_REGISTRY_TOKEN", *token); err != nil {
			return fmt.Errorf("failed to set registry token: %w", err)
		}
	}

	// Check if TLS is requested
	useTLS := strings.TrimSpace(*tlsCert) != "" && strings.TrimSpace(*tlsKey) != ""

	if useTLS {
		// Start HTTPS server
		fmt.Printf("serving registry on https://%s (root=%s) auth=%v\n",
			*addr, regPath, os.Getenv("ORIZON_REGISTRY_TOKEN") != "")

		if err := packagemanager.StartHTTPServerTLS(fileReg, *addr, *tlsCert, *tlsKey); err != nil {
			return fmt.Errorf("failed to start HTTPS server: %w", err)
		}

		return nil
	}

	// Start HTTP server
	fmt.Printf("serving registry on http://%s (root=%s) auth=%v\n",
		*addr, regPath, os.Getenv("ORIZON_REGISTRY_TOKEN") != "")

	if err := packagemanager.StartHTTPServer(fileReg, *addr); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}
