package debug

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

// WriteMachOWithDWARF writes a minimal Mach-O 64-bit object containing the
// provided DWARF sections under the __DWARF segment.
// This is a barebones writer suitable for bundling raw DWARF; it does not
// include relocations or symbols.
func WriteMachOWithDWARF(outPath string, secs DWARFSections) error {
	if outPath == "" {
		return errors.New("empty outPath")
	}
	b, err := buildMachODWARF(secs)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}

// Mach-O 64-bit constants and structures (mirroring mach-o/loader.h minimal set)
const (
	MH_MAGIC_64            = 0xfeedfacf
	CPU_TYPE_X86_64        = 0x01000007
	CPU_SUBTYPE_X86_64_ALL = 0x00000003
	MH_OBJECT              = 0x1
	LC_SEGMENT_64          = 0x19
)

type machHeader64 struct {
	Magic      uint32
	CpuType    uint32
	CpuSubtype uint32
	FileType   uint32
	NCmds      uint32
	SizeOfCmds uint32
	Flags      uint32
	Reserved   uint32
}

type segmentCommand64 struct {
	Cmd      uint32
	Cmdsize  uint32
	Segname  [16]byte
	Vmaddr   uint64
	Vmsize   uint64
	Fileoff  uint64
	Filesize uint64
	Maxprot  int32
	Initprot int32
	Nsects   uint32
	Flags    uint32
}

type section64 struct {
	Sectname  [16]byte
	Segname   [16]byte
	Addr      uint64
	Size      uint64
	Offset    uint32
	Align     uint32
	Reloff    uint32
	Nreloc    uint32
	Flags     uint32
	Reserved1 uint32
	Reserved2 uint32
	Reserved3 uint32
}

func setPaddedName(dst *[16]byte, name string) {
	n := len(name)
	if n > 16 {
		n = 16
	}
	copy(dst[:], []byte(name)[:n])
}

// buildMachODWARF assembles a single-segment Mach-O with __DWARF and four sections.
func buildMachODWARF(secs DWARFSections) ([]byte, error) {
	// Prepare sections and payloads
	secNames := []string{"__debug_abbrev", "__debug_info", "__debug_line", "__debug_str"}
	payloads := [][]byte{secs.Abbrev, secs.Info, secs.Line, secs.Str}
	nsects := len(secNames)

	// Compute command sizes
	const headerSize = 32 + 32 // mach_header_64 is 32 bytes? In Mach-O it's 8*4 = 32 bytes; with Reserved is 8 more? Here struct is 64 bytes in our layout; use binary size
	// We will compute sizes using binary.Size on Go structs to avoid miscount
	mh := machHeader64{}
	seg := segmentCommand64{}
	sec := section64{}
	mhSize := uint32(binary.Size(mh))
	segSize := uint32(binary.Size(seg))
	secSize := uint32(binary.Size(sec))
	cmdsize := segSize + secSize*uint32(nsects)

	// Layout: [mach_header_64][segment_command_64 + n*section_64][section data]
	offset := mhSize + cmdsize
	// Align section offsets to 1 (no alignment) for simplicity; still keep 1-byte alignment field
	secOffsets := make([]uint32, nsects)
	secSizes := make([]uint32, nsects)
	cur := offset
	for i, p := range payloads {
		if len(p) == 0 {
			secOffsets[i] = 0
			secSizes[i] = 0
			continue
		}
		// 1-byte alignment, so no padding
		secOffsets[i] = cur
		secSizes[i] = uint32(len(p))
		cur += secSizes[i]
	}

	buf := &bytes.Buffer{}
	buf.Grow(int(cur))

	// Write header
	mh = machHeader64{
		Magic:      MH_MAGIC_64,
		CpuType:    CPU_TYPE_X86_64,
		CpuSubtype: CPU_SUBTYPE_X86_64_ALL,
		FileType:   MH_OBJECT,
		NCmds:      1,
		SizeOfCmds: cmdsize,
		Flags:      0,
		Reserved:   0,
	}
	if err := binary.Write(buf, binary.LittleEndian, mh); err != nil {
		return nil, err
	}

	// Segment command for __DWARF
	seg = segmentCommand64{
		Cmd:      LC_SEGMENT_64,
		Cmdsize:  cmdsize,
		Vmaddr:   0,
		Vmsize:   0,
		Fileoff:  uint64(offset),
		Filesize: uint64(cur - offset),
		Maxprot:  7,
		Initprot: 7,
		Nsects:   uint32(nsects),
		Flags:    0,
	}
	setPaddedName(&seg.Segname, "__DWARF")
	if err := binary.Write(buf, binary.LittleEndian, seg); err != nil {
		return nil, err
	}

	// Section headers
	for i := 0; i < nsects; i++ {
		s := section64{}
		setPaddedName(&s.Sectname, secNames[i])
		setPaddedName(&s.Segname, "__DWARF")
		s.Addr = 0
		s.Size = uint64(secSizes[i])
		s.Offset = secOffsets[i]
		s.Align = 0 // 2^0 = 1-byte alignment
		s.Reloff = 0
		s.Nreloc = 0
		s.Flags = 0
		if err := binary.Write(buf, binary.LittleEndian, s); err != nil {
			return nil, err
		}
	}

	// Payloads
	for i, p := range payloads {
		if len(p) == 0 {
			continue
		}
		// Ensure we are at the expected offset
		if uint32(buf.Len()) < secOffsets[i] {
			// pad if needed
			for uint32(buf.Len()) < secOffsets[i] {
				buf.WriteByte(0)
			}
		}
		buf.Write(p)
	}
	return buf.Bytes(), nil
}
