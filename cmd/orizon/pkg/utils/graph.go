// Package utils provides graph-related utilities for dependency analysis.
// These functions handle dependency graph construction and traversal.
package utils

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// BuildDependencyGraph constructs a dependency graph from resolved packages.
// It returns a map where keys are package names and values are their dependencies.
func BuildDependencyGraph(ctx context.Context, reg packagemanager.Registry, resolved map[packagemanager.PackageID]struct {
	Version packagemanager.Version
	CID     packagemanager.CID
}) (map[string][]string, error) {
	graph := make(map[string][]string)
	concurrency := GetConcurrencyLimit()
	semaphore := make(chan struct{}, concurrency)

	var mu sync.Mutex

	// Create error group for concurrent operations
	g, gctx := errgroup.WithContext(ctx)

	for name, info := range resolved {
		name, info := name, info

		g.Go(func() error {
			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-semaphore }()

			// Build key for this package
			key := fmt.Sprintf("%s@%s", name, info.Version)

			// Fetch package data
			blob, err := reg.Fetch(gctx, info.CID)
			if err != nil {
				return fmt.Errorf("failed to fetch %s: %w", key, err)
			}

			// Extract dependencies
			dependencies := make([]string, 0)
			for _, dep := range blob.Manifest.Dependencies {
				if depInfo, ok := resolved[dep.Name]; ok {
					dependencies = append(dependencies, fmt.Sprintf("%s@%s", dep.Name, depInfo.Version))
				}
			}

			// Update graph with mutex protection
			mu.Lock()
			graph[key] = dependencies
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Ensure all packages with no dependencies are in the graph
	for name, info := range resolved {
		key := fmt.Sprintf("%s@%s", name, info.Version)
		if _, ok := graph[key]; !ok {
			graph[key] = nil
		}
	}

	return graph, nil
}

// GetRootDependencies extracts root dependencies from a manifest.
// These are the direct dependencies declared in the package manifest.
func GetRootDependencies(manifest types.Manifest) []string {
	roots := make([]string, 0, len(manifest.Dependencies))
	for name := range manifest.Dependencies {
		roots = append(roots, name)
	}
	return roots
}

// FindDependencyPath performs a breadth-first search to find a path from root to target.
// It returns the path as a slice of package names if found, or nil if no path exists.
func FindDependencyPath(graph map[string][]string, roots []string, target string) []string {
	type node struct {
		key  string
		path []string
	}

	visited := make(map[string]bool)
	queue := []node{}

	// Seed queue with root packages that exist in the graph
	for key := range graph {
		name := key
		if i := strings.IndexByte(key, '@'); i >= 0 {
			name = key[:i]
		}

		for _, root := range roots {
			if name == root {
				queue = append(queue, node{
					key:  key,
					path: []string{name},
				})
				visited[key] = true
				break
			}
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentName := current.key
		if i := strings.IndexByte(currentName, '@'); i >= 0 {
			currentName = currentName[:i]
		}

		if currentName == target {
			return current.path
		}

		// Add unvisited dependencies to queue
		for _, dependency := range graph[current.key] {
			if visited[dependency] {
				continue
			}

			visited[dependency] = true
			dependencyName := dependency
			if i := strings.IndexByte(dependency, '@'); i >= 0 {
				dependencyName = dependency[:i]
			}

			newPath := append(append([]string{}, current.path...), dependencyName)
			queue = append(queue, node{
				key:  dependency,
				path: newPath,
			})
		}
	}

	return nil
}
