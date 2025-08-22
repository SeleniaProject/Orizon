// Package utils provides common utility functions for package management operations.
// These functions handle file I/O, path resolution, and other shared functionality.
package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// DefaultManifestPath defines the standard location for package manifest files
const DefaultManifestPath = "orizon.json"

// DefaultLockfilePath defines the standard location for package lock files
const DefaultLockfilePath = "orizon.lock"

// DefaultRegistryPath defines the default local registry storage path
const DefaultRegistryPath = ".orizon/registry"

// DefaultSignaturePath defines the default signature storage path
const DefaultSignaturePath = ".orizon/signatures"

// DefaultCachePath defines the default cache directory for downloaded packages
const DefaultCachePath = ".orizon/cache"

// ReadManifest reads and parses a package manifest from the default location.
// If the file doesn't exist, it returns a default manifest structure.
func ReadManifest() (types.Manifest, error) {
	data, err := os.ReadFile(DefaultManifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Return default manifest if file doesn't exist
			return types.Manifest{
				Name:         "app",
				Version:      "0.1.0",
				Dependencies: make(map[string]string),
			}, nil
		}
		return types.Manifest{}, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest types.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return types.Manifest{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Ensure dependencies map is initialized
	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}

	return manifest, nil
}

// WriteManifest writes a package manifest to the default location with proper formatting.
func WriteManifest(manifest types.Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(DefaultManifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// ReadLockfile reads and parses a package lockfile from the default location.
func ReadLockfile() (types.Lockfile, error) {
	data, err := os.ReadFile(DefaultLockfilePath)
	if err != nil {
		return types.Lockfile{}, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lockfile types.Lockfile
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return types.Lockfile{}, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	return lockfile, nil
}

// WriteLockfile writes a package lockfile to the default location.
func WriteLockfile(data []byte) error {
	if err := os.WriteFile(DefaultLockfilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}
	return nil
}

// EnsureDirectories creates necessary directories for package operations if they don't exist.
func EnsureDirectories() error {
	dirs := []string{
		filepath.Dir(DefaultRegistryPath),
		filepath.Dir(DefaultSignaturePath),
		DefaultCachePath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// SplitAt splits a string at the first occurrence of the '@' character.
// This is commonly used for parsing package name@version strings.
func SplitAt(s string) (name, version string) {
	if i := strings.IndexByte(s, '@'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// GetRegistryPath determines the registry path based on environment variables or defaults.
func GetRegistryPath() string {
	if envPath := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY")); envPath != "" {
		if !strings.HasPrefix(strings.ToLower(envPath), "http://") &&
			!strings.HasPrefix(strings.ToLower(envPath), "https://") {
			return envPath
		}
	}
	return DefaultRegistryPath
}

// GetSignatureStore creates a file-based signature store for the default location.
func GetSignatureStore() (packagemanager.SignatureStore, error) {
	store, err := packagemanager.NewFileSignatureStore(DefaultSignaturePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature store: %w", err)
	}
	return store, nil
}

// ResolveCurrent resolves manifest dependencies and returns pinned versions.
// This function handles the core dependency resolution logic.
func ResolveCurrent(ctx context.Context, reg packagemanager.Registry, manifest types.Manifest) (map[packagemanager.PackageID]struct {
	Version packagemanager.Version
	CID     packagemanager.CID
}, error) {
	reqs := make([]packagemanager.Requirement, 0, len(manifest.Dependencies))
	for name, constraint := range manifest.Dependencies {
		reqs = append(reqs, packagemanager.Requirement{
			Name:      packagemanager.PackageID(name),
			Constraint: constraint,
		})
	}

	manager := packagemanager.NewManager(reg)
	return manager.ResolveAndFetch(ctx, reqs, true)
}

// WriteLockFromManifest re-resolves dependencies and writes a new lockfile.
func WriteLockFromManifest(ctx context.Context, reg packagemanager.Registry, manifest types.Manifest) error {
	pinned, err := ResolveCurrent(ctx, reg, manifest)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	resolution := make(packagemanager.Resolution, len(pinned))
	for name, info := range pinned {
		resolution[name] = info.Version
	}

	_, data, err := packagemanager.GenerateLockfile(ctx, reg, resolution)
	if err != nil {
		return fmt.Errorf("failed to generate lockfile: %w", err)
	}

	return WriteLockfile(data)
}

// GetConcurrencyLimit returns the configured I/O concurrency limit.
// It respects the ORIZON_MAX_CONCURRENCY environment variable with sensible defaults.
func GetConcurrencyLimit() int {
	if envValue := strings.TrimSpace(os.Getenv("ORIZON_MAX_CONCURRENCY")); envValue != "" {
		if limit, err := strconv.Atoi(envValue); err == nil && limit > 0 {
			if limit > 1024 {
				return 1024
			}
			return limit
		}
	}

	// Default to GOMAXPROCS * 8 with reasonable bounds
	limit := runtime.GOMAXPROCS(0) * 8
	if limit < 4 {
		limit = 4
	}
	if limit > 1024 {
		limit = 1024
	}

	return limit
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
