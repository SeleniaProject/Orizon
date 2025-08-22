// Package commands provides a command registry for managing all package management subcommands.
// This centralizes command registration and lookup for clean architecture.
package commands

import (
	"fmt"
	"sort"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
)

// Registry manages all available package management commands.
// It provides centralized command registration and execution.
type Registry struct {
	commands map[string]types.CommandHandler
}

// NewRegistry creates a new command registry and registers all available commands.
func NewRegistry() *Registry {
	registry := &Registry{
		commands: make(map[string]types.CommandHandler),
	}

	// Register all commands
	registry.register("init", NewInitCommand())
	registry.register("add", NewAddCommand())
	registry.register("remove", NewRemoveCommand())
	registry.register("resolve", NewResolveCommand())
	registry.register("lock", NewLockCommand())
	registry.register("verify", NewVerifyCommand())
	registry.register("list", NewListCommand())
	registry.register("fetch", NewFetchCommand())
	registry.register("update", NewUpdateCommand())
	registry.register("publish", NewPublishCommand())
	registry.register("auth", NewAuthCommand())
	registry.register("serve", NewServeCommand())
	registry.register("graph", NewGraphCommand())
	registry.register("why", NewWhyCommand())
	registry.register("outdated", NewOutdatedCommand())
	registry.register("vendor", NewVendorCommand())
	registry.register("sign", NewSignCommand())
	registry.register("verify-sig", NewVerifySigCommand())
	registry.register("audit", NewAuditCommand())

	return registry
}

// register adds a command to the registry.
func (r *Registry) register(name string, command types.CommandHandler) {
	r.commands[name] = command
}

// GetCommand retrieves a command by name.
func (r *Registry) GetCommand(name string) (types.CommandHandler, bool) {
	command, exists := r.commands[name]
	return command, exists
}

// GetAllCommands returns a sorted list of all command names.
func (r *Registry) GetAllCommands() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListCommands prints all available commands with their descriptions.
func (r *Registry) ListCommands() {
	names := r.GetAllCommands()

	fmt.Println("Available subcommands:")
	for _, name := range names {
		if command, exists := r.GetCommand(name); exists {
			fmt.Printf("  %s - %s\n", name, command.Description())
		}
	}
}

// ExecuteCommand executes a command by name with the given context and arguments.
func (r *Registry) ExecuteCommand(name string, ctx types.RegistryContext, args []string) error {
	command, exists := r.GetCommand(name)
	if !exists {
		return fmt.Errorf("unknown subcommand: %s", name)
	}

	return command.Execute(ctx, args)
}
