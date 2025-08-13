// HIR node implementations for the Orizon programming language
// This file contains concrete implementations of HIR nodes with semantic information

package hir

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/position"
)

// =============================================================================
// HIR Declarations
// =============================================================================

// HIRFunctionDeclaration represents a function declaration in HIR
type HIRFunctionDeclaration struct {
	ID         NodeID
	Name       string
	Parameters []*HIRParameter
	ReturnType HIRType
	Body       *HIRBlockStatement
	Generic    bool
	TypeParams []TypeInfo
	Effects    EffectSet
	Regions    RegionSet
	Metadata   IRMetadata
	Span       position.Span
}

func (fd *HIRFunctionDeclaration) GetID() NodeID          { return fd.ID }
func (fd *HIRFunctionDeclaration) GetSpan() position.Span { return fd.Span }
func (fd *HIRFunctionDeclaration) GetType() TypeInfo {
	return TypeInfo{
		Kind: TypeKindFunction,
		Name: fd.Name,
	}
}
func (fd *HIRFunctionDeclaration) GetEffects() EffectSet { return fd.Effects }
func (fd *HIRFunctionDeclaration) GetRegions() RegionSet { return fd.Regions }
func (fd *HIRFunctionDeclaration) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitFunctionDeclaration(fd)
}
func (fd *HIRFunctionDeclaration) GetChildren() []HIRNode {
	children := make([]HIRNode, 0, len(fd.Parameters)+2)
	for _, param := range fd.Parameters {
		children = append(children, param)
	}
	if fd.ReturnType != nil {
		children = append(children, fd.ReturnType)
	}
	if fd.Body != nil {
		children = append(children, fd.Body)
	}
	return children
}
func (fd *HIRFunctionDeclaration) hirDeclarationNode() {}
func (fd *HIRFunctionDeclaration) String() string {
	return fmt.Sprintf("HIRFunctionDeclaration{%s: %d params}", fd.Name, len(fd.Parameters))
}

// HIRParameter represents a function parameter in HIR
type HIRParameter struct {
	ID       NodeID
	Name     string
	Type     HIRType
	Default  HIRExpression // Optional default value
	Metadata IRMetadata
	Span     position.Span
}

func (p *HIRParameter) GetID() NodeID          { return p.ID }
func (p *HIRParameter) GetSpan() position.Span { return p.Span }
func (p *HIRParameter) GetType() TypeInfo {
	return p.Type.GetType()
}
func (p *HIRParameter) GetEffects() EffectSet { return NewEffectSet() }
func (p *HIRParameter) GetRegions() RegionSet { return NewRegionSet() }
func (p *HIRParameter) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitVariableDeclaration(&HIRVariableDeclaration{
		ID:   p.ID,
		Name: p.Name,
		Type: p.Type,
		Span: p.Span,
	})
}
func (p *HIRParameter) GetChildren() []HIRNode {
	children := []HIRNode{p.Type}
	if p.Default != nil {
		children = append(children, p.Default)
	}
	return children
}
func (p *HIRParameter) String() string {
	return fmt.Sprintf("HIRParameter{%s: %s}", p.Name, p.Type.String())
}

// HIRVariableDeclaration represents a variable declaration in HIR
type HIRVariableDeclaration struct {
	ID          NodeID
	Name        string
	Type        HIRType
	Initializer HIRExpression
	Mutable     bool
	Effects     EffectSet
	Regions     RegionSet
	Metadata    IRMetadata
	Span        position.Span
}

func (vd *HIRVariableDeclaration) GetID() NodeID          { return vd.ID }
func (vd *HIRVariableDeclaration) GetSpan() position.Span { return vd.Span }
func (vd *HIRVariableDeclaration) GetType() TypeInfo {
	return vd.Type.GetType()
}
func (vd *HIRVariableDeclaration) GetEffects() EffectSet { return vd.Effects }
func (vd *HIRVariableDeclaration) GetRegions() RegionSet { return vd.Regions }
func (vd *HIRVariableDeclaration) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitVariableDeclaration(vd)
}
func (vd *HIRVariableDeclaration) GetChildren() []HIRNode {
	children := []HIRNode{vd.Type}
	if vd.Initializer != nil {
		children = append(children, vd.Initializer)
	}
	return children
}
func (vd *HIRVariableDeclaration) hirDeclarationNode() {}
func (vd *HIRVariableDeclaration) String() string {
	return fmt.Sprintf("HIRVariableDeclaration{%s: %s}", vd.Name, vd.Type.String())
}

// HIRTypeDeclaration represents a type declaration in HIR
type HIRTypeDeclaration struct {
	ID       NodeID
	Name     string
	Type     HIRType
	Generic  bool
	Params   []TypeInfo
	Metadata IRMetadata
	Span     position.Span
}

func (td *HIRTypeDeclaration) GetID() NodeID          { return td.ID }
func (td *HIRTypeDeclaration) GetSpan() position.Span { return td.Span }
func (td *HIRTypeDeclaration) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindGeneric, Name: td.Name}
}
func (td *HIRTypeDeclaration) GetEffects() EffectSet { return NewEffectSet() }
func (td *HIRTypeDeclaration) GetRegions() RegionSet { return NewRegionSet() }
func (td *HIRTypeDeclaration) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitTypeDeclaration(td)
}
func (td *HIRTypeDeclaration) GetChildren() []HIRNode {
	return []HIRNode{td.Type}
}
func (td *HIRTypeDeclaration) hirDeclarationNode() {}
func (td *HIRTypeDeclaration) String() string {
	return fmt.Sprintf("HIRTypeDeclaration{%s}", td.Name)
}

// HIRConstDeclaration represents a constant declaration in HIR
type HIRConstDeclaration struct {
	ID       NodeID
	Name     string
	Type     HIRType
	Value    HIRExpression
	Metadata IRMetadata
	Span     position.Span
}

func (cd *HIRConstDeclaration) GetID() NodeID          { return cd.ID }
func (cd *HIRConstDeclaration) GetSpan() position.Span { return cd.Span }
func (cd *HIRConstDeclaration) GetType() TypeInfo {
	return cd.Type.GetType()
}
func (cd *HIRConstDeclaration) GetEffects() EffectSet { return NewEffectSet() }
func (cd *HIRConstDeclaration) GetRegions() RegionSet { return NewRegionSet() }
func (cd *HIRConstDeclaration) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitConstDeclaration(cd)
}
func (cd *HIRConstDeclaration) GetChildren() []HIRNode {
	return []HIRNode{cd.Type, cd.Value}
}
func (cd *HIRConstDeclaration) hirDeclarationNode() {}
func (cd *HIRConstDeclaration) String() string {
	return fmt.Sprintf("HIRConstDeclaration{%s: %s}", cd.Name, cd.Type.String())
}

// =============================================================================
// HIR Statements
// =============================================================================

// HIRBlockStatement represents a block statement in HIR
type HIRBlockStatement struct {
	ID         NodeID
	Statements []HIRStatement
	Effects    EffectSet
	Regions    RegionSet
	Metadata   IRMetadata
	Span       position.Span
}

func (bs *HIRBlockStatement) GetID() NodeID          { return bs.ID }
func (bs *HIRBlockStatement) GetSpan() position.Span { return bs.Span }
func (bs *HIRBlockStatement) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (bs *HIRBlockStatement) GetEffects() EffectSet { return bs.Effects }
func (bs *HIRBlockStatement) GetRegions() RegionSet { return bs.Regions }
func (bs *HIRBlockStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitBlockStatement(bs)
}
func (bs *HIRBlockStatement) GetChildren() []HIRNode {
	children := make([]HIRNode, len(bs.Statements))
	for i, stmt := range bs.Statements {
		children[i] = stmt
	}
	return children
}
func (bs *HIRBlockStatement) hirStatementNode() {}
func (bs *HIRBlockStatement) String() string {
	return fmt.Sprintf("HIRBlockStatement{%d statements}", len(bs.Statements))
}

// HIRExpressionStatement represents an expression statement in HIR
type HIRExpressionStatement struct {
	ID         NodeID
	Expression HIRExpression
	Effects    EffectSet
	Regions    RegionSet
	Metadata   IRMetadata
	Span       position.Span
}

func (es *HIRExpressionStatement) GetID() NodeID          { return es.ID }
func (es *HIRExpressionStatement) GetSpan() position.Span { return es.Span }
func (es *HIRExpressionStatement) GetType() TypeInfo {
	return es.Expression.GetType()
}
func (es *HIRExpressionStatement) GetEffects() EffectSet { return es.Effects }
func (es *HIRExpressionStatement) GetRegions() RegionSet { return es.Regions }
func (es *HIRExpressionStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitExpressionStatement(es)
}
func (es *HIRExpressionStatement) GetChildren() []HIRNode {
	return []HIRNode{es.Expression}
}
func (es *HIRExpressionStatement) hirStatementNode() {}
func (es *HIRExpressionStatement) String() string {
	return fmt.Sprintf("HIRExpressionStatement{%s}", es.Expression.String())
}

// HIRReturnStatement represents a return statement in HIR
type HIRReturnStatement struct {
	ID         NodeID
	Expression HIRExpression // Optional return value
	Effects    EffectSet
	Regions    RegionSet
	Metadata   IRMetadata
	Span       position.Span
}

func (rs *HIRReturnStatement) GetID() NodeID          { return rs.ID }
func (rs *HIRReturnStatement) GetSpan() position.Span { return rs.Span }
func (rs *HIRReturnStatement) GetType() TypeInfo {
	if rs.Expression != nil {
		return rs.Expression.GetType()
	}
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (rs *HIRReturnStatement) GetEffects() EffectSet { return rs.Effects }
func (rs *HIRReturnStatement) GetRegions() RegionSet { return rs.Regions }
func (rs *HIRReturnStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitReturnStatement(rs)
}
func (rs *HIRReturnStatement) GetChildren() []HIRNode {
	if rs.Expression != nil {
		return []HIRNode{rs.Expression}
	}
	return []HIRNode{}
}
func (rs *HIRReturnStatement) hirStatementNode() {}
func (rs *HIRReturnStatement) String() string {
	if rs.Expression != nil {
		return fmt.Sprintf("HIRReturnStatement{%s}", rs.Expression.String())
	}
	return "HIRReturnStatement{void}"
}

// HIRIfStatement represents an if statement in HIR
type HIRIfStatement struct {
	ID        NodeID
	Condition HIRExpression
	ThenBlock HIRStatement
	ElseBlock HIRStatement // Optional
	Effects   EffectSet
	Regions   RegionSet
	Metadata  IRMetadata
	Span      position.Span
}

func (is *HIRIfStatement) GetID() NodeID          { return is.ID }
func (is *HIRIfStatement) GetSpan() position.Span { return is.Span }
func (is *HIRIfStatement) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (is *HIRIfStatement) GetEffects() EffectSet { return is.Effects }
func (is *HIRIfStatement) GetRegions() RegionSet { return is.Regions }
func (is *HIRIfStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitIfStatement(is)
}
func (is *HIRIfStatement) GetChildren() []HIRNode {
	children := []HIRNode{is.Condition, is.ThenBlock}
	if is.ElseBlock != nil {
		children = append(children, is.ElseBlock)
	}
	return children
}
func (is *HIRIfStatement) hirStatementNode() {}
func (is *HIRIfStatement) String() string {
	return fmt.Sprintf("HIRIfStatement{%s}", is.Condition.String())
}

// HIRWhileStatement represents a while statement in HIR
type HIRWhileStatement struct {
	ID        NodeID
	Condition HIRExpression
	Body      HIRStatement
	Effects   EffectSet
	Regions   RegionSet
	Metadata  IRMetadata
	Span      position.Span
}

func (ws *HIRWhileStatement) GetID() NodeID          { return ws.ID }
func (ws *HIRWhileStatement) GetSpan() position.Span { return ws.Span }
func (ws *HIRWhileStatement) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (ws *HIRWhileStatement) GetEffects() EffectSet { return ws.Effects }
func (ws *HIRWhileStatement) GetRegions() RegionSet { return ws.Regions }
func (ws *HIRWhileStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitWhileStatement(ws)
}
func (ws *HIRWhileStatement) GetChildren() []HIRNode {
	return []HIRNode{ws.Condition, ws.Body}
}
func (ws *HIRWhileStatement) hirStatementNode() {}
func (ws *HIRWhileStatement) String() string {
	return fmt.Sprintf("HIRWhileStatement{%s}", ws.Condition.String())
}

// HIRForStatement represents a for statement in HIR
type HIRForStatement struct {
	ID        NodeID
	Init      HIRStatement  // Optional initialization
	Condition HIRExpression // Optional condition
	Update    HIRStatement  // Optional update
	Body      HIRStatement
	Effects   EffectSet
	Regions   RegionSet
	Metadata  IRMetadata
	Span      position.Span
}

func (fs *HIRForStatement) GetID() NodeID          { return fs.ID }
func (fs *HIRForStatement) GetSpan() position.Span { return fs.Span }
func (fs *HIRForStatement) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (fs *HIRForStatement) GetEffects() EffectSet { return fs.Effects }
func (fs *HIRForStatement) GetRegions() RegionSet { return fs.Regions }
func (fs *HIRForStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitForStatement(fs)
}
func (fs *HIRForStatement) GetChildren() []HIRNode {
	children := []HIRNode{}
	if fs.Init != nil {
		children = append(children, fs.Init)
	}
	if fs.Condition != nil {
		children = append(children, fs.Condition)
	}
	if fs.Update != nil {
		children = append(children, fs.Update)
	}
	children = append(children, fs.Body)
	return children
}
func (fs *HIRForStatement) hirStatementNode() {}
func (fs *HIRForStatement) String() string {
	return "HIRForStatement{}"
}

// HIRBreakStatement represents a break statement in HIR
type HIRBreakStatement struct {
	ID       NodeID
	Label    string // Optional label
	Metadata IRMetadata
	Span     position.Span
}

func (bs *HIRBreakStatement) GetID() NodeID          { return bs.ID }
func (bs *HIRBreakStatement) GetSpan() position.Span { return bs.Span }
func (bs *HIRBreakStatement) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (bs *HIRBreakStatement) GetEffects() EffectSet { return NewEffectSet() }
func (bs *HIRBreakStatement) GetRegions() RegionSet { return NewRegionSet() }
func (bs *HIRBreakStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitBreakStatement(bs)
}
func (bs *HIRBreakStatement) GetChildren() []HIRNode {
	return []HIRNode{}
}
func (bs *HIRBreakStatement) hirStatementNode() {}
func (bs *HIRBreakStatement) String() string {
	if bs.Label != "" {
		return fmt.Sprintf("HIRBreakStatement{%s}", bs.Label)
	}
	return "HIRBreakStatement{}"
}

// HIRContinueStatement represents a continue statement in HIR
type HIRContinueStatement struct {
	ID       NodeID
	Label    string // Optional label
	Metadata IRMetadata
	Span     position.Span
}

func (cs *HIRContinueStatement) GetID() NodeID          { return cs.ID }
func (cs *HIRContinueStatement) GetSpan() position.Span { return cs.Span }
func (cs *HIRContinueStatement) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (cs *HIRContinueStatement) GetEffects() EffectSet { return NewEffectSet() }
func (cs *HIRContinueStatement) GetRegions() RegionSet { return NewRegionSet() }
func (cs *HIRContinueStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitContinueStatement(cs)
}
func (cs *HIRContinueStatement) GetChildren() []HIRNode {
	return []HIRNode{}
}
func (cs *HIRContinueStatement) hirStatementNode() {}
func (cs *HIRContinueStatement) String() string {
	if cs.Label != "" {
		return fmt.Sprintf("HIRContinueStatement{%s}", cs.Label)
	}
	return "HIRContinueStatement{}"
}

// HIRAssignStatement represents an assignment statement in HIR
type HIRAssignStatement struct {
	ID       NodeID
	Target   HIRExpression
	Value    HIRExpression
	Operator string // Assignment operator (=, +=, -=, etc.)
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

func (as *HIRAssignStatement) GetID() NodeID          { return as.ID }
func (as *HIRAssignStatement) GetSpan() position.Span { return as.Span }
func (as *HIRAssignStatement) GetType() TypeInfo {
	return as.Value.GetType()
}
func (as *HIRAssignStatement) GetEffects() EffectSet { return as.Effects }
func (as *HIRAssignStatement) GetRegions() RegionSet { return as.Regions }
func (as *HIRAssignStatement) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitAssignStatement(as)
}
func (as *HIRAssignStatement) GetChildren() []HIRNode {
	return []HIRNode{as.Target, as.Value}
}
func (as *HIRAssignStatement) hirStatementNode() {}
func (as *HIRAssignStatement) String() string {
	return fmt.Sprintf("HIRAssignStatement{%s %s %s}", as.Target.String(), as.Operator, as.Value.String())
}

// =============================================================================
// HIR Expressions
// =============================================================================

// HIRIdentifier represents an identifier expression in HIR
type HIRIdentifier struct {
	ID           NodeID
	Name         string
	ResolvedDecl HIRDeclaration // Resolved declaration reference
	Type         TypeInfo
	Effects      EffectSet
	Regions      RegionSet
	Metadata     IRMetadata
	Span         position.Span
}

func (id *HIRIdentifier) GetID() NodeID          { return id.ID }
func (id *HIRIdentifier) GetSpan() position.Span { return id.Span }
func (id *HIRIdentifier) GetType() TypeInfo      { return id.Type }
func (id *HIRIdentifier) GetEffects() EffectSet  { return id.Effects }
func (id *HIRIdentifier) GetRegions() RegionSet  { return id.Regions }
func (id *HIRIdentifier) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitIdentifier(id)
}
func (id *HIRIdentifier) GetChildren() []HIRNode {
	return []HIRNode{}
}
func (id *HIRIdentifier) hirExpressionNode() {}
func (id *HIRIdentifier) String() string {
	return fmt.Sprintf("HIRIdentifier{%s: %s}", id.Name, id.Type.String())
}

// HIRLiteral represents a literal expression in HIR
type HIRLiteral struct {
	ID       NodeID
	Value    interface{}
	Type     TypeInfo
	Metadata IRMetadata
	Span     position.Span
}

func (l *HIRLiteral) GetID() NodeID          { return l.ID }
func (l *HIRLiteral) GetSpan() position.Span { return l.Span }
func (l *HIRLiteral) GetType() TypeInfo      { return l.Type }
func (l *HIRLiteral) GetEffects() EffectSet  { return NewEffectSet() }
func (l *HIRLiteral) GetRegions() RegionSet  { return NewRegionSet() }
func (l *HIRLiteral) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitLiteral(l)
}
func (l *HIRLiteral) GetChildren() []HIRNode {
	return []HIRNode{}
}
func (l *HIRLiteral) hirExpressionNode() {}
func (l *HIRLiteral) String() string {
	return fmt.Sprintf("HIRLiteral{%v: %s}", l.Value, l.Type.String())
}

// HIRBinaryExpression represents a binary expression in HIR
type HIRBinaryExpression struct {
	ID       NodeID
	Left     HIRExpression
	Operator string
	Right    HIRExpression
	Type     TypeInfo
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

func (be *HIRBinaryExpression) GetID() NodeID          { return be.ID }
func (be *HIRBinaryExpression) GetSpan() position.Span { return be.Span }
func (be *HIRBinaryExpression) GetType() TypeInfo      { return be.Type }
func (be *HIRBinaryExpression) GetEffects() EffectSet  { return be.Effects }
func (be *HIRBinaryExpression) GetRegions() RegionSet  { return be.Regions }
func (be *HIRBinaryExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitBinaryExpression(be)
}
func (be *HIRBinaryExpression) GetChildren() []HIRNode {
	return []HIRNode{be.Left, be.Right}
}
func (be *HIRBinaryExpression) hirExpressionNode() {}
func (be *HIRBinaryExpression) String() string {
	return fmt.Sprintf("HIRBinaryExpression{%s %s %s}", be.Left.String(), be.Operator, be.Right.String())
}

// HIRUnaryExpression represents a unary expression in HIR
type HIRUnaryExpression struct {
	ID       NodeID
	Operator string
	Operand  HIRExpression
	Type     TypeInfo
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

func (ue *HIRUnaryExpression) GetID() NodeID          { return ue.ID }
func (ue *HIRUnaryExpression) GetSpan() position.Span { return ue.Span }
func (ue *HIRUnaryExpression) GetType() TypeInfo      { return ue.Type }
func (ue *HIRUnaryExpression) GetEffects() EffectSet  { return ue.Effects }
func (ue *HIRUnaryExpression) GetRegions() RegionSet  { return ue.Regions }
func (ue *HIRUnaryExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitUnaryExpression(ue)
}
func (ue *HIRUnaryExpression) GetChildren() []HIRNode {
	return []HIRNode{ue.Operand}
}
func (ue *HIRUnaryExpression) hirExpressionNode() {}
func (ue *HIRUnaryExpression) String() string {
	return fmt.Sprintf("HIRUnaryExpression{%s %s}", ue.Operator, ue.Operand.String())
}

// HIRCallExpression represents a function call expression in HIR
type HIRCallExpression struct {
	ID        NodeID
	Function  HIRExpression
	Arguments []HIRExpression
	Type      TypeInfo
	Effects   EffectSet
	Regions   RegionSet
	Metadata  IRMetadata
	Span      position.Span
}

func (ce *HIRCallExpression) GetID() NodeID          { return ce.ID }
func (ce *HIRCallExpression) GetSpan() position.Span { return ce.Span }
func (ce *HIRCallExpression) GetType() TypeInfo      { return ce.Type }
func (ce *HIRCallExpression) GetEffects() EffectSet  { return ce.Effects }
func (ce *HIRCallExpression) GetRegions() RegionSet  { return ce.Regions }
func (ce *HIRCallExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitCallExpression(ce)
}
func (ce *HIRCallExpression) GetChildren() []HIRNode {
	children := []HIRNode{ce.Function}
	for _, arg := range ce.Arguments {
		children = append(children, arg)
	}
	return children
}
func (ce *HIRCallExpression) hirExpressionNode() {}
func (ce *HIRCallExpression) String() string {
	return fmt.Sprintf("HIRCallExpression{%s: %d args}", ce.Function.String(), len(ce.Arguments))
}

// HIRIndexExpression represents an index expression in HIR
type HIRIndexExpression struct {
	ID       NodeID
	Array    HIRExpression
	Index    HIRExpression
	Type     TypeInfo
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

func (ie *HIRIndexExpression) GetID() NodeID          { return ie.ID }
func (ie *HIRIndexExpression) GetSpan() position.Span { return ie.Span }
func (ie *HIRIndexExpression) GetType() TypeInfo      { return ie.Type }
func (ie *HIRIndexExpression) GetEffects() EffectSet  { return ie.Effects }
func (ie *HIRIndexExpression) GetRegions() RegionSet  { return ie.Regions }
func (ie *HIRIndexExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitIndexExpression(ie)
}
func (ie *HIRIndexExpression) GetChildren() []HIRNode {
	return []HIRNode{ie.Array, ie.Index}
}
func (ie *HIRIndexExpression) hirExpressionNode() {}
func (ie *HIRIndexExpression) String() string {
	return fmt.Sprintf("HIRIndexExpression{%s[%s]}", ie.Array.String(), ie.Index.String())
}

// HIRFieldExpression represents a field access expression in HIR
type HIRFieldExpression struct {
	ID       NodeID
	Object   HIRExpression
	Field    string
	Type     TypeInfo
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

func (fe *HIRFieldExpression) GetID() NodeID          { return fe.ID }
func (fe *HIRFieldExpression) GetSpan() position.Span { return fe.Span }
func (fe *HIRFieldExpression) GetType() TypeInfo      { return fe.Type }
func (fe *HIRFieldExpression) GetEffects() EffectSet  { return fe.Effects }
func (fe *HIRFieldExpression) GetRegions() RegionSet  { return fe.Regions }
func (fe *HIRFieldExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitFieldExpression(fe)
}
func (fe *HIRFieldExpression) GetChildren() []HIRNode {
	return []HIRNode{fe.Object}
}
func (fe *HIRFieldExpression) hirExpressionNode() {}
func (fe *HIRFieldExpression) String() string {
	return fmt.Sprintf("HIRFieldExpression{%s.%s}", fe.Object.String(), fe.Field)
}

// HIRCastExpression represents a type cast expression in HIR
type HIRCastExpression struct {
	ID         NodeID
	Expression HIRExpression
	TargetType HIRType
	Type       TypeInfo
	Effects    EffectSet
	Regions    RegionSet
	Metadata   IRMetadata
	Span       position.Span
}

func (ce *HIRCastExpression) GetID() NodeID          { return ce.ID }
func (ce *HIRCastExpression) GetSpan() position.Span { return ce.Span }
func (ce *HIRCastExpression) GetType() TypeInfo      { return ce.Type }
func (ce *HIRCastExpression) GetEffects() EffectSet  { return ce.Effects }
func (ce *HIRCastExpression) GetRegions() RegionSet  { return ce.Regions }
func (ce *HIRCastExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitCastExpression(ce)
}
func (ce *HIRCastExpression) GetChildren() []HIRNode {
	return []HIRNode{ce.Expression, ce.TargetType}
}
func (ce *HIRCastExpression) hirExpressionNode() {}
func (ce *HIRCastExpression) String() string {
	return fmt.Sprintf("HIRCastExpression{%s as %s}", ce.Expression.String(), ce.TargetType.String())
}

// HIRArrayExpression represents an array literal expression in HIR
type HIRArrayExpression struct {
	ID       NodeID
	Elements []HIRExpression
	Type     TypeInfo
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

func (ae *HIRArrayExpression) GetID() NodeID          { return ae.ID }
func (ae *HIRArrayExpression) GetSpan() position.Span { return ae.Span }
func (ae *HIRArrayExpression) GetType() TypeInfo      { return ae.Type }
func (ae *HIRArrayExpression) GetEffects() EffectSet  { return ae.Effects }
func (ae *HIRArrayExpression) GetRegions() RegionSet  { return ae.Regions }
func (ae *HIRArrayExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitArrayExpression(ae)
}
func (ae *HIRArrayExpression) GetChildren() []HIRNode {
	children := make([]HIRNode, len(ae.Elements))
	for i, elem := range ae.Elements {
		children[i] = elem
	}
	return children
}
func (ae *HIRArrayExpression) hirExpressionNode() {}
func (ae *HIRArrayExpression) String() string {
	return fmt.Sprintf("HIRArrayExpression{%d elements}", len(ae.Elements))
}

// HIRStructExpression represents a struct literal expression in HIR
type HIRStructExpression struct {
	ID       NodeID
	Type     TypeInfo
	Fields   []HIRFieldInit
	Effects  EffectSet
	Regions  RegionSet
	Metadata IRMetadata
	Span     position.Span
}

// HIRFieldInit represents a field initialization in a struct literal
type HIRFieldInit struct {
	Name  string
	Value HIRExpression
	Span  position.Span
}

func (se *HIRStructExpression) GetID() NodeID          { return se.ID }
func (se *HIRStructExpression) GetSpan() position.Span { return se.Span }
func (se *HIRStructExpression) GetType() TypeInfo      { return se.Type }
func (se *HIRStructExpression) GetEffects() EffectSet  { return se.Effects }
func (se *HIRStructExpression) GetRegions() RegionSet  { return se.Regions }
func (se *HIRStructExpression) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitStructExpression(se)
}
func (se *HIRStructExpression) GetChildren() []HIRNode {
	children := make([]HIRNode, len(se.Fields))
	for i, field := range se.Fields {
		children[i] = field.Value
	}
	return children
}
func (se *HIRStructExpression) hirExpressionNode() {}
func (se *HIRStructExpression) String() string {
	return fmt.Sprintf("HIRStructExpression{%s: %d fields}", se.Type.String(), len(se.Fields))
}
