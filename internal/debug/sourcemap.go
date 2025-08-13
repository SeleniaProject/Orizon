package debug

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/orizon-lang/orizon/internal/hir"
)

// SourceMap is a compact mapping from generated code back to source locations.
// This version focuses on function-level and basic line mappings derived from HIR.
type SourceMap struct {
	Version   int              `json:"version"`
	Files     []string         `json:"files"`
	Functions []FunctionRanges `json:"functions"`
}

// FunctionRanges captures the source line ranges that belong to a function per file.
type FunctionRanges struct {
	Module   string          `json:"module"`
	Name     string          `json:"name"`
	Mappings []FileLineRange `json:"mappings"`
}

// FileLineRange is an inclusive line range within a specific file.
type FileLineRange struct {
	File       string `json:"file"`
	StartLine  int    `json:"start_line"`
	StartCol   int    `json:"start_col"`
	EndLine    int    `json:"end_line"`
	EndCol     int    `json:"end_col"`
}

// GenerateSourceMap builds a SourceMap from a HIR program by traversing its spans.
func GenerateSourceMap(p *hir.HIRProgram) (SourceMap, error) {
	if p == nil {
		return SourceMap{}, errors.New("nil program")
	}
	// Collect function-level mappings
	var filesSet = map[string]struct{}{}
	var out SourceMap
	out.Version = 1

	// Stable iterate modules by name
	mods := make([]*hir.HIRModule, 0, len(p.Modules))
	for _, m := range p.Modules {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Name < mods[j].Name })

	for _, m := range mods {
		for _, d := range m.Declarations {
			fn, ok := d.(*hir.HIRFunctionDeclaration)
			if !ok {
				continue
			}
			// Gather per-file line coverage by walking the function subtree
			var entries []LineEntry
			collectLinesFromNode(fn, &entries)
			if len(entries) == 0 {
				continue
			}
			// Group by file and compute min/max line+column ranges
			byFile := map[string]FileLineRange{}
			for _, le := range entries {
				flr, ok := byFile[le.File]
				if !ok {
					flr = FileLineRange{File: le.File, StartLine: le.Line, StartCol: le.Column, EndLine: le.Line, EndCol: le.Column}
				} else {
					// Expand
					if le.Line < flr.StartLine || (le.Line == flr.StartLine && le.Column < flr.StartCol) {
						flr.StartLine, flr.StartCol = le.Line, le.Column
					}
					if le.Line > flr.EndLine || (le.Line == flr.EndLine && le.Column > flr.EndCol) {
						flr.EndLine, flr.EndCol = le.Line, le.Column
					}
				}
				byFile[le.File] = flr
				filesSet[le.File] = struct{}{}
			}
			// Materialize deterministic list of file ranges
			files := make([]string, 0, len(byFile))
			for f := range byFile { files = append(files, f) }
			sort.Strings(files)
			maps := make([]FileLineRange, 0, len(files))
			for _, f := range files { maps = append(maps, byFile[f]) }
			out.Functions = append(out.Functions, FunctionRanges{ Module: m.Name, Name: fn.Name, Mappings: maps })
		}
	}

	// Files array in stable order
	files := make([]string, 0, len(filesSet))
	for f := range filesSet { files = append(files, f) }
	sort.Strings(files)
	out.Files = files
	return out, nil
}

// SerializeSourceMap returns canonical JSON for the SourceMap.
func SerializeSourceMap(sm SourceMap) ([]byte, error) {
	return json.MarshalIndent(sm, "", "  ")
}


