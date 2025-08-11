// Additional AST expression and statement node definitions
// This file extends the existing AST with additional expression and statement types
// needed for a complete Orizon language implementation.

package parser

import (
	"fmt"
	"strings"
)

// ====== Additional Expression Nodes ======

// ArrayExpression represents array literal [expr1, expr2, ...]
type ArrayExpression struct {
	Span     Span
	Elements []Expression
}

func (ae *ArrayExpression) GetSpan() Span { return ae.Span }
func (ae *ArrayExpression) String() string {
	var elements []string
	for _, elem := range ae.Elements {
		elements = append(elements, elem.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}
func (ae *ArrayExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitArrayExpression(ae)
}
func (ae *ArrayExpression) expressionNode()       {}
func (ae *ArrayExpression) GetNodeKind() NodeKind { return NodeKindArrayExpression }
func (ae *ArrayExpression) Clone() TypeSafeNode {
	clone := *ae
	clone.Elements = make([]Expression, len(ae.Elements))
	for i, elem := range ae.Elements {
		clone.Elements[i] = elem.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (ae *ArrayExpression) Equals(other TypeSafeNode) bool {
	if oe, ok := other.(*ArrayExpression); ok {
		if len(ae.Elements) != len(oe.Elements) {
			return false
		}
		for i, elem := range ae.Elements {
			if !elem.(TypeSafeNode).Equals(oe.Elements[i].(TypeSafeNode)) {
				return false
			}
		}
		return true
	}
	return false
}
func (ae *ArrayExpression) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, len(ae.Elements))
	for i, elem := range ae.Elements {
		children[i] = elem.(TypeSafeNode)
	}
	return children
}
func (ae *ArrayExpression) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= len(ae.Elements) {
		return fmt.Errorf("index %d out of range for ArrayExpression children", index)
	}
	if newExpr, ok := newChild.(Expression); ok {
		ae.Elements[index] = newExpr
		return nil
	}
	return fmt.Errorf("expected Expression, got %T", newChild)
}

// IndexExpression represents array/map indexing: expr[index]
type IndexExpression struct {
	Span   Span
	Object Expression
	Index  Expression
}

func (ie *IndexExpression) GetSpan() Span { return ie.Span }
func (ie *IndexExpression) String() string {
	return fmt.Sprintf("%s[%s]", ie.Object.String(), ie.Index.String())
}
func (ie *IndexExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitIndexExpression(ie)
}
func (ie *IndexExpression) expressionNode()       {}
func (ie *IndexExpression) GetNodeKind() NodeKind { return NodeKindIndexExpression }
func (ie *IndexExpression) Clone() TypeSafeNode {
	clone := *ie
	if ie.Object != nil {
		clone.Object = ie.Object.(TypeSafeNode).Clone().(Expression)
	}
	if ie.Index != nil {
		clone.Index = ie.Index.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (ie *IndexExpression) Equals(other TypeSafeNode) bool {
	if oe, ok := other.(*IndexExpression); ok {
		return ((ie.Object == nil && oe.Object == nil) ||
			(ie.Object != nil && oe.Object != nil &&
				ie.Object.(TypeSafeNode).Equals(oe.Object.(TypeSafeNode)))) &&
			((ie.Index == nil && oe.Index == nil) ||
				(ie.Index != nil && oe.Index != nil &&
					ie.Index.(TypeSafeNode).Equals(oe.Index.(TypeSafeNode))))
	}
	return false
}
func (ie *IndexExpression) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 2)
	if ie.Object != nil {
		children = append(children, ie.Object.(TypeSafeNode))
	}
	if ie.Index != nil {
		children = append(children, ie.Index.(TypeSafeNode))
	}
	return children
}
func (ie *IndexExpression) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 2 {
		return fmt.Errorf("index %d out of range for IndexExpression children", index)
	}
	newExpr, ok := newChild.(Expression)
	if !ok {
		return fmt.Errorf("expected Expression, got %T", newChild)
	}
	if index == 0 {
		ie.Object = newExpr
	} else {
		ie.Index = newExpr
	}
	return nil
}

// MemberExpression represents member access: expr.member
type MemberExpression struct {
	Span     Span
	Object   Expression
	Member   *Identifier
	IsMethod bool // true for method calls that don't have parentheses yet
}

func (me *MemberExpression) GetSpan() Span { return me.Span }
func (me *MemberExpression) String() string {
	return fmt.Sprintf("%s.%s", me.Object.String(), me.Member.Value)
}
func (me *MemberExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitMemberExpression(me)
}
func (me *MemberExpression) expressionNode()       {}
func (me *MemberExpression) GetNodeKind() NodeKind { return NodeKindMemberExpression }
func (me *MemberExpression) Clone() TypeSafeNode {
	clone := *me
	if me.Object != nil {
		clone.Object = me.Object.(TypeSafeNode).Clone().(Expression)
	}
	if me.Member != nil {
		clone.Member = me.Member.Clone().(*Identifier)
	}
	return &clone
}
func (me *MemberExpression) Equals(other TypeSafeNode) bool {
	if oe, ok := other.(*MemberExpression); ok {
		return me.IsMethod == oe.IsMethod &&
			((me.Object == nil && oe.Object == nil) ||
				(me.Object != nil && oe.Object != nil &&
					me.Object.(TypeSafeNode).Equals(oe.Object.(TypeSafeNode)))) &&
			((me.Member == nil && oe.Member == nil) ||
				(me.Member != nil && oe.Member != nil &&
					me.Member.Equals(oe.Member)))
	}
	return false
}
func (me *MemberExpression) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 2)
	if me.Object != nil {
		children = append(children, me.Object.(TypeSafeNode))
	}
	if me.Member != nil {
		children = append(children, me.Member)
	}
	return children
}
func (me *MemberExpression) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 2 {
		return fmt.Errorf("index %d out of range for MemberExpression children", index)
	}
	if index == 0 {
		if newExpr, ok := newChild.(Expression); ok {
			me.Object = newExpr
			return nil
		}
		return fmt.Errorf("expected Expression for Object, got %T", newChild)
	}
	if index == 1 {
		if newIdent, ok := newChild.(*Identifier); ok {
			me.Member = newIdent
			return nil
		}
		return fmt.Errorf("expected Identifier for Member, got %T", newChild)
	}
	return fmt.Errorf("invalid child index %d for MemberExpression", index)
}

// StructExpression represents struct literal: StructName { field1: value1, field2: value2 }
type StructExpression struct {
	Span   Span
	Type   Type // Optional type annotation
	Fields []*StructFieldValue
}

type StructFieldValue struct {
	Span  Span
	Name  *Identifier
	Value Expression
}

func (se *StructExpression) GetSpan() Span { return se.Span }
func (se *StructExpression) String() string {
	var fields []string
	for _, field := range se.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", field.Name.Value, field.Value.String()))
	}
	typeName := ""
	if se.Type != nil {
		typeName = se.Type.String() + " "
	}
	return fmt.Sprintf("%s{ %s }", typeName, strings.Join(fields, ", "))
}
func (se *StructExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitStructExpression(se)
}
func (se *StructExpression) expressionNode()       {}
func (se *StructExpression) GetNodeKind() NodeKind { return NodeKindStructExpression }
func (se *StructExpression) Clone() TypeSafeNode {
	clone := *se
	if se.Type != nil {
		clone.Type = se.Type.(TypeSafeNode).Clone().(Type)
	}
	clone.Fields = make([]*StructFieldValue, len(se.Fields))
	for i, field := range se.Fields {
		fieldClone := *field
		if field.Name != nil {
			fieldClone.Name = field.Name.Clone().(*Identifier)
		}
		if field.Value != nil {
			fieldClone.Value = field.Value.(TypeSafeNode).Clone().(Expression)
		}
		clone.Fields[i] = &fieldClone
	}
	return &clone
}
func (se *StructExpression) Equals(other TypeSafeNode) bool {
	if oe, ok := other.(*StructExpression); ok {
		if len(se.Fields) != len(oe.Fields) {
			return false
		}
		if !((se.Type == nil && oe.Type == nil) ||
			(se.Type != nil && oe.Type != nil &&
				se.Type.(TypeSafeNode).Equals(oe.Type.(TypeSafeNode)))) {
			return false
		}
		for i, field := range se.Fields {
			otherField := oe.Fields[i]
			if !field.Name.Equals(otherField.Name) ||
				!field.Value.(TypeSafeNode).Equals(otherField.Value.(TypeSafeNode)) {
				return false
			}
		}
		return true
	}
	return false
}
func (se *StructExpression) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(se.Fields)*2+1)
	if se.Type != nil {
		children = append(children, se.Type.(TypeSafeNode))
	}
	for _, field := range se.Fields {
		if field.Name != nil {
			children = append(children, field.Name)
		}
		if field.Value != nil {
			children = append(children, field.Value.(TypeSafeNode))
		}
	}
	return children
}
func (se *StructExpression) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for StructExpression")
}

// ====== Additional Statement Nodes ======

// ForStatement represents for loops
type ForStatement struct {
	Span      Span
	Init      Statement  // Initialization statement (optional)
	Condition Expression // Loop condition (optional for infinite loops)
	Update    Statement  // Update statement (optional)
	Body      *BlockStatement
	Label     *Identifier // Optional label for break/continue
}

func (fs *ForStatement) GetSpan() Span                      { return fs.Span }
func (fs *ForStatement) String() string                     { return "for (...) { ... }" }
func (fs *ForStatement) Accept(visitor Visitor) interface{} { return visitor.VisitForStatement(fs) }
func (fs *ForStatement) statementNode()                     {}
func (fs *ForStatement) GetNodeKind() NodeKind              { return NodeKindForStatement }
func (fs *ForStatement) Clone() TypeSafeNode {
	clone := *fs
	if fs.Init != nil {
		clone.Init = fs.Init.(TypeSafeNode).Clone().(Statement)
	}
	if fs.Condition != nil {
		clone.Condition = fs.Condition.(TypeSafeNode).Clone().(Expression)
	}
	if fs.Update != nil {
		clone.Update = fs.Update.(TypeSafeNode).Clone().(Statement)
	}
	if fs.Body != nil {
		clone.Body = fs.Body.Clone().(*BlockStatement)
	}
	if fs.Label != nil {
		clone.Label = fs.Label.Clone().(*Identifier)
	}
	return &clone
}
func (fs *ForStatement) Equals(other TypeSafeNode) bool {
	if os, ok := other.(*ForStatement); ok {
		return ((fs.Init == nil && os.Init == nil) ||
			(fs.Init != nil && os.Init != nil &&
				fs.Init.(TypeSafeNode).Equals(os.Init.(TypeSafeNode)))) &&
			((fs.Condition == nil && os.Condition == nil) ||
				(fs.Condition != nil && os.Condition != nil &&
					fs.Condition.(TypeSafeNode).Equals(os.Condition.(TypeSafeNode)))) &&
			((fs.Update == nil && os.Update == nil) ||
				(fs.Update != nil && os.Update != nil &&
					fs.Update.(TypeSafeNode).Equals(os.Update.(TypeSafeNode)))) &&
			((fs.Body == nil && os.Body == nil) ||
				(fs.Body != nil && os.Body != nil &&
					fs.Body.Equals(os.Body))) &&
			((fs.Label == nil && os.Label == nil) ||
				(fs.Label != nil && os.Label != nil &&
					fs.Label.Equals(os.Label)))
	}
	return false
}
func (fs *ForStatement) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 5)
	if fs.Init != nil {
		children = append(children, fs.Init.(TypeSafeNode))
	}
	if fs.Condition != nil {
		children = append(children, fs.Condition.(TypeSafeNode))
	}
	if fs.Update != nil {
		children = append(children, fs.Update.(TypeSafeNode))
	}
	if fs.Body != nil {
		children = append(children, fs.Body)
	}
	if fs.Label != nil {
		children = append(children, fs.Label)
	}
	return children
}
func (fs *ForStatement) ReplaceChild(index int, newChild TypeSafeNode) error {
	children := fs.GetChildren()
	if index < 0 || index >= len(children) {
		return fmt.Errorf("index %d out of range for ForStatement children", index)
	}
	// This implementation would need to track which child is being replaced
	// based on the actual structure and ordering
	return fmt.Errorf("ReplaceChild not fully implemented for ForStatement")
}

// BreakStatement represents break statement
type BreakStatement struct {
	Span  Span
	Label *Identifier // Optional label for labeled break
}

func (bs *BreakStatement) GetSpan() Span { return bs.Span }
func (bs *BreakStatement) String() string {
	if bs.Label != nil {
		return fmt.Sprintf("break %s", bs.Label.Value)
	}
	return "break"
}
func (bs *BreakStatement) Accept(visitor Visitor) interface{} { return visitor.VisitBreakStatement(bs) }
func (bs *BreakStatement) statementNode()                     {}
func (bs *BreakStatement) GetNodeKind() NodeKind              { return NodeKindBreakStatement }
func (bs *BreakStatement) Clone() TypeSafeNode {
	clone := *bs
	if bs.Label != nil {
		clone.Label = bs.Label.Clone().(*Identifier)
	}
	return &clone
}
func (bs *BreakStatement) Equals(other TypeSafeNode) bool {
	if os, ok := other.(*BreakStatement); ok {
		return ((bs.Label == nil && os.Label == nil) ||
			(bs.Label != nil && os.Label != nil &&
				bs.Label.Equals(os.Label)))
	}
	return false
}
func (bs *BreakStatement) GetChildren() []TypeSafeNode {
	if bs.Label != nil {
		return []TypeSafeNode{bs.Label}
	}
	return []TypeSafeNode{}
}
func (bs *BreakStatement) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 || bs.Label == nil {
		return fmt.Errorf("index %d out of range for BreakStatement children", index)
	}
	if newIdent, ok := newChild.(*Identifier); ok {
		bs.Label = newIdent
		return nil
	}
	return fmt.Errorf("expected Identifier, got %T", newChild)
}

// ContinueStatement represents continue statement
type ContinueStatement struct {
	Span  Span
	Label *Identifier // Optional label for labeled continue
}

func (cs *ContinueStatement) GetSpan() Span { return cs.Span }
func (cs *ContinueStatement) String() string {
	if cs.Label != nil {
		return fmt.Sprintf("continue %s", cs.Label.Value)
	}
	return "continue"
}
func (cs *ContinueStatement) Accept(visitor Visitor) interface{} {
	return visitor.VisitContinueStatement(cs)
}
func (cs *ContinueStatement) statementNode()        {}
func (cs *ContinueStatement) GetNodeKind() NodeKind { return NodeKindContinueStatement }
func (cs *ContinueStatement) Clone() TypeSafeNode {
	clone := *cs
	if cs.Label != nil {
		clone.Label = cs.Label.Clone().(*Identifier)
	}
	return &clone
}
func (cs *ContinueStatement) Equals(other TypeSafeNode) bool {
	if os, ok := other.(*ContinueStatement); ok {
		return ((cs.Label == nil && os.Label == nil) ||
			(cs.Label != nil && os.Label != nil &&
				cs.Label.Equals(os.Label)))
	}
	return false
}
func (cs *ContinueStatement) GetChildren() []TypeSafeNode {
	if cs.Label != nil {
		return []TypeSafeNode{cs.Label}
	}
	return []TypeSafeNode{}
}
func (cs *ContinueStatement) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 || cs.Label == nil {
		return fmt.Errorf("index %d out of range for ContinueStatement children", index)
	}
	if newIdent, ok := newChild.(*Identifier); ok {
		cs.Label = newIdent
		return nil
	}
	return fmt.Errorf("expected Identifier, got %T", newChild)
}

// MatchStatement represents pattern matching statement
type MatchStatement struct {
	Span       Span
	Expression Expression
	Arms       []*MatchArm
}

type MatchArm struct {
	Span    Span
	Pattern Expression // Pattern to match against
	Guard   Expression // Optional guard condition
	Body    Statement  // Body to execute if pattern matches
}

func (ms *MatchStatement) GetSpan() Span                      { return ms.Span }
func (ms *MatchStatement) String() string                     { return "match expr { ... }" }
func (ms *MatchStatement) Accept(visitor Visitor) interface{} { return visitor.VisitMatchStatement(ms) }
func (ms *MatchStatement) statementNode()                     {}
func (ms *MatchStatement) GetNodeKind() NodeKind              { return NodeKindMatchStatement }
func (ms *MatchStatement) Clone() TypeSafeNode {
	clone := *ms
	if ms.Expression != nil {
		clone.Expression = ms.Expression.(TypeSafeNode).Clone().(Expression)
	}
	clone.Arms = make([]*MatchArm, len(ms.Arms))
	for i, arm := range ms.Arms {
		armClone := *arm
		if arm.Pattern != nil {
			armClone.Pattern = arm.Pattern.(TypeSafeNode).Clone().(Expression)
		}
		if arm.Guard != nil {
			armClone.Guard = arm.Guard.(TypeSafeNode).Clone().(Expression)
		}
		if arm.Body != nil {
			armClone.Body = arm.Body.(TypeSafeNode).Clone().(Statement)
		}
		clone.Arms[i] = &armClone
	}
	return &clone
}
func (ms *MatchStatement) Equals(other TypeSafeNode) bool {
	if os, ok := other.(*MatchStatement); ok {
		if len(ms.Arms) != len(os.Arms) {
			return false
		}
		if !((ms.Expression == nil && os.Expression == nil) ||
			(ms.Expression != nil && os.Expression != nil &&
				ms.Expression.(TypeSafeNode).Equals(os.Expression.(TypeSafeNode)))) {
			return false
		}
		for i, arm := range ms.Arms {
			otherArm := os.Arms[i]
			if !arm.Pattern.(TypeSafeNode).Equals(otherArm.Pattern.(TypeSafeNode)) ||
				!((arm.Guard == nil && otherArm.Guard == nil) ||
					(arm.Guard != nil && otherArm.Guard != nil &&
						arm.Guard.(TypeSafeNode).Equals(otherArm.Guard.(TypeSafeNode)))) ||
				!arm.Body.(TypeSafeNode).Equals(otherArm.Body.(TypeSafeNode)) {
				return false
			}
		}
		return true
	}
	return false
}
func (ms *MatchStatement) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(ms.Arms)*3+1)
	if ms.Expression != nil {
		children = append(children, ms.Expression.(TypeSafeNode))
	}
	for _, arm := range ms.Arms {
		if arm.Pattern != nil {
			children = append(children, arm.Pattern.(TypeSafeNode))
		}
		if arm.Guard != nil {
			children = append(children, arm.Guard.(TypeSafeNode))
		}
		if arm.Body != nil {
			children = append(children, arm.Body.(TypeSafeNode))
		}
	}
	return children
}
func (ms *MatchStatement) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for MatchStatement")
}

// Add the new node kind
const (
	NodeKindStructExpression NodeKind = iota + 100 // Continue from previous constants
	NodeKindMatchStatement
)

// Update the String method for new node kinds
func (nk NodeKind) StringExtended() string {
	switch nk {
	case NodeKindStructExpression:
		return "StructExpression"
	case NodeKindMatchStatement:
		return "MatchStatement"
	default:
		return nk.String() // Fall back to original String method
	}
}
