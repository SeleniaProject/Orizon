package main

import "fmt"

// Simple debug parser
type SimpleParser struct {
	input    string
	position int
	current  rune
}

func NewSimpleParser(input string) *SimpleParser {
	p := &SimpleParser{
		input:    input,
		position: 0,
		current:  0,
	}
	p.advance()
	return p
}

func (p *SimpleParser) advance() {
	if p.position < len(p.input) {
		p.current = rune(p.input[p.position])
		p.position++
		fmt.Printf("advance: pos=%d, current='%c' (%d)\n", p.position, p.current, p.current)
	} else {
		p.current = 0 // EOF
		fmt.Printf("advance: EOF, pos=%d\n", p.position)
	}
}

func (p *SimpleParser) skipWhitespace() {
	for p.current == ' ' || p.current == '\t' || p.current == '\n' || p.current == '\r' {
		fmt.Printf("skipWhitespace: skipping '%c'\n", p.current)
		p.advance()
	}
}

func (p *SimpleParser) parseToken() (string, error) {
	p.skipWhitespace()

	if p.current == 0 {
		return "", fmt.Errorf("unexpected end of input")
	}

	fmt.Printf("parseToken: start at pos=%d, current='%c'\n", p.position, p.current)

	start := p.position - 1
	for (p.current >= 'a' && p.current <= 'z') ||
		(p.current >= 'A' && p.current <= 'Z') ||
		(p.current >= '0' && p.current <= '9') ||
		p.current == '_' {
		p.advance()
	}

	token := p.input[start : p.position-1]
	fmt.Printf("parseToken: token='%s' (start=%d, end=%d)\n", token, start, p.position-1)

	return token, nil
}

func main() {
	fmt.Println("Testing simple parser:")

	parser := NewSimpleParser("i")
	token, err := parser.parseToken()
	fmt.Printf("Result: token='%s', err=%v\n\n", token, err)

	parser = NewSimpleParser("42")
	token, err = parser.parseToken()
	fmt.Printf("Result: token='%s', err=%v\n\n", token, err)

	parser = NewSimpleParser("i + 1")
	token, err = parser.parseToken()
	fmt.Printf("Result: token='%s', err=%v\n", token, err)
}
