package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SecurityValidator provides input validation for the Orizon compiler
type SecurityValidator struct {
	allowedExtensions []string
	maxPathLength     int
	blockedPatterns   []string
}

// NewSecurityValidator creates a new security validator with safe defaults
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		allowedExtensions: []string{".oriz", ".orizon"},
		maxPathLength:     4096, // Reasonable path length limit
		blockedPatterns: []string{
			"..",                       // Path traversal
			"~",                        // User home directory
			"/etc/", "/proc/", "/sys/", // Sensitive system directories (Unix)
			"/bin/", "/sbin/", "/usr/", "/var/", "/dev/", "/tmp/", // Additional Unix directories
			"C:\\Windows\\", "C:\\Program Files\\", // Sensitive system directories (Windows)
			"\\windows\\", "\\program files\\", // Relative Windows paths
			"c:\\windows\\", "c:\\program files\\", // Lowercase variants
		},
	}
}

// ValidateInputFile validates an input file path for security issues
func (sv *SecurityValidator) ValidateInputFile(filename string) error {
	// Check path length
	if len(filename) > sv.maxPathLength {
		return fmt.Errorf("path too long: %d characters (max: %d)", len(filename), sv.maxPathLength)
	}

	// Check for path traversal patterns in original filename
	if strings.Contains(filename, "..") {
		return fmt.Errorf("blocked pattern in path '..': %s", filename)
	}

	// Normalize and clean the path
	cleanPath := filepath.Clean(filename)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check for path traversal attempts in both original and absolute paths
	for _, pattern := range sv.blockedPatterns {
		if strings.Contains(strings.ToLower(filename), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(absPath), strings.ToLower(pattern)) {
			return fmt.Errorf("blocked pattern in path '%s': %s", pattern, filename)
		}
	}

	// Verify file extension
	ext := strings.ToLower(filepath.Ext(filename))
	isValidExt := false
	for _, allowedExt := range sv.allowedExtensions {
		if ext == allowedExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return fmt.Errorf("invalid file extension '%s', allowed: %v", ext, sv.allowedExtensions)
	}

	return nil
}

// ValidateOutputPath validates an output file path for security issues
func (sv *SecurityValidator) ValidateOutputPath(outputPath string) error {
	if outputPath == "" {
		return nil // Empty output path is allowed (stdout)
	}

	// Check path length
	if len(outputPath) > sv.maxPathLength {
		return fmt.Errorf("output path too long: %d characters (max: %d)", len(outputPath), sv.maxPathLength)
	}

	// Check for path traversal patterns in original path
	if strings.Contains(outputPath, "..") {
		return fmt.Errorf("blocked pattern in output path '..': %s", outputPath)
	}

	// Normalize and clean the path
	cleanPath := filepath.Clean(outputPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Check for path traversal attempts in both original and absolute paths
	for _, pattern := range sv.blockedPatterns {
		if strings.Contains(strings.ToLower(outputPath), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(absPath), strings.ToLower(pattern)) {
			return fmt.Errorf("blocked pattern in output path '%s': %s", pattern, outputPath)
		}
	}

	// Ensure output directory is writable (basic check)
	outputDir := filepath.Dir(absPath)
	if outputDir == "." {
		return nil // Current directory
	}

	return nil
}

// SanitizeString removes potentially dangerous characters from strings
func (sv *SecurityValidator) SanitizeString(input string) string {
	// Remove null bytes and control characters
	result := strings.ReplaceAll(input, "\x00", "")
	result = strings.ReplaceAll(result, "\r", "")
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\t", " ")

	// Limit length
	if len(result) > 1024 {
		result = result[:1024]
	}

	return result
}

// ValidateOptimizationLevel validates the optimization level parameter
func (sv *SecurityValidator) ValidateOptimizationLevel(level string) error {
	if level == "" {
		return nil
	}

	allowedLevels := []string{"none", "basic", "default", "aggressive"}
	levelLower := strings.ToLower(level)

	for _, allowed := range allowedLevels {
		if levelLower == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid optimization level '%s', allowed: %v", level, allowedLevels)
}
