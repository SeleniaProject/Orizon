package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orizon-lang/orizon/internal/cli"
)

type ProjectConfig struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Language     string            `json:"language"`
	BuildOptions BuildOptions      `json:"build_options"`
	TestOptions  TestOptions       `json:"test_options"`
	FmtOptions   FmtOptions        `json:"fmt_options"`
	LSPOptions   LSPOptions        `json:"lsp_options"`
	Custom       map[string]string `json:"custom,omitempty"`
}

type BuildOptions struct {
	OptimizeLevel string   `json:"optimize_level"`
	EmitDebug     bool     `json:"emit_debug"`
	DebugOutDir   string   `json:"debug_out_dir"`
	Targets       []string `json:"targets"`
}

type TestOptions struct {
	Timeout   string `json:"timeout"`
	Parallel  int    `json:"parallel"`
	Race      bool   `json:"race"`
	Verbose   bool   `json:"verbose"`
	CoverMode string `json:"cover_mode"`
	CoverOut  string `json:"cover_out"`
}

type FmtOptions struct {
	UseAST   bool `json:"use_ast"`
	TabWidth int  `json:"tab_width"`
	MaxWidth int  `json:"max_width"`
}

type LSPOptions struct {
	MaxDocumentSize int64  `json:"max_document_size"`
	CacheSize       int    `json:"cache_size"`
	DebugMode       bool   `json:"debug_mode"`
	LogLevel        string `json:"log_level"`
}

func main() {
	var (
		showVersion bool
		showHelp    bool
		jsonOutput  bool
		configFile  string
		init        bool
		validate    bool
		show        bool
		set         string
		get         string
		unset       string
	)

	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&showHelp, "help", false, "show help information")
	flag.BoolVar(&jsonOutput, "json", false, "output in JSON format")
	flag.StringVar(&configFile, "config", "orizon.json", "configuration file path")
	flag.BoolVar(&init, "init", false, "initialize a new configuration file")
	flag.BoolVar(&validate, "validate", false, "validate configuration file")
	flag.BoolVar(&show, "show", false, "show current configuration")
	flag.StringVar(&set, "set", "", "set configuration value (key=value)")
	flag.StringVar(&get, "get", "", "get configuration value by key")
	flag.StringVar(&unset, "unset", "", "unset configuration value by key")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Orizon project configuration manager.\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s --init                        # Initialize new config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --show                        # Show current config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --set name=my-project         # Set project name\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --get build_options.targets   # Get build targets\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --validate                    # Validate config\n", os.Args[0])
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		cli.PrintVersion("Orizon Config Manager", jsonOutput)
		os.Exit(0)
	}

	if init {
		if err := initConfig(configFile); err != nil {
			cli.ExitWithError("Failed to initialize config: %v", err)
		}
		fmt.Printf("Configuration initialized: %s\n", configFile)
		return
	}

	if validate {
		if err := validateConfig(configFile); err != nil {
			cli.ExitWithError("Configuration validation failed: %v", err)
		}
		fmt.Printf("Configuration is valid: %s\n", configFile)
		return
	}

	if show {
		config, err := loadConfig(configFile)
		if err != nil {
			cli.ExitWithError("Failed to load config: %v", err)
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(config, "", "  ")
			fmt.Println(string(data))
		} else {
			showConfigHuman(config)
		}
		return
	}

	if set != "" {
		if err := setConfigValue(configFile, set); err != nil {
			cli.ExitWithError("Failed to set config value: %v", err)
		}
		fmt.Printf("Configuration updated: %s\n", configFile)
		return
	}

	if get != "" {
		value, err := getConfigValue(configFile, get)
		if err != nil {
			cli.ExitWithError("Failed to get config value: %v", err)
		}
		fmt.Println(value)
		return
	}

	if unset != "" {
		if err := unsetConfigValue(configFile, unset); err != nil {
			cli.ExitWithError("Failed to unset config value: %v", err)
		}
		fmt.Printf("Configuration updated: %s\n", configFile)
		return
	}

	flag.Usage()
	os.Exit(1)
}

func initConfig(configFile string) error {
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configFile)
	}

	config := &ProjectConfig{
		Name:        filepath.Base(filepath.Dir(configFile)),
		Version:     "0.1.0",
		Description: "An Orizon project",
		Language:    "orizon",
		BuildOptions: BuildOptions{
			OptimizeLevel: "default",
			EmitDebug:     false,
			DebugOutDir:   "debug",
			Targets:       []string{"x86_64"},
		},
		TestOptions: TestOptions{
			Timeout:   "10m",
			Parallel:  0,
			Race:      false,
			Verbose:   false,
			CoverMode: "set",
			CoverOut:  "coverage.out",
		},
		FmtOptions: FmtOptions{
			UseAST:   true,
			TabWidth: 4,
			MaxWidth: 100,
		},
		LSPOptions: LSPOptions{
			MaxDocumentSize: 10 * 1024 * 1024, // 10MB
			CacheSize:       1000,
			DebugMode:       false,
			LogLevel:        "info",
		},
		Custom: make(map[string]string),
	}

	return saveConfig(configFile, config)
}

func loadConfig(configFile string) (*ProjectConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveConfig(configFile string, config *ProjectConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func validateConfig(configFile string) error {
	config, err := loadConfig(configFile)
	if err != nil {
		return err
	}

	if config.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if config.Version == "" {
		return fmt.Errorf("project version is required")
	}

	if config.Language != "orizon" {
		return fmt.Errorf("unsupported language: %s", config.Language)
	}

	return nil
}

func showConfigHuman(config *ProjectConfig) {
	fmt.Printf("Project: %s v%s\n", config.Name, config.Version)
	fmt.Printf("Description: %s\n", config.Description)
	fmt.Printf("Language: %s\n\n", config.Language)

	fmt.Println("Build Options:")
	fmt.Printf("  Optimize Level: %s\n", config.BuildOptions.OptimizeLevel)
	fmt.Printf("  Emit Debug: %t\n", config.BuildOptions.EmitDebug)
	fmt.Printf("  Debug Output Dir: %s\n", config.BuildOptions.DebugOutDir)
	fmt.Printf("  Targets: %v\n\n", config.BuildOptions.Targets)

	fmt.Println("Test Options:")
	fmt.Printf("  Timeout: %s\n", config.TestOptions.Timeout)
	fmt.Printf("  Parallel: %d\n", config.TestOptions.Parallel)
	fmt.Printf("  Race Detection: %t\n", config.TestOptions.Race)
	fmt.Printf("  Verbose: %t\n", config.TestOptions.Verbose)
	fmt.Printf("  Coverage Mode: %s\n", config.TestOptions.CoverMode)
	fmt.Printf("  Coverage Output: %s\n\n", config.TestOptions.CoverOut)

	fmt.Println("Format Options:")
	fmt.Printf("  Use AST: %t\n", config.FmtOptions.UseAST)
	fmt.Printf("  Tab Width: %d\n", config.FmtOptions.TabWidth)
	fmt.Printf("  Max Width: %d\n\n", config.FmtOptions.MaxWidth)

	fmt.Println("LSP Options:")
	fmt.Printf("  Max Document Size: %d bytes\n", config.LSPOptions.MaxDocumentSize)
	fmt.Printf("  Cache Size: %d\n", config.LSPOptions.CacheSize)
	fmt.Printf("  Debug Mode: %t\n", config.LSPOptions.DebugMode)
	fmt.Printf("  Log Level: %s\n", config.LSPOptions.LogLevel)

	if len(config.Custom) > 0 {
		fmt.Println("\nCustom Settings:")
		for k, v := range config.Custom {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

func setConfigValue(configFile, keyValue string) error {
	// This is a simplified implementation
	// In a real implementation, you'd parse the key path and set nested values
	return fmt.Errorf("set operation not yet implemented")
}

func getConfigValue(configFile, key string) (string, error) {
	// This is a simplified implementation
	// In a real implementation, you'd parse the key path and get nested values
	return "", fmt.Errorf("get operation not yet implemented")
}

func unsetConfigValue(configFile, key string) error {
	// This is a simplified implementation
	// In a real implementation, you'd parse the key path and unset nested values
	return fmt.Errorf("unset operation not yet implemented")
}
