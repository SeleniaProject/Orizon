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
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Location   string        `json:"location"`
	Span       position.Span `json:"span"`
	IsParam    bool          `json:"is_param"`
	IsCaptured bool          `json:"is_captured"`
}

// FunctionInfo describes a function for debug.
type FunctionInfo struct {
	Name      string         `json:"name"`
	Span      position.Span  `json:"span"`
	Lines     []LineEntry    `json:"lines"`
	Variables []VariableInfo `json:"variables"`
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
		// Walk declarations and extract functions and variables
		for _, d := range m.Declarations {
			// Function declarations only
			if fn, ok := d.(*hir.HIRFunctionDeclaration); ok {
				fi := FunctionInfo{Name: fn.Name, Span: fn.Span}
				// Line entries: collect from statement spans
				var lines []LineEntry
				collectLinesFromNode(fn, &lines)
				// Deduplicate and sort
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

				// Variables: parameters only (locals are not modeled as a distinct list in current HIR)
				vars := make([]VariableInfo, 0, len(fn.Parameters))
				for _, p := range fn.Parameters {
					// Resolve TypeInfo string from HIRType
					typeStr := p.Type.GetType().String()
					vars = append(vars, VariableInfo{Name: p.Name, Type: typeStr, Location: "param:" + p.Name, Span: p.Span, IsParam: true})
				}
				// Sort variables for determinism
				sort.Slice(vars, func(i, j int) bool { return vars[i].Name < vars[j].Name })
				fi.Variables = vars
				mdi.Functions = append(mdi.Functions, fi)
			}
		}
		// Deterministic function order
		sort.Slice(mdi.Functions, func(i, j int) bool { return mdi.Functions[i].Name < mdi.Functions[j].Name })
		out.Modules = append(out.Modules, mdi)
	}
	return out, nil
}

// Serialize returns canonical JSON for the debug info.
func Serialize(info ProgramDebugInfo) ([]byte, error) {
	return json.MarshalIndent(info, "", "  ")
}

func itoa(v int) string { return fmtInt(v) }

// fmtInt converts int to string without fmt to avoid overhead.
func fmtInt(v int) string {
	// simple fast path
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

// collectLinesFromNode traverses node to collect source line entries.
func collectLinesFromNode(n hir.HIRNode, out *[]LineEntry) {
	sp := n.GetSpan()
	if sp.IsValid() {
		*out = append(*out, LineEntry{File: sp.Start.Filename, Line: sp.Start.Line, Column: sp.Start.Column})
	}
	for _, ch := range n.GetChildren() {
		collectLinesFromNode(ch, out)
	}
}
