package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Usage: go run fix_lints.go <directory>")
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
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	log.Println("Linting fixes applied successfully!")
}

func fixFile(filename string) error {
	// Validate the filename to prevent path traversal attacks
	if strings.Contains(filename, "..") || strings.Contains(filename, "~") {
		return fmt.Errorf("invalid filename: %s", filename)
	}

	input, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	defer func() {
		if closeErr := input.Close(); closeErr != nil {
			log.Printf("Warning: failed to close input file: %v", closeErr)
		}
	}()

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

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("scanner error: %w", scanErr)
	}

	// Write back to file.
	output, err := os.Create(filepath.Clean(filename))
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	defer func() {
		if closeErr := output.Close(); closeErr != nil {
			log.Printf("Warning: failed to close output file: %v", closeErr)
		}
	}()

	writer := bufio.NewWriter(output)
	for _, line := range lines {
		if _, writeErr := writer.WriteString(line + "\n"); writeErr != nil {
			return fmt.Errorf("failed to write line: %w", writeErr)
		}
	}

	if flushErr := writer.Flush(); flushErr != nil {
		return fmt.Errorf("failed to flush writer: %w", flushErr)
	}

	return nil
}

func fixGodotViolations(line string) string {
	// Pattern for single-line comments that don't end with period.
	commentPattern := regexp.MustCompile(`^(\s*//\s*)([^/.]*[^/.\s])(\s*)$`)

	if commentPattern.MatchString(line) {
		return commentPattern.ReplaceAllString(line, "${1}${2}.${3}")
	}

	return line
}

func fixTestPackageViolations(line, filename string) string {
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
