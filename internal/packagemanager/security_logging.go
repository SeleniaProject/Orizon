package packagemanager

import (
	"crypto/subtle"
	"fmt"
	"log"
	"strings"
	"time"
)

// SecurityLogger provides secure logging with sensitive data protection
type SecurityLogger struct {
	enabled        bool
	redactPatterns []string
}

// NewSecurityLogger creates a new security logger
func NewSecurityLogger() *SecurityLogger {
	// Common patterns that should be redacted from logs
	redactPatterns := []string{
		"password", "passwd", "secret", "key", "token", "auth",
		"credential", "private", "confidential", "sensitive",
		"bearer", "authorization", "session", "cookie",
	}

	return &SecurityLogger{
		enabled:        true,
		redactPatterns: redactPatterns,
	}
}

// LogSecurityEvent logs a security-related event with sanitization
func (sl *SecurityLogger) LogSecurityEvent(event string, details map[string]interface{}) {
	if !sl.enabled {
		return
	}

	// Sanitize the event message
	sanitizedEvent := sl.sanitizeLogMessage(event)

	// Sanitize details
	sanitizedDetails := make(map[string]interface{})
	for key, value := range details {
		sanitizedKey := sl.sanitizeLogMessage(key)
		sanitizedValue := sl.sanitizeValue(value)
		sanitizedDetails[sanitizedKey] = sanitizedValue
	}

	// Log with timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339)
	log.Printf("[SECURITY] %s - %s - Details: %v", timestamp, sanitizedEvent, sanitizedDetails)
}

// LogAuthenticationAttempt logs authentication attempts
func (sl *SecurityLogger) LogAuthenticationAttempt(success bool, userAgent string, remoteAddr string, details map[string]interface{}) {
	status := "FAILED"
	if success {
		status = "SUCCESS"
	}

	// Sanitize user agent and remote address
	sanitizedUA := sl.sanitizeLogMessage(userAgent)
	sanitizedAddr := sl.sanitizeIPAddress(remoteAddr)

	eventDetails := map[string]interface{}{
		"status":      status,
		"user_agent":  sanitizedUA,
		"remote_addr": sanitizedAddr,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	// Add additional details
	for k, v := range details {
		eventDetails[sl.sanitizeLogMessage(k)] = sl.sanitizeValue(v)
	}

	sl.LogSecurityEvent("authentication_attempt", eventDetails)
}

// LogInputValidationFailure logs input validation failures
func (sl *SecurityLogger) LogInputValidationFailure(inputType string, reason string, value string) {
	// Never log the actual invalid value in full - only a hash or prefix
	sanitizedValue := sl.hashOrTruncateValue(value)

	details := map[string]interface{}{
		"input_type": inputType,
		"reason":     reason,
		"value_hash": sanitizedValue,
		"value_len":  len(value),
	}

	sl.LogSecurityEvent("input_validation_failure", details)
}

// LogSuspiciousActivity logs potentially suspicious activity
func (sl *SecurityLogger) LogSuspiciousActivity(activity string, severity string, context map[string]interface{}) {
	details := map[string]interface{}{
		"activity": activity,
		"severity": severity,
	}

	// Add sanitized context
	for k, v := range context {
		details[sl.sanitizeLogMessage(k)] = sl.sanitizeValue(v)
	}

	sl.LogSecurityEvent("suspicious_activity", details)
}

// LogRateLimitExceeded logs rate limit violations
func (sl *SecurityLogger) LogRateLimitExceeded(endpoint string, remoteAddr string, attempts int) {
	details := map[string]interface{}{
		"endpoint":    endpoint,
		"remote_addr": sl.sanitizeIPAddress(remoteAddr),
		"attempts":    attempts,
		"action":      "rate_limit_exceeded",
	}

	sl.LogSecurityEvent("rate_limit_violation", details)
}

// sanitizeLogMessage removes sensitive patterns from log messages
func (sl *SecurityLogger) sanitizeLogMessage(message string) string {
	sanitized := message
	lowerMessage := strings.ToLower(message)

	for _, pattern := range sl.redactPatterns {
		if strings.Contains(lowerMessage, pattern) {
			// Replace sensitive parts with [REDACTED]
			sanitized = strings.ReplaceAll(sanitized, message, "[REDACTED]")
			break
		}
	}

	// Additional sanitization for common patterns
	// Remove potential tokens/keys (alphanumeric strings longer than 20 characters)
	words := strings.Fields(sanitized)
	for i, word := range words {
		if len(word) > 20 && isAlphanumeric(word) {
			words[i] = "[REDACTED_TOKEN]"
		}
	}

	return strings.Join(words, " ")
}

// sanitizeValue sanitizes any value that might contain sensitive data
func (sl *SecurityLogger) sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return sl.sanitizeLogMessage(v)
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for key, val := range v {
			sanitized[sl.sanitizeLogMessage(key)] = sl.sanitizeValue(val)
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, val := range v {
			sanitized[i] = sl.sanitizeValue(val)
		}
		return sanitized
	default:
		return value
	}
}

// sanitizeIPAddress sanitizes IP addresses for logging (partial redaction)
func (sl *SecurityLogger) sanitizeIPAddress(addr string) string {
	// Remove port if present
	if colonIndex := strings.LastIndex(addr, ":"); colonIndex != -1 {
		addr = addr[:colonIndex]
	}

	// For IPv4, redact last octet
	if parts := strings.Split(addr, "."); len(parts) == 4 {
		return strings.Join(parts[:3], ".") + ".xxx"
	}

	// For IPv6 or other formats, redact the last part
	if parts := strings.Split(addr, ":"); len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], ":") + ":xxxx"
	}

	// If we can't parse it, redact most of it
	if len(addr) > 8 {
		return addr[:4] + "xxxx"
	}

	return "xxx.xxx.xxx.xxx"
}

// hashOrTruncateValue creates a safe representation of potentially sensitive values
func (sl *SecurityLogger) hashOrTruncateValue(value string) string {
	if len(value) == 0 {
		return ""
	}

	// For short values, just show length
	if len(value) <= 10 {
		return fmt.Sprintf("[%d chars]", len(value))
	}

	// For longer values, show prefix and suffix with length
	prefix := value[:4]
	suffix := value[len(value)-4:]
	return fmt.Sprintf("%s...%s [%d chars]", prefix, suffix, len(value))
}

// isAlphanumeric checks if a string contains only alphanumeric characters
func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// SecureCompare performs constant-time string comparison to prevent timing attacks
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Global security logger instance
var globalSecurityLogger = NewSecurityLogger()
