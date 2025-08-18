package packagemanager

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalPackage represents a simple local Orizon package
type LocalPackage struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description,omitempty"`
	Author       string                 `json:"author,omitempty"`
	License      string                 `json:"license,omitempty"`
	Dependencies map[string]string      `json:"dependencies,omitempty"`
	DevDeps      map[string]string      `json:"dev_dependencies,omitempty"`
	Scripts      map[string]string      `json:"scripts,omitempty"`
	Repository   string                 `json:"repository,omitempty"`
	Keywords     []string               `json:"keywords,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LocalManifest represents the package.oriz manifest file
type LocalManifest struct {
	Package      LocalPackage      `json:"package"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	DevDeps      map[string]string `json:"dev_dependencies,omitempty"`
	BuildConfig  map[string]string `json:"build,omitempty"`
	TargetConfig map[string]string `json:"target,omitempty"`
}

// LocalResolvedDependency represents a resolved dependency with local path
type LocalResolvedDependency struct {
	Name     string
	Version  string
	Path     string
	Source   string // "local", "git", "registry"
	Checksum string
}

// LocalManager handles simple local Orizon package management
type LocalManager struct {
	rootDir      string
	cacheDir     string
	manifestPath string
	lockPath     string
	resolved     map[string]LocalResolvedDependency
}

// NewLocalManager creates a new local package manager instance
func NewLocalManager(rootDir string) *LocalManager {
	if rootDir == "" {
		rootDir = "."
	}

	return &LocalManager{
		rootDir:      rootDir,
		cacheDir:     filepath.Join(rootDir, ".orizon", "cache"),
		manifestPath: filepath.Join(rootDir, "package.oriz"),
		lockPath:     filepath.Join(rootDir, "package-lock.oriz"),
		resolved:     make(map[string]LocalResolvedDependency),
	}
}

// LoadManifest loads the package manifest from package.oriz
func (pm *LocalManager) LoadManifest() (*LocalManifest, error) {
	data, err := os.ReadFile(pm.manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("package.oriz not found in %s", pm.rootDir)
		}
		return nil, fmt.Errorf("failed to read package.oriz: %w", err)
	}

	var manifest LocalManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse package.oriz: %w", err)
	}

	return &manifest, nil
}

// SaveManifest saves the package manifest to package.oriz
func (pm *LocalManager) SaveManifest(manifest *LocalManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize manifest: %w", err)
	}

	if err := os.WriteFile(pm.manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write package.oriz: %w", err)
	}

	return nil
}

// InitPackage creates a new package.oriz file in the current directory
func (pm *LocalManager) InitPackage(name, version string) error {
	if _, err := os.Stat(pm.manifestPath); err == nil {
		return fmt.Errorf("package.oriz already exists")
	}

	manifest := &LocalManifest{
		Package: LocalPackage{
			Name:        name,
			Version:     version,
			Description: "",
			License:     "MIT",
		},
		Dependencies: make(map[string]string),
		DevDeps:      make(map[string]string),
		BuildConfig:  make(map[string]string),
	}

	return pm.SaveManifest(manifest)
}

// ResolveLocal resolves dependencies using local file system paths
func (pm *LocalManager) ResolveLocal() error {
	manifest, err := pm.LoadManifest()
	if err != nil {
		return err
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(pm.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Resolve dependencies
	allDeps := make(map[string]string)
	for name, version := range manifest.Dependencies {
		allDeps[name] = version
	}
	for name, version := range manifest.DevDeps {
		allDeps[name] = version
	}

	for name, version := range allDeps {
		resolved, err := pm.resolveLocalDependency(name, version)
		if err != nil {
			return fmt.Errorf("failed to resolve dependency %s@%s: %w", name, version, err)
		}
		pm.resolved[name] = resolved
	}

	// Save lock file
	return pm.saveLockFile()
}

// resolveLocalDependency attempts to resolve a dependency locally
func (pm *LocalManager) resolveLocalDependency(name, version string) (LocalResolvedDependency, error) {
	// Try different local resolution strategies

	// 1. Check if it's a relative path
	if strings.HasPrefix(version, "./") || strings.HasPrefix(version, "../") {
		absPath, err := filepath.Abs(filepath.Join(pm.rootDir, version))
		if err != nil {
			return LocalResolvedDependency{}, err
		}

		if _, err := os.Stat(filepath.Join(absPath, "package.oriz")); err == nil {
			return LocalResolvedDependency{
				Name:    name,
				Version: version,
				Path:    absPath,
				Source:  "local",
			}, nil
		}
	}

	// 2. Check in local cache
	cacheDir := filepath.Join(pm.cacheDir, name, version)
	if _, err := os.Stat(filepath.Join(cacheDir, "package.oriz")); err == nil {
		return LocalResolvedDependency{
			Name:    name,
			Version: version,
			Path:    cacheDir,
			Source:  "local",
		}, nil
	}

	// 3. Look for local workspace packages
	workspacePackages := []string{
		filepath.Join(pm.rootDir, "packages", name),
		filepath.Join(pm.rootDir, "..", name),
		filepath.Join(pm.rootDir, "vendor", name),
	}

	for _, pkgPath := range workspacePackages {
		if _, err := os.Stat(filepath.Join(pkgPath, "package.oriz")); err == nil {
			absPath, err := filepath.Abs(pkgPath)
			if err != nil {
				return LocalResolvedDependency{}, err
			}

			return LocalResolvedDependency{
				Name:    name,
				Version: version,
				Path:    absPath,
				Source:  "local",
			}, nil
		}
	}

	return LocalResolvedDependency{}, fmt.Errorf("package %s@%s not found locally", name, version)
}

// Build performs a basic build using resolved dependencies
func (pm *LocalManager) Build() error {
	if err := pm.ResolveLocal(); err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	manifest, err := pm.LoadManifest()
	if err != nil {
		return err
	}

	// Basic build logic - in a real implementation, this would invoke the Orizon compiler
	buildDir := filepath.Join(pm.rootDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Generate build metadata
	buildInfo := map[string]interface{}{
		"package":      manifest.Package.Name,
		"version":      manifest.Package.Version,
		"dependencies": pm.resolved,
		"built_at":     time.Now().UTC(),
	}

	buildInfoData, err := json.MarshalIndent(buildInfo, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(buildDir, "build-info.json"), buildInfoData, 0644)
}

// AddDependency adds a new dependency to the manifest
func (pm *LocalManager) AddDependency(name, version string, dev bool) error {
	manifest, err := pm.LoadManifest()
	if err != nil {
		return err
	}

	if dev {
		if manifest.DevDeps == nil {
			manifest.DevDeps = make(map[string]string)
		}
		manifest.DevDeps[name] = version
	} else {
		if manifest.Dependencies == nil {
			manifest.Dependencies = make(map[string]string)
		}
		manifest.Dependencies[name] = version
	}

	return pm.SaveManifest(manifest)
}

// RemoveDependency removes a dependency from the manifest
func (pm *LocalManager) RemoveDependency(name string) error {
	manifest, err := pm.LoadManifest()
	if err != nil {
		return err
	}

	delete(manifest.Dependencies, name)
	delete(manifest.DevDeps, name)

	return pm.SaveManifest(manifest)
}

// ListPackages lists all packages in the workspace
func (pm *LocalManager) ListPackages() ([]string, error) {
	var packages []string

	// Walk the workspace to find package.oriz files
	err := filepath.WalkDir(pm.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Name() == "package.oriz" {
			relPath, err := filepath.Rel(pm.rootDir, filepath.Dir(path))
			if err != nil {
				return err
			}
			packages = append(packages, relPath)
		}

		return nil
	})

	return packages, err
}

// GetResolvedDependencies returns the current resolved dependencies
func (pm *LocalManager) GetResolvedDependencies() map[string]LocalResolvedDependency {
	return pm.resolved
}

// saveLockFile saves the current resolved dependencies to package-lock.oriz
func (pm *LocalManager) saveLockFile() error {
	lockData := map[string]interface{}{
		"version":      "1.0",
		"generated_at": time.Now().UTC(),
		"dependencies": pm.resolved,
	}

	data, err := json.MarshalIndent(lockData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pm.lockPath, data, 0644)
}

// Clean removes build artifacts and cache
func (pm *LocalManager) Clean() error {
	buildDir := filepath.Join(pm.rootDir, "build")
	if err := os.RemoveAll(buildDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove build directory: %w", err)
	}

	if err := os.RemoveAll(pm.cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	return nil
}
