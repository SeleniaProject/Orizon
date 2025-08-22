package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/orizon-lang/orizon/internal/cli"
	"github.com/orizon-lang/orizon/internal/testrunner"
)

func main() {
	var (
		pkgs             string
		runPat           string
		par              int
		jsonOut          bool
		jsonAug          bool
		short            bool
		race             bool
		timeout          time.Duration
		color            bool
		envList          string
		extra            string
		junit            string
		retries          int
		failFast         bool
		pkgRegex         string
		summaryJSON      string
		fileRegex        string
		listOnly         bool
		failOnFlaky      bool
		updateSnapshots  bool
		snapshotDir      string
		cleanupSnapshots bool
		goldenTests      bool
		showVersion      bool
		showHelp         bool
		jsonOutput       bool
	)

	flag.StringVar(&pkgs, "packages", "./...", "comma-separated package patterns (e.g. ./...,./internal/...)")
	flag.StringVar(&runPat, "run", "", "regex to select tests (forwarded to go test -run)")
	flag.IntVar(&par, "p", 0, "parallel package workers (default: GOMAXPROCS)")
	flag.BoolVar(&jsonOut, "json", false, "stream raw go test -json events")
	flag.BoolVar(&short, "short", false, "pass -short to go test")
	flag.BoolVar(&race, "race", false, "pass -race to go test")
	flag.DurationVar(&timeout, "timeout", 10*time.Minute, "go test timeout")
	flag.BoolVar(&color, "color", true, "colorize output")
	flag.StringVar(&envList, "env", "", "extra env KEY=VAL;KEY2=VAL2")
	flag.StringVar(&extra, "args", "", "extra args to append to go test (space-separated)")
	flag.StringVar(&junit, "junit", "", "optional JUnit XML output path")
	flag.IntVar(&retries, "retries", 0, "re-run failing tests up to N times to detect flakiness")
	flag.BoolVar(&failFast, "fail-fast", false, "stop at first failing package (cancels remaining)")
	flag.BoolVar(&jsonAug, "json-augment", false, "when --json is set, add Orizon augment events (attempts/flaky)")
	flag.StringVar(&pkgRegex, "pkg-regex", "", "optional regex to filter package names after expansion")
	flag.StringVar(&summaryJSON, "json-summary", "", "optional path to write machine-readable summary JSON (with retry attempts)")
	flag.StringVar(&fileRegex, "file-regex", "", "optional regex to include only packages that have files matching this regex")
	flag.BoolVar(&listOnly, "list", false, "list tests without executing (dry run)")
	flag.BoolVar(&failOnFlaky, "fail-on-flaky", false, "exit non-zero if any test recovered after retries (flaky detected)")
	flag.BoolVar(&updateSnapshots, "update-snapshots", false, "update snapshot files instead of comparing")
	flag.StringVar(&snapshotDir, "snapshot-dir", "testdata/snapshots", "directory for snapshot files")
	flag.BoolVar(&cleanupSnapshots, "cleanup-snapshots", false, "remove orphaned snapshot files")
	flag.BoolVar(&goldenTests, "golden", false, "enable golden file testing support")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&showHelp, "help", false, "show help information")
	flag.BoolVar(&jsonOutput, "json-format", false, "output version in JSON format")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Advanced test runner for Orizon projects with retry logic, flakiness detection, and rich reporting.\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s                        # Run all tests\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -packages ./internal   # Run tests in internal packages\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -retries 3 -race       # Run with race detection and 3 retries\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -json -junit out.xml   # Output JSON and JUnit XML\n", os.Args[0])
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		cli.PrintVersion("Orizon Test Runner", jsonOutput)
		os.Exit(0)
	}

	pkgsArr := splitNonEmpty(pkgs, ",")
	env := splitNonEmpty(envList, ";")
	extras := splitNonEmpty(extra, " ")

	runner := testrunner.New(testrunner.Options{
		Packages:         pkgsArr,
		RunPattern:       runPat,
		Parallel:         par,
		JSON:             jsonOut,
		Short:            short,
		Race:             race,
		Timeout:          timeout,
		Env:              env,
		Color:            color,
		ExtraArgs:        extras,
		JUnitPath:        junit,
		Retries:          retries,
		FailFast:         failFast,
		PackageRegex:     pkgRegex,
		SummaryJSON:      summaryJSON,
		AugmentJSON:      jsonAug,
		FileRegex:        fileRegex,
		ListOnly:         listOnly,
		FailOnFlaky:      failOnFlaky,
		UpdateSnapshots:  updateSnapshots,
		SnapshotDir:      snapshotDir,
		CleanupSnapshots: cleanupSnapshots,
		GoldenTests:      goldenTests,
	})
	ctx := context.Background()

	res, err := runner.Run(ctx, os.Stdout)
	if err != nil {
		// non-zero exit on failure.
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	_ = res
}

func splitNonEmpty(s, sep string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}
