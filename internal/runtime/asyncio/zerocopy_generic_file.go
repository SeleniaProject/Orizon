//go:build freebsd || netbsd || openbsd
// +build freebsd netbsd openbsd

package asyncio

import (
	"context"
	"io"
	"net"
	"os"
	"time"
)

// CopyFileToConn generic fallback using io.Copy.
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
	}
	n, err := io.Copy(dst, src)
	if ctx != nil {
		_ = dst.SetWriteDeadline(time.Time{})
	}
	return n, err
}
