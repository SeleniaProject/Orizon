package packagemanager

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SecurityConfig contains security validation configuration
type SecurityConfig struct {
	MaxJSONSize      int64
	MaxStringLength  int
	MaxArrayLength   int
	MaxObjectDepth   int
	AllowedCharsets  []string
	BlockedPatterns  []*regexp.Regexp
	RequireValidUTF8 bool
	SanitizeHTML     bool
}

// DefaultSecurityConfig returns the default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	// Compile blocked patterns for security threats
	blockedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>`),                // XSS
		regexp.MustCompile(`(?i)javascript:`),                  // JavaScript injection
		regexp.MustCompile(`(?i)data:text/html`),               // Data URI XSS
		regexp.MustCompile(`(?i)vbscript:`),                    // VBScript injection
		regexp.MustCompile(`(?i)on\w+\s*=`),                    // Event handlers
		regexp.MustCompile(`\x00`),                             // Null bytes
		regexp.MustCompile(`[\x01-\x08\x0b\x0c\x0e-\x1f\x7f]`), // Control characters
		regexp.MustCompile(`\\.\\./`),                          // Path traversal
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)\s`), // SQL injection
	}

	return &SecurityConfig{
		MaxJSONSize:      50 * 1024 * 1024, // 50MB
		MaxStringLength:  65536,            // 64KB
		MaxArrayLength:   10000,            // 10K items
		MaxObjectDepth:   32,               // 32 levels deep
		AllowedCharsets:  []string{"utf-8", "ascii"},
		BlockedPatterns:  blockedPatterns,
		RequireValidUTF8: true,
		SanitizeHTML:     true,
	}
}

// InputValidator provides comprehensive input validation and sanitization
type InputValidator struct {
	config *SecurityConfig
}

// NewInputValidator creates a new input validator with default configuration
func NewInputValidator() *InputValidator {
	return &InputValidator{
		config: DefaultSecurityConfig(),
	}
}

// NewInputValidatorWithConfig creates a new input validator with custom configuration
func NewInputValidatorWithConfig(config *SecurityConfig) *InputValidator {
	return &InputValidator{
		config: config,
	}
}

// ValidateJSON validates and sanitizes JSON input
func (iv *InputValidator) ValidateJSON(data []byte) error {
	if int64(len(data)) > iv.config.MaxJSONSize {
		return fmt.Errorf("JSON payload too large: %d bytes (max: %d)", len(data), iv.config.MaxJSONSize)
	}

	// Check for null bytes
	if strings.Contains(string(data), "\x00") {
		return fmt.Errorf("null bytes detected in JSON payload")
	}

	// Validate UTF-8 encoding
	if iv.config.RequireValidUTF8 && !utf8.Valid(data) {
		return fmt.Errorf("invalid UTF-8 encoding in JSON payload")
	}

	// Check for blocked patterns
	dataStr := string(data)
	for _, pattern := range iv.config.BlockedPatterns {
		if pattern.MatchString(dataStr) {
			return fmt.Errorf("blocked pattern detected in JSON: %s", pattern.String())
		}
	}

	// Validate JSON structure
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("invalid JSON structure: %w", err)
	}

	// Validate JSON depth and complexity
	if err := iv.validateJSONStructure(parsed, 0); err != nil {
		return fmt.Errorf("JSON structure validation failed: %w", err)
	}

	return nil
}

// validateJSONStructure recursively validates JSON structure
func (iv *InputValidator) validateJSONStructure(obj interface{}, depth int) error {
	if depth > iv.config.MaxObjectDepth {
		return fmt.Errorf("JSON nesting too deep: %d (max: %d)", depth, iv.config.MaxObjectDepth)
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if err := iv.ValidateString(key); err != nil {
				return fmt.Errorf("invalid JSON key '%s': %w", key, err)
			}
			if err := iv.validateJSONStructure(value, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		if len(v) > iv.config.MaxArrayLength {
			return fmt.Errorf("JSON array too large: %d (max: %d)", len(v), iv.config.MaxArrayLength)
		}
		for i, item := range v {
			if err := iv.validateJSONStructure(item, depth+1); err != nil {
				return fmt.Errorf("invalid JSON array item %d: %w", i, err)
			}
		}
	case string:
		if err := iv.ValidateString(v); err != nil {
			return fmt.Errorf("invalid JSON string: %w", err)
		}
	case float64, bool, nil:
		// These types are safe
	default:
		return fmt.Errorf("unsupported JSON value type: %T", v)
	}

	return nil
}

// ValidateString validates and sanitizes string input
func (iv *InputValidator) ValidateString(s string) error {
	if len(s) > iv.config.MaxStringLength {
		return fmt.Errorf("string too long: %d characters (max: %d)", len(s), iv.config.MaxStringLength)
	}

	// Validate UTF-8 encoding
	if iv.config.RequireValidUTF8 && !utf8.ValidString(s) {
		return fmt.Errorf("invalid UTF-8 encoding in string")
	}

	// Check for null bytes
	if strings.Contains(s, "\x00") {
		return fmt.Errorf("null bytes detected in string")
	}

	// Check for control characters (except tab, newline, carriage return)
	for _, r := range s {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("control character detected: U+%04X", r)
		}
	}

	// Check for blocked patterns
	for _, pattern := range iv.config.BlockedPatterns {
		if pattern.MatchString(s) {
			return fmt.Errorf("blocked pattern detected: %s", pattern.String())
		}
	}

	return nil
}

// ValidateURL validates URL input
func (iv *InputValidator) ValidateURL(rawURL string) (*url.URL, error) {
	if err := iv.ValidateString(rawURL); err != nil {
		return nil, fmt.Errorf("invalid URL string: %w", err)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("URL parse error: %w", err)
	}

	// Validate scheme
	allowedSchemes := []string{"http", "https"}
	schemeAllowed := false
	for _, scheme := range allowedSchemes {
		if parsedURL.Scheme == scheme {
			schemeAllowed = true
			break
		}
	}
	if !schemeAllowed {
		return nil, fmt.Errorf("disallowed URL scheme: %s", parsedURL.Scheme)
	}

	// Validate host
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("empty host in URL")
	}

	// Block private/internal networks for external URLs
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		if iv.isPrivateIP(parsedURL.Hostname()) {
			return nil, fmt.Errorf("private IP address not allowed: %s", parsedURL.Hostname())
		}
	}

	return parsedURL, nil
}

// ValidatePackageID validates package identifier
func (iv *InputValidator) ValidatePackageID(id string) error {
	if err := iv.ValidateString(id); err != nil {
		return fmt.Errorf("invalid package ID: %w", err)
	}

	// Package ID specific validation
	if len(id) == 0 {
		return fmt.Errorf("package ID cannot be empty")
	}

	if len(id) > 255 {
		return fmt.Errorf("package ID too long: %d characters (max: 255)", len(id))
	}

	// Allow alphanumeric, hyphens, underscores, dots, slashes (for namespacing)
	validIDPattern := regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("package ID contains invalid characters: %s", id)
	}

	// Must start with alphanumeric
	if !unicode.IsLetter(rune(id[0])) && !unicode.IsDigit(rune(id[0])) {
		return fmt.Errorf("package ID must start with alphanumeric character: %s", id)
	}

	return nil
}

// ValidateVersion validates semantic version string
func (iv *InputValidator) ValidateVersion(version string) error {
	if err := iv.ValidateString(version); err != nil {
		return fmt.Errorf("invalid version string: %w", err)
	}

	// Semantic version pattern (simplified)
	semverPattern := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	if !semverPattern.MatchString(version) {
		return fmt.Errorf("invalid semantic version format: %s", version)
	}

	return nil
}

// ValidateCID validates Content Identifier
func (iv *InputValidator) ValidateCID(cid string) error {
	if err := iv.ValidateString(cid); err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	// CID should be hex-encoded hash
	if len(cid) != 64 { // SHA-256 hex length
		return fmt.Errorf("invalid CID length: %d (expected: 64)", len(cid))
	}

	// Check if valid hex
	hexPattern := regexp.MustCompile(`^[a-fA-F0-9]+$`)
	if !hexPattern.MatchString(cid) {
		return fmt.Errorf("CID contains non-hex characters: %s", cid)
	}

	return nil
}

// SanitizeString removes dangerous characters and patterns from string
func (iv *InputValidator) SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Remove other control characters except tab, newline, carriage return
	var sanitized strings.Builder
	for _, r := range s {
		if !unicode.IsControl(r) || r == '\t' || r == '\n' || r == '\r' {
			sanitized.WriteRune(r)
		}
	}

	result := sanitized.String()

	// Apply HTML sanitization if enabled
	if iv.config.SanitizeHTML {
		result = iv.sanitizeHTML(result)
	}

	return result
}

// sanitizeHTML removes potentially dangerous HTML content
func (iv *InputValidator) sanitizeHTML(s string) string {
	// Remove script tags
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	s = scriptPattern.ReplaceAllString(s, "")

	// Remove javascript: and data: URLs
	jsPattern := regexp.MustCompile(`(?i)javascript:[^"'\s>]*`)
	s = jsPattern.ReplaceAllString(s, "")

	dataPattern := regexp.MustCompile(`(?i)data:text/html[^"'\s>]*`)
	s = dataPattern.ReplaceAllString(s, "")

	// Remove event handlers
	eventPattern := regexp.MustCompile(`(?i)\s*on\w+\s*=\s*["'][^"']*["']`)
	s = eventPattern.ReplaceAllString(s, "")

	return s
}

// isPrivateIP checks if an IP address is in private ranges
func (iv *InputValidator) isPrivateIP(host string) bool {
	// Simplified check for private IP ranges
	privatePatterns := []string{
		"127.",     // Loopback
		"10.",      // Private Class A
		"172.16.",  // Private Class B (simplified)
		"192.168.", // Private Class C
		"localhost",
		"::1", // IPv6 loopback
	}

	for _, pattern := range privatePatterns {
		if strings.HasPrefix(host, pattern) {
			return true
		}
	}

	return false
}

// ValidateHTTPHeaders validates HTTP headers for security
func (iv *InputValidator) ValidateHTTPHeaders(headers map[string]string) error {
	for name, value := range headers {
		if err := iv.ValidateString(name); err != nil {
			return fmt.Errorf("invalid header name '%s': %w", name, err)
		}

		if err := iv.ValidateString(value); err != nil {
			return fmt.Errorf("invalid header value for '%s': %w", name, err)
		}

		// Check header-specific security rules
		lowerName := strings.ToLower(name)
		switch lowerName {
		case "content-length":
			// Validate numeric value
			if !regexp.MustCompile(`^\d+$`).MatchString(value) {
				return fmt.Errorf("invalid Content-Length header: %s", value)
			}
		case "host":
			// Validate hostname
			if _, err := iv.ValidateURL("http://" + value); err != nil {
				return fmt.Errorf("invalid Host header: %w", err)
			}
		}
	}

	return nil
}
