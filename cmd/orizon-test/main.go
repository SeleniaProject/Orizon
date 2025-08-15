package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/orizon-lang/orizon/internal/testrunner"
)

func main() {
	var (
		pkgs    string
		runPat  string
		par     int
		jsonOut bool
		short   bool
		race    bool
		timeout time.Duration
		color   bool
		envList string
		extra   string
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
	flag.Parse()

	pkgsArr := splitNonEmpty(pkgs, ",")
	env := splitNonEmpty(envList, ";")
	extras := splitNonEmpty(extra, " ")

	runner := testrunner.New(testrunner.Options{
		Packages:   pkgsArr,
		RunPattern: runPat,
		Parallel:   par,
		JSON:       jsonOut,
		Short:      short,
		Race:       race,
		Timeout:    timeout,
		Env:        env,
		Color:      color,
		ExtraArgs:  extras,
	})
	ctx := context.Background()
	res, err := runner.Run(ctx, os.Stdout)
	if err != nil {
		// non-zero exit on failure
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
