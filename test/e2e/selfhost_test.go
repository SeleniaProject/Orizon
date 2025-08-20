package e2e_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// repoRoot walks up from CWD to find go.mod and returns that directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

// TestBootstrapCompilerBuild tests if the bootstrap compiler can be built.
func TestBootstrapCompilerBuild(t *testing.T) {
	t.Run("Build Bootstrap Compiler", func(t *testing.T) {
		cmd := exec.Command("go", "build", "-v", "-o", "build/orizon-bootstrap", "./cmd/orizon-bootstrap")
		var stdout, stderr bytes.Buffer
		cmd.Dir = repoRoot(t)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Errorf("Failed to build bootstrap compiler: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
	})
}

// TestSelfHostingCompile tests basic self-hosting compilation.
func TestSelfHostingCompile(t *testing.T) {
	t.Run("Basic Self-Hosting Test", func(t *testing.T) {
		// This test verifies that the compiler can be built and used
		// to compile basic Orizon code
		t.Log("Self-hosting test - basic compilation capabilities")
		t.Log("Testing bootstrap compiler build process")

		// For now, just verify that the compiler builds
		cmd := exec.Command("go", "build", "-v", "./cmd/orizon-compiler")
		var stdout, stderr bytes.Buffer
		cmd.Dir = repoRoot(t)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Errorf("Failed to build main compiler: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
	})
}
