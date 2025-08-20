package core

import (
	gos "strings"
	"unicode/utf8"
)

// Contains reports whether substr is within s.
func Contains(s, substr string) bool { return gos.Contains(s, substr) }

// HasPrefix reports whether s begins with prefix.
func HasPrefix(s, prefix string) bool { return gos.HasPrefix(s, prefix) }

// HasSuffix reports whether s ends with suffix.
func HasSuffix(s, suffix string) bool { return gos.HasSuffix(s, suffix) }

// Join concatenates elements with sep.
func Join(elems []string, sep string) string { return gos.Join(elems, sep) }

// Split splits s by sep.
func Split(s, sep string) []string { return gos.Split(s, sep) }

// Trim trims cutset from both ends.
func Trim(s, cutset string) string { return gos.Trim(s, cutset) }

// ToUpper returns s uppercased.
func ToUpper(s string) string { return gos.ToUpper(s) }

// ToLower returns s lowercased.
func ToLower(s string) string { return gos.ToLower(s) }

// RuneCount counts runes in s (Unicode code points).
func RuneCount(s string) int { return utf8.RuneCountInString(s) }

// Reverse returns a rune-wise reversed string.
func Reverse(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}

	return string(rs)
}
