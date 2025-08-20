package build

import (
	"errors"
	"fmt"
	"runtime"
)

// Platform represents a target platform for cross compilation.
type Platform struct {
	GOOS   string
	GOARCH string
}

func (p Platform) String() string { return p.GOOS + "/" + p.GOARCH }

// Validate performs a conservative validation of the platform tuple.
func (p Platform) Validate() error {
	if p.GOOS == "" || p.GOARCH == "" {
		return errors.New("GOOS/GOARCH must be non-empty")
	}
	// Supported baseline combos without external toolchains.
	supported := map[string]bool{
		"linux/amd64":   true,
		"linux/arm64":   true,
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"windows/amd64": true,
		"windows/arm64": true,
	}
	if !supported[p.String()] {
		return fmt.Errorf("unsupported target platform: %s", p.String())
	}

	return nil
}

// CommandSpec describes a build command to be executed by a runner.
type CommandSpec struct {
	Env     map[string]string
	WorkDir string
	Cmd     string
	Args    []string
}

// GoToolchain provides commands for building Go packages for different platforms.
type GoToolchain struct {
	DefaultFlags   []string // e.g. ["-trimpath"]
	DefaultLdFlags []string // e.g. ["-s", "-w"]
}

// BuildPackage creates a CommandSpec that builds the given package path for target platform.
// output should be a path to the resulting binary or library.
func (tc GoToolchain) BuildPackage(target Platform, packagePath string, output string, extraFlags []string, extraLdFlags []string) (CommandSpec, error) {
	if err := target.Validate(); err != nil {
		return CommandSpec{}, err
	}

	args := []string{"build", "-o", output}
	// Flags.
	args = append(args, tc.DefaultFlags...)

	args = append(args, extraFlags...)
	// ldflags are space-separated after -ldflags.
	var ldflags string

	if len(tc.DefaultLdFlags) > 0 {
		for i, lf := range tc.DefaultLdFlags {
			if i > 0 {
				ldflags += " "
			}

			ldflags += lf
		}
	}

	if len(extraLdFlags) > 0 {
		if ldflags != "" {
			ldflags += " "
		}

		for i, lf := range extraLdFlags {
			if i > 0 {
				ldflags += " "
			}

			ldflags += lf
		}
	}

	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}

	args = append(args, packagePath)

	env := map[string]string{
		"GOOS":   target.GOOS,
		"GOARCH": target.GOARCH,
	}

	return CommandSpec{Cmd: "go", Args: args, Env: env}, nil
}

// HostPlatform returns the current runtime's GOOS/GOARCH.
func HostPlatform() Platform { return Platform{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH} }
