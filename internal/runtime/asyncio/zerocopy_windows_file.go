//go:build windows
// +build windows

package asyncio

import (
    "context"
    "io"
    "net"
    "os"
    "time"
)

// CopyFileToConn on Windows currently falls back to io.Copy.
// Future work: use TransmitFile via syscall for true zero-copy.
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
    if deadline, ok := ctx.Deadline(); ok {
        _ = dst.SetWriteDeadline(deadline)
        defer dst.SetWriteDeadline(time.Time{})
    }
    return io.Copy(dst, src)
}


