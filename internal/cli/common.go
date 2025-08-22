package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Version information for all CLI tools
const (
	Version   = "0.1.0"
	BuildDate = "2025-08-22"
	CommitSHA = "unknown" // Will be set during build
)

// VersionInfo contains version and build information
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	CommitSHA string `json:"commit_sha"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Arch      string `json:"arch"`
	BuildTags string `json:"build_tags,omitempty"`
}

// GetVersionInfo returns structured version information
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		CommitSHA: CommitSHA,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// PrintVersion prints version information in a consistent format
func PrintVersion(toolName string, jsonOutput bool) {
	info := GetVersionInfo()

	if jsonOutput {
		data, err := json.MarshalIndent(map[string]interface{}{
			"tool":         toolName,
			"version_info": info,
		}, "", "  ")
		if err != nil {
			// Fallback to plain text if JSON marshaling fails
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal version info to JSON: %v\n", err)
			jsonOutput = false
		} else {
			fmt.Println(string(data))
			return
		}
	}

	if !jsonOutput {
		fmt.Printf("%s v%s\n", toolName, info.Version)
		fmt.Printf("Build Date: %s\n", info.BuildDate)
		if info.CommitSHA != "unknown" && info.CommitSHA != "" {
			fmt.Printf("Commit: %s\n", info.CommitSHA)
		}
		fmt.Printf("Go Version: %s\n", info.GoVersion)
		fmt.Printf("Platform: %s/%s\n", info.Platform, info.Arch)
	}
}

// ExitWithError prints an error message and exits with code 1
func ExitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// ExitWithCode exits with the specified code and optional message
func ExitWithCode(code int, format string, args ...interface{}) {
	if format != "" {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	os.Exit(code)
}

// Logger provides structured logging for CLI tools
type Logger struct {
	Verbose   bool
	DebugMode bool
}

// NewLogger creates a new logger instance
func NewLogger(verbose, debug bool) *Logger {
	return &Logger{
		Verbose:   verbose,
		DebugMode: debug,
	}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.Verbose {
		fmt.Printf("[INFO] %s: %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.DebugMode {
		fmt.Printf("[DEBUG] %s: %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] %s: %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] %s: %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

// Config represents common configuration for CLI tools
type Config struct {
	Verbose    bool   `json:"verbose"`
	Debug      bool   `json:"debug"`
	ConfigFile string `json:"config_file"`
	WorkDir    string `json:"work_dir"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		WorkDir: ".",
	}

	if configPath == "" {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Default config if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CommandInfo represents information about a CLI command
type CommandInfo struct {
	Name        string
	Usage       string
	Description string
	Examples    []string
	Flags       []FlagInfo
}

// FlagInfo represents information about a command flag
type FlagInfo struct {
	Name     string
	Short    string
	Usage    string
	Default  string
	Required bool
}

// PrintUsage prints a standardized usage message
func PrintUsage(tool string, commands []CommandInfo) {
	fmt.Printf("%s - Orizon Language Tools\n\n", tool)
	fmt.Printf("USAGE:\n")
	fmt.Printf("    %s <command> [OPTIONS]\n\n", tool)

	if len(commands) > 0 {
		fmt.Printf("COMMANDS:\n")
		for _, cmd := range commands {
			fmt.Printf("    %-12s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("GLOBAL OPTIONS:\n")
	fmt.Printf("    --help, -h     Show help information\n")
	fmt.Printf("    --version, -v  Show version information\n")
	fmt.Printf("    --json         Output version in JSON format\n")
	fmt.Printf("\n")
	fmt.Printf("Use '%s <command> --help' for more information about a command.\n", tool)
}

// PrintCommandUsage prints usage for a specific command
func PrintCommandUsage(tool string, cmd CommandInfo) {
	fmt.Printf("%s %s - %s\n\n", tool, cmd.Name, cmd.Description)
	fmt.Printf("USAGE:\n")
	fmt.Printf("    %s\n\n", cmd.Usage)

	if len(cmd.Flags) > 0 {
		fmt.Printf("OPTIONS:\n")
		for _, flag := range cmd.Flags {
			flagStr := fmt.Sprintf("    --%s", flag.Name)
			if flag.Short != "" {
				flagStr += fmt.Sprintf(", -%s", flag.Short)
			}

			required := ""
			if flag.Required {
				required = " (required)"
			}

			fmt.Printf("%-20s %s%s\n", flagStr, flag.Usage, required)
			if flag.Default != "" {
				fmt.Printf("%-20s Default: %s\n", "", flag.Default)
			}
		}
		fmt.Printf("\n")
	}

	if len(cmd.Examples) > 0 {
		fmt.Printf("EXAMPLES:\n")
		for _, example := range cmd.Examples {
			fmt.Printf("    %s\n", example)
		}
		fmt.Printf("\n")
	}
}

// ValidateArgs validates command line arguments
func ValidateArgs(args []string, minArgs int, usage string) error {
	if len(args) < minArgs {
		return fmt.Errorf("insufficient arguments\nUsage: %s", usage)
	}
	return nil
}

// HandleError handles errors in a consistent way
func HandleError(err error, logger *Logger) {
	if err != nil {
		if logger != nil {
			logger.Error("%v", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
