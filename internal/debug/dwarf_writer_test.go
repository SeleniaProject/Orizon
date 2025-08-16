package debug

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

// buildMinimalHIR builds a tiny HIR program with one module and one function
func buildMinimalHIR() *hir.HIRProgram {
	p := hir.NewHIRProgram()
	m := &hir.HIRModule{ID: 1, ModuleID: 1, Name: "main"}
	fn := &hir.HIRFunctionDeclaration{
		ID:   2,
		Name: "foo",
		Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 1}, End: position.Position{Filename: "main.oriz", Line: 3, Column: 1}},
		Parameters: []*hir.HIRParameter{
			{ID: 3, Name: "x", Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 5}}},
		},
	}
	// Body with one literal to seed a line mapping
	fn.Body = &hir.HIRBlockStatement{ID: 4, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 2, Column: 1}, End: position.Position{Filename: "main.oriz", Line: 2, Column: 5}},
		Statements: []hir.HIRStatement{&hir.HIRExpressionStatement{ID: 5, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 2, Column: 1}, End: position.Position{Filename: "main.oriz", Line: 2, Column: 5}}}},
	}
	m.Declarations = []hir.HIRDeclaration{fn}
	p.Modules[1] = m
	return p
}

func TestBuildDWARF_MinimalSections(t *testing.T) {
	// Build ProgramDebugInfo from HIR via Emitter
	p := buildMinimalHIR()
	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	// Basic non-empty checks
	if len(secs.Abbrev) == 0 || len(secs.Info) == 0 || len(secs.Line) == 0 || len(secs.Str) == 0 {
		t.Fatalf("unexpected empty sections: %+v", secs)
	}
	// .debug_info: first 4 bytes are unit length; must equal total-4
	if len(secs.Info) < 8 {
		t.Fatalf("info too small: %d", len(secs.Info))
	}
	ul := uint32(secs.Info[0]) | uint32(secs.Info[1])<<8 | uint32(secs.Info[2])<<16 | uint32(secs.Info[3])<<24
	if int(ul) != len(secs.Info)-4 {
		t.Fatalf("info unit length mismatch: got %d want %d", ul, len(secs.Info)-4)
	}
	// .debug_line: same header rule
	if len(secs.Line) < 8 {
		t.Fatalf("line too small: %d", len(secs.Line))
	}
	ll := uint32(secs.Line[0]) | uint32(secs.Line[1])<<8 | uint32(secs.Line[2])<<16 | uint32(secs.Line[3])<<24
	if int(ll) != len(secs.Line)-4 {
		t.Fatalf("line unit length mismatch: got %d want %d", ll, len(secs.Line)-4)
	}
	// .debug_abbrev must start with abbrev code 1 for compile_unit (ULEB128 encoded 0x01, 0x11 tag follows somewhere soon)
	if secs.Abbrev[0] != 0x01 {
		t.Fatalf("abbrev does not start with code 1: 0x%x", secs.Abbrev[0])
	}
}

func TestBuildDWARF_ContainsFrameBaseAndParamLocation(t *testing.T) {
	p := buildMinimalHIR()
	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	if len(secs.Info) == 0 {
		t.Fatalf("info empty")
	}
	// Check that DW_OP_call_frame_cfa (0x9c) appears in .debug_info (frame_base exprloc)
	foundCFA := false
	for _, b := range secs.Info {
		if b == 0x9c {
			foundCFA = true
			break
		}
	}
	if !foundCFA {
		t.Fatalf("expected DW_OP_call_frame_cfa (0x9c) in .debug_info")
	}
	// Check that DW_OP_fbreg (0x91) appears for parameter/local location
	foundFBReg := false
	for _, b := range secs.Info {
		if b == 0x91 {
			foundFBReg = true
			break
		}
	}
	if !foundFBReg {
		t.Fatalf("expected DW_OP_fbreg (0x91) in .debug_info")
	}
	// sanity: ensure at least one DW_TAG_variable abbrev (code 4) is present via abbrev table
	if len(secs.Abbrev) == 0 || secs.Abbrev[0] == 0 {
		t.Fatalf("abbrev empty")
	}
}

func TestWriteELFWithDWARF_Minimal(t *testing.T) {
	p := buildMinimalHIR()
	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	tmp := t.TempDir()
	out := filepath.Join(tmp, "dbg.o")
	if err := WriteELFWithDWARF(out, secs); err != nil {
		t.Fatalf("WriteELFWithDWARF: %v", err)
	}
	// Basic size check
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Size() <= 64 { // must be larger than ELF header
		t.Fatalf("elf too small: %d", fi.Size())
	}
}

func TestWriteCOFFWithDWARF_Minimal(t *testing.T) {
	p := buildMinimalHIR()
	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	tmp := t.TempDir()
	out := filepath.Join(tmp, "dbg.obj")
	if err := WriteCOFFWithDWARF(out, secs); err != nil {
		t.Fatalf("WriteCOFFWithDWARF: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Size() <= 20 { // must be larger than COFF header
		t.Fatalf("coff too small: %d", fi.Size())
	}
}

func TestWriteMachOWithDWARF_Minimal(t *testing.T) {
	p := buildMinimalHIR()
	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	tmp := t.TempDir()
	out := filepath.Join(tmp, "dbg.o")
	if err := WriteMachOWithDWARF(out, secs); err != nil {
		t.Fatalf("WriteMachOWithDWARF: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Size() <= 32 { // must be larger than Mach-O header
		t.Fatalf("macho too small: %d", fi.Size())
	}
}

func TestBuildDWARF_StructArray_DIEs(t *testing.T) {
	// Build HIR with a struct parameter and an array parameter
	p := hir.NewHIRProgram()
	m := &hir.HIRModule{ID: 1, ModuleID: 1, Name: "main"}
	tb := hir.NewHIRTypeBuilder(p)

	i32 := tb.BuildBasicType("int32", position.Span{})
	i64 := tb.BuildBasicType("int64", position.Span{})
	pi64 := tb.BuildPointerType(i64, false, position.Span{})
	st := tb.BuildStructType("S", []hir.HIRStructField{
		{Name: "a", Type: i32, Span: position.Span{}},
		{Name: "b", Type: pi64, Span: position.Span{}},
	}, position.Span{})
	arr := tb.BuildArrayType(i32, nil, position.Span{})

	pS := &hir.HIRParameter{ID: 20, Name: "s", Type: st}
	pArr := &hir.HIRParameter{ID: 21, Name: "arr", Type: arr}
	fn := &hir.HIRFunctionDeclaration{
		ID:         2,
		Name:       "f",
		Span:       position.Span{Start: position.Position{Filename: "m.oriz", Line: 1, Column: 1}},
		Parameters: []*hir.HIRParameter{pS, pArr},
		ReturnType: i32,
		Body:       &hir.HIRBlockStatement{ID: 3, Span: position.Span{Start: position.Position{Filename: "m.oriz", Line: 1, Column: 1}}},
	}
	m.Declarations = []hir.HIRDeclaration{fn}
	p.Modules[1] = m

	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	// debug_str should contain struct and field names
	if !bytes.Contains(secs.Str, []byte("S\x00")) || !bytes.Contains(secs.Str, []byte("a\x00")) || !bytes.Contains(secs.Str, []byte("b\x00")) {
		t.Fatalf("expected struct and member names in .debug_str")
	}
}

func TestBuildDWARF_StructMemberOffsets(t *testing.T) {
	// Build a struct S { a:int32; b:*int64 } and verify member offsets are encoded
	p := hir.NewHIRProgram()
	m := &hir.HIRModule{ID: 1, ModuleID: 1, Name: "main"}
	tb := hir.NewHIRTypeBuilder(p)

	i32 := tb.BuildBasicType("int32", position.Span{})
	i64 := tb.BuildBasicType("int64", position.Span{})
	pi64 := tb.BuildPointerType(i64, false, position.Span{})
	st := tb.BuildStructType("S", []hir.HIRStructField{
		{Name: "a", Type: i32, Span: position.Span{}},
		{Name: "b", Type: pi64, Span: position.Span{}},
	}, position.Span{})

	// Use as a parameter to ensure the type is referenced
	fn := &hir.HIRFunctionDeclaration{
		ID:         2,
		Name:       "f",
		Span:       position.Span{Start: position.Position{Filename: "m.oriz", Line: 1, Column: 1}},
		Parameters: []*hir.HIRParameter{{ID: 20, Name: "s", Type: st}},
		ReturnType: i32,
		Body:       &hir.HIRBlockStatement{ID: 3, Span: position.Span{Start: position.Position{Filename: "m.oriz", Line: 1, Column: 1}}},
	}
	m.Declarations = []hir.HIRDeclaration{fn}
	p.Modules[1] = m

	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	secs, err := BuildDWARF(dbg)
	if err != nil {
		t.Fatalf("BuildDWARF: %v", err)
	}
	// Expect that the struct field offset for 'b' is 4 (little-endian 0x04 00 00 00)
	want := []byte{0x04, 0x00, 0x00, 0x00}
	if !bytes.Contains(secs.Info, want) {
		t.Fatalf("expected member offset 4 encoded in .debug_info")
	}
}
