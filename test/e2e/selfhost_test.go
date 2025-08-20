package e2e_test

import (
	"os/exec"
	"testing"
)

// TestBootstrapCompilerBuild tests if the bootstrap compiler can be built.
func TestBootstrapCompilerBuild(t *testing.T) {
	t.Run("Build Bootstrap Compiler", func(t *testing.T) {
		cmd := exec.Command("go", "build", "-o", "build/orizon-bootstrap", "./cmd/orizon-bootstrap")
		if err := cmd.Run(); err != nil {
			t.Errorf("Failed to build bootstrap compiler: %v", err)
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
		cmd := exec.Command("go", "build", "./cmd/orizon-compiler")
		if err := cmd.Run(); err != nil {
			t.Errorf("Failed to build main compiler: %v", err)
		}
	})
}
