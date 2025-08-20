package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func Test__Debug_ErrorToleranceTokens(t *testing.T) {
	input := `
	func good1() { return 1; }
	
	func bad syntax here invalid
	
	func good2() { return 2; }
	
	let invalid = ;
	
	let valid = 42;
	`
	l := lexer.New(input)
	p := NewParser(l, "dbg.oriz")
	prog, errs := p.Parse()
	t.Logf("errs=%d decls=%d", len(errs), len(prog.Declarations))

	for i, d := range prog.Declarations {
		if d == nil {
			t.Logf("decl %d: <nil>", i)

			continue
		}

		switch x := d.(type) {
		case *FunctionDeclaration:
			t.Logf("decl %d: func %s", i, x.Name.Value)
		case *VariableDeclaration:
			t.Logf("decl %d: var %s", i, x.Name.Value)
		case *ExpressionStatement:
			t.Logf("decl %d: expr: %T", i, x.Expression)
		default:
			t.Logf("decl %d: %T", i, x)
		}
	}
}

func Test__Debug_DumpTokens(t *testing.T) {
	input := `
func good1() { return 1; }

func bad syntax here invalid

func good2() { return 2; }

let invalid = ;

let valid = 42;
`

	l := lexer.New(input)
	for i := 0; i < 200; i++ {
		tok := l.NextToken()
		t.Logf("%3d: %s %q", i, tok.Type.String(), tok.Literal)

		if tok.Type == lexer.TokenEOF {
			break
		}
	}
}
