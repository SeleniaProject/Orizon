package debug

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

// WriteELFWithDWARF writes a minimal ELF64 relocatable object that contains.
// the provided DWARF sections as .debug_* sections.
// This does not attempt to provide symbol tables or relocations; it packages.
// the raw sections for external tools to consume.
func WriteELFWithDWARF(outPath string, secs DWARFSections) error {
	if outPath == "" {
		return errors.New("empty outPath")
	}

	b, err := buildELF64DWARF(secs)
	if err != nil {
		return err
	}

	return os.WriteFile(outPath, b, 0o644)
}

// buildELF64DWARF constructs a minimal ELF64 (ET_REL) file containing.
// .debug_abbrev, .debug_info, .debug_line, .debug_str sections and a .shstrtab.
func buildELF64DWARF(secs DWARFSections) ([]byte, error) {
	// Section names in desired order.
	names := []string{".debug_abbrev", ".debug_info", ".debug_line", ".debug_str"}
	payloads := [][]byte{secs.Abbrev, secs.Info, secs.Line, secs.Str}
	// Build section header string table.
	shstr := &bytes.Buffer{}
	_ = shstr.WriteByte(0) // first entry is empty

	nameOff := make([]uint32, len(names)+1) // +1 for .shstrtab itself (will be last)
	for i, n := range names {
		nameOff[i] = uint32(shstr.Len())
		shstr.WriteString(n)
		shstr.WriteByte(0)
	}

	shstrtabName := ".shstrtab"
	nameOff[len(names)] = uint32(shstr.Len())
	shstr.WriteString(shstrtabName)
	shstr.WriteByte(0)

	// ELF header constants.
	const (
		EI_NIDENT    = 16
		ehdrSize     = 64
		shdrSize     = 64
		ET_REL       = 1
		EM_X86_64    = 62
		SHT_PROGBITS = 1
		SHT_STRTAB   = 3
	)

	// Layout: [ELF header][section payloads...][.shstrtab][section header table]
	cur := uint64(ehdrSize)
	// Section header table entries: null + debug sections + shstrtab.
	shnum := uint16(1 + len(names) + 1)
	// Record offsets/sizes for sections
	off := make([]uint64, len(names))
	sz := make([]uint64, len(names))

	for i, p := range payloads {
		off[i] = cur
		sz[i] = uint64(len(p))
		cur += sz[i]
	}
	// .shstrtab offset/size
	shstrOff := cur
	shstrSz := uint64(shstr.Len())
	cur += shstrSz
	// Section header table offset.
	shoff := cur
	// Build file.
	file := &bytes.Buffer{}
	file.Grow(int(cur) + int(shdrSize)*int(shnum))
	// Write ELF header.
	ehdr := make([]byte, ehdrSize)
	copy(ehdr[0:4], []byte{0x7f, 'E', 'L', 'F'})
	ehdr[4] = 2 // ELFCLASS64
	ehdr[5] = 1 // ELFDATA2LSB
	ehdr[6] = 1 // EV_CURRENT
	// rest of e_ident zeros.
	binary.LittleEndian.PutUint16(ehdr[16:], ET_REL)
	binary.LittleEndian.PutUint16(ehdr[18:], EM_X86_64)
	binary.LittleEndian.PutUint32(ehdr[20:], 1)                    // e_version
	binary.LittleEndian.PutUint64(ehdr[24:], 0)                    // e_entry
	binary.LittleEndian.PutUint64(ehdr[32:], 0)                    // e_phoff
	binary.LittleEndian.PutUint64(ehdr[40:], shoff)                // e_shoff
	binary.LittleEndian.PutUint32(ehdr[48:], 0)                    // e_flags
	binary.LittleEndian.PutUint16(ehdr[52:], ehdrSize)             // e_ehsize
	binary.LittleEndian.PutUint16(ehdr[54:], 0)                    // e_phentsize
	binary.LittleEndian.PutUint16(ehdr[56:], 0)                    // e_phnum
	binary.LittleEndian.PutUint16(ehdr[58:], shdrSize)             // e_shentsize
	binary.LittleEndian.PutUint16(ehdr[60:], shnum)                // e_shnum
	binary.LittleEndian.PutUint16(ehdr[62:], uint16(1+len(names))) // e_shstrndx (last section)
	file.Write(ehdr)
	// Write payload sections.
	for i, p := range payloads {
		_ = i

		file.Write(p)
	}
	// Write .shstrtab
	file.Write(shstr.Bytes())
	// Write section header table.
	// Null section.
	file.Write(make([]byte, shdrSize))
	// Helpers.
	writeShdr := func(nameOff uint32, shtype uint32, flags uint64, addr uint64, off uint64, size uint64, link uint32, info uint32, addralign uint64, entsize uint64) {
		sh := make([]byte, shdrSize)
		binary.LittleEndian.PutUint32(sh[0:], nameOff)
		binary.LittleEndian.PutUint32(sh[4:], shtype)
		binary.LittleEndian.PutUint64(sh[8:], flags)
		binary.LittleEndian.PutUint64(sh[16:], addr)
		binary.LittleEndian.PutUint64(sh[24:], off)
		binary.LittleEndian.PutUint64(sh[32:], size)
		binary.LittleEndian.PutUint32(sh[40:], link)
		binary.LittleEndian.PutUint32(sh[44:], info)
		binary.LittleEndian.PutUint64(sh[48:], addralign)
		binary.LittleEndian.PutUint64(sh[56:], entsize)
		file.Write(sh)
	}
	for i := range names {
		writeShdr(nameOff[i], SHT_PROGBITS, 0, 0, off[i], sz[i], 0, 0, 1, 0)
	}
	// shstrtab is last.
	writeShdr(nameOff[len(names)], SHT_STRTAB, 0, 0, shstrOff, shstrSz, 0, 0, 1, 0)
	// Patch e_shoff (it was computed earlier, but ensure buffer length matches).
	_ = shoff

	return file.Bytes(), nil
}
