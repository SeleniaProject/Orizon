package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// SecureExecManager provides centralized secure command execution
type SecureExecManager struct {
	mutex    sync.RWMutex
	executed map[string]int // Track command execution for monitoring
}

// NewSecureExecManager creates a new secure execution manager
func NewSecureExecManager() *SecureExecManager {
	return &SecureExecManager{
		executed: make(map[string]int),
	}
}

// ExecuteSecureCommand executes a command with full security validation
func (sem *SecureExecManager) ExecuteSecureCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	sem.mutex.Lock()
	defer sem.mutex.Unlock()

	// Track command execution
	cmdKey := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	sem.executed[cmdKey]++

	// Validate command name
	if err := sem.validateCommandName(name); err != nil {
		return nil, fmt.Errorf("invalid command name: %w", err)
	}

	// Validate arguments
	for i, arg := range args {
		if err := sem.validateCommandArgument(arg); err != nil {
			return nil, fmt.Errorf("invalid argument %d '%s': %w", i, arg, err)
		}
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, name, args...)

	// Set secure environment
	cmd.Env = sem.getSecureEnvironment()

	return cmd, nil
}

// validateCommandName validates the command name for security
func (sem *SecureExecManager) validateCommandName(name string) error {
	// Clean the path
	cleanName := filepath.Clean(name)

	// Check for blocked patterns
	blockedPatterns := []string{
		"..",                               // Path traversal
		"~",                                // Home directory
		"/bin/sh", "/bin/bash", "/bin/zsh", // Shell executables
		"cmd.exe", "powershell.exe", "wscript.exe", // Windows executables
		"python", "perl", "ruby", "node", // Scripting languages
	}

	lowerName := strings.ToLower(cleanName)
	for _, pattern := range blockedPatterns {
		if strings.Contains(lowerName, pattern) {
			return fmt.Errorf("blocked command pattern: %s", pattern)
		}
	}

	// Only allow specific safe commands
	allowedCommands := []string{
		"go",      // Go compiler
		"git",     // Git commands
		"make",    // Build system
		"gcc",     // C compiler
		"clang",   // LLVM compiler
		"ld",      // Linker
		"ar",      // Archive tool
		"objdump", // Object file analyzer
		"nm",      // Symbol table lister
	}

	baseName := filepath.Base(cleanName)
	// Remove extension for Windows compatibility
	if ext := filepath.Ext(baseName); ext != "" {
		baseName = strings.TrimSuffix(baseName, ext)
	}

	for _, allowed := range allowedCommands {
		if strings.EqualFold(baseName, allowed) {
			return nil
		}
	}

	return fmt.Errorf("command not in allowed list: %s", baseName)
}

// validateCommandArgument validates command arguments for security
func (sem *SecureExecManager) validateCommandArgument(arg string) error {
	// Check length
	if len(arg) > 4096 {
		return fmt.Errorf("argument too long")
	}

	// Check for null bytes
	if strings.Contains(arg, "\x00") {
		return fmt.Errorf("null byte in argument")
	}

	// Check for command injection patterns
	injectionPatterns := []string{
		";", "&", "|", "`", "$(", // Command separators and substitution
		"&&", "||", // Logical operators
		"$(", "${", // Variable/command substitution
		">", ">>", "<", // Redirection
	}

	for _, pattern := range injectionPatterns {
		if strings.Contains(arg, pattern) {
			return fmt.Errorf("potential command injection pattern: %s", pattern)
		}
	}

	return nil
}

// getSecureEnvironment returns a secure environment for command execution
func (sem *SecureExecManager) getSecureEnvironment() []string {
	// Start with minimal safe environment
	secureEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TEMP=" + os.Getenv("TEMP"),
		"TMP=" + os.Getenv("TMP"),
	}

	// Add Go-specific environment variables if they exist
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		secureEnv = append(secureEnv, "GOROOT="+goroot)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		secureEnv = append(secureEnv, "GOPATH="+gopath)
	}
	if gocache := os.Getenv("GOCACHE"); gocache != "" {
		secureEnv = append(secureEnv, "GOCACHE="+gocache)
	}

	return secureEnv
}

// ExecuteSecureGoCommand safely executes Go commands
func (sem *SecureExecManager) ExecuteSecureGoCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	// Validate Go-specific arguments
	if len(args) == 0 {
		return nil, fmt.Errorf("no Go command specified")
	}

	// Allow only specific Go subcommands
	allowedGoCommands := []string{
		"build", "run", "test", "fmt", "vet", "get", "mod",
		"version", "env", "list", "clean", "install",
	}

	subCommand := args[0]
	commandAllowed := false
	for _, allowed := range allowedGoCommands {
		if subCommand == allowed {
			commandAllowed = true
			break
		}
	}

	if !commandAllowed {
		return nil, fmt.Errorf("Go subcommand not allowed: %s", subCommand)
	}

	return sem.ExecuteSecureCommand(ctx, "go", args...)
}

// ValidateProjectPath ensures the path is within project boundaries
func (sem *SecureExecManager) ValidateProjectPath(path string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Ensure path is within the project directory
	absProject, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	relPath, err := filepath.Rel(absProject, absPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Check for path traversal attempts
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path traversal attempt detected: %s", path)
	}

	return nil
}

// GetExecutionStats returns command execution statistics for monitoring
func (sem *SecureExecManager) GetExecutionStats() map[string]int {
	sem.mutex.RLock()
	defer sem.mutex.RUnlock()

	stats := make(map[string]int)
	for cmd, count := range sem.executed {
		stats[cmd] = count
	}
	return stats
}

// Global secure execution manager instance
var globalSecureExecManager = NewSecureExecManager()
