package gdbserver

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	dbg "github.com/orizon-lang/orizon/internal/debug"
)

// Server implements a minimal GDB RSP server over TCP/pipe for pseudo-PC execution
// backed by ProgramDebugInfo and PCMap.
type Server struct {
	rwatch map[uint64]bool
	pcmap  *dbg.PCMap
	bp     map[uint64]bool
	hwbp   map[uint64]bool
	wwatch map[uint64]bool
	awatch map[uint64]bool
	mem    map[uint64]byte
	info   dbg.ProgramDebugInfo
	regs   [17]uint64
	pc     uint64
	mu     sync.Mutex
	noAck  bool
	useT   bool
}

// ActorsJSONProvider, when non-nil, supplies a JSON snapshot of actors for qXfer:actors:read.
// The format is application-defined JSON that GDB clients or tools can consume.
var ActorsJSONProvider func() []byte

// ActorMessagesJSONProvider, when non-nil, supplies JSON for recent messages of a given actor.
// Expected to return a compact JSON (array or object). If nil, qXfer:actors-messages:read falls back
// to ActorsJSONProvider output to keep compatibility with older providers.
var ActorMessagesJSONProvider func(actorID uint64, n int) []byte

// ActorsGraphJSONProvider, when non-nil, supplies a JSON graph of the actor system for qXfer:actors-graph:read.
// The JSON should typically include nodes and edges with minimal fields consumable by UI tools.
var ActorsGraphJSONProvider func() []byte

// ActorsGraphJSONProviderEx, when non-nil, can consume optional filters from annex.
// Recognized parameters are implementation-defined (e.g., root, limit, pattern).
var ActorsGraphJSONProviderEx func(params map[string]string) []byte

// DeadlocksJSONProvider, when non-nil, supplies a JSON array/object describing potential deadlocks for qXfer:deadlocks:read.
// This is implementation-defined but should be stable for tooling consumption.
var DeadlocksJSONProvider func() []byte

// DeadlocksJSONProviderEx allows filtered deadlock reports using annex parameters.
var DeadlocksJSONProviderEx func(params map[string]string) []byte

// CorrelationJSONProvider, when non-nil, supplies JSON for events by a correlation id.
// The provider should accept (id, n) and return a JSON array or object chunk.
var CorrelationJSONProvider func(id string, n int) []byte

// CorrelationJSONProviderEx supports filtered correlation queries via annex parameter map.
var CorrelationJSONProviderEx func(params map[string]string) []byte

// LocalsJSONProvider, when non-nil, provides a JSON array/object describing current function locals
// at the server's pc. This demo provider is pluggable so the embedding runtime can compute real values.
var LocalsJSONProvider func(pc uint64) []byte

// PrettyLocalsJSONProvider optionally provides evaluated local variable values (typed) for current pc.
// When nil, the server will try to compute minimal pretty values from its memory map and registers.
var PrettyLocalsJSONProvider func(pc uint64, fp uint64) []byte

// NewServer creates a new RSP server bound to the provided debug info.
func NewServer(info dbg.ProgramDebugInfo) *Server {
	s := &Server{
		info:   info,
		pcmap:  dbg.BuildPCMap(info),
		pc:     0,
		bp:     make(map[uint64]bool),
		hwbp:   make(map[uint64]bool),
		wwatch: make(map[uint64]bool),
		rwatch: make(map[uint64]bool),
		awatch: make(map[uint64]bool),
		mem:    make(map[uint64]byte),
	}
	// Initialize registers to zero; mirror pc into regs[0].
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
		// Send ACK '+' per RSP (skip if no-ack mode is enabled).
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
	// Queries.
	switch {
	case cmd == "?":
		return s.makeStopReply()
	case strings.HasPrefix(cmd, "qSupported"):
		// Advertise minimal features and no-ack support, and xfers (include custom xfers).
		// Also declare software and hardware breakpoint support.
		return "PacketSize=4000;QStartNoAckMode+;swbreak+;hwbreak+;qXfer:features:read+;qXfer:libraries:read+;qXfer:memory-map:read+;qXfer:auxv:read+;qXfer:stack:read+;qXfer:locals:read+;qXfer:pretty-locals:read+;qXfer:pretty-memory:read+;qXfer:actors:read+;qXfer:actors-messages:read+;qXfer:actors-graph:read+;qXfer:deadlocks:read+;qXfer:correlation:read+;qSymbol+"
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
		// Serve a static library list (no real shared libs; include main image only).
		return s.handleQXferLibraries(cmd)
	case strings.HasPrefix(cmd, "qXfer:memory-map:read:"):
		return s.handleQXferMemoryMap(cmd)
	case strings.HasPrefix(cmd, "qXfer:auxv:read:"):
		return s.handleQXferAuxv(cmd)
	case strings.HasPrefix(cmd, "qXfer:stack:read:"):
		return s.handleQXferStack(cmd)
	case strings.HasPrefix(cmd, "qXfer:actors:read:"):
		return s.handleQXferActors(cmd)
	case strings.HasPrefix(cmd, "qXfer:actors-messages:read:"):
		return s.handleQXferActorsMessages(cmd)
	case strings.HasPrefix(cmd, "qXfer:actors-graph:read:"):
		return s.handleQXferActorsGraph(cmd)
	case strings.HasPrefix(cmd, "qXfer:deadlocks:read:"):
		return s.handleQXferDeadlocks(cmd)
	case strings.HasPrefix(cmd, "qXfer:correlation:read:"):
		return s.handleQXferCorrelation(cmd)
	case strings.HasPrefix(cmd, "qXfer:locals:read:"):
		return s.handleQXferLocals(cmd)
	case strings.HasPrefix(cmd, "qXfer:pretty-locals:read:"):
		return s.handleQXferPrettyLocals(cmd)
	case strings.HasPrefix(cmd, "qXfer:pretty-memory:read:"):
		return s.handleQXferPrettyMemory(cmd)
	case strings.HasPrefix(cmd, "qSymbol"):
		// Accept symbol lookups but request none.
		return "OK"
	case strings.HasPrefix(cmd, "qC"):
		// Report current thread id (single-thread model).
		return "QC1"
	case strings.HasPrefix(cmd, "QThreadSuffixSupported"):
		// Acknowledge thread suffix support (harmless in single-thread model).
		return "OK"
	case strings.HasPrefix(cmd, "qHostInfo"):
		// Minimal host info for LLDB clients.
		return "triple:orizon-unknown-unknown;endian:little;ptrsize:8;"
	case strings.HasPrefix(cmd, "qRegisterInfo"):
		return s.handleQRegisterInfo(cmd)
	case strings.HasPrefix(cmd, "qThreadExtraInfo"):
		// hex-encoded string.
		msg := []byte("main thread")

		return hexEncode(msg)
	case strings.HasPrefix(cmd, "qMemoryRegionInfo:"):
		// Reply with a single flat RAM region covering our pseudo memory.
		// Format: start:hex;size:hex;permissions:rw;.
		return "start:0;size:100000;permissions:rw;"
	case strings.HasPrefix(cmd, "qfThreadInfo"):
		// First thread list packet.
		return "m1"
	case strings.HasPrefix(cmd, "qsThreadInfo"):
		// End of thread list.
		return "l"
	case strings.HasPrefix(cmd, "H"):
		return "OK"
	case strings.HasPrefix(cmd, "T"):
		// Thread-alive query always true for our single thread.
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
		// Advertise continue and single-step support.
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
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		s.hwbp[addr] = true
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "Z2,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		s.wwatch[addr] = true
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "Z3,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		s.rwatch[addr] = true
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "Z4,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		s.awatch[addr] = true
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "z0,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		delete(s.bp, addr)
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "z1,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		delete(s.hwbp, addr)
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "z2,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		delete(s.wwatch, addr)
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "z3,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		delete(s.rwatch, addr)
		s.mu.Unlock()

		return "OK"
	case strings.HasPrefix(cmd, "z4,"):
		addr := parseAddrFromTriplet(cmd)

		s.mu.Lock()
		delete(s.awatch, addr)
		s.mu.Unlock()

		return "OK"
	case cmd == "c":
		return s.cont()
	case strings.HasPrefix(cmd, "c"):
		// Optional address form: cADDR.
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
		// Optional address form: sADDR.
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
		// Detach.
		return "OK"
	case cmd == "k":
		// Kill request.
		return "OK"
	case strings.HasPrefix(cmd, "vMustReplyEmpty"):
		return ""
	default:
		return ""
	}
}

// handleQRegisterInfo returns metadata for a register index as used by LLDB's remote protocol.
// Example queries: qRegisterInfo0, qRegisterInfo1, ...
func (s *Server) handleQRegisterInfo(cmd string) string {
	// Extract numeric suffix after "qRegisterInfo".
	idxStr := strings.TrimPrefix(cmd, "qRegisterInfo")
	if idxStr == "" {
		return "E01"
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(s.regs) {
		return "E01"
	}
	// Build response fields.
	// Required: name,bitsize,encoding,format,set.
	name := "r" + strconv.Itoa(idx-1)
	generic := ""

	if idx == 0 {
		name = "pc"
		generic = ";generic:pc"
	}
	// 64-bit little-endian unsigned general register.
	resp := fmt.Sprintf("name:%s;bitsize:64;encoding:uint;format:hex;set:general%s;", name, generic)

	return resp
}

// Format: qXfer:features:read:target.xml:OFFSET,LENGTH.
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
	// Extract offset,length.
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
// Format: qXfer:libraries:read::OFFSET,LENGTH.
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

// handleQXferMemoryMap returns a minimal memory-map XML (single region).
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

// handleQXferAuxv returns empty auxv for now.
func (s *Server) handleQXferAuxv(cmd string) string {
	// Empty content.
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

// handleQXferActors streams a JSON snapshot of actors if a provider is registered.
func (s *Server) handleQXferActors(cmd string) string {
	if ActorsJSONProvider == nil {
		return "l"
	}

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

	data := ActorsJSONProvider()
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

// handleQXferActorsMessages streams a JSON array of recent messages for a given actor id.
// The provider function must be registered by the embedding application. The JSON format
// is implementation-defined but should be consistent with /actors/messages endpoint.
func (s *Server) handleQXferActorsMessages(cmd string) string {
	// Two forms supported:.
	// 1) qXfer:actors-messages:read::<hexOffset>,<hexLength>.
	//    (provider must internally choose a sensible default actor or last N global events).
	// 2) qXfer:actors-messages:read:actor=<decID>,n=<decN>:<hexOffset>,<hexLength>.
	//    (annex carries actor and count hints before the final colon).
	// Parse tail offset,length.
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
	// Extract optional annex between the two last colons.
	// qXfer:actors-messages:read:<annex>:off,len.
	var annex string
	// Find second last colon.
	rest := cmd[:lastColon]
	secondLast := strings.LastIndex(rest, ":")

	if secondLast >= 0 && secondLast+1 < len(rest) {
		annex = rest[secondLast+1:]
	}

	var data []byte

	if ActorMessagesJSONProvider != nil {
		// Parse annex for actor and n.
		var actorID uint64

		n := 100

		if annex != "" {
			// Format: actor=123,n=50 or any order.
			items := strings.Split(annex, ",")
			for _, it := range items {
				if strings.HasPrefix(it, "actor=") {
					v := strings.TrimPrefix(it, "actor=")
					if id, err := strconv.ParseUint(v, 10, 64); err == nil {
						actorID = id
					}
				} else if strings.HasPrefix(it, "n=") {
					v := strings.TrimPrefix(it, "n=")
					if nn, err := strconv.Atoi(v); err == nil && nn > 0 {
						n = nn
					}
				}
			}
		}

		data = ActorMessagesJSONProvider(actorID, n)
	} else if ActorsJSONProvider != nil {
		data = ActorsJSONProvider()
	} else {
		return "l"
	}

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

// handleQXferActorsGraph streams a JSON representation of the actor graph if a provider is registered.
func (s *Server) handleQXferActorsGraph(cmd string) string {
	if ActorsGraphJSONProvider == nil && ActorsGraphJSONProviderEx == nil {
		return "l"
	}

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
	// Extract optional annex between colons: qXfer:actors-graph:read:<annex>:off,len.
	rest := cmd[:lastColon]
	secondLast := strings.LastIndex(rest, ":")
	annex := ""

	if secondLast >= 0 && secondLast+1 < len(rest) {
		annex = rest[secondLast+1:]
	}

	var data []byte

	if annex != "" && ActorsGraphJSONProviderEx != nil {
		params := make(map[string]string)

		for _, kv := range strings.Split(annex, ",") {
			if kv == "" {
				continue
			}

			if i := strings.IndexByte(kv, '='); i >= 0 {
				k := strings.TrimSpace(kv[:i])
				v := strings.TrimSpace(kv[i+1:])
				params[k] = v
			}
		}

		data = ActorsGraphJSONProviderEx(params)
	}

	if len(data) == 0 && ActorsGraphJSONProvider != nil {
		data = ActorsGraphJSONProvider()
	}

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

// handleQXferDeadlocks streams a JSON representation of potential deadlocks if a provider is registered.
func (s *Server) handleQXferDeadlocks(cmd string) string {
	if DeadlocksJSONProvider == nil && DeadlocksJSONProviderEx == nil {
		return "l"
	}

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
	// Optional annex.
	rest := cmd[:lastColon]
	secondLast := strings.LastIndex(rest, ":")
	annex := ""

	if secondLast >= 0 && secondLast+1 < len(rest) {
		annex = rest[secondLast+1:]
	}

	var data []byte

	if annex != "" && DeadlocksJSONProviderEx != nil {
		params := make(map[string]string)

		for _, kv := range strings.Split(annex, ",") {
			if kv == "" {
				continue
			}

			if i := strings.IndexByte(kv, '='); i >= 0 {
				k := strings.TrimSpace(kv[:i])
				v := strings.TrimSpace(kv[i+1:])
				params[k] = v
			}
		}

		data = DeadlocksJSONProviderEx(params)
	}

	if len(data) == 0 && DeadlocksJSONProvider != nil {
		data = DeadlocksJSONProvider()
	}

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

// handleQXferLocals serves JSON-encoded locals for the current PC location.
func (s *Server) handleQXferLocals(cmd string) string {
	if LocalsJSONProvider == nil {
		// Fallback: synthesize locals from ProgramDebugInfo at current pc (names/types only)
		data := s.synthesizeLocals()

		return s.streamChunk(cmd, data)
	}
	// Provider path.
	return s.streamChunk(cmd, LocalsJSONProvider(s.pc))
}

// handleQXferPrettyLocals streams evaluated locals if a provider exists, falling back to synthesized.
// names/types without values when not available.
func (s *Server) handleQXferPrettyLocals(cmd string) string {
	if PrettyLocalsJSONProvider != nil {
		// fp is not modeled separately; use regs[1] (r0) as pseudo-frame or 0 when unknown.
		fp := s.regs[1]

		return s.streamChunk(cmd, PrettyLocalsJSONProvider(s.pc, fp))
	}
	// Fallback: synthesize with approximate values from memory when possible.
	data := s.synthesizePrettyLocals()

	return s.streamChunk(cmd, data)
}

// synthesizeLocals builds a minimal JSON array of variable descriptors at current pc.
func (s *Server) synthesizeLocals() []byte {
	// Determine function index by replicating PCMap construction ordering.
	// Iterate modules/functions in sorted order and accumulate sizes
	mods := make([]dbg.ModuleDebugInfo, len(s.info.Modules))
	copy(mods, s.info.Modules)
	// Sort by ModuleName.
	sort.Slice(mods, func(i, j int) bool { return mods[i].ModuleName < mods[j].ModuleName })

	pc := uint64(0)

	for _, md := range mods {
		fns := make([]dbg.FunctionInfo, len(md.Functions))
		copy(fns, md.Functions)
		sort.Slice(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })

		for _, fn := range fns {
			lines := len(fn.Lines)
			if lines == 0 {
				lines = 1
			}

			low := pc
			high := pc + uint64(lines*4)

			if s.pc >= low && s.pc < high {
				// Build locals array (names/types only)
				type local struct {
					Name string `json:"name"`
					Type string `json:"type"`
					Base string `json:"base,omitempty"`
					Off  int64  `json:"off,omitempty"`
				}

				out := make([]local, 0, len(fn.Variables))

				for _, v := range fn.Variables {
					off := int64(0)
					if v.FrameOffset != 0 {
						off = v.FrameOffset
					}

					base := v.AddressBase
					out = append(out, local{Name: v.Name, Type: chooseType(v), Base: base, Off: off})
				}

				b, _ := json.Marshal(out)

				return b
			}

			pc = high
		}
	}
	// default empty.
	return []byte("[]")
}

// synthesizePrettyLocals evaluates locals using AddressBase/FrameOffset and TypeMeta if available.
func (s *Server) synthesizePrettyLocals() []byte {
	type pretty struct {
		Value interface{} `json:"value,omitempty"`
		Name  string      `json:"name"`
		Type  string      `json:"type"`
		Addr  string      `json:"addr,omitempty"`
	}

	mods := make([]dbg.ModuleDebugInfo, len(s.info.Modules))
	copy(mods, s.info.Modules)
	sort.Slice(mods, func(i, j int) bool { return mods[i].ModuleName < mods[j].ModuleName })

	pc := uint64(0)

	for _, md := range mods {
		fns := make([]dbg.FunctionInfo, len(md.Functions))
		copy(fns, md.Functions)
		sort.Slice(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })

		for _, fn := range fns {
			lines := len(fn.Lines)
			if lines == 0 {
				lines = 1
			}

			low := pc
			high := pc + uint64(lines*4)

			if s.pc >= low && s.pc < high {
				out := make([]pretty, 0, len(fn.Variables))
				fb := s.frameBase()

				for _, v := range fn.Variables {
					// compute address when fbreg.
					var addr uint64

					if v.AddressBase == "fbreg" {
						if v.FrameOffset >= 0 {
							addr = fb + uint64(v.FrameOffset)
						} else {
							addr = fb - uint64(-v.FrameOffset)
						}
					}

					pv := pretty{Name: v.Name, Type: chooseType(v)}
					if addr != 0 {
						pv.Addr = fmt.Sprintf("0x%x", addr)
						pv.Value = s.readTypedValue(addr, v.TypeMeta, pv.Type)
					}

					out = append(out, pv)
				}

				b, _ := json.Marshal(out)

				return b
			}

			pc = high
		}
	}

	return []byte("[]")
}

func (s *Server) frameBase() uint64 {
	// Use regs[1] as pseudo frame pointer when available; fallback to stack top in mem if any.
	if s.regs[1] != 0 {
		return s.regs[1]
	}

	return 0
}

// readTypedValue decodes a minimal pretty value for common base kinds using memory.
func (s *Server) readTypedValue(addr uint64, tm *dbg.TypeMeta, typeStr string) interface{} {
	return s.readTypedValueDepth(addr, tm, typeStr, 0)
}

func (s *Server) readTypedValueDepth(addr uint64, tm *dbg.TypeMeta, typeStr string, depth int) interface{} {
	visited := make(map[uint64]bool)

	return s.readTypedValueDepthVis(addr, tm, typeStr, depth, visited)
}

func (s *Server) readTypedValueDepthVis(addr uint64, tm *dbg.TypeMeta, typeStr string, depth int, visited map[uint64]bool) interface{} {
	if depth > 3 {
		return "<max-depth>"
	}

	kind := ""
	if tm != nil && tm.Kind != "" {
		kind = tm.Kind
	}

	if kind == "" {
		kind = inferKindFromTypeString(typeStr)
	}

	switch kind {
	case "map":
		if tm == nil {
			return "<map>"
		}

		var key, val *dbg.TypeMeta
		if len(tm.Parameters) >= 2 {
			key = &tm.Parameters[0]
			val = &tm.Parameters[1]
		}
		// Try to discover storage fields heuristically.
		var dataOff, lenOff int64 = -1, -1

		for _, f := range tm.Fields {
			name := strings.ToLower(f.Name)
			if name == "data" || name == "entries" || name == "buckets" {
				dataOff = f.Offset
			} else if name == "len" || name == "length" || name == "count" || name == "size" {
				lenOff = f.Offset
			}
		}

		base := uint64(0)
		if dataOff >= 0 {
			base = s.readU64(addr + uint64(dataOff))
		} else {
			// Fallback: assume first word is pointer to entries.
			base = s.readU64(addr)
		}

		ln := 0
		if lenOff >= 0 {
			ln = int(s.readU32(addr + uint64(lenOff)))
		}

		if ln < 0 {
			ln = 0
		}

		if ln > 8 {
			ln = 8
		}

		align8 := func(sz int64) uint64 {
			if sz <= 0 {
				return 8
			}

			if sz%8 == 0 {
				return uint64(sz)
			}

			return uint64(((sz + 7) / 8) * 8)
		}
		kSz := uint64(8)
		vSz := uint64(8)

		if key != nil {
			kSz = align8(key.Size)
		}

		if val != nil {
			vSz = align8(val.Size)
		}

		stride := kSz + vSz
		pairs := make([]map[string]interface{}, 0, ln)

		for i := 0; i < ln; i++ {
			kAddr := base + uint64(i)*stride
			vAddr := kAddr + kSz

			kVal := interface{}(fmt.Sprintf("0x%x", s.readU64(kAddr)))
			if key != nil {
				kVal = s.readTypedValueDepthVis(kAddr, key, key.Name, depth+1, visited)
			}

			vVal := interface{}(fmt.Sprintf("0x%x", s.readU64(vAddr)))
			if val != nil {
				vVal = s.readTypedValueDepthVis(vAddr, val, val.Name, depth+1, visited)
			}

			pairs = append(pairs, map[string]interface{}{"key": kVal, "value": vVal})
		}

		return pairs
	case "string":
		// Treat as pointer to zero-terminated UTF-8 by convention.
		p := s.readU64(addr)

		return s.readCString(p, 256)
	case "slice":
		if tm == nil {
			return "<slice>"
		}

		var dataOff, lenOff int64

		var elem *dbg.TypeMeta
		if len(tm.Parameters) > 0 {
			elem = &tm.Parameters[0]
		}

		for _, f := range tm.Fields {
			switch f.Name {
			case "data":
				dataOff = f.Offset
			case "len":
				lenOff = f.Offset
			}
		}

		dataPtr := s.readU64(addr + uint64(dataOff))

		ln := int(s.readU32(addr + uint64(lenOff)))
		if ln < 0 {
			ln = 0
		}

		if ln > 64 {
			ln = 64
		}

		out := make([]interface{}, 0, ln)

		stride := uint64(8)
		if elem != nil && elem.Size > 0 {
			stride = uint64(elem.Size)
		}

		for i := 0; i < ln; i++ {
			if elem != nil {
				out = append(out, s.readTypedValueDepthVis(dataPtr+uint64(i)*stride, elem, elem.Name, depth+1, visited))
			} else {
				out = append(out, fmt.Sprintf("0x%x", s.readU64(dataPtr+uint64(i)*stride)))
			}
		}

		return out
	case "struct":
		if tm == nil {
			return "<struct>"
		}

		obj := make(map[string]interface{}, len(tm.Fields))
		for _, f := range tm.Fields {
			obj[f.Name] = s.readTypedValueDepthVis(addr+uint64(f.Offset), &f.Type, f.Type.Name, depth+1, visited)
		}

		return obj
	case "tuple":
		if tm == nil {
			return "<tuple>"
		}

		out := make([]interface{}, 0, len(tm.Fields))
		for _, f := range tm.Fields {
			out = append(out, s.readTypedValueDepthVis(addr+uint64(f.Offset), &f.Type, f.Type.Name, depth+1, visited))
		}

		return out
	case "interface":
		vptr := s.readU64(addr)
		data := s.readU64(addr + 8)

		return map[string]interface{}{"vptr": fmt.Sprintf("0x%x", vptr), "data": fmt.Sprintf("0x%x", data)}
	case "array":
		var elem *dbg.TypeMeta
		if tm != nil && len(tm.Parameters) > 0 {
			elem = &tm.Parameters[0]
		}

		count := 0
		if tm != nil && tm.Size > 0 && elem != nil && elem.Size > 0 {
			count = int(tm.Size / elem.Size)
		}

		if count <= 0 {
			count = 8
		}

		if count > 64 {
			count = 64
		}

		out := make([]interface{}, 0, count)

		stride := uint64(8)
		if elem != nil && elem.Size > 0 {
			stride = uint64(elem.Size)
		}

		for i := 0; i < count; i++ {
			if elem != nil {
				out = append(out, s.readTypedValueDepthVis(addr+uint64(i)*stride, elem, elem.Name, depth+1, visited))
			} else {
				out = append(out, fmt.Sprintf("0x%x", s.readU64(addr+uint64(i)*stride)))
			}
		}

		return out
	case "int", "Int", "i32", "i64":
		if tm != nil && tm.Size == 4 {
			return int64(int32(s.readU32(addr)))
		}

		return int64(int64(s.readU64(addr)))
	case "uint", "UInt32", "uint32", "u32":
		return uint64(s.readU32(addr))
	case "UInt64", "uint64", "u64":
		return s.readU64(addr)
	case "bool", "Bool":
		b := s.readU8(addr)

		return b != 0
	case "pointer", "ptr":
		p := s.readU64(addr)
		if p == 0 {
			return map[string]interface{}{"addr": "0x0", "deref": nil}
		}

		if visited[p] {
			return map[string]interface{}{"addr": fmt.Sprintf("0x%x", p), "deref": "<cycle>"}
		}

		visited[p] = true

		if tm != nil && len(tm.Parameters) > 0 {
			elem := &tm.Parameters[0]

			return map[string]interface{}{
				"addr":  fmt.Sprintf("0x%x", p),
				"deref": s.readTypedValueDepthVis(p, elem, elem.Name, depth+1, visited),
			}
		}

		return fmt.Sprintf("0x%x", p)
	case "float", "Float32", "float32":
		u := s.readU32(addr)

		return float32frombits(u)
	case "Float64", "float64":
		u := s.readU64(addr)

		return float64frombits(u)
	default:
		u := s.readU64(addr)

		return fmt.Sprintf("0x%x", u)
	}
}

// readCString reads at most max bytes starting at addr until NUL or missing memory.
func (s *Server) readCString(addr uint64, max int) string {
	if addr == 0 || max <= 0 {
		return ""
	}

	buf := make([]byte, 0, max)

	for i := 0; i < max; i++ {
		if b, ok := s.getMem(addr + uint64(i)); ok {
			if b == 0 {
				break
			}

			buf = append(buf, b)
		} else {
			break
		}
	}

	return string(buf)
}

func inferKindFromTypeString(ts string) string {
	ls := strings.ToLower(ts)
	switch ls {
	case "int32", "i32":
		return "i32"
	case "int64", "i64":
		return "i64"
	case "uint32", "u32":
		return "u32"
	case "uint64", "u64":
		return "u64"
	case "bool":
		return "bool"
	case "float32":
		return "float32"
	case "float64":
		return "float64"
	default:
		return ""
	}
}

func (s *Server) readU8(addr uint64) byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.mem[addr]
}

// getMem returns (value, present).
func (s *Server) getMem(addr uint64) (byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.mem[addr]

	return v, ok
}

func (s *Server) readU32(addr uint64) uint32 {
	var b [4]byte

	s.mu.Lock()
	for i := 0; i < 4; i++ {
		b[i] = s.mem[addr+uint64(i)]
	}
	s.mu.Unlock()

	return binary.LittleEndian.Uint32(b[:])
}

func (s *Server) readU64(addr uint64) uint64 {
	var b [8]byte

	s.mu.Lock()
	for i := 0; i < 8; i++ {
		b[i] = s.mem[addr+uint64(i)]
	}
	s.mu.Unlock()

	return binary.LittleEndian.Uint64(b[:])
}

func float32frombits(u uint32) float32 { return math.Float32frombits(u) }
func float64frombits(u uint64) float64 { return math.Float64frombits(u) }

// handleQXferPrettyMemory pretty-prints a memory range in hex with ASCII side and 16-byte rows.
func (s *Server) handleQXferPrettyMemory(cmd string) string {
	// Annex form: qXfer:pretty-memory:read:addr=<hex>,len=<hex>:<off>,<len>.
	// We ignore outer chunk off/len and directly render requested range, then chunk it.
	lastColon := strings.LastIndex(cmd, ":")
	if lastColon < 0 || lastColon+1 >= len(cmd) {
		return "E01"
	}
	// parse annex.
	rest := cmd[:lastColon]
	second := strings.LastIndex(rest, ":")
	annex := ""

	if second >= 0 && second+1 < len(rest) {
		annex = rest[second+1:]
	}

	var addr, total uint64

	for _, kv := range strings.Split(annex, ",") {
		if strings.HasPrefix(kv, "addr=") {
			v := strings.TrimPrefix(kv, "addr=")
			a, _ := strconv.ParseUint(v, 16, 64)
			addr = a
		} else if strings.HasPrefix(kv, "len=") {
			v := strings.TrimPrefix(kv, "len=")
			l, _ := strconv.ParseUint(v, 16, 64)
			total = l
		}
	}

	if total == 0 {
		total = 256
	}
	// build lines.
	var b strings.Builder

	var i uint64
	for i = 0; i < total; i += 16 {
		cur := addr + i
		fmt.Fprintf(&b, "%08x  ", cur)

		var ascii [16]byte

		for j := uint64(0); j < 16; j++ {
			if v, ok := s.getMem(cur + j); ok {
				fmt.Fprintf(&b, "%02x ", v)

				if v >= 32 && v <= 126 {
					ascii[j] = v
				} else {
					ascii[j] = '.'
				}
			} else {
				b.WriteString("?? ")

				ascii[j] = '.'
			}
		}

		fmt.Fprintf(&b, " |%s|\n", string(ascii[:]))
	}

	rendered := []byte(b.String())
	// chunk with tail off,len.
	tail := cmd[lastColon+1:]

	parts := strings.SplitN(tail, ",", 2)
	if len(parts) != 2 {
		return "E01"
	}

	off, err1 := strconv.ParseUint(parts[0], 16, 64)
	ln, err2 := strconv.ParseUint(parts[1], 16, 64)

	if err1 != nil || err2 != nil {
		return "E01"
	}

	if off >= uint64(len(rendered)) {
		return "l"
	}

	end := off + ln
	if end > uint64(len(rendered)) {
		end = uint64(len(rendered))
	}

	chunk := rendered[off:end]

	marker := byte('m')
	if end == uint64(len(rendered)) {
		marker = 'l'
	}

	return string(marker) + hexEncode(chunk)
}

func chooseType(v dbg.VariableInfo) string {
	if v.Type != "" {
		return v.Type
	}

	if v.TypeMeta != nil {
		if v.TypeMeta.Name != "" {
			return v.TypeMeta.Name
		}

		if v.TypeMeta.Kind != "" {
			return v.TypeMeta.Kind
		}
	}

	return "unknown"
}

// streamChunk is a helper to implement qXfer offset,length chunking for arbitrary data.
func (s *Server) streamChunk(cmd string, data []byte) string {
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

// handleQXferCorrelation streams events for a correlation id. Annex may carry id and n similar to actors-messages.
func (s *Server) handleQXferCorrelation(cmd string) string {
	if CorrelationJSONProvider == nil && CorrelationJSONProviderEx == nil {
		return "l"
	}

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
	// annex.
	rest := cmd[:lastColon]
	second := strings.LastIndex(rest, ":")
	annex := ""

	if second >= 0 && second+1 < len(rest) {
		annex = rest[second+1:]
	}
	// If extended provider present, parse annex key=val pairs.
	if CorrelationJSONProviderEx != nil && annex != "" {
		params := make(map[string]string)

		for _, kv := range strings.Split(annex, ",") {
			if kv == "" {
				continue
			}

			if i := strings.IndexByte(kv, '='); i >= 0 {
				k := strings.TrimSpace(kv[:i])
				v := strings.TrimSpace(kv[i+1:])
				params[k] = v
			}
		}

		data := CorrelationJSONProviderEx(params)
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

	corrID := ""
	n := 100

	if annex != "" {
		for _, it := range strings.Split(annex, ",") {
			if strings.HasPrefix(it, "id=") {
				corrID = strings.TrimPrefix(it, "id=")
			} else if strings.HasPrefix(it, "n=") {
				if nn, err := strconv.Atoi(strings.TrimPrefix(it, "n=")); err == nil && nn > 0 {
					n = nn
				}
			}
		}
	}

	data := CorrelationJSONProvider(corrID, n)
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

// handleQXferStack returns a JSON-encoded pseudo stack trace.
// Format: qXfer:stack:read::OFFSET,LENGTH.
func (s *Server) handleQXferStack(cmd string) string {
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

	st := dbg.BuildStackTrace(s.pcmap, s.info, s.pc)

	data := dbg.EncodeStackTraceJSON(st)
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

// handleReadMemory implements 'm addr,length'.
func (s *Server) handleReadMemory(cmd string) string {
	// mADDR,LENGTH.
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

// handleWriteMemory implements 'M addr,length:hexdata'.
func (s *Server) handleWriteMemory(cmd string) string {
	// Strip prefix 'M'.
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
	// advance until next breakpoint or range end.
	for {
		if s.bp[s.pc] {
			return s.makeStopReplyLocked()
		}
		// single step.
		if !s.stepLocked() {
			return s.makeStopReplyLocked()
		}
		// propagate pc into regs after each step.
		s.regs[0] = s.pc
	}
}

func (s *Server) step() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stepLocked()
	// Keep the pseudo-pc register in sync.
	s.regs[0] = s.pc

	return s.makeStopReplyLocked()
}

func (s *Server) stepLocked() bool {
	// Size policy: 4 bytes per line, bounded by owning range.
	file, _, ok := s.pcmap.AddrToLine(s.pc)
	if !ok {
		// at end or invalid; try to move to next known range start.
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
	// Increment within range.
	for _, r := range s.pcmap.Ranges {
		if s.pc >= r.Low && s.pc < r.High {
			nxt := s.pc + 4
			if nxt >= r.High {
				// move to next range start if exists.
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

// makeStopReply returns a stop reply packet, either Sxx or Txx with thread and pc.
func (s *Server) makeStopReply() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.makeStopReplyLocked()
}

func (s *Server) makeStopReplyLocked() string {
	if !s.useT {
		return "S05"
	}
	// T05;thread:1;pc:<hex>;.
	// Encode pc as 8-byte little-endian hex.
	v := s.pc
	buf := []byte{
		byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24),
		byte(v >> 32), byte(v >> 40), byte(v >> 48), byte(v >> 56),
	}

	return "T05;thread:1;pc:" + hexEncode(buf) + ";"
}

// readPacket reads an RSP packet and sends ACK '+'.
func readPacket(r *bufio.Reader) (string, error) {
	// wait for '$'.
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
	// read checksum (2 hex).
	csum := make([]byte, 2)
	if _, err := r.Read(csum); err != nil {
		return "", err
	}
	// ignore checksum; write ACK '+'.
	// Note: caller writes on same conn after returning.
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

// handleReadAllRegisters returns hex for all registers (little-endian 64-bit each).
func (s *Server) handleReadAllRegisters() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Sync pc.
	s.regs[0] = s.pc
	// Build byte slice.
	buf := make([]byte, 0, len(s.regs)*8)

	for _, v := range s.regs {
		var tmp [8]byte
		// little-endian.
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

// handleWriteAllRegisters parses hex data for all registers.
func (s *Server) handleWriteAllRegisters(cmd string) string {
	// Gxxxxxxxx (hex for all regs); spec uses raw hex after 'G'.
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
	// Keep pc in sync.
	s.pc = s.regs[0]
	s.mu.Unlock()

	return "OK"
}

// handleReadRegister returns the hex-encoded 64-bit value of a single register.
func (s *Server) handleReadRegister(cmd string) string {
	// pNN where NN is hex register index.
	idxHex := cmd[1:]

	idx, err := strconv.ParseUint(idxHex, 16, 64)
	if err != nil || idx >= uint64(len(s.regs)) {
		// Return zero value for unknown.
		return strings.Repeat("0", 16)
	}

	s.mu.Lock()
	// Sync pc.
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

// handleWriteRegister sets a single 64-bit register from hex.
func (s *Server) handleWriteRegister(cmd string) string {
	// PNN=hex.
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
	// Z0,<addr>,<kind>.
	parts := strings.Split(cmd, ",")
	if len(parts) < 2 {
		return 0
	}

	a := parts[1]
	// addr is hex.
	v, _ := strconv.ParseUint(a, 16, 64)

	return v
}

// hexEncode returns hex string for a byte slice.
func hexEncode(b []byte) string { return hex.EncodeToString(b) }
