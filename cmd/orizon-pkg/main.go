package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orizon-lang/orizon/internal/packagemanager"
)

func main() {
	var (
		workDir string
		command string
	)

	flag.StringVar(&workDir, "dir", ".", "working directory")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command = args[0]

	// Change to working directory
	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to change directory to %s: %v\n", workDir, err)
			os.Exit(1)
		}
	}

	switch command {
	case "init":
		handleInit(args[1:])
	case "add":
		handleAdd(args[1:])
	case "remove":
		handleRemove(args[1:])
	case "install":
		handleInstall(args[1:])
	case "build":
		handleBuild(args[1:])
	case "clean":
		handleClean(args[1:])
	case "list":
		handleList(args[1:])
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`orizon-pkg - Orizon Package Manager

Usage: orizon-pkg [options] <command> [args...]

Commands:
  init <name> [version]     Initialize a new package
  add <name> [version]      Add a dependency
  remove <name>             Remove a dependency
  install                   Install dependencies
  build                     Build the package
  clean                     Clean build artifacts
  list                      List workspace packages
  help                      Show this help

Options:
  -dir <directory>          Working directory (default: .)

Examples:
  orizon-pkg init mypackage 1.0.0
  orizon-pkg add utils ./shared/utils
  orizon-pkg add mathlib 1.2.3 --dev
  orizon-pkg install
  orizon-pkg build
  orizon-pkg list
`)
}

func handleInit(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: package name required\n")
		fmt.Fprintf(os.Stderr, "Usage: orizon-pkg init <name> [version]\n")
		os.Exit(1)
	}

	name := args[0]
	version := "1.0.0"
	if len(args) > 1 {
		version = args[1]
	}

	pm := packagemanager.NewLocalManager(".")
	if err := pm.InitPackage(name, version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize package: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized package '%s' version %s\n", name, version)
	fmt.Printf("Created package.oriz\n")
}

func handleAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: package name required\n")
		fmt.Fprintf(os.Stderr, "Usage: orizon-pkg add <name> [version] [--dev]\n")
		os.Exit(1)
	}

	name := args[0]
	version := "latest"
	dev := false

	// Parse remaining arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--dev" {
			dev = true
		} else if !strings.HasPrefix(arg, "--") {
			version = arg
		}
	}

	pm := packagemanager.NewLocalManager(".")
	if err := pm.AddDependency(name, version, dev); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to add dependency: %v\n", err)
		os.Exit(1)
	}

	depType := "dependency"
	if dev {
		depType = "dev dependency"
	}
	fmt.Printf("Added %s '%s' version %s\n", depType, name, version)
}

func handleRemove(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: package name required\n")
		fmt.Fprintf(os.Stderr, "Usage: orizon-pkg remove <name>\n")
		os.Exit(1)
	}

	name := args[0]

	pm := packagemanager.NewLocalManager(".")
	if err := pm.RemoveDependency(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to remove dependency: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed dependency '%s'\n", name)
}

func handleInstall(args []string) {
	pm := packagemanager.NewLocalManager(".")
	if err := pm.ResolveLocal(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to install dependencies: %v\n", err)
		os.Exit(1)
	}

	resolved := pm.GetResolvedDependencies()
	fmt.Printf("Installed %d dependencies:\n", len(resolved))
	for name, dep := range resolved {
		fmt.Printf("  %s@%s (%s) -> %s\n", name, dep.Version, dep.Source, dep.Path)
	}
}

func handleBuild(args []string) {
	pm := packagemanager.NewLocalManager(".")
	if err := pm.Build(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Build completed successfully")
	fmt.Println("Output: ./build/")
}

func handleClean(args []string) {
	pm := packagemanager.NewLocalManager(".")
	if err := pm.Clean(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: clean failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Cleaned build artifacts and cache")
}

func handleList(args []string) {
	pm := packagemanager.NewLocalManager(".")
	packages, err := pm.ListPackages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list packages: %v\n", err)
		os.Exit(1)
	}

	if len(packages) == 0 {
		fmt.Println("No packages found in workspace")
		return
	}

	fmt.Printf("Found %d packages in workspace:\n", len(packages))
	for _, pkg := range packages {
		absPath, _ := filepath.Abs(pkg)
		fmt.Printf("  %s (%s)\n", pkg, absPath)
	}
}
