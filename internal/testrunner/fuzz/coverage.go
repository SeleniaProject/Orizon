package fuzz

import (
	"github.com/orizon-lang/orizon/internal/lexer"
)

// TokenEdgeCoverage computes a simple input-derived coverage: pairs of adjacent token types.
// Each edge is encoded as uint64: (uint64(prev)<<32)|uint64(curr).
func TokenEdgeCoverage(input string) []uint64 {
	lx := lexer.NewWithFilename(input, "coverage.oriz")
	// prime first token
	t := lx.NextToken()
	prev := uint64(t.Type)
	edges := make([]uint64, 0, 256)
	for {
		nt := lx.NextToken()
		curr := uint64(nt.Type)
		edge := (prev << 32) | curr
		edges = append(edges, edge)
		if nt.Type == lexer.TokenEOF {
			break
		}
		prev = curr
	}
	return edges
}

