// Type-safe extensions for existing AST nodes
// This file extends existing AST nodes with TypeSafeNode interface implementations

package parser

import "fmt"

// ====== TypeSafeNode implementations for existing nodes ======

// Program TypeSafeNode implementation
func (p *Program) GetNodeKind() NodeKind { return NodeKindProgram }
func (p *Program) Clone() TypeSafeNode {
	clone := *p
	clone.Declarations = make([]Declaration, len(p.Declarations))
	for i, decl := range p.Declarations {
		clone.Declarations[i] = decl.(TypeSafeNode).Clone().(Declaration)
	}
	return &clone
}
func (p *Program) Equals(other TypeSafeNode) bool {
	if op, ok := other.(*Program); ok {
		if len(p.Declarations) != len(op.Declarations) {
			return false
		}
		for i, decl := range p.Declarations {
			if !decl.(TypeSafeNode).Equals(op.Declarations[i].(TypeSafeNode)) {
				return false
			}
		}
		return true
	}
	return false
}
func (p *Program) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, len(p.Declarations))
	for i, decl := range p.Declarations {
		children[i] = decl.(TypeSafeNode)
	}
	return children
}
func (p *Program) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= len(p.Declarations) {
		return fmt.Errorf("index %d out of range for Program children", index)
	}
	if newDecl, ok := newChild.(Declaration); ok {
		p.Declarations[index] = newDecl
		return nil
	}
	return fmt.Errorf("expected Declaration, got %T", newChild)
}

// FunctionDeclaration TypeSafeNode implementation
func (f *FunctionDeclaration) GetNodeKind() NodeKind { return NodeKindFunctionDeclaration }
func (f *FunctionDeclaration) Clone() TypeSafeNode {
	clone := *f
	if f.Name != nil {
		clone.Name = f.Name.Clone().(*Identifier)
	}
	clone.Parameters = make([]*Parameter, len(f.Parameters))
	for i, param := range f.Parameters {
		clone.Parameters[i] = param.Clone().(*Parameter)
	}
	if f.ReturnType != nil {
		clone.ReturnType = f.ReturnType.(TypeSafeNode).Clone().(Type)
	}
	if f.Body != nil {
		clone.Body = f.Body.Clone().(*BlockStatement)
	}
	return &clone
}
func (f *FunctionDeclaration) Equals(other TypeSafeNode) bool {
	if of, ok := other.(*FunctionDeclaration); ok {
		return f.IsPublic == of.IsPublic &&
			len(f.Parameters) == len(of.Parameters) &&
			((f.Name == nil && of.Name == nil) ||
				(f.Name != nil && of.Name != nil && f.Name.Equals(of.Name))) &&
			((f.ReturnType == nil && of.ReturnType == nil) ||
				(f.ReturnType != nil && of.ReturnType != nil &&
					f.ReturnType.(TypeSafeNode).Equals(of.ReturnType.(TypeSafeNode)))) &&
			((f.Body == nil && of.Body == nil) ||
				(f.Body != nil && of.Body != nil && f.Body.Equals(of.Body)))
	}
	return false
}
func (f *FunctionDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(f.Parameters)+3)
	if f.Name != nil {
		children = append(children, f.Name)
	}
	for _, param := range f.Parameters {
		children = append(children, param)
	}
	if f.ReturnType != nil {
		children = append(children, f.ReturnType.(TypeSafeNode))
	}
	if f.Body != nil {
		children = append(children, f.Body)
	}
	return children
}
func (f *FunctionDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for FunctionDeclaration")
}

// Parameter TypeSafeNode implementation
func (p *Parameter) GetNodeKind() NodeKind { return NodeKindParameter }
func (p *Parameter) Clone() TypeSafeNode {
	clone := *p
	if p.Name != nil {
		clone.Name = p.Name.Clone().(*Identifier)
	}
	if p.TypeSpec != nil {
		clone.TypeSpec = p.TypeSpec.(TypeSafeNode).Clone().(Type)
	}
	return &clone
}
func (p *Parameter) Equals(other TypeSafeNode) bool {
	if op, ok := other.(*Parameter); ok {
		return p.IsMut == op.IsMut &&
			((p.Name == nil && op.Name == nil) ||
				(p.Name != nil && op.Name != nil && p.Name.Equals(op.Name))) &&
			((p.TypeSpec == nil && op.TypeSpec == nil) ||
				(p.TypeSpec != nil && op.TypeSpec != nil &&
					p.TypeSpec.(TypeSafeNode).Equals(op.TypeSpec.(TypeSafeNode))))
	}
	return false
}
func (p *Parameter) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 2)
	if p.Name != nil {
		children = append(children, p.Name)
	}
	if p.TypeSpec != nil {
		children = append(children, p.TypeSpec.(TypeSafeNode))
	}
	return children
}
func (p *Parameter) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 2 {
		return fmt.Errorf("index %d out of range for Parameter children", index)
	}
	if index == 0 {
		if newIdent, ok := newChild.(*Identifier); ok {
			p.Name = newIdent
			return nil
		}
		return fmt.Errorf("expected Identifier for Name, got %T", newChild)
	}
	if index == 1 {
		if newType, ok := newChild.(Type); ok {
			p.TypeSpec = newType
			return nil
		}
		return fmt.Errorf("expected Type for TypeSpec, got %T", newChild)
	}
	return fmt.Errorf("invalid child index %d for Parameter", index)
}

// VariableDeclaration TypeSafeNode implementation
func (v *VariableDeclaration) GetNodeKind() NodeKind { return NodeKindVariableDeclaration }
func (v *VariableDeclaration) Clone() TypeSafeNode {
	clone := *v
	if v.Name != nil {
		clone.Name = v.Name.Clone().(*Identifier)
	}
	if v.TypeSpec != nil {
		clone.TypeSpec = v.TypeSpec.(TypeSafeNode).Clone().(Type)
	}
	if v.Initializer != nil {
		clone.Initializer = v.Initializer.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (v *VariableDeclaration) Equals(other TypeSafeNode) bool {
	if ov, ok := other.(*VariableDeclaration); ok {
		return v.IsMutable == ov.IsMutable &&
			v.IsPublic == ov.IsPublic &&
			((v.Name == nil && ov.Name == nil) ||
				(v.Name != nil && ov.Name != nil && v.Name.Equals(ov.Name))) &&
			((v.TypeSpec == nil && ov.TypeSpec == nil) ||
				(v.TypeSpec != nil && ov.TypeSpec != nil &&
					v.TypeSpec.(TypeSafeNode).Equals(ov.TypeSpec.(TypeSafeNode)))) &&
			((v.Initializer == nil && ov.Initializer == nil) ||
				(v.Initializer != nil && ov.Initializer != nil &&
					v.Initializer.(TypeSafeNode).Equals(ov.Initializer.(TypeSafeNode))))
	}
	return false
}
func (v *VariableDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 3)
	if v.Name != nil {
		children = append(children, v.Name)
	}
	if v.TypeSpec != nil {
		children = append(children, v.TypeSpec.(TypeSafeNode))
	}
	if v.Initializer != nil {
		children = append(children, v.Initializer.(TypeSafeNode))
	}
	return children
}
func (v *VariableDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	children := v.GetChildren()
	if index < 0 || index >= len(children) {
		return fmt.Errorf("index %d out of range for VariableDeclaration children", index)
	}
	if index == 0 && v.Name != nil {
		if newIdent, ok := newChild.(*Identifier); ok {
			v.Name = newIdent
			return nil
		}
		return fmt.Errorf("expected Identifier for Name, got %T", newChild)
	}
	// Complex logic needed for TypeSpec vs Initializer based on actual structure
	return fmt.Errorf("ReplaceChild logic complex for VariableDeclaration")
}

// BlockStatement TypeSafeNode implementation
func (b *BlockStatement) GetNodeKind() NodeKind { return NodeKindBlockStatement }
func (b *BlockStatement) Clone() TypeSafeNode {
	clone := *b
	clone.Statements = make([]Statement, len(b.Statements))
	for i, stmt := range b.Statements {
		clone.Statements[i] = stmt.(TypeSafeNode).Clone().(Statement)
	}
	return &clone
}
func (b *BlockStatement) Equals(other TypeSafeNode) bool {
	if ob, ok := other.(*BlockStatement); ok {
		if len(b.Statements) != len(ob.Statements) {
			return false
		}
		for i, stmt := range b.Statements {
			if !stmt.(TypeSafeNode).Equals(ob.Statements[i].(TypeSafeNode)) {
				return false
			}
		}
		return true
	}
	return false
}
func (b *BlockStatement) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, len(b.Statements))
	for i, stmt := range b.Statements {
		children[i] = stmt.(TypeSafeNode)
	}
	return children
}
func (b *BlockStatement) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= len(b.Statements) {
		return fmt.Errorf("index %d out of range for BlockStatement children", index)
	}
	if newStmt, ok := newChild.(Statement); ok {
		b.Statements[index] = newStmt
		return nil
	}
	return fmt.Errorf("expected Statement, got %T", newChild)
}

// Identifier TypeSafeNode implementation
func (i *Identifier) GetNodeKind() NodeKind { return NodeKindIdentifier }
func (i *Identifier) Clone() TypeSafeNode {
	clone := *i
	return &clone
}
func (i *Identifier) Equals(other TypeSafeNode) bool {
	if oi, ok := other.(*Identifier); ok {
		return i.Value == oi.Value
	}
	return false
}
func (i *Identifier) GetChildren() []TypeSafeNode {
	return []TypeSafeNode{} // Leaf node
}
func (i *Identifier) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("Identifier is a leaf node, no children to replace")
}

// Literal TypeSafeNode implementation
func (l *Literal) GetNodeKind() NodeKind { return NodeKindLiteral }
func (l *Literal) Clone() TypeSafeNode {
	clone := *l
	// Deep copy the value if it's a complex type
	// For now, assume primitive values are safe to copy directly
	return &clone
}
func (l *Literal) Equals(other TypeSafeNode) bool {
	if ol, ok := other.(*Literal); ok {
		return l.Kind == ol.Kind && l.Value == ol.Value
	}
	return false
}
func (l *Literal) GetChildren() []TypeSafeNode {
	return []TypeSafeNode{} // Leaf node
}
func (l *Literal) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("Literal is a leaf node, no children to replace")
}

// BinaryExpression TypeSafeNode implementation
func (b *BinaryExpression) GetNodeKind() NodeKind { return NodeKindBinaryExpression }
func (b *BinaryExpression) Clone() TypeSafeNode {
	clone := *b
	if b.Left != nil {
		clone.Left = b.Left.(TypeSafeNode).Clone().(Expression)
	}
	if b.Right != nil {
		clone.Right = b.Right.(TypeSafeNode).Clone().(Expression)
	}
	if b.Operator != nil {
		operatorClone := *b.Operator
		clone.Operator = &operatorClone
	}
	return &clone
}
func (b *BinaryExpression) Equals(other TypeSafeNode) bool {
	if ob, ok := other.(*BinaryExpression); ok {
		return ((b.Left == nil && ob.Left == nil) ||
			(b.Left != nil && ob.Left != nil &&
				b.Left.(TypeSafeNode).Equals(ob.Left.(TypeSafeNode)))) &&
			((b.Right == nil && ob.Right == nil) ||
				(b.Right != nil && ob.Right != nil &&
					b.Right.(TypeSafeNode).Equals(ob.Right.(TypeSafeNode)))) &&
			((b.Operator == nil && ob.Operator == nil) ||
				(b.Operator != nil && ob.Operator != nil &&
					b.Operator.Value == ob.Operator.Value))
	}
	return false
}
func (b *BinaryExpression) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 3)
	if b.Left != nil {
		children = append(children, b.Left.(TypeSafeNode))
	}
	if b.Right != nil {
		children = append(children, b.Right.(TypeSafeNode))
	}
	if b.Operator != nil {
		children = append(children, b.Operator)
	}
	return children
}
func (b *BinaryExpression) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 2 {
		return fmt.Errorf("index %d out of range for BinaryExpression children", index)
	}
	newExpr, ok := newChild.(Expression)
	if !ok {
		return fmt.Errorf("expected Expression, got %T", newChild)
	}
	if index == 0 {
		b.Left = newExpr
	} else {
		b.Right = newExpr
	}
	return nil
}

// UnaryExpression TypeSafeNode implementation
func (u *UnaryExpression) GetNodeKind() NodeKind { return NodeKindUnaryExpression }
func (u *UnaryExpression) Clone() TypeSafeNode {
	clone := *u
	if u.Operand != nil {
		clone.Operand = u.Operand.(TypeSafeNode).Clone().(Expression)
	}
	if u.Operator != nil {
		operatorClone := *u.Operator
		clone.Operator = &operatorClone
	}
	return &clone
}
func (u *UnaryExpression) Equals(other TypeSafeNode) bool {
	if ou, ok := other.(*UnaryExpression); ok {
		return ((u.Operand == nil && ou.Operand == nil) ||
			(u.Operand != nil && ou.Operand != nil &&
				u.Operand.(TypeSafeNode).Equals(ou.Operand.(TypeSafeNode)))) &&
			((u.Operator == nil && ou.Operator == nil) ||
				(u.Operator != nil && ou.Operator != nil &&
					u.Operator.Value == ou.Operator.Value))
	}
	return false
}
func (u *UnaryExpression) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 2)
	if u.Operand != nil {
		children = append(children, u.Operand.(TypeSafeNode))
	}
	if u.Operator != nil {
		children = append(children, u.Operator)
	}
	return children
}
func (u *UnaryExpression) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 {
		return fmt.Errorf("index %d out of range for UnaryExpression children", index)
	}
	if newExpr, ok := newChild.(Expression); ok {
		u.Operand = newExpr
		return nil
	}
	return fmt.Errorf("expected Expression, got %T", newChild)
}

// BasicType TypeSafeNode implementation
func (bt *BasicType) GetNodeKind() NodeKind { return NodeKindBasicType }
func (bt *BasicType) Clone() TypeSafeNode {
	clone := *bt
	return &clone
}
func (bt *BasicType) Equals(other TypeSafeNode) bool {
	if obt, ok := other.(*BasicType); ok {
		return bt.Name == obt.Name
	}
	return false
}
func (bt *BasicType) GetChildren() []TypeSafeNode {
	return []TypeSafeNode{} // Leaf node
}
func (bt *BasicType) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("BasicType is a leaf node, no children to replace")
}

// Add TypeSafeNode implementation for Operator if it exists
func (o *Operator) GetNodeKind() NodeKind { return NodeKind(-1) } // Special case
func (o *Operator) Clone() TypeSafeNode {
	clone := *o
	return &clone
}
func (o *Operator) Equals(other TypeSafeNode) bool {
	if oo, ok := other.(*Operator); ok {
		return o.Value == oo.Value && o.Kind == oo.Kind
	}
	return false
}
func (o *Operator) GetChildren() []TypeSafeNode {
	return []TypeSafeNode{} // Leaf node
}
func (o *Operator) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("Operator is a leaf node, no children to replace")
}

// ====== TypeSafeNode implementations for new declaration nodes ======

// StructDeclaration
func (d *StructDeclaration) GetNodeKind() NodeKind { return NodeKindStructDeclaration }
func (d *StructDeclaration) Clone() TypeSafeNode {
	clone := *d
	if d.Name != nil {
		clone.Name = d.Name.Clone().(*Identifier)
	}
	clone.Fields = make([]*StructField, len(d.Fields))
	for i, f := range d.Fields {
		fc := *f
		if f.Name != nil {
			fc.Name = f.Name.Clone().(*Identifier)
		}
		if f.Type != nil {
			fc.Type = f.Type.(TypeSafeNode).Clone().(Type)
		}
		if f.Tags != nil {
			fc.Tags = make(map[string]string)
			for k, v := range f.Tags {
				fc.Tags[k] = v
			}
		}
		clone.Fields[i] = &fc
	}
	return &clone
}
func (d *StructDeclaration) Equals(other TypeSafeNode) bool {
	od, ok := other.(*StructDeclaration)
	if !ok {
		return false
	}
	if (d.Name == nil) != (od.Name == nil) {
		return false
	}
	if d.Name != nil && !d.Name.Equals(od.Name) {
		return false
	}
	if len(d.Fields) != len(od.Fields) || d.IsPublic != od.IsPublic {
		return false
	}
	for i := range d.Fields {
		if !d.Fields[i].Name.Equals(od.Fields[i].Name) {
			return false
		}
		if !d.Fields[i].Type.(TypeSafeNode).Equals(od.Fields[i].Type.(TypeSafeNode)) {
			return false
		}
		if d.Fields[i].IsPublic != od.Fields[i].IsPublic {
			return false
		}
	}
	return true
}
func (d *StructDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 1+len(d.Fields)*2)
	if d.Name != nil {
		children = append(children, d.Name)
	}
	for _, f := range d.Fields {
		if f.Name != nil {
			children = append(children, f.Name)
		}
		if f.Type != nil {
			children = append(children, f.Type.(TypeSafeNode))
		}
	}
	return children
}
func (d *StructDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for StructDeclaration")
}

// EnumDeclaration
func (d *EnumDeclaration) GetNodeKind() NodeKind { return NodeKindEnumDeclaration }
func (d *EnumDeclaration) Clone() TypeSafeNode {
	clone := *d
	if d.Name != nil {
		clone.Name = d.Name.Clone().(*Identifier)
	}
	clone.Variants = make([]*EnumVariant, len(d.Variants))
	for i, v := range d.Variants {
		vc := *v
		if v.Name != nil {
			vc.Name = v.Name.Clone().(*Identifier)
		}
		if v.Value != nil {
			vc.Value = v.Value.(TypeSafeNode).Clone().(Expression)
		}
		vc.Fields = make([]*StructField, len(v.Fields))
		for j, f := range v.Fields {
			fc := *f
			if f.Name != nil {
				fc.Name = f.Name.Clone().(*Identifier)
			}
			if f.Type != nil {
				fc.Type = f.Type.(TypeSafeNode).Clone().(Type)
			}
			vc.Fields[j] = &fc
		}
		clone.Variants[i] = &vc
	}
	return &clone
}
func (d *EnumDeclaration) Equals(other TypeSafeNode) bool {
	od, ok := other.(*EnumDeclaration)
	if !ok {
		return false
	}
	if (d.Name == nil) != (od.Name == nil) {
		return false
	}
	if d.Name != nil && !d.Name.Equals(od.Name) {
		return false
	}
	if len(d.Variants) != len(od.Variants) || d.IsPublic != od.IsPublic {
		return false
	}
	// Shallow compare variant names only for now
	for i := range d.Variants {
		if !d.Variants[i].Name.Equals(od.Variants[i].Name) {
			return false
		}
	}
	return true
}
func (d *EnumDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0)
	if d.Name != nil {
		children = append(children, d.Name)
	}
	for _, v := range d.Variants {
		if v.Name != nil {
			children = append(children, v.Name)
		}
		if v.Value != nil {
			children = append(children, v.Value.(TypeSafeNode))
		}
		for _, f := range v.Fields {
			if f.Name != nil {
				children = append(children, f.Name)
			}
			if f.Type != nil {
				children = append(children, f.Type.(TypeSafeNode))
			}
		}
	}
	return children
}
func (d *EnumDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for EnumDeclaration")
}

// TraitDeclaration
func (d *TraitDeclaration) GetNodeKind() NodeKind { return NodeKindTraitDeclaration }
func (d *TraitDeclaration) Clone() TypeSafeNode {
	clone := *d
	if d.Name != nil {
		clone.Name = d.Name.Clone().(*Identifier)
	}
	clone.Methods = make([]*TraitMethod, len(d.Methods))
	for i, m := range d.Methods {
		mc := *m
		if m.Name != nil {
			mc.Name = m.Name.Clone().(*Identifier)
		}
		mc.Parameters = make([]*Parameter, len(m.Parameters))
		for j, p := range m.Parameters {
			mc.Parameters[j] = p.Clone().(*Parameter)
		}
		if m.ReturnType != nil {
			mc.ReturnType = m.ReturnType.(TypeSafeNode).Clone().(Type)
		}
		clone.Methods[i] = &mc
	}
	return &clone
}
func (d *TraitDeclaration) Equals(other TypeSafeNode) bool {
	od, ok := other.(*TraitDeclaration)
	if !ok {
		return false
	}
	if (d.Name == nil) != (od.Name == nil) {
		return false
	}
	if d.Name != nil && !d.Name.Equals(od.Name) {
		return false
	}
	if len(d.Methods) != len(od.Methods) || d.IsPublic != od.IsPublic {
		return false
	}
	return true
}
func (d *TraitDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0)
	if d.Name != nil {
		children = append(children, d.Name)
	}
	for _, m := range d.Methods {
		if m.Name != nil {
			children = append(children, m.Name)
		}
		if m.ReturnType != nil {
			children = append(children, m.ReturnType.(TypeSafeNode))
		}
		for _, p := range m.Parameters {
			children = append(children, p)
		}
	}
	return children
}
func (d *TraitDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for TraitDeclaration")
}

// ImplBlock
func (i *ImplBlock) GetNodeKind() NodeKind { return NodeKindImplBlock }
func (i *ImplBlock) Clone() TypeSafeNode {
	clone := *i
	if i.Trait != nil {
		clone.Trait = i.Trait.(TypeSafeNode).Clone().(Type)
	}
	if i.ForType != nil {
		clone.ForType = i.ForType.(TypeSafeNode).Clone().(Type)
	}
	clone.Items = make([]*FunctionDeclaration, len(i.Items))
	for idx, it := range i.Items {
		clone.Items[idx] = it.Clone().(*FunctionDeclaration)
	}
	return &clone
}
func (i *ImplBlock) Equals(other TypeSafeNode) bool {
	oi, ok := other.(*ImplBlock)
	if !ok {
		return false
	}
	if (i.Trait == nil) != (oi.Trait == nil) {
		return false
	}
	if i.Trait != nil && !i.Trait.(TypeSafeNode).Equals(oi.Trait.(TypeSafeNode)) {
		return false
	}
	if (i.ForType == nil) != (oi.ForType == nil) {
		return false
	}
	if i.ForType != nil && !i.ForType.(TypeSafeNode).Equals(oi.ForType.(TypeSafeNode)) {
		return false
	}
	if len(i.Items) != len(oi.Items) {
		return false
	}
	return true
}
func (i *ImplBlock) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0)
	if i.Trait != nil {
		children = append(children, i.Trait.(TypeSafeNode))
	}
	if i.ForType != nil {
		children = append(children, i.ForType.(TypeSafeNode))
	}
	for _, it := range i.Items {
		children = append(children, it)
	}
	return children
}
func (i *ImplBlock) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for ImplBlock")
}

// ImportDeclaration
func (d *ImportDeclaration) GetNodeKind() NodeKind { return NodeKindImportDeclaration }
func (d *ImportDeclaration) Clone() TypeSafeNode {
	clone := *d
	clone.Path = make([]*Identifier, len(d.Path))
	for i, seg := range d.Path {
		clone.Path[i] = seg.Clone().(*Identifier)
	}
	if d.Alias != nil {
		clone.Alias = d.Alias.Clone().(*Identifier)
	}
	return &clone
}
func (d *ImportDeclaration) Equals(other TypeSafeNode) bool {
	od, ok := other.(*ImportDeclaration)
	if !ok {
		return false
	}
	if len(d.Path) != len(od.Path) || d.IsPublic != od.IsPublic {
		return false
	}
	for i := range d.Path {
		if !d.Path[i].Equals(od.Path[i]) {
			return false
		}
	}
	if (d.Alias == nil) != (od.Alias == nil) {
		return false
	}
	if d.Alias != nil && !d.Alias.Equals(od.Alias) {
		return false
	}
	return true
}
func (d *ImportDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(d.Path)+1)
	for _, seg := range d.Path {
		children = append(children, seg)
	}
	if d.Alias != nil {
		children = append(children, d.Alias)
	}
	return children
}
func (d *ImportDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for ImportDeclaration")
}

// ExportDeclaration
func (d *ExportDeclaration) GetNodeKind() NodeKind { return NodeKindExportDeclaration }
func (d *ExportDeclaration) Clone() TypeSafeNode {
	clone := *d
	clone.Items = make([]*ExportItem, len(d.Items))
	for i, it := range d.Items {
		ic := *it
		if it.Name != nil {
			ic.Name = it.Name.Clone().(*Identifier)
		}
		if it.Alias != nil {
			ic.Alias = it.Alias.Clone().(*Identifier)
		}
		clone.Items[i] = &ic
	}
	return &clone
}
func (d *ExportDeclaration) Equals(other TypeSafeNode) bool {
	od, ok := other.(*ExportDeclaration)
	if !ok {
		return false
	}
	if len(d.Items) != len(od.Items) {
		return false
	}
	for i := range d.Items {
		if !d.Items[i].Name.Equals(od.Items[i].Name) {
			return false
		}
	}
	return true
}
func (d *ExportDeclaration) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0)
	for _, it := range d.Items {
		if it.Name != nil {
			children = append(children, it.Name)
		}
		if it.Alias != nil {
			children = append(children, it.Alias)
		}
	}
	return children
}
func (d *ExportDeclaration) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for ExportDeclaration")
}
