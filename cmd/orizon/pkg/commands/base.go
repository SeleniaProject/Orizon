// Package commands provides implementations for individual package management subcommands.
// Each command is implemented as a separate handler with clean separation of concerns.
package commands

import (
	"fmt"
	"os"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
)

// BaseCommand provides common functionality for all command implementations.
// It implements the CommandHandler interface and provides default behaviors.
type BaseCommand struct {
	description string
	usage       string
}

// NewBaseCommand creates a new base command with the given description and usage.
func NewBaseCommand(description, usage string) *BaseCommand {
	return &BaseCommand{
		description: description,
		usage:       usage,
	}
}

// Description returns the human-readable description of the command.
func (c *BaseCommand) Description() string {
	return c.description
}

// Usage returns the usage information for the command.
func (c *BaseCommand) Usage() string {
	return c.usage
}

// PrintUsage prints the command usage to stderr.
func (c *BaseCommand) PrintUsage() {
	fmt.Fprintf(os.Stderr, "%s\n", c.Usage())
}

// Execute implements the CommandHandler interface.
// This base implementation returns an error indicating the method should be overridden.
func (c *BaseCommand) Execute(ctx types.RegistryContext, args []string) error {
	return fmt.Errorf("command execution not implemented")
}

// ExitWithUsage prints usage information and exits with the given code.
func (c *BaseCommand) ExitWithUsage(code int) {
	c.PrintUsage()
	os.Exit(code)
}

// ExitWithError prints an error message and exits with code 1.
func (c *BaseCommand) ExitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// ExitWithMessage prints a message and exits with the given code.
func (c *BaseCommand) ExitWithMessage(message string, code int) {
	fmt.Println(message)
	os.Exit(code)
}
