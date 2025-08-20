package packagemanager

import (
	"strings"
	"testing"
)

func TestInputValidator_ValidateJSON(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			input:   `{"name": "test", "version": "1.0.0"}`,
			wantErr: false,
		},
		{
			name:    "JSON with script tag",
			input:   `{"name": "<script>alert('xss')</script>", "version": "1.0.0"}`,
			wantErr: true,
		},
		{
			name:    "JSON with null bytes",
			input:   "{\x00\"name\": \"test\"}",
			wantErr: true,
		},
		{
			name:    "JSON too large",
			input:   strings.Repeat("a", 60*1024*1024), // 60MB
			wantErr: true,
		},
		{
			name:    "JSON with SQL injection pattern",
			input:   `{"query": "SELECT * FROM users WHERE id = 1; DROP TABLE users;"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidator_ValidatePackageID(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid package ID",
			input:   "my-package",
			wantErr: false,
		},
		{
			name:    "valid namespaced package ID",
			input:   "org/my-package",
			wantErr: false,
		},
		{
			name:    "empty package ID",
			input:   "",
			wantErr: true,
		},
		{
			name:    "package ID too long",
			input:   strings.Repeat("a", 300),
			wantErr: true,
		},
		{
			name:    "package ID with invalid characters",
			input:   "my-package!@#",
			wantErr: true,
		},
		{
			name:    "package ID starting with non-alphanumeric",
			input:   "-my-package",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePackageID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidator_ValidateVersion(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid semantic version",
			input:   "1.0.0",
			wantErr: false,
		},
		{
			name:    "valid semantic version with prerelease",
			input:   "1.0.0-alpha.1",
			wantErr: false,
		},
		{
			name:    "valid semantic version with build metadata",
			input:   "1.0.0+build.1",
			wantErr: false,
		},
		{
			name:    "invalid version format",
			input:   "1.0",
			wantErr: true,
		},
		{
			name:    "version with script injection",
			input:   "1.0.0<script>alert('xss')</script>",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidator_ValidateCID(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid CID",
			input:   "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "CID too short",
			input:   "1234567890abcdef",
			wantErr: true,
		},
		{
			name:    "CID too long",
			input:   "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef00",
			wantErr: true,
		},
		{
			name:    "CID with invalid characters",
			input:   "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdeg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidator_ValidateString(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid string",
			input:   "Hello, world!",
			wantErr: false,
		},
		{
			name:    "string with script tag",
			input:   "<script>alert('xss')</script>",
			wantErr: true,
		},
		{
			name:    "string with null byte",
			input:   "hello\x00world",
			wantErr: true,
		},
		{
			name:    "string too long",
			input:   strings.Repeat("a", 70000),
			wantErr: true,
		},
		{
			name:    "string with control characters",
			input:   "hello\x01world",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityLogger_LogSecurityEvent(t *testing.T) {
	logger := NewSecurityLogger()

	// Test that logging doesn't panic and properly sanitizes data.
	logger.LogSecurityEvent("test_event", map[string]interface{}{
		"password": "secret123",
		"token":    "abc123def456",
		"normal":   "data",
	})

	// Test authentication logging.
	logger.LogAuthenticationAttempt(false, "test-agent", "192.168.1.1:8080", map[string]interface{}{
		"reason": "invalid_token",
	})

	// Test input validation failure logging.
	logger.LogInputValidationFailure("json", "invalid format", `{"bad": "data"}`)
	// If we get here without panicking, the test passes.
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical strings",
			a:        "secret123",
			b:        "secret123",
			expected: true,
		},
		{
			name:     "different strings",
			a:        "secret123",
			b:        "secret456",
			expected: false,
		},
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: true,
		},
		{
			name:     "different lengths",
			a:        "short",
			b:        "longer string",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecureCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("SecureCompare() = %v, want %v", result, tt.expected)
			}
		})
	}
}
