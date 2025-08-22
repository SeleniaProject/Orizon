// Package xml provides XML parsing and generation capabilities.
// This package supports XML marshaling/unmarshaling, XPath queries,
// and XML validation with a developer-friendly API.
package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Parser provides XML parsing functionality.
type Parser struct {
	decoder *xml.Decoder
}

// NewParser creates a new XML parser.
func NewParser(reader io.Reader) *Parser {
	return &Parser{
		decoder: xml.NewDecoder(reader),
	}
}

// Parse parses XML into a struct.
func (p *Parser) Parse(v interface{}) error {
	return p.decoder.Decode(v)
}

// ParseToMap parses XML into a map structure.
func (p *Parser) ParseToMap() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for {
		token, err := p.decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			element, err := p.parseElement(t)
			if err != nil {
				return nil, err
			}
			result[t.Name.Local] = element
		}
	}

	return result, nil
}

func (p *Parser) parseElement(start xml.StartElement) (interface{}, error) {
	element := make(map[string]interface{})

	// Add attributes
	if len(start.Attr) > 0 {
		attrs := make(map[string]string)
		for _, attr := range start.Attr {
			attrs[attr.Name.Local] = attr.Value
		}
		element["@attributes"] = attrs
	}

	var content strings.Builder
	var children []interface{}

	for {
		token, err := p.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				// End of current element
				if content.Len() > 0 {
					element["text"] = strings.TrimSpace(content.String())
				}
				if len(children) > 0 {
					element["children"] = children
				}
				if len(element) == 0 {
					return "", nil
				}
				if len(element) == 1 && element["text"] != nil {
					return element["text"], nil
				}
				return element, nil
			}
		case xml.StartElement:
			child, err := p.parseElement(t)
			if err != nil {
				return nil, err
			}
			children = append(children, map[string]interface{}{
				t.Name.Local: child,
			})
		case xml.CharData:
			content.Write(t)
		}
	}
}

// Generator provides XML generation functionality.
type Generator struct {
	encoder *xml.Encoder
	buffer  *bytes.Buffer
}

// NewGenerator creates a new XML generator.
func NewGenerator() *Generator {
	buffer := &bytes.Buffer{}
	encoder := xml.NewEncoder(buffer)
	encoder.Indent("", "  ")

	return &Generator{
		encoder: encoder,
		buffer:  buffer,
	}
}

// Generate generates XML from a struct.
func (g *Generator) Generate(v interface{}) ([]byte, error) {
	g.buffer.Reset()

	if err := g.encoder.Encode(v); err != nil {
		return nil, err
	}

	if err := g.encoder.Flush(); err != nil {
		return nil, err
	}

	return g.buffer.Bytes(), nil
}

// GenerateFromMap generates XML from a map structure.
func (g *Generator) GenerateFromMap(data map[string]interface{}, rootName string) ([]byte, error) {
	g.buffer.Reset()

	if err := g.writeMapElement(rootName, data); err != nil {
		return nil, err
	}

	if err := g.encoder.Flush(); err != nil {
		return nil, err
	}

	return g.buffer.Bytes(), nil
}

func (g *Generator) writeMapElement(name string, data interface{}) error {
	start := xml.StartElement{Name: xml.Name{Local: name}}

	switch v := data.(type) {
	case map[string]interface{}:
		// Handle attributes
		if attrs, ok := v["@attributes"].(map[string]string); ok {
			for key, value := range attrs {
				start.Attr = append(start.Attr, xml.Attr{
					Name:  xml.Name{Local: key},
					Value: value,
				})
			}
		}

		if err := g.encoder.EncodeToken(start); err != nil {
			return err
		}

		// Handle text content
		if text, ok := v["text"]; ok {
			if err := g.encoder.EncodeToken(xml.CharData(fmt.Sprintf("%v", text))); err != nil {
				return err
			}
		}

		// Handle children
		if children, ok := v["children"].([]interface{}); ok {
			for _, child := range children {
				if childMap, ok := child.(map[string]interface{}); ok {
					for childName, childData := range childMap {
						if err := g.writeMapElement(childName, childData); err != nil {
							return err
						}
					}
				}
			}
		}

		// Handle other properties as child elements
		for key, value := range v {
			if key != "@attributes" && key != "text" && key != "children" {
				if err := g.writeMapElement(key, value); err != nil {
					return err
				}
			}
		}

		return g.encoder.EncodeToken(xml.EndElement{Name: start.Name})

	case []interface{}:
		for _, item := range v {
			if err := g.writeMapElement(name, item); err != nil {
				return err
			}
		}
		return nil

	default:
		// Simple value
		if err := g.encoder.EncodeToken(start); err != nil {
			return err
		}
		if err := g.encoder.EncodeToken(xml.CharData(fmt.Sprintf("%v", v))); err != nil {
			return err
		}
		return g.encoder.EncodeToken(xml.EndElement{Name: start.Name})
	}
}

// Document represents an XML document.
type Document struct {
	Root     *Element
	Version  string
	Encoding string
}

// Element represents an XML element.
type Element struct {
	Name       string
	Attributes map[string]string
	Text       string
	Children   []*Element
	Parent     *Element
}

// NewDocument creates a new XML document.
func NewDocument(rootName string) *Document {
	return &Document{
		Root:     NewElement(rootName),
		Version:  "1.0",
		Encoding: "UTF-8",
	}
}

// NewElement creates a new XML element.
func NewElement(name string) *Element {
	return &Element{
		Name:       name,
		Attributes: make(map[string]string),
		Children:   make([]*Element, 0),
	}
}

// SetAttribute sets an attribute on the element.
func (e *Element) SetAttribute(name, value string) *Element {
	e.Attributes[name] = value
	return e
}

// GetAttribute gets an attribute value.
func (e *Element) GetAttribute(name string) string {
	return e.Attributes[name]
}

// SetText sets the text content of the element.
func (e *Element) SetText(text string) *Element {
	e.Text = text
	return e
}

// AddChild adds a child element.
func (e *Element) AddChild(child *Element) *Element {
	child.Parent = e
	e.Children = append(e.Children, child)
	return e
}

// FindChild finds a child element by name.
func (e *Element) FindChild(name string) *Element {
	for _, child := range e.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// FindChildren finds all child elements by name.
func (e *Element) FindChildren(name string) []*Element {
	var result []*Element
	for _, child := range e.Children {
		if child.Name == name {
			result = append(result, child)
		}
	}
	return result
}

// ToString converts the document to XML string.
func (d *Document) ToString() (string, error) {
	generator := NewGenerator()
	data, err := generator.Generate(d.Root)
	if err != nil {
		return "", err
	}

	header := fmt.Sprintf("<?xml version=\"%s\" encoding=\"%s\"?>\n", d.Version, d.Encoding)
	return header + string(data), nil
}

// XPath provides simple XPath-like querying.
type XPath struct {
	document *Document
}

// NewXPath creates a new XPath instance.
func NewXPath(doc *Document) *XPath {
	return &XPath{document: doc}
}

// Select selects elements using a simple XPath-like syntax.
func (x *XPath) Select(path string) []*Element {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	return x.selectFromElement(x.document.Root, parts)
}

func (x *XPath) selectFromElement(element *Element, parts []string) []*Element {
	if len(parts) == 0 {
		return []*Element{element}
	}

	part := parts[0]
	remaining := parts[1:]

	var results []*Element

	if part == "*" {
		// Wildcard - select all children
		for _, child := range element.Children {
			results = append(results, x.selectFromElement(child, remaining)...)
		}
	} else if strings.Contains(part, "[") {
		// Attribute selector: element[@attr='value']
		elementName := part[:strings.Index(part, "[")]
		selector := part[strings.Index(part, "[")+1 : strings.LastIndex(part, "]")]

		for _, child := range element.Children {
			if child.Name == elementName && x.matchesSelector(child, selector) {
				results = append(results, x.selectFromElement(child, remaining)...)
			}
		}
	} else {
		// Simple element name
		for _, child := range element.Children {
			if child.Name == part {
				results = append(results, x.selectFromElement(child, remaining)...)
			}
		}
	}

	return results
}

func (x *XPath) matchesSelector(element *Element, selector string) bool {
	if strings.HasPrefix(selector, "@") {
		// Attribute selector
		if strings.Contains(selector, "=") {
			parts := strings.SplitN(selector[1:], "=", 2)
			attrName := parts[0]
			attrValue := strings.Trim(parts[1], "'\"")
			return element.GetAttribute(attrName) == attrValue
		} else {
			// Just check if attribute exists
			attrName := selector[1:]
			_, exists := element.Attributes[attrName]
			return exists
		}
	}

	return false
}

// Validator provides XML validation functionality.
type Validator struct {
	rules []ValidationRule
}

// ValidationRule represents a validation rule.
type ValidationRule struct {
	Path     string
	Type     string
	Required bool
	Pattern  string
}

// NewValidator creates a new XML validator.
func NewValidator() *Validator {
	return &Validator{
		rules: make([]ValidationRule, 0),
	}
}

// AddRule adds a validation rule.
func (v *Validator) AddRule(rule ValidationRule) *Validator {
	v.rules = append(v.rules, rule)
	return v
}

// Validate validates an XML document against the rules.
func (v *Validator) Validate(doc *Document) []ValidationError {
	var errors []ValidationError

	for _, rule := range v.rules {
		xpath := NewXPath(doc)
		elements := xpath.Select(rule.Path)

		if rule.Required && len(elements) == 0 {
			errors = append(errors, ValidationError{
				Path:    rule.Path,
				Message: "Required element not found",
			})
			continue
		}

		for _, element := range elements {
			if err := v.validateElement(element, rule); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func (v *Validator) validateElement(element *Element, rule ValidationRule) *ValidationError {
	switch rule.Type {
	case "string":
		// String validation (already string)
		return nil
	case "int":
		if _, err := strconv.Atoi(element.Text); err != nil {
			return &ValidationError{
				Path:    rule.Path,
				Message: "Value is not a valid integer",
			}
		}
	case "float":
		if _, err := strconv.ParseFloat(element.Text, 64); err != nil {
			return &ValidationError{
				Path:    rule.Path,
				Message: "Value is not a valid float",
			}
		}
	case "bool":
		if _, err := strconv.ParseBool(element.Text); err != nil {
			return &ValidationError{
				Path:    rule.Path,
				Message: "Value is not a valid boolean",
			}
		}
	}

	return nil
}

// ValidationError represents a validation error.
type ValidationError struct {
	Path    string
	Message string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("Validation error at %s: %s", e.Path, e.Message)
}

// Convenience functions

// ParseXML parses XML from a string into a struct.
func ParseXML(xmlData string, v interface{}) error {
	parser := NewParser(strings.NewReader(xmlData))
	return parser.Parse(v)
}

// ParseXMLToMap parses XML from a string into a map.
func ParseXMLToMap(xmlData string) (map[string]interface{}, error) {
	parser := NewParser(strings.NewReader(xmlData))
	return parser.ParseToMap()
}

// GenerateXML generates XML from a struct.
func GenerateXML(v interface{}) ([]byte, error) {
	generator := NewGenerator()
	return generator.Generate(v)
}

// GenerateXMLFromMap generates XML from a map.
func GenerateXMLFromMap(data map[string]interface{}, rootName string) ([]byte, error) {
	generator := NewGenerator()
	return generator.GenerateFromMap(data, rootName)
}

// PrettyPrint formats XML with proper indentation.
func PrettyPrint(xmlData []byte) ([]byte, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")

	decoder := xml.NewDecoder(bytes.NewReader(xmlData))

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if err := encoder.EncodeToken(token); err != nil {
			return nil, err
		}
	}

	if err := encoder.Flush(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Transform provides XSLT-like transformation capabilities.
type Transform struct {
	rules map[string]TransformRule
}

// TransformRule represents a transformation rule.
type TransformRule struct {
	Match    string
	Template func(*Element) *Element
}

// NewTransform creates a new XML transformer.
func NewTransform() *Transform {
	return &Transform{
		rules: make(map[string]TransformRule),
	}
}

// AddRule adds a transformation rule.
func (t *Transform) AddRule(match string, template func(*Element) *Element) *Transform {
	t.rules[match] = TransformRule{
		Match:    match,
		Template: template,
	}
	return t
}

// Apply applies the transformation to a document.
func (t *Transform) Apply(doc *Document) *Document {
	newRoot := t.transformElement(doc.Root)
	return &Document{
		Root:     newRoot,
		Version:  doc.Version,
		Encoding: doc.Encoding,
	}
}

func (t *Transform) transformElement(element *Element) *Element {
	// Check if there's a rule for this element
	if rule, exists := t.rules[element.Name]; exists {
		return rule.Template(element)
	}

	// Default transformation - copy element and transform children
	newElement := NewElement(element.Name)
	newElement.Text = element.Text
	newElement.Attributes = make(map[string]string)
	for k, v := range element.Attributes {
		newElement.Attributes[k] = v
	}

	for _, child := range element.Children {
		newChild := t.transformElement(child)
		newElement.AddChild(newChild)
	}

	return newElement
}
