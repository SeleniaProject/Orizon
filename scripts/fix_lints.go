package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run fix_lints.go <directory>")
		os.Exit(1)
	}

	rootDir := os.Args[1]

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor and .git directories
		if strings.Contains(path, "vendor/") || strings.Contains(path, ".git/") {
			return nil
		}

		return fixFile(path)
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Linting fixes applied successfully!")
}

func fixFile(filename string) error {
	input, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer input.Close()

	var lines []string

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		// Fix godot: Add period to comment endings.
		line = fixGodotViolations(line)

		// Fix testpackage violations.
		line = fixTestPackageViolations(line, filename)

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Write back to file.
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}

func fixGodotViolations(line string) string {
	// Pattern for single-line comments that don't end with period.
	commentPattern := regexp.MustCompile(`^(\s*//\s*)([^/.]*[^/.\s])(\s*)$`)

	if commentPattern.MatchString(line) {
		return commentPattern.ReplaceAllString(line, "${1}${2}.${3}")
	}

	return line
}

func fixTestPackageViolations(line string, filename string) string {
	// Fix test package names.
	if strings.Contains(filename, "/test/") && strings.HasPrefix(strings.TrimSpace(line), "package ") {
		if strings.Contains(filename, "/benchmark/") {
			return strings.Replace(line, "package benchmark", "package benchmark_test", 1)
		}

		if strings.Contains(filename, "/e2e/") {
			return strings.Replace(line, "package e2e", "package e2e_test", 1)
		}

		if strings.Contains(filename, "/golden/") {
			return strings.Replace(line, "package golden", "package golden_test", 1)
		}

		if strings.Contains(filename, "/integration/") {
			return strings.Replace(line, "package integration", "package integration_test", 1)
		}

		if strings.Contains(filename, "/unit/") {
			return strings.Replace(line, "package unit", "package unit_test", 1)
		}

		if strings.HasSuffix(filename, "_test.go") && strings.Contains(line, "package test") {
			return strings.Replace(line, "package test", "package test_test", 1)
		}
	}

	return line
}
