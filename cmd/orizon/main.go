// Package main provides the main entry point for the Orizon CLI tool.
// It handles command parsing, subcommand routing, and delegates to specific
// command handlers while maintaining clean separation of concerns.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	semver "github.com/Masterminds/semver/v3"
	"golang.org/x/sync/errgroup"

	"github.com/orizon-lang/orizon/internal/cli"
	pm "github.com/orizon-lang/orizon/internal/packagemanager"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/commands"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/cmd/orizon/pkg/utils"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	case "help", "-h", "--help":
		usage()
	case "version", "-v", "--version":
		// Check for JSON output flag
		jsonOutput := false
		for _, arg := range args {
			if arg == "--json" || arg == "-j" {
				jsonOutput = true
				break
			}
		}
		cli.PrintVersion("Orizon CLI", jsonOutput)
		os.Exit(0)
	case "build":
		must(runToolOrRun("orizon-compiler", args...))
	case "test":
		// Prefer dedicated tester; fallback to `go test ./...`
		if err := runToolOrRun("orizon-test", args...); err != nil {
			fmt.Println("[info] falling back to `go test ./...`")
			must(runCmd(context.Background(), "go", append([]string{"test", "./..."}, args...)...))
		}
	case "fmt":
		must(runToolOrRun("orizon-fmt", args...))
	case "fuzz":
		must(runToolOrRun("orizon-fuzz", args...))
	case "mockgen":
		must(runToolOrRun("orizon-mockgen", args...))
	case "summary":
		must(runToolOrRun("orizon-summary", args...))
	case "lsp":
		must(runToolOrRun("orizon-lsp", args...))
	case "repl":
		must(runToolOrRun("orizon-repl", args...))
	case "doc":
		must(runToolOrRun("orizon-doc", args...))
	case "profile":
		must(runToolOrRun("orizon-profile", args...))
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		timeout := fs.Duration("timeout", 0, "optional timeout (e.g., 30s)")
		_ = fs.Parse(args)

		rest := fs.Args()
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "usage: orizon run <file.oriz> [args...]")
			os.Exit(2)
		}

		file := rest[0]
		runArgs := rest[1:]
		ctx := context.Background()

		if *timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, *timeout)
			defer cancel()
		}
		// Delegate to compiler with the source; additional args are passed through.
		must(runCmd(ctx, resolveTool("orizon-compiler"), append([]string{file}, runArgs...)...))
	case "pkg":
		pkg(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		usage()
		os.Exit(2)
	}
}

func usage() {
	commands := []cli.CommandInfo{
		{
			Name:        "build",
			Description: "Build Orizon sources",
		},
		{
			Name:        "run",
			Description: "Compile and run a source file",
		},
		{
			Name:        "test",
			Description: "Run tests",
		},
		{
			Name:        "fmt",
			Description: "Format source code",
		},
		{
			Name:        "lsp",
			Description: "Start language server",
		},
		{
			Name:        "fuzz",
			Description: "Run fuzzer",
		},
		{
			Name:        "mockgen",
			Description: "Generate mocks",
		},
		{
			Name:        "summary",
			Description: "Print project summary",
		},
		{
			Name:        "repl",
			Description: "Start interactive REPL",
		},
		{
			Name:        "doc",
			Description: "Generate documentation",
		},
		{
			Name:        "profile",
			Description: "Performance profiling",
		},
		{
			Name:        "pkg",
			Description: "Package operations (init, publish, add, etc.)",
		},
	}

	cli.PrintUsage("orizon", commands)
}

type manifest struct {
	Dependencies map[string]string `json:"dependencies,omitempty"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
}

type lockfile struct {
	Entries []pm.LockEntry `json:"entries"`
}

// utils_pkg provides a bridge to the refactored utilities
type utils_pkg struct{}

// CreateRegistryContext creates a registry context using the refactored utilities
func (u *utils_pkg) CreateRegistryContext() (types.RegistryContext, error) {
	return utils.CreateRegistryContext()
}

func pkg(args []string) {
	// Import the new command registry and utilities
	registry := commands.NewRegistry()
	utils := &utils_pkg{} // This would need to be defined

	if len(args) == 0 {
		registry.ListCommands()
		return
	}

	// Create registry context
	ctx, err := utils.CreateRegistryContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create registry context: %v\n", err)
		os.Exit(1)
	}

	// Execute the command
	subcommand := args[0]
	subArgs := args[1:]

	if err := registry.ExecuteCommand(subcommand, ctx, subArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command '%s': %v\n", subcommand, err)
		os.Exit(1)
	}
	}

			return
		}

		switch args[1] {
		case "login":
			fs := flag.NewFlagSet("auth login", flag.ExitOnError)
			regURL := fs.String("registry", os.Getenv("ORIZON_REGISTRY"), "registry url (http/https)")
			token := fs.String("token", "", "bearer token")
			_ = fs.Parse(args[2:])

			if strings.TrimSpace(*regURL) == "" || strings.TrimSpace(*token) == "" {
				fmt.Fprintln(os.Stderr, "--registry and --token required")
				os.Exit(2)
			}

			_ = os.MkdirAll(".orizon", 0o755)
			path := filepath.Join(".orizon", "credentials.json")
			// read existing.
			creds := struct {
				Registries map[string]struct {
					Token string `json:"token"`
				} `json:"registries"`
			}{Registries: map[string]struct {
				Token string `json:"token"`
			}{}}
			if b, err := os.ReadFile(path); err == nil {
				_ = json.Unmarshal(b, &creds)
			}

			creds.Registries[strings.TrimRight(*regURL, "/")] = struct {
				Token string `json:"token"`
			}{Token: *token}

			b, _ := json.MarshalIndent(creds, "", "  ")
			if err := os.WriteFile(path, b, 0o600); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Println("credentials updated")
		default:
			fmt.Fprintln(os.Stderr, "unknown auth subcommand")
			os.Exit(2)
		}

		return
	case "serve":
		// Start HTTP registry server backed by local file registry (.orizon/registry or ORIZON_REGISTRY path)
		regPath := regEnv
		if regPath == "" || strings.HasPrefix(strings.ToLower(regPath), "http") {
			regPath = filepath.Join(".orizon", "registry")
		}

		fileReg, err := pm.NewFileRegistry(regPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := fs.String("addr", ":9321", "listen address")
		token := fs.String("token", "", "optional bearer token (also reads ORIZON_REGISTRY_TOKEN)")
		tlsCert := fs.String("tls-cert", "", "path to TLS certificate (PEM)")
		tlsKey := fs.String("tls-key", "", "path to TLS private key (PEM)")
		_ = fs.Parse(args[1:])
		// if --token provided, set env for server process lifetime.
		if *token != "" {
			_ = os.Setenv("ORIZON_REGISTRY_TOKEN", *token)
		}

		useTLS := strings.TrimSpace(*tlsCert) != "" && strings.TrimSpace(*tlsKey) != ""
		if useTLS {
			fmt.Printf("serving registry on https://%s (root=%s) auth=%v\n", *addr, regPath, os.Getenv("ORIZON_REGISTRY_TOKEN") != "")

			if err := pm.StartHTTPServerTLS(fileReg, *addr, *tlsCert, *tlsKey); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			return
		}

		fmt.Printf("serving registry on http://%s (root=%s) auth=%v\n", *addr, regPath, os.Getenv("ORIZON_REGISTRY_TOKEN") != "")
		// blocking.
		if err := pm.StartHTTPServer(fileReg, *addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return
	case "init":
		// Create or update manifest skeleton.
		if _, err := os.Stat("orizon.json"); errors.Is(err, os.ErrNotExist) {
			m := manifest{Name: "app", Version: "0.1.0", Dependencies: map[string]string{}}
			b, _ := json.MarshalIndent(m, "", "  ")
			_ = os.WriteFile("orizon.json", b, 0o644)

			fmt.Println("created orizon.json")
		} else {
			fmt.Println("orizon.json exists")
		}
		// Ensure .orizon dir
		_ = os.MkdirAll(".orizon", 0o755)
	case "publish":
		fs := flag.NewFlagSet("publish", flag.ExitOnError)
		name := fs.String("name", "", "package name")
		ver := fs.String("version", "", "package version (semver)")
		file := fs.String("file", "", "payload file (e.g., .tar) to publish")
		_ = fs.Parse(args[1:])

		if *name == "" || *ver == "" || *file == "" {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg publish --name <id> --version <semver> --file <path>")
			os.Exit(2)
		}

		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read file:", err)
			os.Exit(1)
		}

		blob := pm.PackageBlob{Manifest: pm.PackageManifest{Name: pm.PackageID(*name), Version: pm.Version(*ver)}, Data: data}

		cid, err := reg.Publish(ctx, blob)
		if err != nil {
			fmt.Fprintln(os.Stderr, "publish:", err)
			os.Exit(1)
		}

		fmt.Printf("published %s@%s cid=%s\n", *name, *ver, cid)
	case "add":
		// Add dependency to manifest (local project).
		fs := flag.NewFlagSet("add", flag.ExitOnError)
		dep := fs.String("dep", "", "dependency in form name@constraint (e.g., foo@^1.2.0)")
		_ = fs.Parse(args[1:])

		if *dep == "" {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg add --dep name@constraint")
			os.Exit(2)
		}

		m := readManifest()
		n, c := splitAt(*dep)

		if m.Dependencies == nil {
			m.Dependencies = map[string]string{}
		}

		m.Dependencies[n] = c
		writeManifest(m)
		fmt.Printf("added %s -> %s\n", n, c)
	case "resolve":
		// Resolve current manifest dependencies against registry and print plan.
		m := readManifest()
		reqs := make([]pm.Requirement, 0, len(m.Dependencies))

		for name, con := range m.Dependencies {
			reqs = append(reqs, pm.Requirement{Name: pm.PackageID(name), Constraint: con})
		}

		man := pm.NewManager(reg)

		out, err := man.ResolveAndFetch(ctx, reqs, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "resolve:", err)
			os.Exit(1)
		}
		// Print result as JSON name -> {version,cid}.
		type pinned struct {
			Version string `json:"version"`
			CID     string `json:"cid"`
		}

		res := map[string]pinned{}
		for n, v := range out {
			res[string(n)] = pinned{Version: string(v.Version), CID: string(v.CID)}
		}

		b, _ := json.MarshalIndent(res, "", "  ")
		os.Stdout.Write(b)
		fmt.Println()
	case "lock":
		// Generate lockfile from current resolved state.
		m := readManifest()
		reqs := make([]pm.Requirement, 0, len(m.Dependencies))

		for name, con := range m.Dependencies {
			reqs = append(reqs, pm.Requirement{Name: pm.PackageID(name), Constraint: con})
		}

		man := pm.NewManager(reg)

		out, err := man.ResolveAndFetch(ctx, reqs, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "resolve:", err)
			os.Exit(1)
		}
		// Convert to Resolution.
		rr := make(pm.Resolution)
		for n, v := range out {
			rr[n] = v.Version
		}

		lock, b, err := pm.GenerateLockfile(ctx, reg, rr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "lock:", err)
			os.Exit(1)
		}

		if err := os.WriteFile("orizon.lock", b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write lock:", err)
			os.Exit(1)
		}

		fmt.Printf("lockfile written (%d entries)\n", len(lock.Entries))
	case "verify":
		b, err := os.ReadFile("orizon.lock")
		if err != nil {
			fmt.Fprintln(os.Stderr, "verify read:", err)
			os.Exit(1)
		}

		var lf pm.Lockfile
		if err := json.Unmarshal(b, &lf); err != nil {
			fmt.Fprintln(os.Stderr, "verify parse:", err)
			os.Exit(1)
		}

		if err := pm.VerifyLockfile(ctx, reg, lf); err != nil {
			fmt.Fprintln(os.Stderr, "verify:", err)
			os.Exit(1)
		}

		fmt.Println("lockfile verified")
	case "list":
		// List all known manifests in registry (name@version).
		mans, err := reg.All(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "list:", err)
			os.Exit(1)
		}

		for _, m := range mans {
			fmt.Printf("%s@%s\n", m.Name, m.Version)
		}
	case "fetch":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg fetch <name>@<constraint>")
			os.Exit(2)
		}

		name, con := splitAt(args[1])

		c, err := semver.NewConstraint(con)
		if err != nil {
			fmt.Fprintln(os.Stderr, "constraint:", err)
			os.Exit(1)
		}

		cid, man, err := reg.Find(ctx, pm.PackageID(name), c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "find:", err)
			os.Exit(1)
		}

		blob, err := reg.Fetch(ctx, cid)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fetch:", err)
			os.Exit(1)
		}

		out := filepath.Join(".orizon", "cache", string(cid))
		_ = os.MkdirAll(filepath.Dir(out), 0o755)

		if err := os.WriteFile(out, blob.Data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}

		fmt.Printf("fetched %s@%s -> %s\n", man.Name, man.Version, out)
	case "update":
		// Re-resolve and rewrite lockfile.
		fs := flag.NewFlagSet("update", flag.ExitOnError)
		only := fs.String("dep", "", "comma-separated dependency names to update (others pinned to lock)")
		_ = fs.Parse(args[1:])
		m := readManifest()
		deps := []string{}

		if strings.TrimSpace(*only) != "" {
			for _, t := range strings.Split(*only, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					deps = append(deps, t)
				}
			}
		}

		if len(deps) == 0 {
			if err := writeLockFromManifest(ctx, reg, m); err != nil {
				fmt.Fprintln(os.Stderr, "update:", err)
				os.Exit(1)
			}

			fmt.Println("dependencies updated and lockfile rewritten")

			break
		}
		// targeted update: keep others pinned to lockfile versions.
		// load lockfile if present; otherwise compute pinned now.
		var locked map[string]string // name -> version

		if b, err := os.ReadFile("orizon.lock"); err == nil {
			var lf pm.Lockfile
			if json.Unmarshal(b, &lf) == nil {
				locked = map[string]string{}
				for _, e := range lf.Entries {
					locked[string(e.Name)] = string(e.Version)
				}
			}
		}

		if locked == nil {
			cur, err := resolveCurrent(ctx, reg, m)
			if err != nil {
				fmt.Fprintln(os.Stderr, "update resolve:", err)
				os.Exit(1)
			}

			locked = map[string]string{}
			for n, v := range cur {
				locked[string(n)] = string(v.Version)
			}
		}
		// build requirements: selected use manifest constraints; others pinned to =version.
		sel := map[string]bool{}
		for _, d := range deps {
			sel[d] = true
		}

		reqs := make([]pm.Requirement, 0, len(m.Dependencies))

		for name := range m.Dependencies {
			if sel[name] {
				reqs = append(reqs, pm.Requirement{Name: pm.PackageID(name), Constraint: m.Dependencies[name]})
			} else if v, ok := locked[name]; ok {
				reqs = append(reqs, pm.Requirement{Name: pm.PackageID(name), Constraint: "=" + v})
			} else {
				// fallback to manifest constraint.
				reqs = append(reqs, pm.Requirement{Name: pm.PackageID(name), Constraint: m.Dependencies[name]})
			}
		}

		man := pm.NewManager(reg)

		out, err := man.ResolveAndFetch(ctx, reqs, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "update resolve:", err)
			os.Exit(1)
		}
		// write lock.
		rr := make(pm.Resolution)
		for n, v := range out {
			rr[n] = v.Version
		}

		_, b, err := pm.GenerateLockfile(ctx, reg, rr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "update lock:", err)
			os.Exit(1)
		}

		if err := os.WriteFile("orizon.lock", b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "update write:", err)
			os.Exit(1)
		}

		fmt.Printf("updated %s and rewrote lockfile\n", *only)
	case "remove":
		fs := flag.NewFlagSet("remove", flag.ExitOnError)
		name := fs.String("dep", "", "dependency name to remove")
		relock := fs.Bool("lock", true, "rewrite lockfile after removal")
		_ = fs.Parse(args[1:])

		if *name == "" {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg remove --dep <name> [--lock=true]")
			os.Exit(2)
		}

		m := readManifest()
		delete(m.Dependencies, *name)
		writeManifest(m)

		if *relock {
			if err := writeLockFromManifest(ctx, reg, m); err != nil {
				fmt.Fprintln(os.Stderr, "remove lock:", err)
				os.Exit(1)
			}
		}

		fmt.Printf("removed %s\n", *name)
	case "graph":
		// Build resolved graph and print edges.
		fs := flag.NewFlagSet("graph", flag.ExitOnError)
		dot := fs.Bool("dot", false, "print Graphviz DOT instead of edges")
		outPath := fs.String("output", "", "optional output file path")
		_ = fs.Parse(args[1:])
		m := readManifest()

		pinned, err := resolveCurrent(ctx, reg, m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		g, err := buildGraph(ctx, reg, pinned)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var b strings.Builder
		if *dot {
			// Graphviz DOT with roots highlighted.
			b.WriteString("digraph deps {\n")
			b.WriteString("  rankdir=LR;\n")

			roots := rootsFromManifest(m)
			isRoot := map[string]bool{}

			for _, r := range roots {
				isRoot[r] = true
			}

			for from, tos := range g {
				name := from
				if i := strings.IndexByte(name, '@'); i >= 0 {
					name = name[:i]
				}

				if isRoot[name] {
					fmt.Fprintf(&b, "  \"%s\" [shape=box,style=bold];\n", from)
				} else if len(tos) == 0 {
					fmt.Fprintf(&b, "  \"%s\";\n", from)
				}

				for _, to := range tos {
					fmt.Fprintf(&b, "  \"%s\" -> \"%s\";\n", from, to)
				}
			}

			b.WriteString("}\n")
		} else {
			for from, tos := range g {
				if len(tos) == 0 {
					fmt.Fprintln(&b, from)
				} else {
					fmt.Fprintf(&b, "%s -> %s\n", from, strings.Join(tos, ", "))
				}
			}
		}

		if *outPath != "" {
			if err := os.WriteFile(*outPath, []byte(b.String()), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write:", err)
				os.Exit(1)
			}
		} else {
			fmt.Print(b.String())
		}
	case "why":
		// Explain why a package is included (path from roots).
		fs := flag.NewFlagSet("why", flag.ExitOnError)
		verbose := fs.Bool("verbose", false, "print versions along the path")
		showCID := fs.Bool("cid", false, "include CIDs when --verbose is set")
		_ = fs.Parse(args[1:])

		rest := fs.Args()
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg why [--verbose] [--cid] <name>")
			os.Exit(2)
		}

		target := rest[0]
		m := readManifest()

		pinned, err := resolveCurrent(ctx, reg, m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		g, err := buildGraph(ctx, reg, pinned)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		path := whyPath(g, rootsFromManifest(m), target)
		if len(path) == 0 {
			fmt.Printf("no path to %s\n", target)
		} else {
			if *verbose {
				parts := make([]string, 0, len(path))

				for _, name := range path {
					if pv, ok := pinned[pm.PackageID(name)]; ok {
						if *showCID {
							parts = append(parts, fmt.Sprintf("%s@%s (%s)", name, pv.Version, pv.CID))
						} else {
							parts = append(parts, fmt.Sprintf("%s@%s", name, pv.Version))
						}
					} else {
						parts = append(parts, name)
					}
				}

				fmt.Println(strings.Join(parts, " -> "))
			} else {
				fmt.Println(strings.Join(path, " -> "))
			}
		}
	case "outdated":
		// Show current vs latest allowed vs latest overall for manifest deps.
		m := readManifest()

		pinned, err := resolveCurrent(ctx, reg, m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("name  current  allowed  latest")

		for name, con := range m.Dependencies {
			cur := string(pinned[pm.PackageID(name)].Version)
			// highest satisfying.
			c, _ := semver.NewConstraint(con)
			mans, _ := reg.List(ctx, pm.PackageID(name))

			var bestAllowed, bestOverall string

			var bestAllowedVer, bestOverallVer *semver.Version

			for _, mf := range mans {
				sv, err := semver.NewVersion(string(mf.Version))
				if err != nil {
					continue
				}

				if bestOverallVer == nil || sv.GreaterThan(bestOverallVer) {
					bestOverallVer = sv
					bestOverall = sv.String()
				}

				if c != nil && !c.Check(sv) {
					continue
				}

				if bestAllowedVer == nil || sv.GreaterThan(bestAllowedVer) {
					bestAllowedVer = sv
					bestAllowed = sv.String()
				}
			}

			if bestAllowed == "" {
				bestAllowed = "-"
			}

			if bestOverall == "" {
				bestOverall = "-"
			}

			fmt.Printf("%s  %s  %s  %s\n", name, cur, bestAllowed, bestOverall)
		}
	case "vendor":
		// Download all lockfile entries into .orizon/vendor for offline builds
		b, err := os.ReadFile("orizon.lock")
		if err != nil {
			fmt.Fprintln(os.Stderr, "vendor read lock:", err)
			os.Exit(1)
		}

		var lf pm.Lockfile
		if err := json.Unmarshal(b, &lf); err != nil {
			fmt.Fprintln(os.Stderr, "vendor parse lock:", err)
			os.Exit(1)
		}

		dir := filepath.Join(".orizon", "vendor")
		_ = os.MkdirAll(dir, 0o755)

		for _, e := range lf.Entries {
			blob, err := reg.Fetch(ctx, e.CID)
			if err != nil {
				fmt.Fprintln(os.Stderr, "vendor fetch:", e.Name, err)
				os.Exit(1)
			}

			name := fmt.Sprintf("%s-%s.blob", e.Name, e.Version)
			if err := os.WriteFile(filepath.Join(dir, name), blob.Data, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "vendor write:", err)
				os.Exit(1)
			}
		}

		fmt.Printf("vendored %d packages into %s\n", len(lf.Entries), dir)
	case "sign":
		// Sign a CID with an ephemeral self-signed root (demo). In real projects, load keys from disk.
		fs := flag.NewFlagSet("sign", flag.ExitOnError)
		cidStr := fs.String("cid", "", "content id to sign")
		subject := fs.String("subject", "dev", "certificate subject")
		_ = fs.Parse(args[1:])

		if *cidStr == "" {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg sign --cid <cid> [--subject subj]")
			os.Exit(2)
		}

		pub, priv, err := pm.GenerateEd25519Keypair()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		root, err := pm.SelfSignRoot(*subject, pub, priv, 24*60*60*365*10)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// Build descriptor and sign.
		bundle, err := pm.SignPackage(ctx, reg, pm.CID(*cidStr), priv, []pm.Certificate{root}, sigStore)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("signed %s with key %s (chain len %d)\n", *cidStr, bundle.KeyID, len(bundle.Chain))
	case "verify-sig":
		fs := flag.NewFlagSet("verify-sig", flag.ExitOnError)
		cidStr := fs.String("cid", "", "content id to verify")
		_ = fs.Parse(args[1:])

		if *cidStr == "" {
			fmt.Fprintln(os.Stderr, "usage: orizon pkg verify-sig --cid <cid>")
			os.Exit(2)
		}
		// Build trust store by trusting all roots from bundles on disk (demo).
		ts := pm.NewTrustStore()

		bundles, _ := sigStore.List(pm.CID(*cidStr))
		for _, b := range bundles {
			if len(b.Chain) > 0 {
				ts.AddRoot(b.Chain[len(b.Chain)-1].PublicKey)
			}
		}

		if err := pm.VerifyPackage(ctx, reg, ts, pm.CID(*cidStr), sigStore); err != nil {
			fmt.Fprintln(os.Stderr, "verify-sig:", err)
			os.Exit(1)
		}

		fmt.Println("signature verified (at least one)")
	case "audit":
		// Simple vulnerability audit using in-memory advisory list (demo).
		mans, err := reg.All(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		_ = pm.NewInMemoryAdvisoryScanner()
		// No built-in advisories; this is a placeholder for later wiring. Show non-failing audit summary
		fmt.Printf("audited %d packages: no advisories configured\n", len(mans))
	default:
		fmt.Fprintf(os.Stderr, "unknown pkg subcommand: %s\n", args[0])
		os.Exit(2)
	}
}
	b, err := os.ReadFile("orizon.json")
	if err != nil {
		// default manifest if missing.
		return manifest{Name: "app", Version: "0.1.0", Dependencies: map[string]string{}}
	}

	var m manifest
	_ = json.Unmarshal(b, &m)

	if m.Dependencies == nil {
		m.Dependencies = map[string]string{}
	}

	return m
}

func writeManifest(m manifest) {
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile("orizon.json", b, 0o644)
}

func splitAt(s string) (string, string) {
	i := strings.IndexByte(s, '@')
	if i < 0 {
		return s, ""
	}

	return s[:i], s[i+1:]
}

// runToolOrRun tries to run a built binary under build/, then falls back to `go run ./cmd/<tool>`.
func runToolOrRun(tool string, args ...string) error {
	exe := resolveTool(tool)
	if fileExists(exe) {
		return runCmd(context.Background(), exe, args...)
	}
	// Fallback to `go run ./cmd/<tool>`
	pkgPath := filepath.Join("./cmd", tool)

	return runCmd(context.Background(), "go", append([]string{"run", pkgPath}, args...)...)
}

func resolveTool(tool string) string {
	bin := tool
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	return filepath.Join("build", bin)
}
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	return c.Run()
}

func must(err error) {
	if err != nil {
		os.Exit(codeFromErr(err))
	}
}

func codeFromErr(err error) int {
	if err == nil {
		return 0
	}
	// best-effort.
	return 1
}

func fileExists(path string) bool {
	st, err := os.Stat(path)

	return err == nil && !st.IsDir()
}

// --- pkg helpers ---.
// resolveCurrent resolves the manifest deps and returns pinned versions map.
func resolveCurrent(ctx context.Context, reg pm.Registry, m manifest) (map[pm.PackageID]struct {
	Version pm.Version
	CID     pm.CID
}, error,
) {
	reqs := make([]pm.Requirement, 0, len(m.Dependencies))
	for name, con := range m.Dependencies {
		reqs = append(reqs, pm.Requirement{Name: pm.PackageID(name), Constraint: con})
	}

	man := pm.NewManager(reg)

	return man.ResolveAndFetch(ctx, reqs, true)
}

// writeLockFromManifest re-resolves and writes orizon.lock.
func writeLockFromManifest(ctx context.Context, reg pm.Registry, m manifest) error {
	pinned, err := resolveCurrent(ctx, reg, m)
	if err != nil {
		return err
	}

	rr := make(pm.Resolution, len(pinned))
	for n, v := range pinned {
		rr[n] = v.Version
	}

	_, b, err := pm.GenerateLockfile(ctx, reg, rr)
	if err != nil {
		return err
	}

	return os.WriteFile("orizon.lock", b, 0o644)
}

// buildGraph: name@version -> []name@version edges.
func buildGraph(ctx context.Context, reg pm.Registry, pinned map[pm.PackageID]struct {
	Version pm.Version
	CID     pm.CID
},
) (map[string][]string, error) {
	out := make(map[string][]string)
	lim := ioConcurrency()
	sem := make(chan struct{}, lim)
	g, gctx := errgroup.WithContext(ctx)

	var mu sync.Mutex

	for name, pv := range pinned {
		name, pv := name, pv

		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}

			defer func() { <-sem }()

			key := fmt.Sprintf("%s@%s", name, pv.Version)

			blob, err := reg.Fetch(gctx, pv.CID)
			if err != nil {
				return err
			}

			edges := make([]string, 0)

			for _, d := range blob.Manifest.Dependencies {
				if dep, ok := pinned[d.Name]; ok {
					edges = append(edges, fmt.Sprintf("%s@%s", d.Name, dep.Version))
				}
			}

			mu.Lock()
			out[key] = edges
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	// ensure keys with no edges exist.
	for name, pv := range pinned {
		key := fmt.Sprintf("%s@%s", name, pv.Version)
		if _, ok := out[key]; !ok {
			out[key] = nil
		}
	}

	return out, nil
}