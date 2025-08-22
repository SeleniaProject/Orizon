// Package commands provides the auth command implementation for package management.
// This handles registry authentication and credential management.
package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

// AuthCommand handles authentication operations for package registries.
// It supports login functionality with token-based authentication.
type AuthCommand struct {
	*BaseCommand
}

// NewAuthCommand creates a new auth command handler.
func NewAuthCommand() *AuthCommand {
	return &AuthCommand{
		BaseCommand: NewBaseCommand(
			"Manage registry authentication",
			"usage: orizon pkg auth login",
		),
	}
}

// Execute implements the CommandHandler interface for auth operations.
func (c *AuthCommand) Execute(ctx types.RegistryContext, args []string) error {
	if len(args) < 1 {
		c.PrintUsage()
		return nil
	}

	switch args[0] {
	case "login":
		return c.handleLogin(args[1:])
	default:
		return fmt.Errorf("unknown auth subcommand: %s", args[0])
	}
}

// handleLogin processes the login subcommand for registry authentication.
func (c *AuthCommand) handleLogin(args []string) error {
	fs := flag.NewFlagSet("auth login", flag.ExitOnError)
	registryURL := fs.String("registry", "", "registry URL (http/https)")
	token := fs.String("token", "", "bearer token")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse login flags: %w", err)
	}

	if strings.TrimSpace(*registryURL) == "" || strings.TrimSpace(*token) == "" {
		return fmt.Errorf("--registry and --token are required")
	}

	// Load existing credentials
	credentials, err := utils.LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Update credentials with new token
	credentials[strings.TrimRight(*registryURL, "/")] = struct {
		Token string `json:"token"`
	}{
		Token: *token,
	}

	// Save updated credentials
	if err := utils.SaveCredentials(credentials); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("credentials updated")
	return nil
}
