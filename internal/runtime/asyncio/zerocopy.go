package asyncio

import (
	"context"
	"io"
	"net"
	"time"
)

// CopyPreferZeroCopy copies from src to dst preferring zero-copy paths provided
// by the standard library (ReaderFrom/WriterTo). It falls back automatically.
func CopyPreferZeroCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

// CopyConnToConn copies stream data between two connections. If the underlying
// runtime supports zero-copy (e.g., TCP sendfile on Unix), io.Copy will use it.
// Deadlines from ctx (if any) are applied as best-effort read/write deadlines.
func CopyConnToConn(ctx context.Context, dst net.Conn, src net.Conn) (int64, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		_ = src.SetReadDeadline(deadline)
	}
	n, err := io.Copy(dst, src)
	// Clear deadlines after copy
	if ctx != nil {
		_ = dst.SetWriteDeadline(time.Time{})
		_ = src.SetReadDeadline(time.Time{})
	}
	return n, err
}

// CopyFileToConn streams file content to a connection. Platforms that support
// zero-copy sendfile may be used under the hood by io.Copy.
// Platform-specific optimized CopyFileToConn is provided per-OS. See
// zerocopy_unix_file.go and zerocopy_generic_file.go for implementations.
