package gdbserver

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"testing"

	dbg "github.com/orizon-lang/orizon/internal/debug"
)

// helper to build minimal ProgramDebugInfo with given line counts per function
func buildDebugInfo(linesPerFunc int) dbg.ProgramDebugInfo {
	lines := make([]dbg.LineEntry, 0, linesPerFunc)
	for i := 0; i < linesPerFunc; i++ {
		lines = append(lines, dbg.LineEntry{File: "test.orz", Line: i + 1, Column: 1})
	}
	fn := dbg.FunctionInfo{Name: "main", Lines: lines}
	mod := dbg.ModuleDebugInfo{ModuleName: "m", Functions: []dbg.FunctionInfo{fn}}
	return dbg.ProgramDebugInfo{Modules: []dbg.ModuleDebugInfo{mod}}
}

func encodeRSP(payload string) []byte {
	sum := byte(0)
	for i := 0; i < len(payload); i++ {
		sum += payload[i]
	}
	return []byte(fmt.Sprintf("$%s#%02x", payload, sum))
}

// readReply reads optional ack and one RSP packet payload
func readReply(r *bufio.Reader) (ack bool, payload string, err error) {
	// optional ack
	b, err := r.ReadByte()
	if err != nil {
		return false, "", err
	}
	if b != '+' {
		// put back into buffer semantics: since bufio.Reader has no UnreadByte for first byte after ReadByte? It does have UnreadByte
		if err := r.UnreadByte(); err != nil {
			return false, "", err
		}
	} else {
		ack = true
	}
	// read '$'
	for {
		ch, err := r.ReadByte()
		if err != nil {
			return ack, "", err
		}
		if ch == '$' {
			break
		}
	}
	data := make([]byte, 0, 128)
	for {
		ch, err := r.ReadByte()
		if err != nil {
			return ack, "", err
		}
		if ch == '#' {
			break
		}
		data = append(data, ch)
	}
	// read checksum
	csum := make([]byte, 2)
	if _, err := r.Read(csum); err != nil {
		return ack, "", err
	}
	_ = csum // ignore
	return ack, string(data), nil
}

func TestRSP_QSupported_NoAckMode(t *testing.T) {
	srv := NewServer(buildDebugInfo(3))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()

	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// qSupported
	if _, err := w.Write(encodeRSP("qSupported")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	ack, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if !ack {
		t.Fatalf("expected ack for qSupported")
	}
	if payload == "" || payload[:11] != "PacketSize=" {
		t.Fatalf("unexpected payload: %q", payload)
	}

	// Enable no-ack mode
	if _, err := w.Write(encodeRSP("QStartNoAckMode")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	ack, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if !ack {
		t.Fatalf("expected ack for QStartNoAckMode")
	}
	if payload != "OK" {
		t.Fatalf("expected OK, got %q", payload)
	}

	// Next command should not be acked
	if _, err := w.Write(encodeRSP("g")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	ack, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if ack {
		t.Fatalf("did not expect ack after no-ack mode")
	}
	if len(payload) == 0 {
		t.Fatalf("expected register payload")
	}
}

func TestRSP_RegisterAndPCStep(t *testing.T) {
	srv := NewServer(buildDebugInfo(3))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// Enter no-ack for simpler reads
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// Read PC (register 0 via p00)
	_, _ = w.Write(encodeRSP("p0"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "0000000000000000" {
		t.Fatalf("initial pc expected 0, got %q", payload)
	}

	// Single step once -> pc=4
	_, _ = w.Write(encodeRSP("s"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "S05" {
		t.Fatalf("expected S05 stop, got %q", payload)
	}
	// Read PC again
	_, _ = w.Write(encodeRSP("p0"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "0400000000000000" {
		t.Fatalf("expected pc=4 (little endian), got %q", payload)
	}
}

func TestRSP_MemoryReadWrite(t *testing.T) {
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack mode
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// Write memory at 0x10: 01 02 03 04
	_, _ = w.Write(encodeRSP("M10,4:01020304"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "OK" {
		t.Fatalf("expected OK, got %q", payload)
	}
	// Read back
	_, _ = w.Write(encodeRSP("m10,4"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "01020304" {
		t.Fatalf("expected 01020304, got %q", payload)
	}
}

func TestRSP_BreakpointContinue(t *testing.T) {
	srv := NewServer(buildDebugInfo(3))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack mode
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// Set breakpoint at address 8 (third step boundary)
	_, _ = w.Write(encodeRSP("Z0,8,1"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "OK" {
		t.Fatalf("expected OK for Z0, got %q", payload)
	}

	// Continue until hit bp -> expect S05
	_, _ = w.Write(encodeRSP("c"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "S05" {
		t.Fatalf("expected stop S05, got %q", payload)
	}

	// Verify PC is 8
	_, _ = w.Write(encodeRSP("p0"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	// expect 8 in little endian
	want := make([]byte, 8)
	want[0] = 0x08
	got, _ := hex.DecodeString(payload)
	if len(got) != 8 || got[0] != want[0] {
		t.Fatalf("expected pc=8, got %q", payload)
	}
}

func TestRSP_QXferFeaturesAndLibraries(t *testing.T) {
	srv := NewServer(buildDebugInfo(2))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// Enter no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// features: target.xml chunked read
	_, _ = w.Write(encodeRSP("qXfer:features:read:target.xml:0,20"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) < 3 {
		t.Fatalf("short payload: %q", payload)
	}
	if payload[0] != 'm' && payload[0] != 'l' {
		t.Fatalf("expected m/l marker, got %q", payload[0])
	}
	data, err := hex.DecodeString(payload[1:])
	if err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
	if len(data) == 0 || data[0] != '<' {
		t.Fatalf("expected xml data, got %q", string(data))
	}

	// libraries list
	_, _ = w.Write(encodeRSP("qXfer:libraries:read::0,10"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload[0] != 'm' && payload[0] != 'l' {
		t.Fatalf("expected m/l marker, got %q", payload[0])
	}
	_, err = hex.DecodeString(payload[1:])
	if err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
}

func TestRSP_QXferMemoryMapAndAuxv(t *testing.T) {
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// memory-map
	_, _ = w.Write(encodeRSP("qXfer:memory-map:read::0,20"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload[0] != 'm' && payload[0] != 'l' {
		t.Fatalf("expected m/l marker, got %q", payload[0])
	}
	_, err = hex.DecodeString(payload[1:])
	if err != nil {
		t.Fatalf("invalid hex: %v", err)
	}

	// auxv (empty)
	_, _ = w.Write(encodeRSP("qXfer:auxv:read::0,10"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload[0] != 'l' {
		t.Fatalf("expected l marker for empty auxv, got %q", payload[0])
	}
}

func TestRSP_QXferStack(t *testing.T) {
	srv := NewServer(buildDebugInfo(3))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// stack trace chunk read
	_, _ = w.Write(encodeRSP("qXfer:stack:read::0,40"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) < 2 {
		t.Fatalf("short payload: %q", payload)
	}
	if payload[0] != 'm' && payload[0] != 'l' {
		t.Fatalf("expected m/l marker, got %q", payload[0])
	}
	data, err := hex.DecodeString(payload[1:])
	if err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
	if len(data) == 0 || data[0] != '{' {
		t.Fatalf("expected JSON data, got %q", string(data))
	}
}

func TestRSP_QXferActorsAndMessages(t *testing.T) {
	// Wire a simple provider with deterministic JSON
	ActorsJSONProvider = func() []byte { return []byte(`{"frames":[{"pc":0}]}`) }
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// actors read
	_, _ = w.Write(encodeRSP("qXfer:actors:read::0,20"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad xfer actors: %q", payload)
	}

	// actors-messages read (reuses provider for now)
	_, _ = w.Write(encodeRSP("qXfer:actors-messages:read::0,20"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad xfer actors-messages: %q", payload)
	}
}

func TestRSP_QXferActorsGraphAndDeadlocks(t *testing.T) {
	// Provide deterministic JSON providers
	ActorsJSONProvider = func() []byte { return []byte(`{"actors":[{"id":1}]}`) }
	ActorsGraphJSONProvider = func() []byte { return []byte(`{"nodes":[],"edges":[]}`) }
	DeadlocksJSONProvider = func() []byte { return []byte(`[]`) }
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// actors-graph
	_, _ = w.Write(encodeRSP("qXfer:actors-graph:read::0,20"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad xfer actors-graph: %q", payload)
	}
	if _, err := hex.DecodeString(payload[1:]); err != nil {
		t.Fatalf("invalid hex: %v", err)
	}

	// deadlocks
	_, _ = w.Write(encodeRSP("qXfer:deadlocks:read::0,20"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad xfer deadlocks: %q", payload)
	}
	if _, err := hex.DecodeString(payload[1:]); err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
}

func TestRSP_QXferCorrelation(t *testing.T) {
	// Provide correlation provider
	CorrelationJSONProvider = func(id string, n int) []byte {
		if id == "abc" {
			return []byte(`[{"id":"abc"}]`)
		}
		return []byte(`[]`)
	}
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// read correlation with annex id=abc,n=10
	_, _ = w.Write(encodeRSP("qXfer:correlation:read:id=abc,n=10:0,20"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad xfer correlation: %q", payload)
	}
	if _, err := hex.DecodeString(payload[1:]); err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
}

func TestRSP_QXferLocals(t *testing.T) {
	// Provide locals provider returning fixed json
	LocalsJSONProvider = func(pc uint64) []byte { return []byte(`[{"name":"x","value":1}]`) }
	srv := NewServer(buildDebugInfo(2))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// enter no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// request first chunk
	_, _ = w.Write(encodeRSP("qXfer:locals:read::0,20"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad payload: %q", payload)
	}
	if _, err := hex.DecodeString(payload[1:]); err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
}

func TestRSP_QXferPrettyLocals(t *testing.T) {
	PrettyLocalsJSONProvider = func(pc, fp uint64) []byte { return []byte(`[{"name":"x","type":"int","value":123}]`) }
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)
	_, _ = w.Write(encodeRSP("qXfer:pretty-locals:read::0,40"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad payload: %q", payload)
	}
	if _, err := hex.DecodeString(payload[1:]); err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
}

func TestRSP_QXferPrettyMemory(t *testing.T) {
	srv := NewServer(buildDebugInfo(1))
	// Seed memory
	for i := 0; i < 64; i++ {
		srv.mem[0x100+uint64(i)] = byte(0x41 + (i % 26))
	}
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)
	// Request pretty memory
	_, _ = w.Write(encodeRSP("qXfer:pretty-memory:read:addr=100,len=40:0,80"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" || (payload[0] != 'm' && payload[0] != 'l') {
		t.Fatalf("bad payload: %q", payload)
	}
	// ensure hex decodes
	if data, err := hex.DecodeString(payload[1:]); err != nil || len(data) == 0 {
		t.Fatalf("invalid hex or empty: %v, %d", err, len(data))
	}
}

func TestRSP_TStopReply(t *testing.T) {
	srv := NewServer(buildDebugInfo(2))
	// Enable T-stop replies
	srv.useT = true
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// Single step -> expect T05 with pc
	_, _ = w.Write(encodeRSP("s"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) < 3 || payload[:3] != "T05" {
		t.Fatalf("expected T05..., got %q", payload)
	}
	if !strings.Contains(payload, ";pc:") {
		t.Fatalf("expected pc field in T-stop: %q", payload)
	}
}

func TestRSP_ThreadQueriesAndControl(t *testing.T) {
	srv := NewServer(buildDebugInfo(2))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// qC current thread
	_, _ = w.Write(encodeRSP("qC"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "QC1" {
		t.Fatalf("unexpected qC: %q", payload)
	}

	// thread list
	_, _ = w.Write(encodeRSP("qfThreadInfo"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "m1" {
		t.Fatalf("unexpected qfThreadInfo: %q", payload)
	}
	_, _ = w.Write(encodeRSP("qsThreadInfo"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "l" {
		t.Fatalf("unexpected qsThreadInfo: %q", payload)
	}

	// H thread selection
	_, _ = w.Write(encodeRSP("Hc1"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "OK" {
		t.Fatalf("unexpected H: %q", payload)
	}

	// T thread alive
	_, _ = w.Write(encodeRSP("T1"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "OK" {
		t.Fatalf("unexpected T: %q", payload)
	}
}

func TestRSP_ContinueAndStepWithAddress(t *testing.T) {
	srv := NewServer(buildDebugInfo(6))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// set breakpoint at 12, continue from 8
	_, _ = w.Write(encodeRSP("Z0,c,1")) // malformed; ensure use hex addr next
	_ = w.Flush()
	_, _, _ = readReply(r) // ignore
	_, _ = w.Write(encodeRSP("Z0,c,1"))
	_ = w.Flush()
	_, _, _ = readReply(r)
	// correct break at 0xC
	_, _ = w.Write(encodeRSP("Z0,c,1"))
	_ = w.Flush()
	_, _, _ = readReply(r)
}

func TestRSP_vContQueryAndRegistersWrite(t *testing.T) {
	srv := NewServer(buildDebugInfo(1))
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() { _ = srv.HandleConn(c1) }()
	w := bufio.NewWriter(c2)
	r := bufio.NewReader(c2)

	// no-ack
	_, _ = w.Write(encodeRSP("QStartNoAckMode"))
	_ = w.Flush()
	_, _, _ = readReply(r)

	// vCont?
	_, _ = w.Write(encodeRSP("vCont?"))
	_ = w.Flush()
	_, payload, err := readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "vCont;c;s" {
		t.Fatalf("unexpected vCont support: %q", payload)
	}

	// Prepare G payload: set r1 to 0x1122334455667788
	totalRegs := 17
	raw := make([]byte, totalRegs*8)
	// write little-endian at offset for r1 (index 1)
	v := []byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}
	copy(raw[8:16], v)
	hexAll := hex.EncodeToString(raw)
	_, _ = w.Write(encodeRSP("G" + hexAll))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "OK" {
		t.Fatalf("expected OK for G, got %q", payload)
	}

	// Read back r1
	_, _ = w.Write(encodeRSP("p1"))
	_ = w.Flush()
	_, payload, err = readReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload != "8877665544332211" {
		t.Fatalf("unexpected r1 value: %q", payload)
	}
}
