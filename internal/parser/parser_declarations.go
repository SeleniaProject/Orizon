package parser

import "github.com/orizon-lang/orizon/internal/lexer"

// =============================================================================
// Declaration Parsing
// =============================================================================

// parseDeclaration parses any kind of declaration
func (p *Parser) parseDeclarationHelper() Declaration {
	// This is a helper that delegates to the main parseDeclaration
	// We cannot move the actual parseDeclaration here due to complex dependencies
	return p.parseDeclaration()
}

// Declaration parsing helper functions

// parseTypeAliasDeclarationHelper helps with type alias declaration parsing
func (p *Parser) parseTypeAliasDeclarationHelper() *TypeAliasDeclaration {
	return p.parseTypeAliasDeclaration()
}

// parseNewtypeDeclarationHelper helps with newtype declaration parsing
func (p *Parser) parseNewtypeDeclarationHelper() *NewtypeDeclaration {
	return p.parseNewtypeDeclaration()
}

// parseImportDeclarationHelper helps with import declaration parsing
func (p *Parser) parseImportDeclarationHelper() *ImportDeclaration {
	return p.parseImportDeclaration()
}

// parseExportDeclarationHelper helps with export declaration parsing
func (p *Parser) parseExportDeclarationHelper() *ExportDeclaration {
	return p.parseExportDeclaration()
}

// parseEffectDeclarationHelper helps with effect declaration parsing
func (p *Parser) parseEffectDeclarationHelper() *EffectDeclaration {
	return p.parseEffectDeclaration()
}

// parseStructDeclarationHelper helps with struct declaration parsing
func (p *Parser) parseStructDeclarationHelper() *StructDeclaration {
	return p.parseStructDeclaration()
}

// parseEnumDeclarationHelper helps with enum declaration parsing
func (p *Parser) parseEnumDeclarationHelper() *EnumDeclaration {
	return p.parseEnumDeclaration()
}

// parseTraitDeclarationHelper helps with trait declaration parsing
func (p *Parser) parseTraitDeclarationHelper() *TraitDeclaration {
	return p.parseTraitDeclaration()
}

// parseImplBlockHelper helps with impl block parsing
func (p *Parser) parseImplBlockHelper() *ImplBlock {
	return p.parseImplBlock()
}

// parseFunctionDeclarationHelper helps with function declaration parsing
func (p *Parser) parseFunctionDeclarationHelper() *FunctionDeclaration {
	return p.parseFunctionDeclaration()
}

// parseVariableDeclarationHelper helps with variable declaration parsing
func (p *Parser) parseVariableDeclarationHelper() *VariableDeclaration {
	return p.parseVariableDeclaration()
}

// =============================================================================
// Declaration Validation Helpers
// =============================================================================

// validateDeclaration validates a parsed declaration
func (p *Parser) validateDeclaration(decl Declaration) error {
	// Implementation for declaration validation
	return nil
}

// checkDeclarationConsistency checks consistency across declarations
func (p *Parser) checkDeclarationConsistency(decls []Declaration) []error {
	// Implementation for consistency checking
	return nil
}

// optimizeDeclaration optimizes a declaration during parsing
func (p *Parser) optimizeDeclaration(decl Declaration) Declaration {
	// Implementation for declaration optimization
	return decl
}

// =============================================================================
// Declaration Error Recovery
// =============================================================================

// recoverFromDeclarationError attempts to recover from declaration parsing errors
func (p *Parser) recoverFromDeclarationError() bool {
	// Look for the next declaration keyword
	for !p.currentTokenIs(lexer.TokenEOF) {
		switch p.current.Type {
		case lexer.TokenFunc, lexer.TokenStruct, lexer.TokenEnum,
			lexer.TokenTrait, lexer.TokenImpl, lexer.TokenTypeKeyword,
			lexer.TokenImport, lexer.TokenExport, lexer.TokenEffect:
			return true
		default:
			p.nextToken()
		}
	}
	return false
}

// suggestDeclarationFix suggests fixes for declaration errors
func (p *Parser) suggestDeclarationFix(expectedType string) []Suggestion {
	suggestions := []Suggestion{}

	// Add common declaration suggestions
	switch expectedType {
	case "function":
		suggestions = append(suggestions, Suggestion{
			Type:        ErrorFix,
			Message:     "Add 'func' keyword before function name",
			Replacement: "func ",
		})
	case "struct":
		suggestions = append(suggestions, Suggestion{
			Type:        ErrorFix,
			Message:     "Add 'struct' keyword before struct name",
			Replacement: "struct ",
		})
	case "type":
		suggestions = append(suggestions, Suggestion{
			Type:        ErrorFix,
			Message:     "Add 'type' keyword for type alias",
			Replacement: "type ",
		})
	}

	return suggestions
}

// =============================================================================
// Declaration Parsing Utilities
// =============================================================================

// parseDeclarationWithRecovery parses a declaration with error recovery
func (p *Parser) parseDeclarationWithRecovery() Declaration {
	// Save current position for recovery
	savedErrors := len(p.errors)

	decl := p.parseDeclaration()

	// If parsing failed, attempt recovery
	if len(p.errors) > savedErrors {
		if p.recoverFromDeclarationError() {
			// Try parsing again after recovery
			decl = p.parseDeclaration()
		}
	}

	return decl
}

// getDeclarationType determines the type of declaration from current token
func (p *Parser) getDeclarationType() string {
	switch p.current.Type {
	case lexer.TokenFunc:
		return "function"
	case lexer.TokenStruct:
		return "struct"
	case lexer.TokenEnum:
		return "enum"
	case lexer.TokenTrait:
		return "trait"
	case lexer.TokenImpl:
		return "impl"
	case lexer.TokenTypeKeyword:
		return "type"
	case lexer.TokenNewtype:
		return "newtype"
	case lexer.TokenImport:
		return "import"
	case lexer.TokenExport:
		return "export"
	case lexer.TokenEffect:
		return "effect"
	case lexer.TokenLet, lexer.TokenConst:
		return "variable"
	default:
		return "unknown"
	}
}

// parseDeclarationBody parses the body of a declaration
func (p *Parser) parseDeclarationBody(declType string) interface{} {
	switch declType {
	case "function":
		return p.parseFunctionDeclaration()
	case "struct":
		return p.parseStructDeclaration()
	case "enum":
		return p.parseEnumDeclaration()
	case "trait":
		return p.parseTraitDeclaration()
	case "impl":
		return p.parseImplBlock()
	case "type":
		return p.parseTypeAliasDeclaration()
	case "newtype":
		return p.parseNewtypeDeclaration()
	case "import":
		return p.parseImportDeclaration()
	case "export":
		return p.parseExportDeclaration()
	case "effect":
		return p.parseEffectDeclaration()
	case "variable":
		return p.parseVariableDeclaration()
	default:
		return nil
	}
}
