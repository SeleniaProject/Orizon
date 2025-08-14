package gdbserver

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	dbg "github.com/orizon-lang/orizon/internal/debug"
)

// Server implements a minimal GDB RSP server over TCP/pipe for pseudo-PC execution
// backed by ProgramDebugInfo and PCMap.
type Server struct {
	info  dbg.ProgramDebugInfo
	pcmap *dbg.PCMap
	pc    uint64
	bp    map[uint64]bool
	mu    sync.Mutex
	// When true, do not send acknowledgements ('+') for received packets
	noAck bool
	// Simple memory backing store for RSP 'm'/'M' commands
	mem map[uint64]byte
	// Flat register file: [0]=pc, [1..16]=r0..r15 (64-bit each)
	regs [17]uint64
	// When true, return T-stop replies instead of S-stop
	useT bool
}

// NewServer creates a new RSP server bound to the provided debug info.
func NewServer(info dbg.ProgramDebugInfo) *Server {
	s := &Server{info: info, pcmap: dbg.BuildPCMap(info), pc: 0, bp: make(map[uint64]bool), mem: make(map[uint64]byte)}
	// Initialize registers to zero; mirror pc into regs[0]
	s.regs[0] = 0
	return s
}

// HandleConn serves a single RSP session over conn.
func (s *Server) HandleConn(conn net.Conn) error {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		pkt, err := readPacket(r)
		if err != nil {
			return err
		}
		// Send ACK '+' per RSP (skip if no-ack mode is enabled)
		s.mu.Lock()
		noAck := s.noAck
		s.mu.Unlock()
		if !noAck {
			_, _ = conn.Write([]byte("+"))
		}
		resp := s.dispatch(pkt)
		if err := writePacket(conn, resp); err != nil {
			return err
		}
	}
}

func (s *Server) dispatch(cmd string) string {
	// Queries
	switch {
	case cmd == "?":
		return s.makeStopReply()
	case strings.HasPrefix(cmd, "qSupported"):
		// Advertise minimal features and no-ack support, and xfers
		return "PacketSize=4000;QStartNoAckMode+;qXfer:features:read+;qXfer:libraries:read+;qXfer:memory-map:read+;qXfer:auxv:read+;qSymbol+"
	case strings.HasPrefix(cmd, "QStartNoAckMode"):
		s.mu.Lock()
		s.noAck = true
		s.mu.Unlock()
		return "OK"
	case strings.HasPrefix(cmd, "qAttached"):
		return "1"
	case strings.HasPrefix(cmd, "qOffsets"):
		return "Text=0;Data=0;Bss=0"
	case strings.HasPrefix(cmd, "qXfer:features:read:target.xml:"):
		// Serve a static target.xml describing a pseudo 64-bit register set
		return s.handleQXferFeatures(cmd)
	case strings.HasPrefix(cmd, "qXfer:libraries:read:"):
		// Serve a static library list (no real shared libs; include main image only)
		return s.handleQXferLibraries(cmd)
	case strings.HasPrefix(cmd, "qXfer:memory-map:read:"):
		return s.handleQXferMemoryMap(cmd)
	case strings.HasPrefix(cmd, "qXfer:auxv:read:"):
		return s.handleQXferAuxv(cmd)
	case strings.HasPrefix(cmd, "qSymbol"):
		// Accept symbol lookups but request none
		return "OK"
	case strings.HasPrefix(cmd, "qC"):
		// Report current thread id (single-thread model)
		return "QC1"
	case strings.HasPrefix(cmd, "qfThreadInfo"):
		// First thread list packet
		return "m1"
	case strings.HasPrefix(cmd, "qsThreadInfo"):
		// End of thread list
		return "l"
	case strings.HasPrefix(cmd, "H"):
		return "OK"
	case strings.HasPrefix(cmd, "T"):
		// Thread-alive query always true for our single thread
		return "OK"
	case cmd == "g":
		return s.handleReadAllRegisters()
	case strings.HasPrefix(cmd, "G"):
		return s.handleWriteAllRegisters(cmd)
	case strings.HasPrefix(cmd, "p"):
		return s.handleReadRegister(cmd)
	case strings.HasPrefix(cmd, "P"):
		return s.handleWriteRegister(cmd)
	case strings.HasPrefix(cmd, "m"):
		return s.handleReadMemory(cmd)
	case strings.HasPrefix(cmd, "M"):
		return s.handleWriteMemory(cmd)
	case strings.HasPrefix(cmd, "vCont?"):
		// Advertise continue and single-step support
		return "vCont;c;s"
	case strings.HasPrefix(cmd, "vCont;c"):
		return s.cont()
	case strings.HasPrefix(cmd, "vCont;s"):
		return s.step()
	case strings.HasPrefix(cmd, "Z0,"):
		addr := parseAddrFromTriplet(cmd)
		s.mu.Lock()
		s.bp[addr] = true
		s.mu.Unlock()
		return "OK"
	case strings.HasPrefix(cmd, "Z1,"):
		return "OK"
	case strings.HasPrefix(cmd, "Z2,"):
		return "OK"
	case strings.HasPrefix(cmd, "Z3,"):
		return "OK"
	case strings.HasPrefix(cmd, "Z4,"):
		return "OK"
	case strings.HasPrefix(cmd, "z0,"):
		addr := parseAddrFromTriplet(cmd)
		s.mu.Lock()
		delete(s.bp, addr)
		s.mu.Unlock()
		return "OK"
	case strings.HasPrefix(cmd, "z1,"):
		return "OK"
	case strings.HasPrefix(cmd, "z2,"):
		return "OK"
	case strings.HasPrefix(cmd, "z3,"):
		return "OK"
	case strings.HasPrefix(cmd, "z4,"):
		return "OK"
	case cmd == "c":
		return s.cont()
	case strings.HasPrefix(cmd, "c"):
		// Optional address form: cADDR
		if len(cmd) > 1 {
			if addr, err := strconv.ParseUint(cmd[1:], 16, 64); err == nil {
				s.mu.Lock()
				s.pc = addr
				s.regs[0] = addr
				s.mu.Unlock()
			}
		}
		return s.cont()
	case cmd == "s":
		return s.step()
	case strings.HasPrefix(cmd, "s"):
		// Optional address form: sADDR
		if len(cmd) > 1 {
			if addr, err := strconv.ParseUint(cmd[1:], 16, 64); err == nil {
				s.mu.Lock()
				s.pc = addr
				s.regs[0] = addr
				s.mu.Unlock()
			}
		}
		return s.step()
	case cmd == "D":
		// Detach
		return "OK"
	case cmd == "k":
		// Kill request
		return "OK"
	case strings.HasPrefix(cmd, "vMustReplyEmpty"):
		return ""
	default:
		return ""
	}
}

// handleQXferFeatures serves a static target.xml via qXfer semantics.
// Format: qXfer:features:read:target.xml:OFFSET,LENGTH
func (s *Server) handleQXferFeatures(cmd string) string {
	const targetXML = "<?xml version=\"1.0\"?>" +
		"<target version=\"1.0\">" +
		"<architecture>orizon-pseudo</architecture>" +
		"<feature name=\"org.orizon.pseudo\">" +
		"<reg name=\"pc\" bitsize=\"64\" type=\"code_ptr\"/>" +
		"<reg name=\"r0\" bitsize=\"64\"/>" +
		"<reg name=\"r1\" bitsize=\"64\"/>" +
		"<reg name=\"r2\" bitsize=\"64\"/>" +
		"<reg name=\"r3\" bitsize=\"64\"/>" +
		"<reg name=\"r4\" bitsize=\"64\"/>" +
		"<reg name=\"r5\" bitsize=\"64\"/>" +
		"<reg name=\"r6\" bitsize=\"64\"/>" +
		"<reg name=\"r7\" bitsize=\"64\"/>" +
		"<reg name=\"r8\" bitsize=\"64\"/>" +
		"<reg name=\"r9\" bitsize=\"64\"/>" +
		"<reg name=\"r10\" bitsize=\"64\"/>" +
		"<reg name=\"r11\" bitsize=\"64\"/>" +
		"<reg name=\"r12\" bitsize=\"64\"/>" +
		"<reg name=\"r13\" bitsize=\"64\"/>" +
		"<reg name=\"r14\" bitsize=\"64\"/>" +
		"<reg name=\"r15\" bitsize=\"64\"/>" +
		"</feature></target>"
	// Extract offset,length
	// cmd example: qXfer:features:read:target.xml:0,fff
	lastColon := strings.LastIndex(cmd, ":")
	if lastColon < 0 || lastColon+1 >= len(cmd) {
		return "E01"
	}
	offLen := cmd[lastColon+1:]
	parts := strings.SplitN(offLen, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}
	off, err1 := strconv.ParseUint(parts[0], 16, 64)
	ln, err2 := strconv.ParseUint(parts[1], 16, 64)
	if err1 != nil || err2 != nil {
		return "E01"
	}
	data := []byte(targetXML)
	if off >= uint64(len(data)) {
		return "l" // no more data
	}
	end := off + ln
	if end > uint64(len(data)) {
		end = uint64(len(data))
	}
	chunk := data[off:end]
	marker := byte('m')
	if end == uint64(len(data)) {
		marker = 'l'
	}
	return string(marker) + hexEncode(chunk)
}

// handleQXferLibraries returns a minimal static XML list for libraries.
// Format: qXfer:libraries:read::OFFSET,LENGTH
func (s *Server) handleQXferLibraries(cmd string) string {
	const libsXML = "<?xml version=\"1.0\"?>" +
		"<library-list version=\"1.0\">" +
		"<library name=\"/proc/self/exe\"/>" +
		"</library-list>"
	lastColon := strings.LastIndex(cmd, ":")
	if lastColon < 0 || lastColon+1 >= len(cmd) {
		return "E01"
	}
	offLen := cmd[lastColon+1:]
	parts := strings.SplitN(offLen, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}
	off, err1 := strconv.ParseUint(parts[0], 16, 64)
	ln, err2 := strconv.ParseUint(parts[1], 16, 64)
	if err1 != nil || err2 != nil {
		return "E01"
	}
	data := []byte(libsXML)
	if off >= uint64(len(data)) {
		return "l"
	}
	end := off + ln
	if end > uint64(len(data)) {
		end = uint64(len(data))
	}
	chunk := data[off:end]
	marker := byte('m')
	if end == uint64(len(data)) {
		marker = 'l'
	}
	return string(marker) + hexEncode(chunk)
}

// handleQXferMemoryMap returns a minimal memory-map XML (single region)
func (s *Server) handleQXferMemoryMap(cmd string) string {
	const memXML = "<?xml version=\"1.0\"?>" +
		"<memory-map>" +
		"<memory type=\"ram\" start=\"0x0\" length=\"0x100000\"/>" +
		"</memory-map>"
	lastColon := strings.LastIndex(cmd, ":")
	if lastColon < 0 || lastColon+1 >= len(cmd) {
		return "E01"
	}
	offLen := cmd[lastColon+1:]
	parts := strings.SplitN(offLen, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}
	off, err1 := strconv.ParseUint(parts[0], 16, 64)
	ln, err2 := strconv.ParseUint(parts[1], 16, 64)
	if err1 != nil || err2 != nil {
		return "E01"
	}
	data := []byte(memXML)
	if off >= uint64(len(data)) {
		return "l"
	}
	end := off + ln
	if end > uint64(len(data)) {
		end = uint64(len(data))
	}
	chunk := data[off:end]
	marker := byte('m')
	if end == uint64(len(data)) {
		marker = 'l'
	}
	return string(marker) + hexEncode(chunk)
}

// handleQXferAuxv returns empty auxv for now
func (s *Server) handleQXferAuxv(cmd string) string {
	// Empty content
	lastColon := strings.LastIndex(cmd, ":")
	if lastColon < 0 || lastColon+1 >= len(cmd) {
		return "E01"
	}
	offLen := cmd[lastColon+1:]
	parts := strings.SplitN(offLen, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}
	off, err1 := strconv.ParseUint(parts[0], 16, 64)
	ln, err2 := strconv.ParseUint(parts[1], 16, 64)
	if err1 != nil || err2 != nil {
		return "E01"
	}
	_ = ln
	if off == 0 {
		return "l"
	}
	return "l"
}

// handleReadMemory implements 'm addr,length'
func (s *Server) handleReadMemory(cmd string) string {
	// mADDR,LENGTH
	body := cmd[1:]
	parts := strings.SplitN(body, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}
	addr, err1 := strconv.ParseUint(parts[0], 16, 64)
	n, err2 := strconv.ParseUint(parts[1], 16, 64)
	if err1 != nil || err2 != nil {
		return "E01"
	}
	buf := make([]byte, n)
	s.mu.Lock()
	for i := uint64(0); i < n; i++ {
		if v, ok := s.mem[addr+i]; ok {
			buf[i] = v
		} else {
			buf[i] = 0
		}
	}
	s.mu.Unlock()
	return hexEncode(buf)
}

// handleWriteMemory implements 'M addr,length:hexdata'
func (s *Server) handleWriteMemory(cmd string) string {
	// Strip prefix 'M'
	body := cmd[1:]
	// Split at ':' to separate address/length and data
	idx := strings.IndexByte(body, ':')
	if idx < 0 {
		return "E01"
	}
	hdr := body[:idx]
	dataHex := body[idx+1:]
	parts := strings.SplitN(hdr, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}
	addr, err1 := strconv.ParseUint(parts[0], 16, 64)
	n, err2 := strconv.ParseUint(parts[1], 16, 64)
	if err1 != nil || err2 != nil {
		return "E01"
	}
	data, err := hex.DecodeString(dataHex)
	if err != nil || uint64(len(data)) != n {
		return "E02"
	}
	s.mu.Lock()
	for i := uint64(0); i < n; i++ {
		s.mem[addr+i] = data[i]
	}
	s.mu.Unlock()
	return "OK"
}

func (s *Server) cont() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// advance until next breakpoint or range end
	for {
		if s.bp[s.pc] {
			return s.makeStopReplyLocked()
		}
		// single step
		if !s.stepLocked() {
			return s.makeStopReplyLocked()
		}
		// propagate pc into regs after each step
		s.regs[0] = s.pc
	}
}

func (s *Server) step() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stepLocked()
	// Keep the pseudo-pc register in sync
	s.regs[0] = s.pc
	return s.makeStopReplyLocked()
}

func (s *Server) stepLocked() bool {
	// Size policy: 4 bytes per line, bounded by owning range
	file, _, ok := s.pcmap.AddrToLine(s.pc)
	if !ok {
		// at end or invalid; try to move to next known range start
		for _, r := range s.pcmap.Ranges {
			if r.Low > s.pc {
				s.pc = r.Low
				s.regs[0] = s.pc
				return true
			}
		}
		return false
	}
	_ = file
	// Increment within range
	for _, r := range s.pcmap.Ranges {
		if s.pc >= r.Low && s.pc < r.High {
			nxt := s.pc + 4
			if nxt >= r.High {
				// move to next range start if exists
				moved := false
				for _, rr := range s.pcmap.Ranges {
					if rr.Low > r.Low {
						s.pc = rr.Low
						moved = true
						break
					}
				}
				if !moved {
					s.pc = r.High
				}
			} else {
				s.pc = nxt
			}
			s.regs[0] = s.pc
			return true
		}
	}
	return false
}

// makeStopReply returns a stop reply packet, either Sxx or Txx with thread and pc
func (s *Server) makeStopReply() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.makeStopReplyLocked()
}

func (s *Server) makeStopReplyLocked() string {
	if !s.useT {
		return "S05"
	}
	// T05;thread:1;pc:<hex>;
	// Encode pc as 8-byte little-endian hex
	v := s.pc
	buf := []byte{
		byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24),
		byte(v >> 32), byte(v >> 40), byte(v >> 48), byte(v >> 56),
	}
	return "T05;thread:1;pc:" + hexEncode(buf) + ";"
}

// readPacket reads an RSP packet and sends ACK '+'
func readPacket(r *bufio.Reader) (string, error) {
	// wait for '$'
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '$' {
			break
		}
	}
	data := make([]byte, 0, 256)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '#' {
			break
		}
		data = append(data, b)
	}
	// read checksum (2 hex)
	csum := make([]byte, 2)
	if _, err := r.Read(csum); err != nil {
		return "", err
	}
	// ignore checksum; write ACK '+'
	// Note: caller writes on same conn after returning
	return string(data), nil
}

func writePacket(w net.Conn, payload string) error {
	sum := byte(0)
	for i := 0; i < len(payload); i++ {
		sum += payload[i]
	}
	pkt := fmt.Sprintf("$%s#%02x", payload, sum)
	_, err := w.Write([]byte(pkt))
	return err
}

// handleReadAllRegisters returns hex for all registers (little-endian 64-bit each)
func (s *Server) handleReadAllRegisters() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Sync pc
	s.regs[0] = s.pc
	// Build byte slice
	buf := make([]byte, 0, len(s.regs)*8)
	for _, v := range s.regs {
		var tmp [8]byte
		// little-endian
		tmp[0] = byte(v)
		tmp[1] = byte(v >> 8)
		tmp[2] = byte(v >> 16)
		tmp[3] = byte(v >> 24)
		tmp[4] = byte(v >> 32)
		tmp[5] = byte(v >> 40)
		tmp[6] = byte(v >> 48)
		tmp[7] = byte(v >> 56)
		buf = append(buf, tmp[:]...)
	}
	return hexEncode(buf)
}

// handleWriteAllRegisters parses hex data for all registers
func (s *Server) handleWriteAllRegisters(cmd string) string {
	// Gxxxxxxxx (hex for all regs); spec uses raw hex after 'G'
	dataHex := cmd[1:]
	data, err := hex.DecodeString(dataHex)
	if err != nil {
		return "E01"
	}
	if len(data) != len(s.regs)*8 {
		return "E02"
	}
	s.mu.Lock()
	for i := 0; i < len(s.regs); i++ {
		off := i * 8
		v := uint64(data[off]) |
			uint64(data[off+1])<<8 |
			uint64(data[off+2])<<16 |
			uint64(data[off+3])<<24 |
			uint64(data[off+4])<<32 |
			uint64(data[off+5])<<40 |
			uint64(data[off+6])<<48 |
			uint64(data[off+7])<<56
		s.regs[i] = v
	}
	// Keep pc in sync
	s.pc = s.regs[0]
	s.mu.Unlock()
	return "OK"
}

// handleReadRegister returns the hex-encoded 64-bit value of a single register
func (s *Server) handleReadRegister(cmd string) string {
	// pNN where NN is hex register index
	idxHex := cmd[1:]
	idx, err := strconv.ParseUint(idxHex, 16, 64)
	if err != nil || idx >= uint64(len(s.regs)) {
		// Return zero value for unknown
		return strings.Repeat("0", 16)
	}
	s.mu.Lock()
	// Sync pc
	s.regs[0] = s.pc
	v := s.regs[idx]
	s.mu.Unlock()
	var tmp [8]byte
	tmp[0] = byte(v)
	tmp[1] = byte(v >> 8)
	tmp[2] = byte(v >> 16)
	tmp[3] = byte(v >> 24)
	tmp[4] = byte(v >> 32)
	tmp[5] = byte(v >> 40)
	tmp[6] = byte(v >> 48)
	tmp[7] = byte(v >> 56)
	return hexEncode(tmp[:])
}

// handleWriteRegister sets a single 64-bit register from hex
func (s *Server) handleWriteRegister(cmd string) string {
	// PNN=hex
	body := cmd[1:]
	if i := strings.IndexByte(body, '='); i >= 0 {
		idxHex := body[:i]
		valHex := body[i+1:]
		idx, err := strconv.ParseUint(idxHex, 16, 64)
		if err != nil || idx >= uint64(len(s.regs)) {
			return "E01"
		}
		data, err := hex.DecodeString(valHex)
		if err != nil || len(data) != 8 {
			return "E02"
		}
		v := uint64(data[0]) |
			uint64(data[1])<<8 |
			uint64(data[2])<<16 |
			uint64(data[3])<<24 |
			uint64(data[4])<<32 |
			uint64(data[5])<<40 |
			uint64(data[6])<<48 |
			uint64(data[7])<<56
		s.mu.Lock()
		s.regs[idx] = v
		if idx == 0 {
			s.pc = v
		}
		s.mu.Unlock()
		return "OK"
	}
	return "E01"
}

func parseAddrFromTriplet(cmd string) uint64 {
	// Z0,<addr>,<kind>
	parts := strings.Split(cmd, ",")
	if len(parts) < 2 {
		return 0
	}
	a := parts[1]
	// addr is hex
	v, _ := strconv.ParseUint(a, 16, 64)
	return v
}

// hexEncode returns hex string for a byte slice
func hexEncode(b []byte) string { return hex.EncodeToString(b) }
