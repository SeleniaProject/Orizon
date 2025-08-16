package debug

import (
	"sort"
)

// PCRange represents a contiguous pseudo-PC range for a function.
type PCRange struct {
	Low       uint64
	High      uint64
	FileLines []LineEntry // sorted by appearance; each step assumed fixed size
}

// PCMap provides pseudo address to source mapping based on ProgramDebugInfo.
type PCMap struct {
	Ranges []PCRange
}

// BuildPCMap builds a PC map from ProgramDebugInfo mirroring the same policy
// used when generating DWARF (4 bytes per line entry, min size 4 bytes).
func BuildPCMap(info ProgramDebugInfo) *PCMap {
	m := &PCMap{}
	mods := make([]ModuleDebugInfo, len(info.Modules))
	copy(mods, info.Modules)
	sort.Slice(mods, func(i, j int) bool { return mods[i].ModuleName < mods[j].ModuleName })
	pc := uint64(0)
	for _, md := range mods {
		fns := make([]FunctionInfo, len(md.Functions))
		copy(fns, md.Functions)
		sort.Slice(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })
		for _, fn := range fns {
			lines := make([]LineEntry, len(fn.Lines))
			copy(lines, fn.Lines)
			szLines := len(lines)
			if szLines == 0 {
				szLines = 1
			}
			size := uint64(szLines * 4)
			r := PCRange{Low: pc, High: pc + size, FileLines: lines}
			m.Ranges = append(m.Ranges, r)
			pc += size
		}
	}
	return m
}

// AddrToLine resolves a pseudo address to a file/line pair using a constant
// 4-byte step per line entry within the owning function range.
func (m *PCMap) AddrToLine(addr uint64) (file string, line int, ok bool) {
	for _, r := range m.Ranges {
		if addr < r.Low || addr >= r.High {
			continue
		}
		if len(r.FileLines) == 0 {
			// no line entries; return unknown but ok
			return "", 0, true
		}
		offset := addr - r.Low
		idx := int(offset / 4)
		if idx >= len(r.FileLines) {
			idx = len(r.FileLines) - 1
		}
		le := r.FileLines[idx]
		return le.File, le.Line, true
	}
	return "", 0, false
}
