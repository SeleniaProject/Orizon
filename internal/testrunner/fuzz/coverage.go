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

// WeightedTokenEdgeCoverage adds a simple weighting for variety: multiply edges by
// a small prime that depends on token class bands. This helps differentiate inputs
// that share structure but vary in operator/identifier density.
func WeightedTokenEdgeCoverage(input string) []uint64 {
	lx := lexer.NewWithFilename(input, "coverage_weighted.oriz")
	t := lx.NextToken()
	prev := uint64(t.Type)
	edges := make([]uint64, 0, 256)
	for {
		nt := lx.NextToken()
		curr := uint64(nt.Type)
		edge := (prev << 32) | curr
		// weight by coarse token band (ident/literal/op/keyword)
		var w uint64 = 1
		switch nt.Type {
		case lexer.TokenIdentifier:
			w = 3
		case lexer.TokenInteger, lexer.TokenFloat, lexer.TokenString, lexer.TokenChar, lexer.TokenBool:
			w = 5
		case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMul, lexer.TokenDiv, lexer.TokenMod, lexer.TokenPower,
			lexer.TokenAssign, lexer.TokenPlusAssign, lexer.TokenMinusAssign, lexer.TokenMulAssign, lexer.TokenDivAssign, lexer.TokenModAssign,
			lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe,
			lexer.TokenAnd, lexer.TokenOr, lexer.TokenNot,
			lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor, lexer.TokenBitNot, lexer.TokenShl, lexer.TokenShr,
			lexer.TokenBitAndAssign, lexer.TokenBitOrAssign, lexer.TokenBitXorAssign, lexer.TokenShlAssign, lexer.TokenShrAssign:
			w = 7
		default:
			w = 2
		}
		edges = append(edges, edge*w)
		if nt.Type == lexer.TokenEOF {
			break
		}
		prev = curr
	}
	return edges
}

// TokenTrigramCoverage computes coverage for token trigrams (prev, mid, curr) by
// packing three token types into a uint64. Packing uses 21 bits per token type
// (sufficient for typical enum ranges): (prev<<42) | (mid<<21) | curr.
func TokenTrigramCoverage(input string) []uint64 {
	lx := lexer.NewWithFilename(input, "coverage_trigram.oriz")
	a := lx.NextToken()
	b := lx.NextToken()
	trigrams := make([]uint64, 0, 256)
	prev := uint64(a.Type)
	mid := uint64(b.Type)
	for {
		c := lx.NextToken()
		curr := uint64(c.Type)
		trig := (prev << 42) | (mid << 21) | curr
		trigrams = append(trigrams, trig)
		if c.Type == lexer.TokenEOF {
			break
		}
		prev, mid = mid, curr
	}
	return trigrams
}

// ComputeCoverage computes coverage based on the given mode:
//   - "edge": TokenEdgeCoverage
//   - "weighted": WeightedTokenEdgeCoverage (default)
//   - "trigram": TokenTrigramCoverage
//   - "both": union of WeightedTokenEdgeCoverage and TokenEdgeCoverage
func ComputeCoverage(mode, input string) []uint64 {
	switch mode {
	case "edge":
		return TokenEdgeCoverage(input)
	case "trigram":
		return TokenTrigramCoverage(input)
	case "both":
		e := TokenEdgeCoverage(input)
		w := WeightedTokenEdgeCoverage(input)
		// Union by simple append (dedup handled by caller via set if needed)
		return append(w, e...)
	case "weighted", "":
		fallthrough
	default:
		return WeightedTokenEdgeCoverage(input)
	}
}
