package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SecureCommandExecutor provides secure command execution with input validation.
type SecureCommandExecutor struct {
	validator *SecurityValidator
}

// NewSecureCommandExecutor creates a new secure command executor.
func NewSecureCommandExecutor() *SecureCommandExecutor {
	return &SecureCommandExecutor{
		validator: NewSecurityValidator(),
	}
}

// ExecuteCommand executes a command with security validation.
func (sce *SecureCommandExecutor) ExecuteCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	// Validate command name.
	if err := sce.validateCommandName(name); err != nil {
		return nil, fmt.Errorf("invalid command name: %w", err)
	}

	// Validate arguments.
	for i, arg := range args {
		if err := sce.validateCommandArgument(arg); err != nil {
			return nil, fmt.Errorf("invalid argument %d '%s': %w", i, arg, err)
		}
	}

	// Create command with context.
	cmd := exec.CommandContext(ctx, name, args...)

	// Set secure environment.
	cmd.Env = sce.getSecureEnvironment()

	return cmd, nil
}

// validateCommandName validates the command name for security.
func (sce *SecureCommandExecutor) validateCommandName(name string) error {
	// Clean the path.
	cleanName := filepath.Clean(name)

	// Check for blocked patterns.
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

	// Only allow specific safe commands.
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
	// Remove extension for Windows compatibility.
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

// validateCommandArgument validates command arguments for security.
func (sce *SecureCommandExecutor) validateCommandArgument(arg string) error {
	// Check length.
	if len(arg) > 4096 {
		return fmt.Errorf("argument too long")
	}

	// Check for null bytes.
	if strings.Contains(arg, "\x00") {
		return fmt.Errorf("null byte in argument")
	}

	// Check for command injection patterns.
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

	// If it looks like a file path, validate it.
	if strings.Contains(arg, "/") || strings.Contains(arg, "\\") {
		if err := sce.validator.ValidateInputFile(arg); err != nil {
			// Try as output path if input validation fails.
			if err2 := sce.validator.ValidateOutputPath(arg); err2 != nil {
				return fmt.Errorf("invalid file path argument: %w", err)
			}
		}
	}

	return nil
}

// getSecureEnvironment returns a secure environment for command execution.
func (sce *SecureCommandExecutor) getSecureEnvironment() []string {
	// Start with minimal safe environment.
	secureEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TEMP=" + os.Getenv("TEMP"),
		"TMP=" + os.Getenv("TMP"),
	}

	// Add Go-specific environment variables if they exist.
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

// ExecuteGoCommand is a convenience method for executing Go commands securely.
func (sce *SecureCommandExecutor) ExecuteGoCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	return sce.ExecuteCommand(ctx, "go", args...)
}

// ExecuteGitCommand is a convenience method for executing Git commands securely.
func (sce *SecureCommandExecutor) ExecuteGitCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	return sce.ExecuteCommand(ctx, "git", args...)
}
