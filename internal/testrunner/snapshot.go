package testrunner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SnapshotOptions controls snapshot test behavior.
type SnapshotOptions struct {
	BaseDir  string
	Format   string
	DiffTool string
	Update   bool
	Cleanup  bool
}

// DefaultSnapshotOptions returns default snapshot configuration.
func DefaultSnapshotOptions() SnapshotOptions {
	return SnapshotOptions{
		BaseDir:  "testdata/snapshots",
		Update:   false,
		Format:   "text",
		DiffTool: "",
		Cleanup:  false,
	}
}

// SnapshotManager handles snapshot testing functionality.
type SnapshotManager struct {
	usedFiles   map[string]bool
	testResults map[string]SnapshotResult
	options     SnapshotOptions
}

// SnapshotResult represents the result of a snapshot test.
type SnapshotResult struct {
	Created      time.Time
	TestName     string
	SnapshotPath string
	Status       string
	Diff         string
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager(options SnapshotOptions) *SnapshotManager {
	return &SnapshotManager{
		options:     options,
		usedFiles:   make(map[string]bool),
		testResults: make(map[string]SnapshotResult),
	}
}

// VerifySnapshot checks if actual output matches the stored snapshot.
func (sm *SnapshotManager) VerifySnapshot(testName, actual string) (bool, error) {
	snapshotPath := sm.getSnapshotPath(testName)

	// Mark this snapshot file as used.
	sm.usedFiles[snapshotPath] = true

	// Ensure snapshot directory exists.
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
		return false, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Check if snapshot file exists.
	expected, err := os.ReadFile(snapshotPath)
	if os.IsNotExist(err) {
		// Create new snapshot if update mode or first run.
		if sm.options.Update {
			if err := sm.writeSnapshot(snapshotPath, actual); err != nil {
				return false, err
			}

			sm.testResults[testName] = SnapshotResult{
				TestName:     testName,
				SnapshotPath: snapshotPath,
				Status:       "created",
				Created:      time.Now(),
			}

			return true, nil
		} else {
			sm.testResults[testName] = SnapshotResult{
				TestName:     testName,
				SnapshotPath: snapshotPath,
				Status:       "fail",
				Diff:         fmt.Sprintf("Snapshot file %s does not exist. Run with --update-snapshots to create it.", snapshotPath),
				Created:      time.Now(),
			}

			return false, fmt.Errorf("snapshot file %s does not exist", snapshotPath)
		}
	} else if err != nil {
		return false, fmt.Errorf("failed to read snapshot file %s: %w", snapshotPath, err)
	}

	// Compare actual vs expected.
	expectedStr := string(expected)
	if actual == expectedStr {
		sm.testResults[testName] = SnapshotResult{
			TestName:     testName,
			SnapshotPath: snapshotPath,
			Status:       "pass",
			Created:      time.Now(),
		}

		return true, nil
	}

	// Content differs.
	if sm.options.Update {
		// Update snapshot.
		if err := sm.writeSnapshot(snapshotPath, actual); err != nil {
			return false, err
		}

		sm.testResults[testName] = SnapshotResult{
			TestName:     testName,
			SnapshotPath: snapshotPath,
			Status:       "updated",
			Created:      time.Now(),
		}

		return true, nil
	} else {
		// Generate diff.
		diff := sm.generateDiff(expectedStr, actual)
		sm.testResults[testName] = SnapshotResult{
			TestName:     testName,
			SnapshotPath: snapshotPath,
			Status:       "fail",
			Diff:         diff,
			Created:      time.Now(),
		}

		return false, fmt.Errorf("snapshot mismatch for %s", testName)
	}
}

// VerifyGoldenFile checks if actual output matches a golden file.
func (sm *SnapshotManager) VerifyGoldenFile(goldenPath, actual string) (bool, error) {
	// Mark this golden file as used.
	sm.usedFiles[goldenPath] = true

	expected, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		if sm.options.Update {
			// Create new golden file.
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				return false, fmt.Errorf("failed to create golden file directory: %w", err)
			}

			if err := sm.writeSnapshot(goldenPath, actual); err != nil {
				return false, err
			}

			return true, nil
		} else {
			return false, fmt.Errorf("golden file %s does not exist", goldenPath)
		}
	} else if err != nil {
		return false, fmt.Errorf("failed to read golden file %s: %w", goldenPath, err)
	}

	expectedStr := string(expected)
	if actual == expectedStr {
		return true, nil
	}

	if sm.options.Update {
		// Update golden file.
		if err := sm.writeSnapshot(goldenPath, actual); err != nil {
			return false, err
		}

		return true, nil
	} else {
		diff := sm.generateDiff(expectedStr, actual)

		return false, fmt.Errorf("golden file mismatch for %s:\n%s", goldenPath, diff)
	}
}

// CleanupOrphanedSnapshots removes snapshot files that weren't used in tests.
func (sm *SnapshotManager) CleanupOrphanedSnapshots() error {
	if !sm.options.Cleanup {
		return nil
	}

	snapshotDir := sm.options.BaseDir
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return nil // No snapshots directory
	}

	var orphanedFiles []string

	err := filepath.WalkDir(snapshotDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if this file was used in tests.
		if !sm.usedFiles[path] && strings.HasSuffix(path, ".snap") {
			orphanedFiles = append(orphanedFiles, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk snapshot directory: %w", err)
	}

	// Remove orphaned files.
	for _, file := range orphanedFiles {
		if err := os.Remove(file); err != nil {
			fmt.Printf("Warning: failed to remove orphaned snapshot %s: %v\n", file, err)
		} else {
			fmt.Printf("Removed orphaned snapshot: %s\n", file)
		}
	}

	return nil
}

// GetResults returns all snapshot test results.
func (sm *SnapshotManager) GetResults() map[string]SnapshotResult {
	return sm.testResults
}

// GenerateReport creates a summary report of snapshot tests.
func (sm *SnapshotManager) GenerateReport() string {
	var report strings.Builder

	report.WriteString("Snapshot Test Report\n")
	report.WriteString("===================\n\n")

	passed := 0
	failed := 0
	updated := 0
	created := 0

	for _, result := range sm.testResults {
		switch result.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "updated":
			updated++
		case "created":
			created++
		}
	}

	report.WriteString(fmt.Sprintf("Total: %d\n", len(sm.testResults)))
	report.WriteString(fmt.Sprintf("Passed: %d\n", passed))
	report.WriteString(fmt.Sprintf("Failed: %d\n", failed))
	report.WriteString(fmt.Sprintf("Updated: %d\n", updated))
	report.WriteString(fmt.Sprintf("Created: %d\n", created))

	if failed > 0 {
		report.WriteString("\nFailed Tests:\n")

		for _, result := range sm.testResults {
			if result.Status == "fail" {
				report.WriteString(fmt.Sprintf("- %s\n", result.TestName))

				if result.Diff != "" {
					report.WriteString(fmt.Sprintf("  %s\n", result.Diff))
				}
			}
		}
	}

	return report.String()
}

// getSnapshotPath generates the path for a snapshot file.
func (sm *SnapshotManager) getSnapshotPath(testName string) string {
	// Sanitize test name for filesystem.
	safeName := strings.ReplaceAll(testName, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")

	// Add appropriate extension based on format.
	var ext string

	switch sm.options.Format {
	case "json":
		ext = ".json"
	case "binary":
		ext = ".bin"
	default:
		ext = ".snap"
	}

	return filepath.Join(sm.options.BaseDir, safeName+ext)
}

// writeSnapshot writes content to a snapshot file.
func (sm *SnapshotManager) writeSnapshot(path, content string) error {
	switch sm.options.Format {
	case "binary":
		// For binary format, treat content as hex-encoded.
		data, err := hex.DecodeString(content)
		if err != nil {
			return fmt.Errorf("invalid hex content for binary snapshot: %w", err)
		}

		return os.WriteFile(path, data, 0o644)
	default:
		return os.WriteFile(path, []byte(content), 0o644)
	}
}

// generateDiff creates a simple diff between expected and actual content.
func (sm *SnapshotManager) generateDiff(expected, actual string) string {
	if sm.options.DiffTool != "" {
		// TODO: Implement external diff tool support.
		return fmt.Sprintf("Use %s to view differences", sm.options.DiffTool)
	}

	// Simple line-by-line diff.
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff strings.Builder

	diff.WriteString("Expected vs Actual:\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		var expectedLine, actualLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}

		if i < len(actualLines) {
			actualLine = actualLines[i]
		}

		if expectedLine != actualLine {
			diff.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			diff.WriteString(fmt.Sprintf("- %s\n", expectedLine))
			diff.WriteString(fmt.Sprintf("+ %s\n", actualLine))
		}
	}

	return diff.String()
}

// HashContent generates a content hash for snapshot comparison.
func HashContent(content string) string {
	hash := sha256.Sum256([]byte(content))

	return hex.EncodeToString(hash[:])
}
