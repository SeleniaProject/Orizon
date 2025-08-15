package debug

import (
	"bytes"
	"encoding/binary"
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// DWARFSections holds raw DWARF section payloads.
type DWARFSections struct {
	Abbrev []byte
	Info   []byte
	Line   []byte
	Str    []byte
}

// BuildDWARF builds minimal DWARF v4 sections from ProgramDebugInfo.
// The generated data contains:
// - .debug_abbrev: abbrev for compile_unit (code 1) and subprogram (code 2)
// - .debug_info: one CU per module and child DIEs for functions
// - .debug_line: a valid empty line program per CU (single end_sequence)
// - .debug_str: string table used by references
func BuildDWARF(info ProgramDebugInfo) (DWARFSections, error) {
	if len(info.Modules) == 0 {
		return DWARFSections{}, errors.New("no modules")
	}

	// Deterministic module order
	mods := make([]ModuleDebugInfo, len(info.Modules))
	copy(mods, info.Modules)
	sort.Slice(mods, func(i, j int) bool { return mods[i].ModuleName < mods[j].ModuleName })

	// Build .debug_line first and record stmt_list offsets per CU
	line := &bytes.Buffer{}
	stmtOffsets := make([]uint32, 0, len(mods))
	cuFileIndex := make([]map[string]uint32, 0, len(mods))
	for _, m := range mods {
		unitStart := uint32(line.Len())
		body := &bytes.Buffer{}
		// version
		binary.Write(body, binary.LittleEndian, uint16(2))
		// header
		hdr := &bytes.Buffer{}
		hdr.WriteByte(1)   // minimum_instruction_length
		hdr.WriteByte(1)   // default_is_stmt (true)
		hdr.WriteByte(251) // line_base = -5 (two's complement)
		hdr.WriteByte(14)  // line_range
		hdr.WriteByte(13)  // opcode_base
		for i := 0; i < 12; i++ {
			hdr.WriteByte(0)
		}
		// include_directories from function line entries
		fileSet := map[string]struct{}{}
		for _, fn := range m.Functions {
			for _, le := range fn.Lines {
				if le.File != "" {
					fileSet[le.File] = struct{}{}
				}
			}
		}
		files := make([]string, 0, len(fileSet))
		for f := range fileSet {
			files = append(files, f)
		}
		sort.Strings(files)
		dirIndex := map[string]uint32{}
		dirs := make([]string, 0)
		for _, f := range files {
			d := filepath.ToSlash(filepath.Dir(f))
			if d == "." {
				d = ""
			}
			if _, ok := dirIndex[d]; !ok {
				dirIndex[d] = uint32(len(dirs) + 1)
				dirs = append(dirs, d)
			}
		}
		for _, d := range dirs {
			hdr.WriteString(d)
			hdr.WriteByte(0)
		}
		hdr.WriteByte(0) // end of include_directories
		// file_names entries
		for _, f := range files {
			name := filepath.Base(f)
			hdr.WriteString(name)
			hdr.WriteByte(0)
			uleb128(hdr, uint64(dirIndex[filepath.ToSlash(filepath.Dir(f))])) // dir index
			uleb128(hdr, 0)                                                   // mtime
			uleb128(hdr, 0)                                                   // length
		}
		hdr.WriteByte(0) // end of file table
		// header_length and header bytes
		binary.Write(body, binary.LittleEndian, uint32(hdr.Len()))
		body.Write(hdr.Bytes())

		// Derive per-file ordered line entries and emit rows
		type row struct {
			file      string
			line, col int
		}
		rows := make([]row, 0)
		for _, fn := range m.Functions {
			for _, le := range fn.Lines {
				if le.File == "" || le.Line <= 0 {
					continue
				}
				rows = append(rows, row{file: le.File, line: le.Line, col: le.Column})
			}
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].file != rows[j].file {
				return rows[i].file < rows[j].file
			}
			if rows[i].line != rows[j].line {
				return rows[i].line < rows[j].line
			}
			return rows[i].col < rows[j].col
		})
		currFile := 1
		currLine := 1
		for _, r := range rows {
			fi := indexOf(files, r.file) + 1
			if fi <= 0 {
				continue
			}
			if fi != currFile {
				body.WriteByte(4) // DW_LNS_set_file
				uleb128(body, uint64(fi))
				currFile = fi
			}
			// set column
			body.WriteByte(5) // DW_LNS_set_column
			uleb128(body, uint64(max(1, r.col)))
			// advance line by delta
			delta := r.line - currLine
			if delta != 0 {
				body.WriteByte(3) // DW_LNS_advance_line
				sleb128(body, int64(delta))
				currLine = r.line
			}
			body.WriteByte(1) // DW_LNS_copy (emit row)
		}
		// end sequence
		body.WriteByte(0) // extended
		body.WriteByte(1) // length
		body.WriteByte(1) // DW_LNE_end_sequence

		// prepend unit length and append to section
		unit := &bytes.Buffer{}
		binary.Write(unit, binary.LittleEndian, uint32(body.Len()))
		unit.Write(body.Bytes())
		line.Write(unit.Bytes())
		stmtOffsets = append(stmtOffsets, unitStart)
		fmap := make(map[string]uint32, len(files))
		for i, f := range files {
			fmap[f] = uint32(i + 1)
		}
		cuFileIndex = append(cuFileIndex, fmap)
	}

	// Build .debug_str
	str := &bytes.Buffer{}
	_ = str.WriteByte(0)
	strIndex := map[string]uint32{}
	writeStr := func(s string) uint32 {
		if off, ok := strIndex[s]; ok {
			return off
		}
		off := uint32(str.Len())
		str.WriteString(s)
		str.WriteByte(0)
		strIndex[s] = off
		return off
	}

	// Build .debug_abbrev
	ab := &bytes.Buffer{}
	// Abbrev 1: compile_unit with children, attrs: name(strp), stmt_list(sec_offset), comp_dir(strp)
	uleb128(ab, 1)
	uleb128(ab, 0x11)
	ab.WriteByte(1)
	uleb128(ab, 0x03)
	uleb128(ab, 0x0e)
	uleb128(ab, 0x10)
	uleb128(ab, 0x17)
	uleb128(ab, 0x1b)
	uleb128(ab, 0x0e)
	uleb128(ab, 0)
	uleb128(ab, 0)
	// Abbrev 2: subprogram, children yes
	// attrs: name(strp), decl_file(data4), decl_line(data4), low_pc(addr), high_pc(data4 as size), frame_base(exprloc)
	uleb128(ab, 2)
	uleb128(ab, 0x2e)
	ab.WriteByte(1)
	uleb128(ab, 0x03)
	uleb128(ab, 0x0e)
	uleb128(ab, 0x3f)
	uleb128(ab, 0x06)
	uleb128(ab, 0x3a)
	uleb128(ab, 0x06)
	uleb128(ab, 0x11)
	uleb128(ab, 0x01)
	uleb128(ab, 0x12)
	uleb128(ab, 0x06)
	uleb128(ab, 0x40)
	uleb128(ab, 0x18)
	uleb128(ab, 0)
	uleb128(ab, 0)
	// Abbrev 3: formal_parameter, no children
	// attrs: name(strp), decl_file(data4), decl_line(data4), location(exprloc), type(ref4)
	uleb128(ab, 3)
	uleb128(ab, 0x05)
	// Abbrev 5: base_type, no children; attrs: name(strp), encoding(data1), byte_size(data1)
	uleb128(ab, 5)
	uleb128(ab, 0x24)
	ab.WriteByte(0)
	uleb128(ab, 0x03)
	uleb128(ab, 0x0e)
	uleb128(ab, 0x3e)
	uleb128(ab, 0x0b)
	uleb128(ab, 0x0b)
	uleb128(ab, 0x0b)
	uleb128(ab, 0)
	uleb128(ab, 0)
	ab.WriteByte(0)
	uleb128(ab, 0x03)
	uleb128(ab, 0x0e)
	uleb128(ab, 0x3f)
	uleb128(ab, 0x06)
	uleb128(ab, 0x3a)
	uleb128(ab, 0x06)
	uleb128(ab, 0x02)
	uleb128(ab, 0x18)
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)
	// Abbrev 4: variable, no children
	// attrs: name(strp), decl_file(data4), decl_line(data4), location(exprloc), type(ref4)
	uleb128(ab, 4)
	uleb128(ab, 0x34)
	ab.WriteByte(0)
	uleb128(ab, 0x03)
	uleb128(ab, 0x0e)
	uleb128(ab, 0x3f)
	uleb128(ab, 0x06)
	uleb128(ab, 0x3a)
	uleb128(ab, 0x06)
	uleb128(ab, 0x02)
	uleb128(ab, 0x18)
	uleb128(ab, 0x49)
	uleb128(ab, 0x13)
	uleb128(ab, 0)
	uleb128(ab, 0)
	ab.WriteByte(0)

	// Abbrev 13: typedef, no children; attrs: name(strp), type(ref4)
	uleb128(ab, 13)
	uleb128(ab, 0x16) // DW_TAG_typedef
	ab.WriteByte(0)
	uleb128(ab, 0x03) // DW_AT_name
	uleb128(ab, 0x0e) // DW_FORM_strp
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 14: const_type, no children; attrs: type(ref4)
	uleb128(ab, 14)
	uleb128(ab, 0x26) // DW_TAG_const_type
	ab.WriteByte(0)
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 15: volatile_type, no children; attrs: type(ref4)
	uleb128(ab, 15)
	uleb128(ab, 0x25) // DW_TAG_volatile_type
	ab.WriteByte(0)
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 6: pointer_type, no children; attrs: byte_size(data1)
	uleb128(ab, 6)
	uleb128(ab, 0x0f) // DW_TAG_pointer_type
	ab.WriteByte(0)
	uleb128(ab, 0x0b) // DW_AT_byte_size
	uleb128(ab, 0x0b) // DW_FORM_data1
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 7: structure_type, children yes; attrs: name(strp), byte_size(data4)
	uleb128(ab, 7)
	uleb128(ab, 0x13) // DW_TAG_structure_type
	ab.WriteByte(1)
	uleb128(ab, 0x03) // DW_AT_name
	uleb128(ab, 0x0e) // DW_FORM_strp
	uleb128(ab, 0x0b) // DW_AT_byte_size
	uleb128(ab, 0x06) // DW_FORM_data4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 8: member, no children; attrs: name(strp), type(ref4), data_member_location(data4)
	uleb128(ab, 8)
	uleb128(ab, 0x0d) // DW_TAG_member
	ab.WriteByte(0)
	uleb128(ab, 0x03) // DW_AT_name
	uleb128(ab, 0x0e) // DW_FORM_strp
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0x38) // DW_AT_data_member_location
	uleb128(ab, 0x06) // DW_FORM_data4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 9: subroutine_type, children yes; attrs: type(ref4)
	uleb128(ab, 9)
	uleb128(ab, 0x15) // DW_TAG_subroutine_type
	ab.WriteByte(1)
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 10: formal_parameter (type-only), no children; attrs: type(ref4)
	uleb128(ab, 10)
	uleb128(ab, 0x05) // DW_TAG_formal_parameter
	ab.WriteByte(0)
	uleb128(ab, 0x49) // DW_AT_type
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 11: array_type, children yes; attrs: type(ref4)
	uleb128(ab, 11)
	uleb128(ab, 0x01) // DW_TAG_array_type
	ab.WriteByte(1)
	uleb128(ab, 0x49) // DW_AT_type (element type)
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Abbrev 12: subrange_type, no children; attrs: type(ref4), count(data4)
	uleb128(ab, 12)
	uleb128(ab, 0x21) // DW_TAG_subrange_type
	ab.WriteByte(0)
	uleb128(ab, 0x49) // DW_AT_type (index type)
	uleb128(ab, 0x13) // DW_FORM_ref4
	uleb128(ab, 0x55) // DW_AT_count
	uleb128(ab, 0x06) // DW_FORM_data4
	uleb128(ab, 0)
	uleb128(ab, 0)

	// Build .debug_info using recorded stmt_list offsets
	inf := &bytes.Buffer{}
	pcCursor := uint64(0)
	for i, m := range mods {
		cu := &bytes.Buffer{}
		binary.Write(cu, binary.LittleEndian, uint16(4)) // version
		binary.Write(cu, binary.LittleEndian, uint32(0)) // abbrev offset
		cu.WriteByte(8)                                  // address size
		// CU DIE
		uleb128(cu, 1)
		nameOff := writeStr(m.ModuleName)
		binary.Write(cu, binary.LittleEndian, nameOff)        // DW_AT_name
		binary.Write(cu, binary.LittleEndian, stmtOffsets[i]) // DW_AT_stmt_list
		compOff := writeStr("")
		binary.Write(cu, binary.LittleEndian, compOff) // DW_AT_comp_dir
		// Pre-emit type DIEs (base/pointer/struct) used in this CU
		// Use string signature key to deduplicate
		typeOffsets := map[string]uint32{}
		internType := func(sig string, emit func()) uint32 {
			if off, ok := typeOffsets[sig]; ok {
				return off
			}
			off := uint32(cu.Len())
			typeOffsets[sig] = off
			emit()
			return off
		}
		// Scan variables to identify base/pointer/struct/array signatures
		for _, fn := range m.Functions {
			for _, v := range fn.Variables {
				if v.TypeMeta == nil {
					// fallback to base type by name
					if v.Type != "" {
						name := v.Type
						if _, ok := typeOffsets[name]; !ok {
							enc, size, ok2 := mapBaseType(name)
							if ok2 {
								internType(name, func() {
									uleb128(cu, 5)
									no := writeStr(name)
									binary.Write(cu, binary.LittleEndian, no)
									cu.WriteByte(enc)
									cu.WriteByte(size)
								})
							}
						}
					}
					continue
				}
				var emitType func(t TypeMeta) uint32
				emitType = func(t TypeMeta) uint32 {
					// Build a stable signature
					sig := t.Kind + ":" + t.Name
					if len(t.Parameters) > 0 {
						var b strings.Builder
						b.WriteString(sig)
						b.WriteString("<")
						for i, p := range t.Parameters {
							if i > 0 {
								b.WriteString(",")
							}
							b.WriteString(p.Kind)
							b.WriteString(":")
							b.WriteString(p.Name)
						}
						b.WriteString(">")
						sig = b.String()
					}
					if off, ok := typeOffsets[sig]; ok {
						return off
					}
					// Handle qualifiers and typedef aliases before kind-specific emission
					if len(t.Qualifiers) > 0 {
						// For const/volatile, we wrap the underlying type DIE with const_type/volatile_type
						// Build the unqualified base first
						base := t
						base.Qualifiers = nil
						baseOff := emitType(base)
						for _, q := range t.Qualifiers {
							switch q {
							case "const":
								// Abbrev 14: const_type
								qualifiedSig := sig + ";const"
								return internType(qualifiedSig, func() {
									uleb128(cu, 14)
									binary.Write(cu, binary.LittleEndian, baseOff)
								})
							case "volatile":
								// Abbrev 15: volatile_type
								qualifiedSig := sig + ";volatile"
								return internType(qualifiedSig, func() {
									uleb128(cu, 15)
									binary.Write(cu, binary.LittleEndian, baseOff)
								})
							default:
								// Unknown qualifier: fall back to base type
								return baseOff
							}
						}
					}
					if t.AliasOf != nil {
						// typedef: refer to underlying type and provide a distinct name
						uOff := emitType(*t.AliasOf)
						typedefSig := sig + ";typedef"
						return internType(typedefSig, func() {
							uleb128(cu, 13) // typedef
							no := writeStr(t.Name)
							binary.Write(cu, binary.LittleEndian, no)
							binary.Write(cu, binary.LittleEndian, uOff)
						})
					}
					switch t.Kind {
					case "int", "float", "bool", "string":
						// map to base_type
						name := t.Name
						if name == "" {
							name = t.Kind
						}
						enc, size, ok := mapBaseType(name)
						if !ok {
							enc, size = 0x05, 8
						}
						return internType(sig, func() {
							uleb128(cu, 5)
							no := writeStr(name)
							binary.Write(cu, binary.LittleEndian, no)
							cu.WriteByte(enc)
							cu.WriteByte(byte(max(1, int(size))))
						})
					case "pointer":
						// Represent as pointer_type of size 8
						return internType(sig, func() {
							uleb128(cu, 6) // pointer_type
							cu.WriteByte(8)
						})
					case "struct":
						// Emit structure_type and members
						off := internType(sig, func() {
							uleb128(cu, 7) // structure_type
							no := writeStr(t.Name)
							binary.Write(cu, binary.LittleEndian, no)
							binary.Write(cu, binary.LittleEndian, uint32(max(0, int(t.Size))))
							// members
							for _, f := range t.Fields {
								uleb128(cu, 8) // member
								fn := writeStr(f.Name)
								binary.Write(cu, binary.LittleEndian, fn)
								// member type: ensure emitted
								mtoff := emitType(f.Type)
								binary.Write(cu, binary.LittleEndian, mtoff)
								binary.Write(cu, binary.LittleEndian, uint32(max(0, int(f.Offset))))
							}
							cu.WriteByte(0) // end of children
						})
						return off
					case "slice":
						// Represent slice as structure { data:*T, len:int32, cap:int32 }
						off := internType(sig, func() {
							uleb128(cu, 7) // structure_type
							no := writeStr(t.Name)
							binary.Write(cu, binary.LittleEndian, no)
							// Approx size 24 (ptr+len+cap on 64-bit: 8+4+4 rounded) - use 24
							binary.Write(cu, binary.LittleEndian, uint32(24))
							// member: data
							uleb128(cu, 8)
							mn := writeStr("data")
							binary.Write(cu, binary.LittleEndian, mn)
							// pointer_type (8 bytes)
							ptoff := internType("ptr", func() { uleb128(cu, 6); cu.WriteByte(8) })
							binary.Write(cu, binary.LittleEndian, ptoff)
							binary.Write(cu, binary.LittleEndian, uint32(0))
							// member: len
							uleb128(cu, 8)
							mln := writeStr("len")
							binary.Write(cu, binary.LittleEndian, mln)
							// int32 base_type
							enc, sz, _ := mapBaseType("int32")
							i32off := internType("int32", func() {
								uleb128(cu, 5)
								no := writeStr("int32")
								binary.Write(cu, binary.LittleEndian, no)
								cu.WriteByte(enc)
								cu.WriteByte(sz)
							})
							binary.Write(cu, binary.LittleEndian, i32off)
							binary.Write(cu, binary.LittleEndian, uint32(8))
							// member: cap
							uleb128(cu, 8)
							mc := writeStr("cap")
							binary.Write(cu, binary.LittleEndian, mc)
							binary.Write(cu, binary.LittleEndian, i32off)
							binary.Write(cu, binary.LittleEndian, uint32(12))
							cu.WriteByte(0)
						})
						return off
					case "interface":
						// Represent interface as structure { vptr:*void, data:*void }
						off := internType(sig, func() {
							uleb128(cu, 7)
							no := writeStr(t.Name)
							binary.Write(cu, binary.LittleEndian, no)
							binary.Write(cu, binary.LittleEndian, uint32(16))
							// vptr
							uleb128(cu, 8)
							mv := writeStr("vptr")
							binary.Write(cu, binary.LittleEndian, mv)
							ptoff := internType("ptr", func() { uleb128(cu, 6); cu.WriteByte(8) })
							binary.Write(cu, binary.LittleEndian, ptoff)
							binary.Write(cu, binary.LittleEndian, uint32(0))
							// data
							uleb128(cu, 8)
							md := writeStr("data")
							binary.Write(cu, binary.LittleEndian, md)
							binary.Write(cu, binary.LittleEndian, ptoff)
							binary.Write(cu, binary.LittleEndian, uint32(8))
							cu.WriteByte(0)
						})
						return off
					case "tuple":
						// Model tuple as an anonymous structure with members _0, _1, ...
						off := internType(sig, func() {
							uleb128(cu, 7) // structure_type
							no := writeStr(t.Name)
							binary.Write(cu, binary.LittleEndian, no)
							// approximate size: sum of param sizes if known
							var tsum int
							var offCur int
							for _, p := range t.Parameters {
								if p.Size > 0 {
									tsum += int(p.Size)
								}
							}
							binary.Write(cu, binary.LittleEndian, uint32(max(0, tsum)))
							for i, p := range t.Parameters {
								uleb128(cu, 8) // member
								fn := writeStr("_" + strings.TrimLeft(strings.TrimPrefix(strings.Repeat("0", 0), ""), "") + strconv.Itoa(i))
								binary.Write(cu, binary.LittleEndian, fn)
								mtoff := emitType(p)
								binary.Write(cu, binary.LittleEndian, mtoff)
								// naive contiguous layout
								binary.Write(cu, binary.LittleEndian, uint32(max(0, offCur)))
								if p.Size > 0 {
									offCur += int(p.Size)
								}
							}
							cu.WriteByte(0)
						})
						return off
					case "array":
						// array_type with one subrange; use element type from Parameters[0] if present
						var elOff uint32
						if len(t.Parameters) > 0 {
							elOff = emitType(t.Parameters[0])
						}
						asig := sig
						return internType(asig, func() {
							uleb128(cu, 11) // array_type
							// element type
							binary.Write(cu, binary.LittleEndian, elOff)
							// child: subrange_type with count if size known; index type = base_type 'int'
							// Ensure an index base type exists (use signed int32)
							idxName := "int32"
							idxEnc, idxSz, _ := mapBaseType(idxName)
							idxSig := "int:" + idxName
							idxOff := internType(idxSig, func() {
								uleb128(cu, 5)
								no := writeStr(idxName)
								binary.Write(cu, binary.LittleEndian, no)
								cu.WriteByte(idxEnc)
								cu.WriteByte(idxSz)
							})
							uleb128(cu, 12) // subrange_type
							binary.Write(cu, binary.LittleEndian, idxOff)
							// Count: derive roughly from t.Size / element size if both known (>0)
							var count uint32 = 0
							if t.Size > 0 && len(t.Parameters) > 0 && t.Parameters[0].Size > 0 {
								es := uint64(t.Parameters[0].Size)
								if es > 0 {
									count = uint32(uint64(t.Size) / es)
								}
							}
							binary.Write(cu, binary.LittleEndian, count)
							cu.WriteByte(0) // end of children
						})
					default:
						// Fallback to base type by name/kind
						name := t.Name
						if name == "" {
							name = t.Kind
						}
						enc, size, ok := mapBaseType(name)
						if !ok {
							enc, size = 0x05, 8
						}
						return internType(sig, func() {
							uleb128(cu, 5)
							no := writeStr(name)
							binary.Write(cu, binary.LittleEndian, no)
							cu.WriteByte(enc)
							cu.WriteByte(byte(max(1, int(size))))
						})
					}
				}
				// ensure type DIE exists for this variable
				_ = emitType(*v.TypeMeta)
			}
		}
		// Children: functions
		for _, fn := range m.Functions {
			uleb128(cu, 2)
			fnOff := writeStr(fn.Name)
			binary.Write(cu, binary.LittleEndian, fnOff)
			// decl_file/decl_line
			var fileIdx uint32
			if fn.Span.Start.Filename != "" {
				if idx, ok := cuFileIndex[i][fn.Span.Start.Filename]; ok {
					fileIdx = idx
				}
			}
			binary.Write(cu, binary.LittleEndian, fileIdx)
			binary.Write(cu, binary.LittleEndian, uint32(fn.Span.Start.Line))
			// low_pc / high_pc(size)
			low := pcCursor
			// Compute approximate size from line entries for determinism
			// Each line contributes 4 bytes; minimum size 1
			szLines := len(fn.Lines)
			if szLines <= 0 {
				szLines = 1
			}
			size := uint32(szLines * 4)
			binary.Write(cu, binary.LittleEndian, low)  // DW_AT_low_pc (addr)
			binary.Write(cu, binary.LittleEndian, size) // DW_AT_high_pc (as size)
			// frame_base = DW_OP_call_frame_cfa
			// exprloc: [len=1][0x9c]
			cu.WriteByte(1)    // ULEB128 length=1 (fits in single byte)
			cu.WriteByte(0x9c) // DW_OP_call_frame_cfa
			// Optionally, attach subroutine_type to describe signature (return + params)
			if fn.ReturnType != nil || len(fn.ParamTypes) > 0 {
				uleb128(cu, 9) // subroutine_type
				// Return type
				var retOff uint32
				if fn.ReturnType != nil {
					// Emit/lookup type for return type
					// Reuse emitType by rebuilding meta → for that we need a small adapter, so skip detailed mapping here
					// Fall back to base 'int32' if unknown
					name := "int32"
					if fn.ReturnType.Name != "" {
						name = fn.ReturnType.Name
					}
					enc, sz, ok := mapBaseType(name)
					if !ok {
						enc, sz = 0x05, 8
					}
					sig := "ret:" + name
					retOff = internType(sig, func() {
						uleb128(cu, 5)
						no := writeStr(name)
						binary.Write(cu, binary.LittleEndian, no)
						cu.WriteByte(enc)
						cu.WriteByte(sz)
					})
				}
				binary.Write(cu, binary.LittleEndian, retOff)
				// children: parameter types only
				for _, pt := range fn.ParamTypes {
					uleb128(cu, 10) // formal_parameter (type-only)
					// Simplify: map by name/kind using base types as needed
					nm := pt.Name
					if nm == "" {
						nm = pt.Kind
					}
					enc, sz, ok := mapBaseType(nm)
					if !ok {
						enc, sz = 0x05, 8
					}
					psig := "param:" + nm
					poff := internType(psig, func() {
						uleb128(cu, 5)
						no := writeStr(nm)
						binary.Write(cu, binary.LittleEndian, no)
						cu.WriteByte(enc)
						cu.WriteByte(sz)
					})
					binary.Write(cu, binary.LittleEndian, poff)
				}
				cu.WriteByte(0) // end of subroutine_type children
			}
			// formal parameters and locals (params first, then locals)
			// Parameters: compute precise frame-base offsets based on declared sizes when available
			paramOffset := int64(0)
			paramIndex := 0
			for _, v := range fn.Variables {
				if !v.IsParam {
					continue
				}
				uleb128(cu, 3)
				nameOff := writeStr(v.Name)
				binary.Write(cu, binary.LittleEndian, nameOff)
				var pfileIdx uint32
				if v.Span.Start.Filename != "" {
					if idx, ok := cuFileIndex[i][v.Span.Start.Filename]; ok {
						pfileIdx = idx
					}
				}
				binary.Write(cu, binary.LittleEndian, pfileIdx)
				binary.Write(cu, binary.LittleEndian, uint32(v.Span.Start.Line))
				// location: DW_OP_fbreg <sleb128 offset>
				// Use accumulated slot size to compute a realistic frame-base offset for parameters
				sz := int64(computeSlotSize(v))
				// Place parameter at current offset from CFA
				off := paramOffset + sz
				// expr length = 1 opcode + sleb128(off)
				expr := &bytes.Buffer{}
				expr.WriteByte(0x91) // DW_OP_fbreg
				sleb128(expr, off)
				// write length then bytes
				uleb128(cu, uint64(expr.Len()))
				cu.Write(expr.Bytes())
				// type ref: prefer structured meta
				if v.TypeMeta != nil {
					// rebuild signature like in emitter
					sig := v.TypeMeta.Kind + ":" + v.TypeMeta.Name
					if len(v.TypeMeta.Parameters) > 0 {
						var b strings.Builder
						b.WriteString(sig)
						b.WriteString("<")
						for i, p := range v.TypeMeta.Parameters {
							if i > 0 {
								b.WriteString(",")
							}
							b.WriteString(p.Kind)
							b.WriteString(":")
							b.WriteString(p.Name)
						}
						b.WriteString(">")
						sig = b.String()
					}
					if toff, ok := typeOffsets[sig]; ok {
						binary.Write(cu, binary.LittleEndian, toff)
					} else {
						binary.Write(cu, binary.LittleEndian, uint32(0))
					}
				} else if toff, ok := typeOffsets[v.Type]; ok {
					binary.Write(cu, binary.LittleEndian, toff)
				} else {
					binary.Write(cu, binary.LittleEndian, uint32(0))
				}
				paramOffset += sz
				paramIndex++
			}
			// locals: emit as DW_TAG_variable (abbrev 4) with fbreg negative offsets (stack grows down)
			localOffset := int64(0)
			for _, v := range fn.Variables {
				if v.IsParam {
					continue
				}
				uleb128(cu, 4)
				nameOff := writeStr(v.Name)
				binary.Write(cu, binary.LittleEndian, nameOff)
				var lfileIdx uint32
				if v.Span.Start.Filename != "" {
					if idx, ok := cuFileIndex[i][v.Span.Start.Filename]; ok {
						lfileIdx = idx
					}
				}
				binary.Write(cu, binary.LittleEndian, lfileIdx)
				binary.Write(cu, binary.LittleEndian, uint32(v.Span.Start.Line))
				// Stack grows down: accumulate slot sizes and assign negative offsets from CFA
				sz := int64(computeSlotSize(v))
				localOffset += sz
				loff := -localOffset
				lexpr := &bytes.Buffer{}
				lexpr.WriteByte(0x91)
				sleb128(lexpr, loff)
				uleb128(cu, uint64(lexpr.Len()))
				cu.Write(lexpr.Bytes())
				// type ref (if available)
				if v.TypeMeta != nil {
					sig := v.TypeMeta.Kind + ":" + v.TypeMeta.Name
					if len(v.TypeMeta.Parameters) > 0 {
						var b strings.Builder
						b.WriteString(sig)
						b.WriteString("<")
						for i, p := range v.TypeMeta.Parameters {
							if i > 0 {
								b.WriteString(",")
							}
							b.WriteString(p.Kind)
							b.WriteString(":")
							b.WriteString(p.Name)
						}
						b.WriteString(">")
						sig = b.String()
					}
					if toff, ok := typeOffsets[sig]; ok {
						binary.Write(cu, binary.LittleEndian, toff)
					} else {
						binary.Write(cu, binary.LittleEndian, uint32(0))
					}
				} else if toff, ok := typeOffsets[v.Type]; ok {
					binary.Write(cu, binary.LittleEndian, toff)
				} else {
					binary.Write(cu, binary.LittleEndian, uint32(0))
				}
			}
			cu.WriteByte(0) // end of subprogram children
			// advance pc cursor
			pcCursor += uint64(size)
		}
		cu.WriteByte(0)
		// unit length
		unit := &bytes.Buffer{}
		binary.Write(unit, binary.LittleEndian, uint32(cu.Len()))
		unit.Write(cu.Bytes())
		inf.Write(unit.Bytes())
	}

	return DWARFSections{Abbrev: ab.Bytes(), Info: inf.Bytes(), Line: line.Bytes(), Str: str.Bytes()}, nil
}

// computeSlotSize returns a reasonable stack slot size for a variable based on its type metadata/name.
// This is a best-effort heuristic for accurate DW_OP_fbreg offsets in the absence of concrete layout.
func computeSlotSize(v VariableInfo) int {
	// Prefer structured type metadata sizes
	if v.TypeMeta != nil && v.TypeMeta.Size > 0 {
		// Round up to 8-byte slot alignment for 64-bit targets
		sz := int(v.TypeMeta.Size)
		if sz%8 != 0 {
			sz = ((sz + 7) / 8) * 8
		}
		return sz
	}
	// Fallback to common base types inferred from name strings
	switch v.Type {
	case "int64", "uint64", "float64", "i64", "u64", "f64":
		return 8
	case "int32", "uint32", "float32", "i32", "u32", "f32":
		return 4
	case "bool", "u8", "i8", "byte", "char":
		return 1
	default:
		// Default conservative slot size: 8 bytes
		return 8
	}
}

// mapBaseType returns DWARF base type encoding and size for a given simple type name.
// This is a minimal mapping for demo purposes.
func mapBaseType(name string) (enc byte, size byte, ok bool) {
	switch name {
	case "Int", "Int32", "int32":
		return 0x05, 4, true // DW_ATE_signed, 4 bytes
	case "Int64", "int64":
		return 0x05, 8, true
	case "UInt32", "uint32":
		return 0x07, 4, true // DW_ATE_unsigned
	case "UInt64", "uint64":
		return 0x07, 8, true
	case "Float32", "float32":
		return 0x04, 4, true // DW_ATE_float
	case "Float64", "float64":
		return 0x04, 8, true
	case "Bool", "bool":
		return 0x02, 1, true // DW_ATE_boolean
	case "Char", "char", "byte", "u8":
		return 0x08, 1, true // DW_ATE_unsigned_char
	case "String", "string":
		// Model as pointer-sized unsigned for now
		return 0x07, 8, true
	default:
		return 0, 0, false
	}
}

// uleb128 encodes an unsigned integer in LEB128 format.
func uleb128(b *bytes.Buffer, v uint64) {
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			c |= 0x80
		}
		b.WriteByte(c)
		if v == 0 {
			break
		}
	}
}

// sleb128 encodes a signed integer in LEB128 format.
func sleb128(b *bytes.Buffer, v int64) {
	for {
		c := byte(v & 0x7f)
		sign := (c & 0x40) != 0
		v >>= 7
		done := (v == 0 && !sign) || (v == -1 && sign)
		if !done {
			c |= 0x80
		}
		b.WriteByte(c)
		if done {
			break
		}
	}
}

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
