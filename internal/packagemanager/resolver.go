package packagemanager

import (
	"fmt"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

// PackageID represents a package name.
type PackageID string

// Version is a semantic version string.
type Version string

// Dependency declares a constraint on another package.
type Dependency struct {
	Name       PackageID
	Constraint string // SemVer constraint (e.g., ">=1.2.0, <2.0.0")
}

// PackageVersion represents one concrete version of a package and its deps.
type PackageVersion struct {
	Name         PackageID
	Version      Version
	Dependencies []Dependency
}

// PackageIndex lists all published versions per package.
type PackageIndex map[PackageID][]PackageVersion

// Requirement describes a root constraint for resolution.
type Requirement struct {
	Name       PackageID
	Constraint string
}

// Resolution is the final mapping of package -> pinned version.
type Resolution map[PackageID]Version

// ResolveOptions controls resolution behavior.
type ResolveOptions struct {
	// PreferHigher, when true, picks highest versions that satisfy constraints; otherwise lowest.
	PreferHigher bool
	// MaxDepth guards against infinite recursion; 0 means unlimited.
	MaxDepth int
}

// ConflictError indicates that constraints cannot be satisfied.
type ConflictError struct {
	Package PackageID
	Reason  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("resolution conflict for %s: %s", e.Package, e.Reason)
}

// CycleError indicates a dependency cycle.
type CycleError struct {
	Stack []PackageID
}

func (e *CycleError) Error() string {
	parts := make([]string, len(e.Stack))
	for i, p := range e.Stack {
		parts[i] = string(p)
	}
	return fmt.Sprintf("dependency cycle detected: %s", strings.Join(parts, " -> "))
}

// Resolver performs version constraint resolution with backtracking.
type Resolver struct {
	index PackageIndex
	opts  ResolveOptions
}

// NewResolver constructs a resolver over a given index.
func NewResolver(index PackageIndex, opts ResolveOptions) *Resolver {
	return &Resolver{index: index, opts: opts}
}

// Resolve computes a version assignment satisfying all requirements and dependencies.
func (r *Resolver) Resolve(reqs []Requirement) (Resolution, error) {
	// Normalize requirements by package
	// We merge multiple constraints for the same package via intersection.
	merged := make(map[PackageID]*semver.Constraints)
	for _, q := range reqs {
		c, err := parseConstraint(q.Constraint)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", q.Name, err)
		}
		if ex, ok := merged[q.Name]; ok {
			// Masterminds/semver does not expose a direct Intersect API universally,
			// so we AND-join the textual constraints and re-parse.
			cc, err := semver.NewConstraint(ex.String() + ", " + c.String())
			if err != nil {
				return nil, fmt.Errorf("%s: %w", q.Name, err)
			}
			merged[q.Name] = cc
		} else {
			merged[q.Name] = c
		}
	}

	result := make(Resolution)
	visiting := make(map[PackageID]bool)

	// Determine initial worklist sorted by name for determinism
	roots := make([]PackageID, 0, len(merged))
	for id := range merged {
		roots = append(roots, id)
	}
	sort.Slice(roots, func(i, j int) bool { return string(roots[i]) < string(roots[j]) })

	for _, root := range roots {
		if _, ok := result[root]; ok {
			continue
		}
		if err := r.selectVersion(root, merged[root], result, visiting, 0); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// selectVersion chooses a version for pkg that satisfies con plus transitive deps.
func (r *Resolver) selectVersion(pkg PackageID, con *semver.Constraints, out Resolution, visiting map[PackageID]bool, depth int) error {
	if r.opts.MaxDepth > 0 && depth > r.opts.MaxDepth {
		return &ConflictError{Package: pkg, Reason: "max depth exceeded"}
	}
	if visiting[pkg] { // cycle
		// Build cycle stack for error message
		stack := make([]PackageID, 0, len(visiting)+1)
		for id, v := range visiting {
			if v {
				stack = append(stack, id)
			}
		}
		stack = append(stack, pkg)
		sort.Slice(stack, func(i, j int) bool { return string(stack[i]) < string(stack[j]) })
		return &CycleError{Stack: stack}
	}
	// Already pinned: verify compatibility
	if v, ok := out[pkg]; ok {
		sv, err := semver.NewVersion(string(v))
		if err != nil {
			return fmt.Errorf("%s pinned invalid version: %w", pkg, err)
		}
		if con != nil && !con.Check(sv) {
			return &ConflictError{Package: pkg, Reason: fmt.Sprintf("pinned %s violates %s", v, con.String())}
		}
		return nil
	}

	candidates := r.index[pkg]
	if len(candidates) == 0 {
		return &ConflictError{Package: pkg, Reason: "no versions in index"}
	}
	// Sort by semver per preference
	sort.Slice(candidates, func(i, j int) bool {
		vi := mustSemver(candidates[i].Version)
		vj := mustSemver(candidates[j].Version)
		if r.opts.PreferHigher {
			return vi.GreaterThan(vj)
		}
		return vi.LessThan(vj)
	})

	// Try candidates until one fits
	for _, pv := range candidates {
		sv := mustSemver(pv.Version)
		if con != nil && !con.Check(sv) {
			continue
		}
		// Tentatively pin
		out[pkg] = pv.Version
		visiting[pkg] = true
		// Resolve dependencies
		depsOK := true
		for _, d := range pv.Dependencies {
			dc, err := parseConstraint(d.Constraint)
			if err != nil {
				depsOK = false
				break
			}
			if err := r.selectVersion(d.Name, dc, out, visiting, depth+1); err != nil {
				depsOK = false
				// Unwind tentative selection for the dependency to permit other choices
				// Note: selections are overwritten in recursive calls; here we simply mark as failed.
				// We will backtrack by trying next candidate version of pkg.
				break
			}
		}
		visiting[pkg] = false
		if depsOK {
			return nil
		}
		// Backtrack this candidate
		delete(out, pkg)
	}
	return &ConflictError{Package: pkg, Reason: fmt.Sprintf("no candidate satisfies %s", humanConstraint(con))}
}

func parseConstraint(expr string) (*semver.Constraints, error) {
	if strings.TrimSpace(expr) == "" {
		// Empty means any version
		return semver.NewConstraint(">=0.0.0")
	}
	return semver.NewConstraint(expr)
}

func mustSemver(v Version) *semver.Version {
	sv, err := semver.NewVersion(string(v))
	if err != nil {
		panic(err)
	}
	return sv
}

func humanConstraint(c *semver.Constraints) string {
	if c == nil {
		return "<any>"
	}
	return c.String()
}
