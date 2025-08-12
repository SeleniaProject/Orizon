//go:build windows
// +build windows

package asyncio

import (
	"context"
	"io"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

var (
	mswsock          = windows.NewLazySystemDLL("mswsock.dll")
	procTransmitFile = mswsock.NewProc("TransmitFile")
)

// CopyFileToConn on Windows uses TransmitFile for zero-copy when available, falling back to io.Copy.
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	// deadline
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
	}
	// attempt TransmitFile path
	var s uintptr
	if sc, ok := dst.(interface {
		SyscallConn() (syscall.RawConn, error)
	}); ok {
		rc, err := sc.SyscallConn()
		if err == nil {
			_ = rc.Control(func(fd uintptr) { s = fd })
		}
	}
	if s != 0 && procTransmitFile.Find() == nil {
		// call TransmitFile(s, hFile, bytesToWrite=0 => send entire file)
		r1, _, e1 := procTransmitFile.Call(s, src.Fd(), 0, 0, 0, 0, 0)
		if r1 != 0 { // success
			// best-effort stat for bytes
			st, _ := src.Stat()
			return st.Size(), nil
		}
		// fallback on known-not-implemented or other errors
		_ = e1
	}
	// fallback
	return io.Copy(dst, src)
}
