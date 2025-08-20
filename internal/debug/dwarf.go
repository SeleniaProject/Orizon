package debug

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

// LineEntry maps an address (abstract) to a source line.
type LineEntry struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// VariableInfo describes a variable with scope and type.
type VariableInfo struct {
	TypeMeta    *TypeMeta     `json:"type_meta,omitempty"`
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Location    string        `json:"location"`
	AddressBase string        `json:"address_base,omitempty"`
	Span        position.Span `json:"span"`
	FrameOffset int64         `json:"frame_offset,omitempty"`
	IsParam     bool          `json:"is_param"`
	IsCaptured  bool          `json:"is_captured"`
}

// FunctionInfo describes a function for debug.
type FunctionInfo struct {
	ReturnType *TypeMeta      `json:"return_type,omitempty"`
	Name       string         `json:"name"`
	Lines      []LineEntry    `json:"lines"`
	Variables  []VariableInfo `json:"variables"`
	ParamTypes []TypeMeta     `json:"param_types,omitempty"`
	Span       position.Span  `json:"span"`
}

// ModuleDebugInfo aggregates module-level debug info.
type ModuleDebugInfo struct {
	ModuleName string         `json:"module_name"`
	Functions  []FunctionInfo `json:"functions"`
}

// ProgramDebugInfo is the top-level debug info artifact.
type ProgramDebugInfo struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Modules     []ModuleDebugInfo `json:"modules"`
}

// TypeMeta provides a lightweight, JSON-serializable snapshot of a type.
type TypeMeta struct {
	AliasOf    *TypeMeta   `json:"alias_of,omitempty"`
	Kind       string      `json:"kind"`
	Name       string      `json:"name"`
	Parameters []TypeMeta  `json:"parameters,omitempty"`
	Fields     []TypeField `json:"fields,omitempty"`
	Qualifiers []string    `json:"qualifiers,omitempty"`
	Size       int64       `json:"size"`
	Alignment  int64       `json:"alignment"`
}

// TypeField describes a struct/record field.
type TypeField struct {
	Type   TypeMeta `json:"type"`
	Name   string   `json:"name"`
	Offset int64    `json:"offset"`
}

// Emitter builds debug information from HIR.
type Emitter struct{}

func NewEmitter() *Emitter { return &Emitter{} }

// Emit constructs ProgramDebugInfo from a HIR program in a deterministic order.
func (e *Emitter) Emit(p *hir.HIRProgram) (ProgramDebugInfo, error) {
	if p == nil {
		return ProgramDebugInfo{}, errors.New("nil program")
	}

	mods := make([]*hir.HIRModule, 0, len(p.Modules))
	for _, m := range p.Modules {
		mods = append(mods, m)
	}

	sort.Slice(mods, func(i, j int) bool { return mods[i].Name < mods[j].Name })

	out := ProgramDebugInfo{GeneratedAt: time.Now().UTC()}

	for _, m := range mods {
		mdi := ModuleDebugInfo{ModuleName: m.Name}
		// Walk declarations and extract functions and variables.
		for _, d := range m.Declarations {
			// Function declarations only.
			if fn, ok := d.(*hir.HIRFunctionDeclaration); ok {
				fi := FunctionInfo{Name: fn.Name, Span: fn.Span}
				// Line entries: collect from statement spans.
				var lines []LineEntry

				collectLinesFromNode(fn, &lines)
				// Deduplicate and sort.
				dedup := make(map[string]LineEntry)

				for _, le := range lines {
					key := le.File + "#" + itoa(le.Line) + ":" + itoa(le.Column)
					dedup[key] = le
				}

				uniq := make([]LineEntry, 0, len(dedup))
				for _, v := range dedup {
					uniq = append(uniq, v)
				}

				sort.Slice(uniq, func(i, j int) bool {
					if uniq[i].File != uniq[j].File {
						return uniq[i].File < uniq[j].File
					}

					if uniq[i].Line != uniq[j].Line {
						return uniq[i].Line < uniq[j].Line
					}

					return uniq[i].Column < uniq[j].Column
				})

				fi.Lines = uniq

				// Variables: parameters and locals (collect from function body).
				vars := make([]VariableInfo, 0, len(fn.Parameters))

				for _, p := range fn.Parameters {
					// Resolve TypeInfo.
					var typeStr string

					var tm *TypeMeta

					if p.Type != nil {
						ti := p.Type.GetType()
						typeStr = ti.String()
						mt := convertTypeInfoToMeta(ti)
						tm = &mt
					}

					vars = append(vars, VariableInfo{Name: p.Name, Type: typeStr, TypeMeta: tm, Location: "param:" + p.Name, Span: p.Span, IsParam: true})
				}
				// collect locals from body.
				if fn.Body != nil {
					locals := make([]VariableInfo, 0)
					collectLocalsFromNode(fn.Body, &locals)
					vars = append(vars, locals...)
				}
				// Sort variables for determinism.
				sort.Slice(vars, func(i, j int) bool { return vars[i].Name < vars[j].Name })
				// Compute approximate frame offsets from a virtual frame base.
				// Parameters at non-negative increasing offsets; locals at negative offsets.
				var paramOffset int64

				var localOffset int64

				for i := range vars {
					sz := int64(computeVarSlotSize(vars[i]))

					if vars[i].IsParam {
						vars[i].AddressBase = "fbreg"
						vars[i].FrameOffset = paramOffset
						// advance offset by slot size (8-byte aligned).
						paramOffset += sz
					} else {
						vars[i].AddressBase = "fbreg"
						localOffset += sz
						vars[i].FrameOffset = -localOffset
					}
				}

				fi.Variables = vars
				// Basic function type summary (return/params) if available
				if fn.ReturnType != nil {
					rt := fn.ReturnType.GetType()
					mt := convertTypeInfoToMeta(rt)
					fi.ReturnType = &mt
				}

				if len(fn.Parameters) > 0 {
					fi.ParamTypes = make([]TypeMeta, 0, len(fn.Parameters))

					for _, p := range fn.Parameters {
						if p.Type != nil {
							fi.ParamTypes = append(fi.ParamTypes, convertTypeInfoToMeta(p.Type.GetType()))
						}
					}
				}

				mdi.Functions = append(mdi.Functions, fi)
			}
		}
		// Deterministic function order.
		sort.Slice(mdi.Functions, func(i, j int) bool { return mdi.Functions[i].Name < mdi.Functions[j].Name })
		out.Modules = append(out.Modules, mdi)
	}

	return out, nil
}

// Serialize returns canonical JSON for the debug info.
func Serialize(info ProgramDebugInfo) ([]byte, error) {
	return json.MarshalIndent(info, "", "  ")
}

// Deserialize parses ProgramDebugInfo from JSON.
func Deserialize(b []byte) (ProgramDebugInfo, error) {
	var info ProgramDebugInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return ProgramDebugInfo{}, err
	}

	return info, nil
}

func itoa(v int) string { return fmtInt(v) }

// fmtInt converts int to string without fmt to avoid overhead.
func fmtInt(v int) string {
	// simple fast path.
	if v == 0 {
		return "0"
	}

	neg := false
	if v < 0 {
		neg = true
		v = -v
	}

	var buf [20]byte

	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}

	if neg {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}

// convertTypeInfoToMeta translates HIR TypeInfo into a serializable TypeMeta tree.
func convertTypeInfoToMeta(t hir.TypeInfo) TypeMeta {
	tm := TypeMeta{
		Kind:      typeKindToString(t.Kind),
		Name:      t.Name,
		Size:      t.Size,
		Alignment: t.Alignment,
	}
	if len(t.Parameters) > 0 {
		tm.Parameters = make([]TypeMeta, len(t.Parameters))
		for i := range t.Parameters {
			tm.Parameters[i] = convertTypeInfoToMeta(t.Parameters[i])
		}
	}

	if len(t.Fields) > 0 {
		tm.Fields = make([]TypeField, len(t.Fields))
		for i := range t.Fields {
			tm.Fields[i] = TypeField{
				Name:   t.Fields[i].Name,
				Offset: t.Fields[i].Offset,
				Type:   convertTypeInfoToMeta(t.Fields[i].Type),
			}
		}
	}

	return tm
}

// computeVarSlotSize estimates a stack slot size for a variable for frame offset modeling.
// It prefers TypeMeta.Size when present; otherwise falls back to common base type sizes.
func computeVarSlotSize(v VariableInfo) int {
	if v.TypeMeta != nil && v.TypeMeta.Size > 0 {
		sz := int(v.TypeMeta.Size)
		if sz%8 != 0 {
			sz = ((sz + 7) / 8) * 8
		}

		return sz
	}

	switch v.Type {
	case "int64", "uint64", "float64", "i64", "u64", "f64":
		return 8
	case "int32", "uint32", "float32", "i32", "u32", "f32":
		return 4
	case "bool", "u8", "i8", "byte", "char":
		return 1
	default:
		return 8
	}
}

func typeKindToString(k hir.TypeKind) string {
	switch k {
	case hir.TypeKindUnknown:
		return "unknown"
	case hir.TypeKindInvalid:
		return "invalid"
	case hir.TypeKindVoid:
		return "void"
	case hir.TypeKindBoolean:
		return "bool"
	case hir.TypeKindInteger:
		return "int"
	case hir.TypeKindFloat:
		return "float"
	case hir.TypeKindString:
		return "string"
	case hir.TypeKindArray:
		return "array"
	case hir.TypeKindSlice:
		return "slice"
	case hir.TypeKindPointer:
		return "pointer"
	case hir.TypeKindFunction:
		return "function"
	case hir.TypeKindStruct:
		return "struct"
	case hir.TypeKindInterface:
		return "interface"
	case hir.TypeKindGeneric:
		return "generic"
	case hir.TypeKindTypeParameter:
		return "type_parameter"
	case hir.TypeKindVariable:
		return "type_variable"
	case hir.TypeKindTuple:
		return "tuple"
	case hir.TypeKindSkolem:
		return "skolem"
	case hir.TypeKindHigherRank:
		return "higher_rank"
	case hir.TypeKindDependent:
		return "dependent"
	case hir.TypeKindEffect:
		return "effect"
	case hir.TypeKindLinear:
		return "linear"
	case hir.TypeKindRefinement:
		return "refinement"
	case hir.TypeKindType:
		return "type"
	case hir.TypeKindApplication:
		return "application"
	default:
		return "unknown"
	}
}

// collectLinesFromNode traverses node to collect source line entries.
func collectLinesFromNode(n hir.HIRNode, out *[]LineEntry) {
	if n == nil {
		return
	}

	sp := n.GetSpan()
	if sp.IsValid() {
		*out = append(*out, LineEntry{File: sp.Start.Filename, Line: sp.Start.Line, Column: sp.Start.Column})
	}

	for _, ch := range n.GetChildren() {
		collectLinesFromNode(ch, out)
	}
}

// collectLocalsFromNode traverses to collect local variable declarations.
func collectLocalsFromNode(n hir.HIRNode, out *[]VariableInfo) {
	if n == nil {
		return
	}
	// Match variable declarations.
	if vd, ok := n.(*hir.HIRVariableDeclaration); ok {
		typeStr := ""

		var tm *TypeMeta

		if vd.Type != nil {
			ti := vd.Type.GetType()
			typeStr = ti.String()
			mt := convertTypeInfoToMeta(ti)
			tm = &mt
		}

		*out = append(*out, VariableInfo{
			Name:       vd.Name,
			Type:       typeStr,
			TypeMeta:   tm,
			Location:   "local:" + vd.Name,
			Span:       vd.Span,
			IsParam:    false,
			IsCaptured: false,
		})
	}

	for _, ch := range n.GetChildren() {
		collectLocalsFromNode(ch, out)
	}
}
