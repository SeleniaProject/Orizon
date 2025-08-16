package debug

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"strconv"
)

// WriteCOFFWithDWARF writes a minimal COFF (PE COFF object) file that contains
// the provided DWARF sections as initialized data sections. This writer targets
// x86-64 (Machine=0x8664) and does not emit symbols or relocations.
func WriteCOFFWithDWARF(outPath string, secs DWARFSections) error {
	if outPath == "" {
		return errors.New("empty outPath")
	}
	b, err := buildCOFFDWARF(secs)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}

// buildCOFFDWARF constructs a COFF object with .debug_* sections and a string table.
func buildCOFFDWARF(secs DWARFSections) ([]byte, error) {
	// COFF constants
	const (
		imageFileHeaderSize    = 20
		imageSectionHeaderSize = 40
		machineAMD64           = 0x8664
		// Section characteristics
		IMAGE_SCN_CNT_INITIALIZED_DATA = 0x00000040
		IMAGE_SCN_MEM_READ             = 0x40000000
		IMAGE_SCN_ALIGN_1BYTES         = 0x00100000
	)

	// Section order
	names := []string{".debug_abbrev", ".debug_info", ".debug_line", ".debug_str"}
	payloads := [][]byte{secs.Abbrev, secs.Info, secs.Line, secs.Str}
	sectionCount := len(names)

	// Prepare string table for long section names (>8 chars) using "/offset" notation.
	strtabData := &bytes.Buffer{}
	nameToOff := map[string]uint32{}
	for _, n := range names {
		if len(n) > 8 {
			// Offset counts from start of string table data (after the 4-byte size field)
			off := uint32(strtabData.Len()) + 4
			nameToOff[n] = off
			strtabData.WriteString(n)
			strtabData.WriteByte(0)
		}
	}

	// Layout: [COFF file header][section headers][section data][string table]
	headerSize := imageFileHeaderSize + imageSectionHeaderSize*sectionCount
	cur := uint32(headerSize)
	// 4-byte alignment for section data
	align4 := func(v uint32) uint32 { return (v + 3) &^ 3 }

	ptr := make([]uint32, sectionCount)
	sz := make([]uint32, sectionCount)
	for i, p := range payloads {
		if len(p) == 0 {
			ptr[i] = 0
			sz[i] = 0
			continue
		}
		cur = align4(cur)
		ptr[i] = cur
		sz[i] = uint32(len(p))
		cur += sz[i]
	}
	// String table starts at cur; size = 4 + len(data)
	strtabSize := uint32(4 + strtabData.Len())
	fileSize := align4(cur) + strtabSize

	buf := &bytes.Buffer{}
	buf.Grow(int(fileSize))

	// IMAGE_FILE_HEADER (20 bytes)
	// struct fields: Machine(2), NumberOfSections(2), TimeDateStamp(4),
	// PointerToSymbolTable(4), NumberOfSymbols(4), SizeOfOptionalHeader(2), Characteristics(2)
	binary.Write(buf, binary.LittleEndian, uint16(machineAMD64))
	binary.Write(buf, binary.LittleEndian, uint16(sectionCount))
	binary.Write(buf, binary.LittleEndian, uint32(0)) // TimeDateStamp
	binary.Write(buf, binary.LittleEndian, uint32(0)) // PointerToSymbolTable
	binary.Write(buf, binary.LittleEndian, uint32(0)) // NumberOfSymbols
	binary.Write(buf, binary.LittleEndian, uint16(0)) // SizeOfOptionalHeader
	binary.Write(buf, binary.LittleEndian, uint16(0)) // Characteristics

	// IMAGE_SECTION_HEADERs (40 bytes each)
	for i := 0; i < sectionCount; i++ {
		name := names[i]
		nameField := make([]byte, 8)
		if len(name) <= 8 {
			copy(nameField, []byte(name))
		} else {
			// "/<offset>" decimal
			off := nameToOff[name]
			ref := []byte("/" + strconv.FormatUint(uint64(off), 10))
			copy(nameField, ref)
		}
		buf.Write(nameField)
		// Misc/VirtualSize
		binary.Write(buf, binary.LittleEndian, uint32(len(payloads[i])))
		// VirtualAddress (0 for object)
		binary.Write(buf, binary.LittleEndian, uint32(0))
		// SizeOfRawData
		binary.Write(buf, binary.LittleEndian, sz[i])
		// PointerToRawData
		binary.Write(buf, binary.LittleEndian, ptr[i])
		// PointerToRelocations / PointerToLinenumbers
		binary.Write(buf, binary.LittleEndian, uint32(0))
		binary.Write(buf, binary.LittleEndian, uint32(0))
		// NumberOfRelocations / NumberOfLinenumbers
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		// Characteristics
		binary.Write(buf, binary.LittleEndian, uint32(IMAGE_SCN_CNT_INITIALIZED_DATA|IMAGE_SCN_MEM_READ|IMAGE_SCN_ALIGN_1BYTES))
	}

	// Section data (aligned to 4)
	padTo := func(pos uint32) {
		for uint32(buf.Len()) < pos {
			buf.WriteByte(0)
		}
	}
	for i, p := range payloads {
		if len(p) == 0 {
			continue
		}
		padTo(ptr[i])
		buf.Write(p)
	}
	// Align before string table
	padTo(align4(uint32(buf.Len())))
	// String table: 4-byte size + data
	binary.Write(buf, binary.LittleEndian, uint32(4+strtabData.Len()))
	buf.Write(strtabData.Bytes())

	return buf.Bytes(), nil
}
