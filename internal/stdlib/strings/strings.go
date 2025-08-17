package stringsx

// Package stringsx provides small string utilities for stdlib.

// TrimPrefix returns s without the provided leading prefix string.
// If s doesn't start with prefix, s is returned unchanged.
func TrimPrefix(s, prefix string) string {
	if len(prefix) == 0 || len(s) < len(prefix) {
		return s
	}
	if s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix returns s without the provided trailing suffix string.
// If s doesn't end with suffix, s is returned unchanged.
func TrimSuffix(s, suffix string) string {
	if len(suffix) == 0 || len(s) < len(suffix) {
		return s
	}
	if s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// SplitOnce splits s around the first instance of sep, returning the text
// before and after sep. If sep is not found, (s, "") is returned.
func SplitOnce(s, sep string) (string, string) {
	if sep == "" {
		return s, ""
	}
	idx := -1
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx < 0 {
		return s, ""
	}
	return s[:idx], s[idx+len(sep):]
}
