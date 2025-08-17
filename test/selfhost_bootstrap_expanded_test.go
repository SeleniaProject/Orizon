package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// Verifies that the bootstrap tool can run with --expand-macros and produce outputs
// without relying on golden comparisons. This ensures the macro expansion path is stable.
func TestSelfHost_BootstrapExpandMacros(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	// Repo root
	root := filepath.Clean("..")
	buildDir := filepath.Join(root, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("mkdir build failed: %v", err)
	}

	// Build bootstrap tool
	outPath := filepath.Join(buildDir, "orizon-bootstrap")
	if runtime.GOOS == "windows" {
		outPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", outPath, "./cmd/orizon-bootstrap")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build bootstrap failed: %v\n%s", err, string(out))
	}

	// Run expansion on macro example without golden compare; just ensure success
	outDir := filepath.Join(root, "artifacts", "selfhost_expanded_test")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir outDir failed: %v", err)
	}
	run := exec.Command(outPath, "--out-dir", outDir, "--expand-macros", filepath.Join("examples", "macro_example.oriz"))
	run.Dir = root
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap (expand-macros) failed: %v\n%s", err, string(out))
	}
}
