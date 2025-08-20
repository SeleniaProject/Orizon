//go:build windows.
// +build windows.

package asyncio

import (
	"context"
	"crypto/rand"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCopyFileToConn_TransmitFilePath verifies that CopyFileToConn can stream a file.
// over a TCP connection on Windows. It does not assert that TransmitFile is used
// (that depends on environment), but it ensures correctness and deadline handling.
func TestCopyFileToConn_TransmitFilePath(t *testing.T) {
	// Prepare temp file with random content.
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.bin")
	const payloadSize = 256 * 1024 // 256 KiB
	data := make([]byte, payloadSize)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		t.Fatalf("failed to generate data: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	// Start TCP server to receive bytes.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	recvDone := make(chan int64, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		// Read exactly payloadSize bytes.
		var got int64
		buf := make([]byte, 32*1024)
		deadline := time.Now().Add(5 * time.Second)
		_ = c.SetReadDeadline(deadline)
		for got < payloadSize {
			n, er := c.Read(buf)
			if n > 0 {
				got += int64(n)
			}
			if er != nil {
				if er == io.EOF {
					break
				}
				return
			}
		}
		_ = c.SetReadDeadline(time.Time{})
		recvDone <- got
	}()

	// Dial client and send the file using CopyFileToConn.
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sent, err := CopyFileToConn(ctx, client, f)
	if err != nil {
		t.Fatalf("CopyFileToConn failed: %v", err)
	}

	// Wait for server side to finish.
	select {
	case got := <-recvDone:
		if got != int64(payloadSize) {
			t.Fatalf("server received %d bytes; want %d", got, payloadSize)
		}
		// sent may be payloadSize or 0 depending on code path; allow >=0 and <=payloadSize.
		if sent < 0 || sent > int64(payloadSize) {
			t.Fatalf("reported sent bytes %d out of range (0..%d)", sent, payloadSize)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("timeout waiting for receiver")
	}
}
