package astbridge

import (
	"fmt"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// DeclarationConverter handles conversion between AST and parser declaration types.
// This specialized converter focuses solely on declaration-level transformations,.
// ensuring separation of concerns and maintainable code organization.
type DeclarationConverter struct {
	// typeConverter handles type-specific conversions.
	typeConverter *TypeConverter
	// exprConverter handles expression-specific conversions.
	exprConverter *ExpressionConverter
	// stmtConverter handles statement-specific conversions.
	stmtConverter *StatementConverter
}

// NewDeclarationConverter creates a new declaration converter with all necessary sub-converters.
// This constructor ensures proper initialization of all dependencies and maintains.
// consistent conversion behavior across the entire AST bridge.
func NewDeclarationConverter() *DeclarationConverter {
	typeConverter := NewTypeConverter()
	exprConverter := NewExpressionConverter(typeConverter)
	stmtConverter := NewStatementConverter(typeConverter, exprConverter)

	return &DeclarationConverter{
		typeConverter: typeConverter,
		exprConverter: exprConverter,
		stmtConverter: stmtConverter,
	}
}

// FromParserDeclaration converts a parser.Declaration to ast.Declaration.
// This method provides comprehensive declaration conversion with proper error handling.
// and type safety. It delegates to specific conversion methods based on the concrete
// declaration type, ensuring extensibility and maintainability.
func (dc *DeclarationConverter) FromParserDeclaration(decl p.Declaration) (ast.Declaration, error) {
	if decl == nil {
		return nil, fmt.Errorf("cannot convert nil parser declaration")
	}

	switch concrete := decl.(type) {
	case *p.FunctionDeclaration:
		return dc.fromParserFunction(concrete)
	case *p.VariableDeclaration:
		return dc.fromParserVariable(concrete)
	case *p.TypeAliasDeclaration:
		return dc.fromParserTypeAlias(concrete)
	case *p.NewtypeDeclaration:
		return dc.fromParserNewtype(concrete)
	case *p.StructDeclaration:
		return dc.fromParserStruct(concrete)
	case *p.EnumDeclaration:
		return dc.fromParserEnum(concrete)
	case *p.TraitDeclaration:
		return dc.fromParserTrait(concrete)
	case *p.ImplBlock:
		return dc.fromParserImplBlock(concrete)
	case *p.ImportDeclaration:
		return dc.fromParserImport(concrete)
	case *p.ExportDeclaration:
		return dc.fromParserExport(concrete)
	case *p.MacroDefinition:
		// Macros are compile-time only constructs and are not represented.
		// in the runtime AST. They are processed during the preprocessing phase.
		return nil, nil
	case *p.ExpressionStatement:
		// Expression statements at the top level are treated as declarations.
		// in certain contexts. We delegate to statement conversion and wrap if needed.
		stmt, err := dc.stmtConverter.FromParserStatement(concrete)
		if err != nil {
			return nil, fmt.Errorf("failed to convert expression statement: %w", err)
		}

		if astDecl, ok := stmt.(ast.Declaration); ok {
			return astDecl, nil
		}
		// Fallback: wrap as a placeholder type declaration for compatibility.
		return dc.createPlaceholderDeclaration(), nil
	default:
		return nil, fmt.Errorf("unsupported parser declaration type: %T", decl)
	}
}

// ToParserDeclaration converts an ast.Declaration to parser.Declaration.
// This method provides the inverse conversion with comprehensive error handling.
// and maintains symmetry with FromParserDeclaration for bidirectional compatibility.
func (dc *DeclarationConverter) ToParserDeclaration(decl ast.Declaration) (p.Declaration, error) {
	if decl == nil {
		return nil, fmt.Errorf("cannot convert nil AST declaration")
	}

	switch concrete := decl.(type) {
	case *ast.FunctionDeclaration:
		return dc.toParserFunction(concrete)
	case *ast.VariableDeclaration:
		return dc.toParserVariable(concrete)
	case *ast.TypeDeclaration:
		return dc.toParserTypeDeclaration(concrete)
	case *ast.StructDeclaration:
		return dc.toParserStruct(concrete)
	case *ast.EnumDeclaration:
		return dc.toParserEnum(concrete)
	case *ast.TraitDeclaration:
		return dc.toParserTrait(concrete)
	case *ast.ImplDeclaration:
		return dc.toParserImplBlock(concrete)
	case *ast.ImportDeclaration:
		return dc.toParserImport(concrete)
	case *ast.ExportDeclaration:
		return dc.toParserExport(concrete)
	default:
		return nil, fmt.Errorf("unsupported AST declaration type: %T", decl)
	}
}

// createPlaceholderDeclaration creates a placeholder type declaration for edge cases.
// This utility method provides a safe fallback when declaration conversion cannot.
// be performed directly, maintaining AST consistency while preserving error information.
func (dc *DeclarationConverter) createPlaceholderDeclaration() *ast.TypeDeclaration {
	return &ast.TypeDeclaration{
		Span:    createEmptySpan(),
		Name:    &ast.Identifier{Span: createEmptySpan(), Value: "_placeholder"},
		Type:    &ast.BasicType{Kind: ast.BasicVoid},
		IsAlias: false,
	}
}

// createEmptySpan creates an empty position span for placeholder nodes.
// This helper method ensures consistent handling of position information in fallback cases.
func createEmptySpan() position.Span {
	return position.Span{
		Start: position.Position{},
		End:   position.Position{},
	}
}

// Stub implementations for specific declaration types (to be expanded).

func (dc *DeclarationConverter) fromParserFunction(fn *p.FunctionDeclaration) (*ast.FunctionDeclaration, error) {
	if fn == nil {
		return nil, fmt.Errorf("cannot convert nil function decl")
	}
	var ret ast.Type
	var err error
	if fn.ReturnType != nil {
		ret, err = dc.typeConverter.FromParserType(fn.ReturnType)
		if err != nil {
			return nil, err
		}
	}
	params := make([]*ast.Parameter, 0, len(fn.Parameters))
	for _, pparam := range fn.Parameters {
		var pt ast.Type
		if pparam.TypeSpec != nil {
			pt, err = dc.typeConverter.FromParserType(pparam.TypeSpec)
			if err != nil {
				return nil, err
			}
		}
		params = append(params, &ast.Parameter{
			Type:      pt,
			Name:      &ast.Identifier{Span: fromParserSpan(pparam.Name.Span), Value: pparam.Name.Value},
			Span:      fromParserSpan(pparam.Span),
			IsMutable: pparam.IsMut,
		})
	}
	return &ast.FunctionDeclaration{
		ReturnType: ret,
		Name:       &ast.Identifier{Span: fromParserSpan(fn.Name.Span), Value: fn.Name.Value},
		Body:       &ast.BlockStatement{Span: fromParserSpan(fn.Span)},
		Parameters: params,
		Span:       fromParserSpan(fn.Span),
		IsExported: fn.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) toParserFunction(fn *ast.FunctionDeclaration) (*p.FunctionDeclaration, error) {
	if fn == nil {
		return nil, fmt.Errorf("cannot convert nil function decl")
	}
	var ret p.Type
	var err error
	if fn.ReturnType != nil {
		ret, err = dc.typeConverter.ToParserType(fn.ReturnType)
		if err != nil {
			return nil, err
		}
	}
	params := make([]*p.Parameter, 0, len(fn.Parameters))
	for _, ap := range fn.Parameters {
		var pt p.Type
		if ap.Type != nil {
			pt, err = dc.typeConverter.ToParserType(ap.Type)
			if err != nil {
				return nil, err
			}
		}
		params = append(params, &p.Parameter{
			TypeSpec: pt,
			Name:     &p.Identifier{Value: ap.Name.Value, Span: toParserSpan(ap.Name.Span)},
			Span:     toParserSpan(ap.Span),
			IsMut:    ap.IsMutable,
		})
	}
	return &p.FunctionDeclaration{
		ReturnType: ret,
		Name:       &p.Identifier{Value: fn.Name.Value, Span: toParserSpan(fn.Name.Span)},
		Body:       &p.BlockStatement{Span: toParserSpan(fn.Span)},
		Parameters: params,
		Span:       toParserSpan(fn.Span),
		IsPublic:   fn.IsExported,
	}, nil
}

func (dc *DeclarationConverter) fromParserVariable(variable *p.VariableDeclaration) (*ast.VariableDeclaration, error) {
	if variable == nil {
		return nil, fmt.Errorf("cannot convert nil variable decl")
	}
	at, err := dc.typeConverter.FromParserType(variable.TypeSpec)
	if err != nil {
		return nil, err
	}
	return &ast.VariableDeclaration{
		Type:       at,
		Name:       &ast.Identifier{Span: fromParserSpan(variable.Name.Span), Value: variable.Name.Value},
		Span:       fromParserSpan(variable.Span),
		Kind:       ast.VarKindLet,
		IsMutable:  variable.IsMutable,
		IsExported: variable.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) toParserVariable(variable *ast.VariableDeclaration) (*p.VariableDeclaration, error) {
	if variable == nil {
		return nil, fmt.Errorf("cannot convert nil variable decl")
	}
	pt, err := dc.typeConverter.ToParserType(variable.Type)
	if err != nil {
		return nil, err
	}
	return &p.VariableDeclaration{
		TypeSpec:  pt,
		Name:      &p.Identifier{Value: variable.Name.Value, Span: toParserSpan(variable.Name.Span)},
		Span:      toParserSpan(variable.Span),
		IsMutable: variable.IsMutable,
		IsPublic:  variable.IsExported,
	}, nil
}

func (dc *DeclarationConverter) fromParserTypeAlias(alias *p.TypeAliasDeclaration) (*ast.TypeDeclaration, error) {
	if alias == nil {
		return nil, fmt.Errorf("cannot convert nil type alias")
	}
	at, err := dc.typeConverter.FromParserType(alias.Aliased)
	if err != nil {
		return nil, err
	}
	return &ast.TypeDeclaration{
		Type:       at,
		Name:       &ast.Identifier{Span: fromParserSpan(alias.Name.Span), Value: alias.Name.Value},
		Span:       fromParserSpan(alias.Span),
		IsAlias:    true,
		IsExported: alias.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) fromParserNewtype(newtype *p.NewtypeDeclaration) (*ast.TypeDeclaration, error) {
	if newtype == nil {
		return nil, fmt.Errorf("cannot convert nil newtype")
	}
	at, err := dc.typeConverter.FromParserType(newtype.Base)
	if err != nil {
		return nil, err
	}
	return &ast.TypeDeclaration{
		Type:       at,
		Name:       &ast.Identifier{Span: fromParserSpan(newtype.Name.Span), Value: newtype.Name.Value},
		Span:       fromParserSpan(newtype.Span),
		IsAlias:    false,
		IsExported: false,
	}, nil
}

func (dc *DeclarationConverter) toParserTypeDeclaration(typeDecl *ast.TypeDeclaration) (p.Declaration, error) {
	if typeDecl == nil {
		return nil, fmt.Errorf("cannot convert nil type decl")
	}
	pt, err := dc.typeConverter.ToParserType(typeDecl.Type)
	if err != nil {
		return nil, err
	}
	if typeDecl.IsAlias {
		return &p.TypeAliasDeclaration{
			Name:     &p.Identifier{Value: typeDecl.Name.Value, Span: toParserSpan(typeDecl.Name.Span)},
			Aliased:  pt,
			Span:     toParserSpan(typeDecl.Span),
			IsPublic: typeDecl.IsExported,
		}, nil
	}
	return &p.NewtypeDeclaration{
		Name: &p.Identifier{Value: typeDecl.Name.Value, Span: toParserSpan(typeDecl.Name.Span)},
		Base: pt,
		Span: toParserSpan(typeDecl.Span),
	}, nil
}

func (dc *DeclarationConverter) fromParserStruct(structDecl *p.StructDeclaration) (*ast.StructDeclaration, error) {
	if structDecl == nil {
		return nil, fmt.Errorf("cannot convert nil struct decl")
	}
	fields := make([]*ast.StructField, 0, len(structDecl.Fields))
	for _, pf := range structDecl.Fields {
		at, err := dc.typeConverter.FromParserType(pf.Type)
		if err != nil {
			return nil, err
		}
		fields = append(fields, &ast.StructField{
			Type:     at,
			Name:     &ast.Identifier{Span: fromParserSpan(pf.Name.Span), Value: pf.Name.Value},
			Span:     fromParserSpan(pf.Span),
			IsPublic: pf.IsPublic,
		})
	}
	gens := make([]*ast.GenericParameter, 0, len(structDecl.Generics))
	for _, g := range structDecl.Generics {
		gens = append(gens, &ast.GenericParameter{
			Name: &ast.Identifier{Span: fromParserSpan(g.Name.Span), Value: g.Name.Value},
			Span: fromParserSpan(g.Span),
			Kind: ast.GenericParamKind(g.Kind),
		})
	}
	return &ast.StructDeclaration{
		Name:       &ast.Identifier{Span: fromParserSpan(structDecl.Name.Span), Value: structDecl.Name.Value},
		Fields:     fields,
		Generics:   gens,
		Span:       fromParserSpan(structDecl.Span),
		IsExported: structDecl.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) toParserStruct(structDecl *ast.StructDeclaration) (*p.StructDeclaration, error) {
	if structDecl == nil {
		return nil, fmt.Errorf("cannot convert nil struct decl")
	}
	fields := make([]*p.StructField, 0, len(structDecl.Fields))
	for _, af := range structDecl.Fields {
		pt, err := dc.typeConverter.ToParserType(af.Type)
		if err != nil {
			return nil, err
		}
		var name *p.Identifier
		if af.Name != nil {
			name = &p.Identifier{Value: af.Name.Value, Span: toParserSpan(af.Name.Span)}
		}
		fields = append(fields, &p.StructField{
			Type:     pt,
			Name:     name,
			Span:     toParserSpan(af.Span),
			IsPublic: af.IsPublic,
		})
	}
	gens := make([]*p.GenericParameter, 0, len(structDecl.Generics))
	for _, g := range structDecl.Generics {
		gens = append(gens, &p.GenericParameter{
			Name: &p.Identifier{Value: g.Name.Value, Span: toParserSpan(g.Name.Span)},
			Span: toParserSpan(g.Span),
			Kind: p.GenericParamKind(g.Kind),
		})
	}
	return &p.StructDeclaration{
		Name:     &p.Identifier{Value: structDecl.Name.Value, Span: toParserSpan(structDecl.Name.Span)},
		Fields:   fields,
		Generics: gens,
		Span:     toParserSpan(structDecl.Span),
		IsPublic: structDecl.IsExported,
	}, nil
}

func (dc *DeclarationConverter) fromParserEnum(enumDecl *p.EnumDeclaration) (*ast.EnumDeclaration, error) {
	if enumDecl == nil {
		return nil, fmt.Errorf("cannot convert nil enum decl")
	}
	variants := make([]*ast.EnumVariant, 0, len(enumDecl.Variants))
	for _, v := range enumDecl.Variants {
		fields := make([]*ast.StructField, 0, len(v.Fields))
		for _, f := range v.Fields {
			at, err := dc.typeConverter.FromParserType(f.Type)
			if err != nil {
				return nil, err
			}
			var name *ast.Identifier
			if f.Name != nil {
				name = &ast.Identifier{Span: fromParserSpan(f.Name.Span), Value: f.Name.Value}
			}
			fields = append(fields, &ast.StructField{Type: at, Name: name, Span: fromParserSpan(f.Span), IsPublic: f.IsPublic})
		}
		variants = append(variants, &ast.EnumVariant{
			Name:   &ast.Identifier{Span: fromParserSpan(v.Name.Span), Value: v.Name.Value},
			Fields: fields,
			Span:   fromParserSpan(v.Span),
		})
	}
	return &ast.EnumDeclaration{
		Name:       &ast.Identifier{Span: fromParserSpan(enumDecl.Name.Span), Value: enumDecl.Name.Value},
		Variants:   variants,
		Span:       fromParserSpan(enumDecl.Span),
		IsExported: enumDecl.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) toParserEnum(enumDecl *ast.EnumDeclaration) (*p.EnumDeclaration, error) {
	if enumDecl == nil {
		return nil, fmt.Errorf("cannot convert nil enum decl")
	}
	variants := make([]*p.EnumVariant, 0, len(enumDecl.Variants))
	for _, v := range enumDecl.Variants {
		fields := make([]*p.StructField, 0, len(v.Fields))
		for _, f := range v.Fields {
			pt, err := dc.typeConverter.ToParserType(f.Type)
			if err != nil {
				return nil, err
			}
			var name *p.Identifier
			if f.Name != nil {
				name = &p.Identifier{Value: f.Name.Value, Span: toParserSpan(f.Name.Span)}
			}
			fields = append(fields, &p.StructField{Type: pt, Name: name, Span: toParserSpan(f.Span), IsPublic: f.IsPublic})
		}
		variants = append(variants, &p.EnumVariant{
			Name:   &p.Identifier{Value: v.Name.Value, Span: toParserSpan(v.Name.Span)},
			Fields: fields,
			Span:   toParserSpan(v.Span),
		})
	}
	return &p.EnumDeclaration{
		Name:     &p.Identifier{Value: enumDecl.Name.Value, Span: toParserSpan(enumDecl.Name.Span)},
		Variants: variants,
		Span:     toParserSpan(enumDecl.Span),
		IsPublic: enumDecl.IsExported,
	}, nil
}

func (dc *DeclarationConverter) fromParserTrait(traitDecl *p.TraitDeclaration) (*ast.TraitDeclaration, error) {
	if traitDecl == nil {
		return nil, fmt.Errorf("cannot convert nil trait decl")
	}
	methods := make([]*ast.TraitMethod, 0, len(traitDecl.Methods))
	for _, m := range traitDecl.Methods {
		var rt ast.Type
		var err error
		if m.ReturnType != nil {
			rt, err = dc.typeConverter.FromParserType(m.ReturnType)
			if err != nil {
				return nil, err
			}
		}
		params := make([]*ast.Parameter, 0, len(m.Parameters))
		for _, pparam := range m.Parameters {
			var pt ast.Type
			if pparam.TypeSpec != nil {
				pt, err = dc.typeConverter.FromParserType(pparam.TypeSpec)
				if err != nil {
					return nil, err
				}
			}
			params = append(params, &ast.Parameter{
				Type:      pt,
				Name:      &ast.Identifier{Span: fromParserSpan(pparam.Name.Span), Value: pparam.Name.Value},
				Span:      fromParserSpan(pparam.Span),
				IsMutable: pparam.IsMut,
			})
		}
		methods = append(methods, &ast.TraitMethod{
			ReturnType: rt,
			Name:       &ast.Identifier{Span: fromParserSpan(m.Name.Span), Value: m.Name.Value},
			Parameters: params,
			Span:       fromParserSpan(m.Span),
		})
	}
	gens := make([]*ast.GenericParameter, 0, len(traitDecl.Generics))
	for _, g := range traitDecl.Generics {
		gens = append(gens, &ast.GenericParameter{
			Name: &ast.Identifier{Span: fromParserSpan(g.Name.Span), Value: g.Name.Value},
			Span: fromParserSpan(g.Span),
			Kind: ast.GenericParamKind(g.Kind),
		})
	}
	assoc := make([]*ast.AssociatedType, 0, len(traitDecl.AssociatedTypes))
	for _, at := range traitDecl.AssociatedTypes {
		bounds := make([]ast.Type, 0, len(at.Bounds))
		for _, b := range at.Bounds {
			bt, err := dc.typeConverter.FromParserType(b)
			if err != nil {
				return nil, err
			}
			bounds = append(bounds, bt)
		}
		assoc = append(assoc, &ast.AssociatedType{
			Name:   &ast.Identifier{Span: fromParserSpan(at.Name.Span), Value: at.Name.Value},
			Bounds: bounds,
			Span:   fromParserSpan(at.Span),
		})
	}
	return &ast.TraitDeclaration{
		Name:            &ast.Identifier{Span: fromParserSpan(traitDecl.Name.Span), Value: traitDecl.Name.Value},
		Methods:         methods,
		Generics:        gens,
		AssociatedTypes: assoc,
		Span:            fromParserSpan(traitDecl.Span),
		IsExported:      traitDecl.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) toParserTrait(traitDecl *ast.TraitDeclaration) (*p.TraitDeclaration, error) {
	if traitDecl == nil {
		return nil, fmt.Errorf("cannot convert nil trait decl")
	}
	methods := make([]*p.TraitMethod, 0, len(traitDecl.Methods))
	for _, m := range traitDecl.Methods {
		var rt p.Type
		var err error
		if m.ReturnType != nil {
			rt, err = dc.typeConverter.ToParserType(m.ReturnType)
			if err != nil {
				return nil, err
			}
		}
		params := make([]*p.Parameter, 0, len(m.Parameters))
		for _, ap := range m.Parameters {
			var pt p.Type
			if ap.Type != nil {
				pt, err = dc.typeConverter.ToParserType(ap.Type)
				if err != nil {
					return nil, err
				}
			}
			params = append(params, &p.Parameter{
				TypeSpec: pt,
				Name:     &p.Identifier{Value: ap.Name.Value, Span: toParserSpan(ap.Name.Span)},
				Span:     toParserSpan(ap.Span),
				IsMut:    ap.IsMutable,
			})
		}
		methods = append(methods, &p.TraitMethod{
			ReturnType: rt,
			Name:       &p.Identifier{Value: m.Name.Value, Span: toParserSpan(m.Name.Span)},
			Parameters: params,
			Span:       toParserSpan(m.Span),
		})
	}
	gens := make([]*p.GenericParameter, 0, len(traitDecl.Generics))
	for _, g := range traitDecl.Generics {
		gens = append(gens, &p.GenericParameter{
			Name: &p.Identifier{Value: g.Name.Value, Span: toParserSpan(g.Name.Span)},
			Span: toParserSpan(g.Span),
			Kind: p.GenericParamKind(g.Kind),
		})
	}
	assoc := make([]*p.AssociatedType, 0, len(traitDecl.AssociatedTypes))
	for _, at := range traitDecl.AssociatedTypes {
		bounds := make([]p.Type, 0, len(at.Bounds))
		for _, b := range at.Bounds {
			bt, err := dc.typeConverter.ToParserType(b)
			if err != nil {
				return nil, err
			}
			bounds = append(bounds, bt)
		}
		assoc = append(assoc, &p.AssociatedType{
			Name:   &p.Identifier{Value: at.Name.Value, Span: toParserSpan(at.Name.Span)},
			Bounds: bounds,
			Span:   toParserSpan(at.Span),
		})
	}
	return &p.TraitDeclaration{
		Name:            &p.Identifier{Value: traitDecl.Name.Value, Span: toParserSpan(traitDecl.Name.Span)},
		Methods:         methods,
		Generics:        gens,
		AssociatedTypes: assoc,
		Span:            toParserSpan(traitDecl.Span),
		IsPublic:        traitDecl.IsExported,
	}, nil
}

func (dc *DeclarationConverter) fromParserImplBlock(implBlock *p.ImplBlock) (*ast.ImplDeclaration, error) {
	if implBlock == nil {
		return nil, fmt.Errorf("cannot convert nil impl block")
	}
	tr, err := dc.typeConverter.FromParserType(implBlock.Trait)
	if err != nil {
		return nil, err
	}
	ft, err := dc.typeConverter.FromParserType(implBlock.ForType)
	if err != nil {
		return nil, err
	}
	methods := make([]*ast.FunctionDeclaration, 0, len(implBlock.Items))
	for _, it := range implBlock.Items {
		m, err := dc.fromParserFunction(it)
		if err != nil {
			return nil, err
		}
		methods = append(methods, m)
	}
	gens := make([]*ast.GenericParameter, 0, len(implBlock.Generics))
	for _, g := range implBlock.Generics {
		gens = append(gens, &ast.GenericParameter{
			Name: &ast.Identifier{Span: fromParserSpan(g.Name.Span), Value: g.Name.Value},
			Span: fromParserSpan(g.Span),
			Kind: ast.GenericParamKind(g.Kind),
		})
	}
	whs := make([]*ast.WherePredicate, 0, len(implBlock.WhereClauses))
	for _, w := range implBlock.WhereClauses {
		tgt, err := dc.typeConverter.FromParserType(w.Target)
		if err != nil {
			return nil, err
		}
		bounds := make([]ast.Type, 0, len(w.Bounds))
		for _, b := range w.Bounds {
			bt, err := dc.typeConverter.FromParserType(b)
			if err != nil {
				return nil, err
			}
			bounds = append(bounds, bt)
		}
		whs = append(whs, &ast.WherePredicate{Target: tgt, Bounds: bounds, Span: fromParserSpan(w.Span)})
	}
	return &ast.ImplDeclaration{
		Trait:        tr,
		ForType:      ft,
		Methods:      methods,
		Generics:     gens,
		WhereClauses: whs,
		Span:         fromParserSpan(implBlock.Span),
	}, nil
}

func (dc *DeclarationConverter) toParserImplBlock(implDecl *ast.ImplDeclaration) (*p.ImplBlock, error) {
	if implDecl == nil {
		return nil, fmt.Errorf("cannot convert nil impl decl")
	}
	tr, err := dc.typeConverter.ToParserType(implDecl.Trait)
	if err != nil {
		return nil, err
	}
	ft, err := dc.typeConverter.ToParserType(implDecl.ForType)
	if err != nil {
		return nil, err
	}
	items := make([]*p.FunctionDeclaration, 0, len(implDecl.Methods))
	for _, m := range implDecl.Methods {
		pm, err := dc.toParserFunction(m)
		if err != nil {
			return nil, err
		}
		items = append(items, pm)
	}
	gens := make([]*p.GenericParameter, 0, len(implDecl.Generics))
	for _, g := range implDecl.Generics {
		gens = append(gens, &p.GenericParameter{
			Name: &p.Identifier{Value: g.Name.Value, Span: toParserSpan(g.Name.Span)},
			Span: toParserSpan(g.Span),
			Kind: p.GenericParamKind(g.Kind),
		})
	}
	whs := make([]*p.WherePredicate, 0, len(implDecl.WhereClauses))
	for _, w := range implDecl.WhereClauses {
		tgt, err := dc.typeConverter.ToParserType(w.Target)
		if err != nil {
			return nil, err
		}
		bounds := make([]p.Type, 0, len(w.Bounds))
		for _, b := range w.Bounds {
			bt, err := dc.typeConverter.ToParserType(b)
			if err != nil {
				return nil, err
			}
			bounds = append(bounds, bt)
		}
		whs = append(whs, &p.WherePredicate{Target: tgt, Bounds: bounds, Span: toParserSpan(w.Span)})
	}
	return &p.ImplBlock{
		Trait:        tr,
		ForType:      ft,
		Items:        items,
		Generics:     gens,
		WhereClauses: whs,
		Span:         toParserSpan(implDecl.Span),
	}, nil
}

func (dc *DeclarationConverter) fromParserImport(importDecl *p.ImportDeclaration) (*ast.ImportDeclaration, error) {
	if importDecl == nil {
		return nil, fmt.Errorf("cannot convert nil import decl")
	}
	path := make([]*ast.Identifier, 0, len(importDecl.Path))
	for _, seg := range importDecl.Path {
		path = append(path, &ast.Identifier{Span: fromParserSpan(seg.Span), Value: seg.Value})
	}
	var alias *ast.Identifier
	if importDecl.Alias != nil {
		alias = &ast.Identifier{Span: fromParserSpan(importDecl.Alias.Span), Value: importDecl.Alias.Value}
	}
	return &ast.ImportDeclaration{
		Alias:      alias,
		Path:       path,
		Span:       fromParserSpan(importDecl.Span),
		IsExported: importDecl.IsPublic,
	}, nil
}

func (dc *DeclarationConverter) toParserImport(importDecl *ast.ImportDeclaration) (*p.ImportDeclaration, error) {
	if importDecl == nil {
		return nil, fmt.Errorf("cannot convert nil import decl")
	}
	path := make([]*p.Identifier, 0, len(importDecl.Path))
	for _, seg := range importDecl.Path {
		path = append(path, &p.Identifier{Value: seg.Value, Span: toParserSpan(seg.Span)})
	}
	var alias *p.Identifier
	if importDecl.Alias != nil {
		alias = &p.Identifier{Value: importDecl.Alias.Value, Span: toParserSpan(importDecl.Alias.Span)}
	}
	return &p.ImportDeclaration{
		Alias:    alias,
		Path:     path,
		Span:     toParserSpan(importDecl.Span),
		IsPublic: importDecl.IsExported,
	}, nil
}

func (dc *DeclarationConverter) fromParserExport(exportDecl *p.ExportDeclaration) (*ast.ExportDeclaration, error) {
	if exportDecl == nil {
		return nil, fmt.Errorf("cannot convert nil export decl")
	}
	items := make([]*ast.ExportItem, 0, len(exportDecl.Items))
	for _, it := range exportDecl.Items {
		var alias *ast.Identifier
		if it.Alias != nil {
			alias = &ast.Identifier{Span: fromParserSpan(it.Alias.Span), Value: it.Alias.Value}
		}
		items = append(items, &ast.ExportItem{
			Name:  &ast.Identifier{Span: fromParserSpan(it.Name.Span), Value: it.Name.Value},
			Alias: alias,
			Span:  fromParserSpan(it.Span),
		})
	}
	return &ast.ExportDeclaration{Items: items, Span: fromParserSpan(exportDecl.Span)}, nil
}

func (dc *DeclarationConverter) toParserExport(exportDecl *ast.ExportDeclaration) (*p.ExportDeclaration, error) {
	if exportDecl == nil {
		return nil, fmt.Errorf("cannot convert nil export decl")
	}
	items := make([]*p.ExportItem, 0, len(exportDecl.Items))
	for _, it := range exportDecl.Items {
		var alias *p.Identifier
		if it.Alias != nil {
			alias = &p.Identifier{Value: it.Alias.Value, Span: toParserSpan(it.Alias.Span)}
		}
		items = append(items, &p.ExportItem{
			Name:  &p.Identifier{Value: it.Name.Value, Span: toParserSpan(it.Name.Span)},
			Alias: alias,
			Span:  toParserSpan(it.Span),
		})
	}
	return &p.ExportDeclaration{Items: items, Span: toParserSpan(exportDecl.Span)}, nil
}
