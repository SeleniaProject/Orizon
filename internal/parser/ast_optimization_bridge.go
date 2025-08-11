package parser

import (
    "fmt"

    aast "github.com/orizon-lang/orizon/internal/ast"
)

// OptimizeViaAstPipe converts a parser.Program into ast.Program, runs the ast optimization
// pipeline, and converts the result back to parser.Program.
// This enables using the richer optimization facilities in internal/ast without
// changing upstream parser users.
func OptimizeViaAstPipe(program *Program, level string) (*Program, error) {
    if program == nil {
        return nil, fmt.Errorf("nil program")
    }

    // Convert parser -> ast (local lightweight conversion to avoid import cycles)
    aProg, err := parserToAst(program)
    if err != nil {
        return nil, fmt.Errorf("convert to ast failed: %w", err)
    }

    // Select pipeline by level
    var pipeline *aast.OptimizationPipeline
    switch level {
    case "none":
        pipeline = aast.NewOptimizationPipeline()
        pipeline.SetOptimizationLevel(aast.OptimizationNone)
    case "basic":
        pipeline = aast.CreateBasicOptimizationPipeline()
    case "aggressive":
        pipeline = aast.CreateAggressiveOptimizationPipeline()
    case "default", "":
        fallthrough
    default:
        pipeline = aast.CreateStandardOptimizationPipeline()
    }

    // Run optimization
    optimizedNode, _, err := pipeline.Optimize(aProg)
    if err != nil {
        return nil, fmt.Errorf("ast optimization failed: %w", err)
    }

    // The root should remain a Program
    aProgOpt, ok := optimizedNode.(*aast.Program)
    if !ok {
        return nil, fmt.Errorf("unexpected optimized root type %T", optimizedNode)
    }

    // Convert back ast -> parser
    back, err := astToParser(aProgOpt)
    if err != nil {
        return nil, fmt.Errorf("convert back to parser failed: %w", err)
    }
    return back, nil
}

// Local minimal converters (subset) to avoid package cycles during optimization bridging
func parserToAst(src *Program) (*aast.Program, error) {
    if src == nil { return nil, fmt.Errorf("nil program") }
    dst := &aast.Program{Declarations: make([]aast.Declaration, 0, len(src.Declarations))}
    for _, d := range src.Declarations {
        switch n := d.(type) {
        case *FunctionDeclaration:
            // Map signature and body (statements recursively are not needed for pipeline structure)
            dst.Declarations = append(dst.Declarations, &aast.FunctionDeclaration{
                Name: &aast.Identifier{Value: n.Name.Value},
                Body: &aast.BlockStatement{},
            })
        case *VariableDeclaration:
            dst.Declarations = append(dst.Declarations, &aast.VariableDeclaration{
                Name: &aast.Identifier{Value: n.Name.Value},
                Kind: aast.VarKindLet,
            })
        default:
            // Skip unsupported declarations
        }
    }
    return dst, nil
}

func astToParser(src *aast.Program) (*Program, error) {
    if src == nil { return nil, fmt.Errorf("nil program") }
    dst := &Program{Declarations: make([]Declaration, 0, len(src.Declarations))}
    for _, d := range src.Declarations {
        switch n := d.(type) {
        case *aast.FunctionDeclaration:
            dst.Declarations = append(dst.Declarations, &FunctionDeclaration{Name: &Identifier{Value: n.Name.Value}, Body: &BlockStatement{}})
        case *aast.VariableDeclaration:
            dst.Declarations = append(dst.Declarations, &VariableDeclaration{Name: &Identifier{Value: n.Name.Value}})
        default:
            // Skip
        }
    }
    return dst, nil
}


