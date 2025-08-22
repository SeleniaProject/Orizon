// Package regex provides regular expression support for Orizon.
// This package implements a full-featured regex engine with support for
// pattern matching, capturing groups, and text replacement.
package regex

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Engine represents a regular expression engine.
type Engine struct {
	pattern  string
	compiled *regexp.Regexp
	flags    CompileFlags
}

// CompileFlags represents regex compilation flags.
type CompileFlags uint32

const (
	None       CompileFlags = 0
	IgnoreCase CompileFlags = 1 << iota
	Multiline
	DotAll
	Unicode
	Extended
)

// Match represents a regex match result.
type Match struct {
	Text   string
	Start  int
	End    int
	Groups []Group
}

// Group represents a capturing group in a match.
type Group struct {
	Name  string
	Text  string
	Start int
	End   int
}

// Compile compiles a regular expression pattern.
func Compile(pattern string) (*Engine, error) {
	return CompileWithFlags(pattern, None)
}

// CompileWithFlags compiles a regular expression pattern with flags.
func CompileWithFlags(pattern string, flags CompileFlags) (*Engine, error) {
	// Convert flags to Go regex flags
	goPattern := pattern

	if flags&IgnoreCase != 0 {
		goPattern = "(?i)" + goPattern
	}
	if flags&Multiline != 0 {
		goPattern = "(?m)" + goPattern
	}
	if flags&DotAll != 0 {
		goPattern = "(?s)" + goPattern
	}

	compiled, err := regexp.Compile(goPattern)
	if err != nil {
		return nil, fmt.Errorf("regex compile error: %w", err)
	}

	return &Engine{
		pattern:  pattern,
		compiled: compiled,
		flags:    flags,
	}, nil
}

// MustCompile compiles a pattern and panics on error.
func MustCompile(pattern string) *Engine {
	engine, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return engine
}

// IsMatch tests if the pattern matches the input string.
func (e *Engine) IsMatch(input string) bool {
	return e.compiled.MatchString(input)
}

// FindFirst finds the first match in the input string.
func (e *Engine) FindFirst(input string) *Match {
	matches := e.compiled.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}

	indices := e.compiled.FindStringSubmatchIndex(input)
	if indices == nil {
		return nil
	}

	match := &Match{
		Text:   matches[0],
		Start:  indices[0],
		End:    indices[1],
		Groups: make([]Group, 0, len(matches)-1),
	}

	// Add capturing groups
	groupNames := e.compiled.SubexpNames()
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			group := Group{
				Text:  matches[i],
				Start: indices[i*2],
				End:   indices[i*2+1],
			}

			if i < len(groupNames) && groupNames[i] != "" {
				group.Name = groupNames[i]
			}

			match.Groups = append(match.Groups, group)
		}
	}

	return match
}

// FindAll finds all matches in the input string.
func (e *Engine) FindAll(input string) []*Match {
	allMatches := e.compiled.FindAllStringSubmatch(input, -1)
	if allMatches == nil {
		return nil
	}

	allIndices := e.compiled.FindAllStringSubmatchIndex(input, -1)
	if allIndices == nil {
		return nil
	}

	results := make([]*Match, 0, len(allMatches))
	groupNames := e.compiled.SubexpNames()

	for i, matches := range allMatches {
		indices := allIndices[i]

		match := &Match{
			Text:   matches[0],
			Start:  indices[0],
			End:    indices[1],
			Groups: make([]Group, 0, len(matches)-1),
		}

		// Add capturing groups
		for j := 1; j < len(matches); j++ {
			if matches[j] != "" {
				group := Group{
					Text:  matches[j],
					Start: indices[j*2],
					End:   indices[j*2+1],
				}

				if j < len(groupNames) && groupNames[j] != "" {
					group.Name = groupNames[j]
				}

				match.Groups = append(match.Groups, group)
			}
		}

		results = append(results, match)
	}

	return results
}

// Replace replaces all matches with the replacement string.
func (e *Engine) Replace(input, replacement string) string {
	return e.compiled.ReplaceAllString(input, replacement)
}

// ReplaceFunc replaces all matches using a function.
func (e *Engine) ReplaceFunc(input string, fn func(*Match) string) string {
	return e.compiled.ReplaceAllStringFunc(input, func(match string) string {
		indices := e.compiled.FindStringSubmatchIndex(input)
		if indices == nil {
			return match
		}

		m := &Match{
			Text:  match,
			Start: indices[0],
			End:   indices[1],
		}

		return fn(m)
	})
}

// Split splits the input string by the regex pattern.
func (e *Engine) Split(input string, limit int) []string {
	if limit < 0 {
		return e.compiled.Split(input, -1)
	}
	return e.compiled.Split(input, limit)
}

// Utility functions for common regex operations

// QuoteMeta returns a string that escapes all regex metacharacters.
func QuoteMeta(s string) string {
	return regexp.QuoteMeta(s)
}

// IsValidPattern checks if a pattern is valid.
func IsValidPattern(pattern string) bool {
	_, err := regexp.Compile(pattern)
	return err == nil
}

// Common regex patterns
var (
	EmailPattern    = MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	URLPattern      = MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	IPV4Pattern     = MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	IPV6Pattern     = MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
	PhonePattern    = MustCompile(`^\+?[1-9]\d{1,14}$`)
	DatePattern     = MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	TimePattern     = MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	UUIDPattern     = MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	HexColorPattern = MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
)

// Convenience functions for common validations

// IsEmail checks if the input is a valid email address.
func IsEmail(input string) bool {
	return EmailPattern.IsMatch(input)
}

// IsURL checks if the input is a valid URL.
func IsURL(input string) bool {
	return URLPattern.IsMatch(input)
}

// IsIPV4 checks if the input is a valid IPv4 address.
func IsIPV4(input string) bool {
	return IPV4Pattern.IsMatch(input)
}

// IsIPV6 checks if the input is a valid IPv6 address.
func IsIPV6(input string) bool {
	return IPV6Pattern.IsMatch(input)
}

// IsPhoneNumber checks if the input is a valid phone number.
func IsPhoneNumber(input string) bool {
	return PhonePattern.IsMatch(input)
}

// IsDate checks if the input is a valid date (YYYY-MM-DD).
func IsDate(input string) bool {
	return DatePattern.IsMatch(input)
}

// IsTime checks if the input is a valid time (HH:MM:SS).
func IsTime(input string) bool {
	return TimePattern.IsMatch(input)
}

// IsUUID checks if the input is a valid UUID.
func IsUUID(input string) bool {
	return UUIDPattern.IsMatch(input)
}

// IsHexColor checks if the input is a valid hex color.
func IsHexColor(input string) bool {
	return HexColorPattern.IsMatch(input)
}

// Builder provides a fluent interface for building regex patterns.
type Builder struct {
	parts []string
}

// NewBuilder creates a new regex builder.
func NewBuilder() *Builder {
	return &Builder{
		parts: make([]string, 0),
	}
}

// Literal adds a literal string to the pattern.
func (b *Builder) Literal(s string) *Builder {
	b.parts = append(b.parts, QuoteMeta(s))
	return b
}

// Any adds a wildcard (.) to the pattern.
func (b *Builder) Any() *Builder {
	b.parts = append(b.parts, ".")
	return b
}

// Digit adds a digit matcher (\d) to the pattern.
func (b *Builder) Digit() *Builder {
	b.parts = append(b.parts, `\d`)
	return b
}

// Word adds a word character matcher (\w) to the pattern.
func (b *Builder) Word() *Builder {
	b.parts = append(b.parts, `\w`)
	return b
}

// Whitespace adds a whitespace matcher (\s) to the pattern.
func (b *Builder) Whitespace() *Builder {
	b.parts = append(b.parts, `\s`)
	return b
}

// OneOrMore adds a + quantifier to the last element.
func (b *Builder) OneOrMore() *Builder {
	if len(b.parts) > 0 {
		b.parts[len(b.parts)-1] += "+"
	}
	return b
}

// ZeroOrMore adds a * quantifier to the last element.
func (b *Builder) ZeroOrMore() *Builder {
	if len(b.parts) > 0 {
		b.parts[len(b.parts)-1] += "*"
	}
	return b
}

// Optional adds a ? quantifier to the last element.
func (b *Builder) Optional() *Builder {
	if len(b.parts) > 0 {
		b.parts[len(b.parts)-1] += "?"
	}
	return b
}

// Group creates a capturing group.
func (b *Builder) Group(pattern string) *Builder {
	b.parts = append(b.parts, "("+pattern+")")
	return b
}

// NamedGroup creates a named capturing group.
func (b *Builder) NamedGroup(name, pattern string) *Builder {
	b.parts = append(b.parts, "(?P<"+name+">"+pattern+")")
	return b
}

// Or adds an alternation.
func (b *Builder) Or(pattern string) *Builder {
	b.parts = append(b.parts, "|"+pattern)
	return b
}

// StartOfLine adds a ^ anchor.
func (b *Builder) StartOfLine() *Builder {
	b.parts = append(b.parts, "^")
	return b
}

// EndOfLine adds a $ anchor.
func (b *Builder) EndOfLine() *Builder {
	b.parts = append(b.parts, "$")
	return b
}

// CharClass adds a character class.
func (b *Builder) CharClass(chars string) *Builder {
	b.parts = append(b.parts, "["+chars+"]")
	return b
}

// NegCharClass adds a negated character class.
func (b *Builder) NegCharClass(chars string) *Builder {
	b.parts = append(b.parts, "[^"+chars+"]")
	return b
}

// Build returns the final regex pattern.
func (b *Builder) Build() string {
	return strings.Join(b.parts, "")
}

// Compile builds and compiles the regex pattern.
func (b *Builder) Compile() (*Engine, error) {
	pattern := b.Build()
	return Compile(pattern)
}

// MustCompile builds and compiles the regex pattern, panicking on error.
func (b *Builder) MustCompile() *Engine {
	engine, err := b.Compile()
	if err != nil {
		panic(err)
	}
	return engine
}

// Advanced regex functionality

// NamedCaptures extracts named captures from a match.
func (m *Match) NamedCaptures() map[string]string {
	captures := make(map[string]string)
	for _, group := range m.Groups {
		if group.Name != "" {
			captures[group.Name] = group.Text
		}
	}
	return captures
}

// GetGroup returns a specific capturing group by index.
func (m *Match) GetGroup(index int) (*Group, error) {
	if index < 0 || index >= len(m.Groups) {
		return nil, errors.New("group index out of range")
	}
	return &m.Groups[index], nil
}

// GetNamedGroup returns a specific capturing group by name.
func (m *Match) GetNamedGroup(name string) (*Group, error) {
	for _, group := range m.Groups {
		if group.Name == name {
			return &group, nil
		}
	}
	return nil, fmt.Errorf("named group '%s' not found", name)
}

// Length returns the length of the match.
func (m *Match) Length() int {
	return m.End - m.Start
}

// IsEmpty returns true if the match is empty.
func (m *Match) IsEmpty() bool {
	return m.Length() == 0
}
