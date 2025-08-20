package debug

import (
	"encoding/json"
	"sort"
)

// Frame represents a single stack frame in a pseudo execution context.
type Frame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	PC       uint64 `json:"pc"`
	Line     int    `json:"line"`
}

// StackTrace is a collection of frames ordered from top (current) to bottom (older).
type StackTrace struct {
	Frames []Frame `json:"frames"`
}

// BuildStackTrace constructs a best-effort stack trace from the current pc using the PCMap and ProgramDebugInfo.
// Since the Orizon pseudo-execution model does not record real call stacks at this layer yet, this function.
// produces at least the current frame and, when possible, a small context of neighboring function boundaries.
func BuildStackTrace(pcmap *PCMap, info ProgramDebugInfo, pc uint64) StackTrace {
	// Build deterministic module/function ordering like PCMap
	mods := make([]ModuleDebugInfo, len(info.Modules))
	copy(mods, info.Modules)
	sort.Slice(mods, func(i, j int) bool { return mods[i].ModuleName < mods[j].ModuleName })

	// Find current range and line.
	var curFn string

	var curFile string

	var curLine int

	for _, r := range pcmap.Ranges {
		if pc >= r.Low && pc < r.High {
			// find the owning function name by matching lines against FunctionInfo collections.
			// fallback to unknown if not found.
			if len(r.FileLines) > 0 {
				off := int((pc - r.Low) / 4)
				if off >= len(r.FileLines) {
					off = len(r.FileLines) - 1
				}

				curFile = r.FileLines[off].File
				curLine = r.FileLines[off].Line
			}
			// attempt to map to function name by scanning ordered functions and accumulating sizes.
			var pcCursor uint64

			for _, md := range mods {
				fns := make([]FunctionInfo, len(md.Functions))
				copy(fns, md.Functions)
				sort.Slice(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })

				for _, fn := range fns {
					sz := len(fn.Lines)
					if sz == 0 {
						sz = 1
					}

					low := pcCursor
					high := pcCursor + uint64(sz*4)

					if pc >= low && pc < high {
						curFn = fn.Name

						break
					}

					pcCursor = high
				}

				if curFn != "" {
					break
				}
			}

			break
		}
	}

	frames := make([]Frame, 0, 3)
	frames = append(frames, Frame{PC: pc, Function: curFn, File: curFile, Line: curLine})

	// Provide one previous and one next boundary as context if available.
	// Previous.
	var prev *PCRange

	var next *PCRange

	for i := range pcmap.Ranges {
		r := pcmap.Ranges[i]
		if pc >= r.Low && pc < r.High {
			if i > 0 {
				prev = &pcmap.Ranges[i-1]
			}

			if i+1 < len(pcmap.Ranges) {
				next = &pcmap.Ranges[i+1]
			}

			break
		}
	}

	if prev != nil {
		file := ""
		line := 0

		if len(prev.FileLines) > 0 {
			file = prev.FileLines[len(prev.FileLines)-1].File
			line = prev.FileLines[len(prev.FileLines)-1].Line
		}

		frames = append(frames, Frame{PC: prev.High - 4, Function: "", File: file, Line: line})
	}

	if next != nil {
		file := ""
		line := 0

		if len(next.FileLines) > 0 {
			file = next.FileLines[0].File
			line = next.FileLines[0].Line
		}

		frames = append(frames, Frame{PC: next.Low, Function: "", File: file, Line: line})
	}

	return StackTrace{Frames: frames}
}

// EncodeStackTraceJSON encodes the stack trace into JSON bytes.
func EncodeStackTraceJSON(st StackTrace) []byte {
	b, _ := json.Marshal(st)

	return b
}
