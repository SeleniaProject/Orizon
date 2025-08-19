package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestSecurityValidator_ValidateInputFile(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_orizon_file",
			filename:    "test.oriz",
			expectError: false,
		},
		{
			name:        "valid_nested_path",
			filename:    "src/main.oriz",
			expectError: false,
		},
		{
			name:        "path_traversal_attack",
			filename:    "../../../etc/passwd.oriz",
			expectError: true,
			errorMsg:    "blocked pattern",
		},
		{
			name:        "invalid_extension",
			filename:    "malicious.exe",
			expectError: true,
			errorMsg:    "invalid file extension",
		},
		{
			name:        "windows_system_path",
			filename:    "C:\\Windows\\System32\\cmd.exe",
			expectError: true,
			errorMsg:    "blocked pattern",
		},
		{
			name:        "unix_system_path",
			filename:    "/etc/shadow.oriz",
			expectError: true,
			errorMsg:    "blocked pattern",
		},
		{
			name:        "home_directory_traversal",
			filename:    "~/secret.oriz",
			expectError: true,
			errorMsg:    "blocked pattern",
		},
		{
			name:        "very_long_path",
			filename:    strings.Repeat("a/", 2500) + "test.oriz", // Over 4096 chars
			expectError: true,
			errorMsg:    "path too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateInputFile(tt.filename)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecurityValidator_ValidateOutputPath(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		outputPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty_path_stdout",
			outputPath:  "",
			expectError: false,
		},
		{
			name:        "valid_output_file",
			outputPath:  "output.json",
			expectError: false,
		},
		{
			name:        "valid_output_directory",
			outputPath:  "build/debug.json",
			expectError: false,
		},
		{
			name:        "path_traversal_output",
			outputPath:  "../../../etc/malicious.json",
			expectError: true,
			errorMsg:    "blocked pattern",
		},
		{
			name:        "system_directory_output",
			outputPath:  "/proc/self/mem.json",
			expectError: true,
			errorMsg:    "blocked pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateOutputPath(tt.outputPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecurityValidator_ValidateOptimizationLevel(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		level       string
		expectError bool
	}{
		{
			name:        "empty_level",
			level:       "",
			expectError: false,
		},
		{
			name:        "valid_none",
			level:       "none",
			expectError: false,
		},
		{
			name:        "valid_basic",
			level:       "basic",
			expectError: false,
		},
		{
			name:        "valid_default",
			level:       "default",
			expectError: false,
		},
		{
			name:        "valid_aggressive",
			level:       "aggressive",
			expectError: false,
		},
		{
			name:        "valid_case_insensitive",
			level:       "BASIC",
			expectError: false,
		},
		{
			name:        "invalid_level",
			level:       "ultra",
			expectError: true,
		},
		{
			name:        "malicious_injection",
			level:       "basic; rm -rf /",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateOptimizationLevel(tt.level)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecurityValidator_SanitizeString(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean_string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "null_bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "control_characters",
			input:    "hello\r\n\tworld",
			expected: "hello  world",
		},
		{
			name:     "very_long_string",
			input:    strings.Repeat("a", 2000),
			expected: strings.Repeat("a", 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSecurityValidator_PlatformSpecific(t *testing.T) {
	validator := NewSecurityValidator()

	// Test platform-specific blocked patterns
	if runtime.GOOS == "windows" {
		err := validator.ValidateInputFile("C:\\Windows\\System32\\kernel32.dll")
		if err == nil {
			t.Error("expected error for Windows system file access")
		}
	} else {
		err := validator.ValidateInputFile("/etc/passwd")
		if err == nil {
			t.Error("expected error for Unix system file access")
		}
	}
}

func BenchmarkSecurityValidator_ValidateInputFile(b *testing.B) {
	validator := NewSecurityValidator()
	filename := "test.oriz"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateInputFile(filename)
	}
}
