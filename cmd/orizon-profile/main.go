package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orizon-lang/orizon/internal/cli"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "show version information")
		showHelp    = flag.Bool("help", false, "show help information")
		jsonOutput  = flag.Bool("json", false, "output version in JSON format")
		profileType = flag.String("type", "cpu", "profile type: cpu, memory, block, mutex, goroutine, trace")
		duration    = flag.Duration("duration", 30*time.Second, "profiling duration")
		outputFile  = flag.String("output", "", "output file (default: profile_<type>_<timestamp>)")
		httpAddr    = flag.String("http", "", "HTTP server address for live profiling (e.g., :6060)")
		packagePath = flag.String("package", ".", "package to profile")
		benchmark   = flag.String("bench", "", "benchmark pattern to profile")
		memProfile  = flag.String("memprofile", "", "write memory profile to file")
		cpuProfile  = flag.String("cpuprofile", "", "write CPU profile to file")
		blockRate   = flag.Int("blockprofilerate", 0, "block profile rate")
		mutexFrac   = flag.Int("mutexprofilefraction", 0, "mutex profile fraction")
		verbose     = flag.Bool("verbose", false, "verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [COMMAND]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Orizon performance profiling tool.\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nPROFILE TYPES:\n")
		fmt.Fprintf(os.Stderr, "  cpu        CPU profiling (default)\n")
		fmt.Fprintf(os.Stderr, "  memory     Memory allocation profiling\n")
		fmt.Fprintf(os.Stderr, "  block      Block contention profiling\n")
		fmt.Fprintf(os.Stderr, "  mutex      Mutex contention profiling\n")
		fmt.Fprintf(os.Stderr, "  goroutine  Goroutine stack profiling\n")
		fmt.Fprintf(os.Stderr, "  trace      Execution trace\n")
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s --type cpu --duration 60s       # Profile CPU for 60 seconds\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --type memory --bench .          # Profile memory during benchmarks\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --http :6060                     # Start HTTP profiling server\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --type trace --output trace.out  # Generate execution trace\n", os.Args[0])
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		cli.PrintVersion("Orizon Performance Profiler", *jsonOutput)
		os.Exit(0)
	}

	profiler := &Profiler{
		Type:        *profileType,
		Duration:    *duration,
		OutputFile:  *outputFile,
		HTTPAddr:    *httpAddr,
		PackagePath: *packagePath,
		Benchmark:   *benchmark,
		MemProfile:  *memProfile,
		CPUProfile:  *cpuProfile,
		BlockRate:   *blockRate,
		MutexFrac:   *mutexFrac,
		Verbose:     *verbose,
	}

	if err := profiler.Run(); err != nil {
		cli.ExitWithError("profiling failed: %v", err)
	}
}

type Profiler struct {
	Type        string
	Duration    time.Duration
	OutputFile  string
	HTTPAddr    string
	PackagePath string
	Benchmark   string
	MemProfile  string
	CPUProfile  string
	BlockRate   int
	MutexFrac   int
	Verbose     bool
}

type ProfileResult struct {
	Type       string    `json:"type"`
	Duration   string    `json:"duration"`
	OutputFile string    `json:"output_file"`
	Size       int64     `json:"size_bytes"`
	Timestamp  time.Time `json:"timestamp"`
	Command    string    `json:"command"`
}

func (p *Profiler) Run() error {
	if p.HTTPAddr != "" {
		return p.startHTTPServer()
	}

	if p.Benchmark != "" {
		return p.profileBenchmark()
	}

	return p.profilePackage()
}

func (p *Profiler) startHTTPServer() error {
	fmt.Printf("Starting HTTP profiling server on %s\n", p.HTTPAddr)
	fmt.Printf("Access profiles at:\n")
	fmt.Printf("  http://%s/debug/pprof/\n", strings.TrimPrefix(p.HTTPAddr, ":"))
	fmt.Printf("  http://%s/debug/pprof/goroutine\n", strings.TrimPrefix(p.HTTPAddr, ":"))
	fmt.Printf("  http://%s/debug/pprof/heap\n", strings.TrimPrefix(p.HTTPAddr, ":"))
	fmt.Printf("  http://%s/debug/pprof/profile\n", strings.TrimPrefix(p.HTTPAddr, ":"))
	fmt.Printf("\nPress Ctrl+C to stop\n")

	// Create a simple HTTP server with pprof endpoints
	cmd := exec.Command("go", "run", "-tags", "profile", ".")
	cmd.Dir = p.PackagePath
	cmd.Env = append(os.Environ(), fmt.Sprintf("HTTP_ADDR=%s", p.HTTPAddr))

	return cmd.Run()
}

func (p *Profiler) profileBenchmark() error {
	if p.Verbose {
		fmt.Printf("Profiling benchmarks matching pattern: %s\n", p.Benchmark)
	}

	outputFile := p.getOutputFile()
	args := []string{"test", "-bench", p.Benchmark, "-run", "^$"}

	switch p.Type {
	case "cpu":
		args = append(args, "-cpuprofile", outputFile)
	case "memory":
		args = append(args, "-memprofile", outputFile)
	case "block":
		args = append(args, "-blockprofile", outputFile)
		if p.BlockRate > 0 {
			args = append(args, fmt.Sprintf("-blockprofilerate=%d", p.BlockRate))
		}
	case "mutex":
		args = append(args, "-mutexprofile", outputFile)
		if p.MutexFrac > 0 {
			args = append(args, fmt.Sprintf("-mutexprofilefraction=%d", p.MutexFrac))
		}
	case "trace":
		args = append(args, "-trace", outputFile)
	default:
		return fmt.Errorf("unsupported profile type for benchmarks: %s", p.Type)
	}

	args = append(args, p.PackagePath)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if p.Verbose {
		fmt.Printf("Running: go %s\n", strings.Join(args, " "))
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("benchmark profiling failed: %w", err)
	}

	return p.reportResult(outputFile)
}

func (p *Profiler) profilePackage() error {
	if p.Verbose {
		fmt.Printf("Profiling package: %s for %v\n", p.PackagePath, p.Duration)
	}

	outputFile := p.getOutputFile()

	// For package profiling, we need to instrument the code
	// This is a simplified version - real implementation would be more complex
	switch p.Type {
	case "cpu":
		return p.profileCPU(outputFile)
	case "memory":
		return p.profileMemory(outputFile)
	case "trace":
		return p.profileTrace(outputFile)
	default:
		return fmt.Errorf("unsupported profile type for packages: %s", p.Type)
	}
}

func (p *Profiler) profileCPU(outputFile string) error {
	// Create a temporary main file for profiling
	tempDir, err := os.MkdirTemp("", "orizon-profile")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	mainContent := fmt.Sprintf(`package main

import (
	"os"
	"runtime/pprof"
	"time"
)

func main() {
	f, err := os.Create("%s")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		panic(err)
	}
	defer pprof.StopCPUProfile()

	// Simulate work for the specified duration
	time.Sleep(%d * time.Second)
}`, outputFile, int(p.Duration.Seconds()))

	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		return err
	}

	cmd := exec.Command("go", "run", mainFile)
	if p.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("CPU profiling failed: %w", err)
	}

	return p.reportResult(outputFile)
}

func (p *Profiler) profileMemory(outputFile string) error {
	// Similar to CPU profiling but for memory
	tempDir, err := os.MkdirTemp("", "orizon-profile")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	mainContent := fmt.Sprintf(`package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

func main() {
	// Simulate work
	time.Sleep(%d * time.Second)

	f, err := os.Create("%s")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		panic(err)
	}
}`, int(p.Duration.Seconds()), outputFile)

	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		return err
	}

	cmd := exec.Command("go", "run", mainFile)
	if p.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("memory profiling failed: %w", err)
	}

	return p.reportResult(outputFile)
}

func (p *Profiler) profileTrace(outputFile string) error {
	tempDir, err := os.MkdirTemp("", "orizon-profile")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	mainContent := fmt.Sprintf(`package main

import (
	"os"
	"runtime/trace"
	"time"
)

func main() {
	f, err := os.Create("%s")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		panic(err)
	}
	defer trace.Stop()

	// Simulate work
	time.Sleep(%d * time.Second)
}`, outputFile, int(p.Duration.Seconds()))

	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		return err
	}

	cmd := exec.Command("go", "run", mainFile)
	if p.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("trace profiling failed: %w", err)
	}

	return p.reportResult(outputFile)
}

func (p *Profiler) getOutputFile() string {
	if p.OutputFile != "" {
		return p.OutputFile
	}

	timestamp := time.Now().Format("20060102_150405")
	ext := "prof"
	if p.Type == "trace" {
		ext = "trace"
	}

	return fmt.Sprintf("profile_%s_%s.%s", p.Type, timestamp, ext)
}

func (p *Profiler) reportResult(outputFile string) error {
	stat, err := os.Stat(outputFile)
	if err != nil {
		return fmt.Errorf("failed to stat output file: %w", err)
	}

	result := ProfileResult{
		Type:       p.Type,
		Duration:   p.Duration.String(),
		OutputFile: outputFile,
		Size:       stat.Size(),
		Timestamp:  time.Now(),
		Command:    fmt.Sprintf("go tool pprof %s", outputFile),
	}

	fmt.Printf("\nProfiling completed successfully!\n")
	fmt.Printf("Profile type: %s\n", result.Type)
	fmt.Printf("Output file: %s\n", result.OutputFile)
	fmt.Printf("File size: %d bytes\n", result.Size)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("\nTo analyze the profile, run:\n")

	if p.Type == "trace" {
		fmt.Printf("  go tool trace %s\n", outputFile)
	} else {
		fmt.Printf("  go tool pprof %s\n", outputFile)
		fmt.Printf("  go tool pprof -http=:8080 %s\n", outputFile)
	}

	// Also create a JSON report
	jsonFile := strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + "_report.json"
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		return err
	}

	fmt.Printf("\nReport saved to: %s\n", jsonFile)
	return nil
}
