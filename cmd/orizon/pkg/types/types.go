// Package types defines common data structures used across package management operations.
// These types provide a clean interface for manifest and lockfile handling.
package types

import (
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// Manifest represents a package manifest with dependencies and metadata.
// It defines the package identity and its dependency requirements.
type Manifest struct {
	// Dependencies maps package names to their version constraints
	Dependencies map[string]string `json:"dependencies,omitempty"`
	// Name is the package identifier
	Name string `json:"name"`
	// Version is the semantic version of the package
	Version string `json:"version"`
}

// Lockfile represents a resolved dependency tree with exact versions and content IDs.
// It ensures reproducible builds by pinning exact package versions.
type Lockfile struct {
	// Entries contains the resolved package information
	Entries []packagemanager.LockEntry `json:"entries"`
}

// RegistryContext holds the runtime context for package operations.
// It encapsulates the registry, signature store, and other shared resources.
type RegistryContext struct {
	// Registry provides access to package storage and retrieval
	Registry packagemanager.Registry
	// SignatureStore handles package signature verification
	SignatureStore packagemanager.SignatureStore
}

// CommandHandler defines the interface for package subcommand implementations.
// Each subcommand should implement this interface for consistent execution.
type CommandHandler interface {
	// Execute runs the command with the given arguments and context
	Execute(ctx RegistryContext, args []string) error
	// Description returns a human-readable description of the command
	Description() string
	// Usage returns usage information for the command
	Usage() string
}
