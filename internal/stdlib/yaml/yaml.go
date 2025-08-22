// Package yaml provides YAML parsing and generation capabilities.
// This package supports YAML marshaling/unmarshaling with a clean API
// that's compatible with JSON struct tags for easy migration.
package yaml

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// Parser provides YAML parsing functionality.
type Parser struct {
	scanner *bufio.Scanner
	line    int
	indent  []int
}

// NewParser creates a new YAML parser.
func NewParser(reader io.Reader) *Parser {
	return &Parser{
		scanner: bufio.NewScanner(reader),
		line:    0,
		indent:  make([]int, 0),
	}
}

// Parse parses YAML into a Go value.
func (p *Parser) Parse() (interface{}, error) {
	return p.parseValue(0)
}

func (p *Parser) parseValue(expectedIndent int) (interface{}, error) {
	if !p.scanner.Scan() {
		return nil, io.EOF
	}

	p.line++
	line := p.scanner.Text()

	// Skip empty lines and comments
	if strings.TrimSpace(line) == "" || strings.TrimSpace(line)[0] == '#' {
		return p.parseValue(expectedIndent)
	}

	indent := p.getIndent(line)
	content := strings.TrimSpace(line)

	// Check for list item
	if strings.HasPrefix(content, "- ") {
		return p.parseList(indent)
	}

	// Check for key-value pair
	if strings.Contains(content, ":") {
		return p.parseMap(indent)
	}

	// Scalar value
	return p.parseScalar(content), nil
}

func (p *Parser) parseList(indent int) (interface{}, error) {
	var list []interface{}

	// Go back one line to process the current line
	p.line--

	for p.scanner.Scan() {
		p.line++
		line := p.scanner.Text()

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.TrimSpace(line)[0] == '#' {
			continue
		}

		currentIndent := p.getIndent(line)
		content := strings.TrimSpace(line)

		if currentIndent < indent {
			// End of list, go back one line
			p.line--
			break
		}

		if currentIndent == indent && strings.HasPrefix(content, "- ") {
			// List item
			itemContent := strings.TrimSpace(content[2:])
			if itemContent == "" {
				// Complex item on next lines
				item, err := p.parseValue(indent + 2)
				if err != nil && err != io.EOF {
					return nil, err
				}
				if item != nil {
					list = append(list, item)
				}
			} else if strings.Contains(itemContent, ":") {
				// Inline map
				item := p.parseInlineMap(itemContent)
				list = append(list, item)
			} else {
				// Simple scalar
				list = append(list, p.parseScalar(itemContent))
			}
		} else if currentIndent > indent {
			// Continue with previous parsing
			p.line--
			break
		}
	}

	return list, nil
}

func (p *Parser) parseMap(indent int) (interface{}, error) {
	result := make(map[interface{}]interface{})

	// Go back one line to process the current line
	p.line--

	for p.scanner.Scan() {
		p.line++
		line := p.scanner.Text()

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.TrimSpace(line)[0] == '#' {
			continue
		}

		currentIndent := p.getIndent(line)
		content := strings.TrimSpace(line)

		if currentIndent < indent {
			// End of map, go back one line
			p.line--
			break
		}

		if currentIndent == indent && strings.Contains(content, ":") {
			// Key-value pair
			parts := strings.SplitN(content, ":", 2)
			key := strings.TrimSpace(parts[0])
			valueStr := ""
			if len(parts) > 1 {
				valueStr = strings.TrimSpace(parts[1])
			}

			if valueStr == "" {
				// Value on next lines
				value, err := p.parseValue(indent + 2)
				if err != nil && err != io.EOF {
					return nil, err
				}
				result[key] = value
			} else {
				// Inline value
				result[key] = p.parseScalar(valueStr)
			}
		}
	}

	return result, nil
}

func (p *Parser) parseInlineMap(content string) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})

	if strings.Contains(content, ":") {
		parts := strings.SplitN(content, ":", 2)
		key := strings.TrimSpace(parts[0])
		value := ""
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}
		result[key] = p.parseScalar(value)
	}

	return result
}

func (p *Parser) parseScalar(value string) interface{} {
	value = strings.TrimSpace(value)

	// Remove quotes
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		value = value[1 : len(value)-1]
		return value
	}

	// Boolean values
	switch strings.ToLower(value) {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	case "null", "~", "":
		return nil
	}

	// Number values
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal
	}

	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// String value
	return value
}

func (p *Parser) getIndent(line string) int {
	indent := 0
	for _, char := range line {
		if char == ' ' {
			indent++
		} else if char == '\t' {
			indent += 4 // Treat tab as 4 spaces
		} else {
			break
		}
	}
	return indent
}

// Generator provides YAML generation functionality.
type Generator struct {
	indent string
}

// NewGenerator creates a new YAML generator.
func NewGenerator() *Generator {
	return &Generator{
		indent: "  ", // 2 spaces by default
	}
}

// SetIndent sets the indentation string.
func (g *Generator) SetIndent(indent string) *Generator {
	g.indent = indent
	return g
}

// Generate generates YAML from a Go value.
func (g *Generator) Generate(value interface{}) ([]byte, error) {
	return g.generateValue(value, 0), nil
}

func (g *Generator) generateValue(value interface{}, depth int) []byte {
	if value == nil {
		return []byte("null")
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		return g.generateString(v.String())
	case reflect.Bool:
		if v.Bool() {
			return []byte("true")
		}
		return []byte("false")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64))
	case reflect.Slice, reflect.Array:
		return g.generateList(v, depth)
	case reflect.Map:
		return g.generateMap(v, depth)
	case reflect.Struct:
		return g.generateStruct(v, depth)
	case reflect.Ptr:
		if v.IsNil() {
			return []byte("null")
		}
		return g.generateValue(v.Elem().Interface(), depth)
	case reflect.Interface:
		if v.IsNil() {
			return []byte("null")
		}
		return g.generateValue(v.Elem().Interface(), depth)
	default:
		return []byte(fmt.Sprintf("%v", value))
	}
}

func (g *Generator) generateString(s string) []byte {
	// Check if string needs quoting
	if g.needsQuoting(s) {
		return []byte(fmt.Sprintf("\"%s\"", g.escapeString(s)))
	}
	return []byte(s)
}

func (g *Generator) needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	// Check for special YAML values
	switch strings.ToLower(s) {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return true
	}

	// Check if it looks like a number
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}

	// Check for special characters
	for _, char := range s {
		if char == ':' || char == '#' || char == '-' || char == '[' || char == ']' ||
			char == '{' || char == '}' || char == '|' || char == '>' ||
			unicode.IsControl(char) {
			return true
		}
	}

	// Check if starts/ends with whitespace
	if strings.TrimSpace(s) != s {
		return true
	}

	return false
}

func (g *Generator) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (g *Generator) generateList(v reflect.Value, depth int) []byte {
	if v.Len() == 0 {
		return []byte("[]")
	}

	var result strings.Builder

	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			result.WriteString("\n")
		}

		result.WriteString(strings.Repeat(g.indent, depth))
		result.WriteString("- ")

		item := g.generateValue(v.Index(i).Interface(), depth+1)
		itemStr := string(item)

		// Handle multi-line values
		if strings.Contains(itemStr, "\n") {
			lines := strings.Split(itemStr, "\n")
			result.WriteString(lines[0])
			for _, line := range lines[1:] {
				result.WriteString("\n")
				result.WriteString(strings.Repeat(g.indent, depth+1))
				result.WriteString(line)
			}
		} else {
			result.WriteString(itemStr)
		}
	}

	return []byte(result.String())
}

func (g *Generator) generateMap(v reflect.Value, depth int) []byte {
	if v.Len() == 0 {
		return []byte("{}")
	}

	var result strings.Builder
	keys := v.MapKeys()

	for i, key := range keys {
		if i > 0 {
			result.WriteString("\n")
		}

		result.WriteString(strings.Repeat(g.indent, depth))
		result.WriteString(fmt.Sprintf("%v: ", key.Interface()))

		value := g.generateValue(v.MapIndex(key).Interface(), depth+1)
		valueStr := string(value)

		// Handle multi-line values
		if strings.Contains(valueStr, "\n") {
			result.WriteString("\n")
			lines := strings.Split(valueStr, "\n")
			for _, line := range lines {
				result.WriteString(strings.Repeat(g.indent, depth+1))
				result.WriteString(line)
				result.WriteString("\n")
			}
			// Remove the last newline
			resultStr := result.String()
			result.Reset()
			result.WriteString(strings.TrimSuffix(resultStr, "\n"))
		} else {
			result.WriteString(valueStr)
		}
	}

	return []byte(result.String())
}

func (g *Generator) generateStruct(v reflect.Value, depth int) []byte {
	t := v.Type()
	var result strings.Builder

	fieldCount := 0
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get field name from tag or use field name
		fieldName := fieldType.Name
		if tag := fieldType.Tag.Get("yaml"); tag != "" && tag != "-" {
			if commaIndex := strings.Index(tag, ","); commaIndex > 0 {
				fieldName = tag[:commaIndex]
			} else {
				fieldName = tag
			}
		} else if tag := fieldType.Tag.Get("json"); tag != "" && tag != "-" {
			if commaIndex := strings.Index(tag, ","); commaIndex > 0 {
				fieldName = tag[:commaIndex]
			} else {
				fieldName = tag
			}
		}

		if fieldCount > 0 {
			result.WriteString("\n")
		}

		result.WriteString(strings.Repeat(g.indent, depth))
		result.WriteString(fmt.Sprintf("%s: ", fieldName))

		value := g.generateValue(field.Interface(), depth+1)
		valueStr := string(value)

		// Handle multi-line values
		if strings.Contains(valueStr, "\n") {
			result.WriteString("\n")
			lines := strings.Split(valueStr, "\n")
			for _, line := range lines {
				result.WriteString(strings.Repeat(g.indent, depth+1))
				result.WriteString(line)
				result.WriteString("\n")
			}
			// Remove the last newline
			resultStr := result.String()
			result.Reset()
			result.WriteString(strings.TrimSuffix(resultStr, "\n"))
		} else {
			result.WriteString(valueStr)
		}

		fieldCount++
	}

	return []byte(result.String())
}

// Convenience functions

// Parse parses YAML from a string.
func Parse(yamlData string) (interface{}, error) {
	parser := NewParser(strings.NewReader(yamlData))
	return parser.Parse()
}

// Generate generates YAML from a Go value.
func Generate(value interface{}) ([]byte, error) {
	generator := NewGenerator()
	return generator.Generate(value)
}

// Marshal marshals a Go value to YAML.
func Marshal(v interface{}) ([]byte, error) {
	return Generate(v)
}

// Unmarshal unmarshals YAML data into a Go value.
func Unmarshal(data []byte, v interface{}) error {
	value, err := Parse(string(data))
	if err != nil {
		return err
	}

	return assignValue(reflect.ValueOf(v).Elem(), reflect.ValueOf(value))
}

func assignValue(dest, src reflect.Value) error {
	if !src.IsValid() {
		return nil
	}

	destType := dest.Type()
	srcType := src.Type()

	// Direct assignment if types match
	if srcType.AssignableTo(destType) {
		dest.Set(src)
		return nil
	}

	// Type conversion
	switch dest.Kind() {
	case reflect.String:
		dest.SetString(fmt.Sprintf("%v", src.Interface()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := convertToInt(src.Interface()); err == nil {
			dest.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := convertToUint(src.Interface()); err == nil {
			dest.SetUint(uintVal)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := convertToFloat(src.Interface()); err == nil {
			dest.SetFloat(floatVal)
		}
	case reflect.Bool:
		if boolVal, err := convertToBool(src.Interface()); err == nil {
			dest.SetBool(boolVal)
		}
	case reflect.Slice:
		return assignSlice(dest, src)
	case reflect.Map:
		return assignMap(dest, src)
	case reflect.Struct:
		return assignStruct(dest, src)
	case reflect.Ptr:
		if dest.IsNil() {
			dest.Set(reflect.New(destType.Elem()))
		}
		return assignValue(dest.Elem(), src)
	}

	return nil
}

func assignSlice(dest, src reflect.Value) error {
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return fmt.Errorf("cannot assign %s to slice", src.Kind())
	}

	dest.Set(reflect.MakeSlice(dest.Type(), src.Len(), src.Len()))

	for i := 0; i < src.Len(); i++ {
		if err := assignValue(dest.Index(i), src.Index(i)); err != nil {
			return err
		}
	}

	return nil
}

func assignMap(dest, src reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("cannot assign %s to map", src.Kind())
	}

	dest.Set(reflect.MakeMap(dest.Type()))

	for _, key := range src.MapKeys() {
		srcValue := src.MapIndex(key)
		destValue := reflect.New(dest.Type().Elem()).Elem()

		if err := assignValue(destValue, srcValue); err != nil {
			return err
		}

		dest.SetMapIndex(key, destValue)
	}

	return nil
}

func assignStruct(dest, src reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("cannot assign %s to struct", src.Kind())
	}

	destType := dest.Type()

	for i := 0; i < dest.NumField(); i++ {
		field := dest.Field(i)
		fieldType := destType.Field(i)

		if !field.CanSet() {
			continue
		}

		// Get field name from tag or use field name
		fieldName := fieldType.Name
		if tag := fieldType.Tag.Get("yaml"); tag != "" && tag != "-" {
			if commaIndex := strings.Index(tag, ","); commaIndex > 0 {
				fieldName = tag[:commaIndex]
			} else {
				fieldName = tag
			}
		} else if tag := fieldType.Tag.Get("json"); tag != "" && tag != "-" {
			if commaIndex := strings.Index(tag, ","); commaIndex > 0 {
				fieldName = tag[:commaIndex]
			} else {
				fieldName = tag
			}
		}

		// Look for the field in the source map
		for _, key := range src.MapKeys() {
			if fmt.Sprintf("%v", key.Interface()) == fieldName {
				srcValue := src.MapIndex(key)
				if err := assignValue(field, srcValue); err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

// Type conversion helpers
func convertToInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

func convertToUint(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case uint:
		return uint64(v), nil
	case int64:
		if v >= 0 {
			return uint64(v), nil
		}
	case float64:
		if v >= 0 {
			return uint64(v), nil
		}
	case string:
		return strconv.ParseUint(v, 10, 64)
	}
	return 0, fmt.Errorf("cannot convert %T to uint", value)
}

func convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float", value)
	}
}

func convertToBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}
