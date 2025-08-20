//go:build linux.
// +build linux.

package asyncio

import (
	"context"
	"io"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// CopyFileToConn uses sendfile on Linux for zero-copy file->socket transfer.
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	// obtain dst fd.
	var dfd int
	if sc, ok := dst.(interface {
		SyscallConn() (syscall.RawConn, error)
	}); ok {
		rc, err := sc.SyscallConn()
		if err != nil {
			return 0, err
		}
		_ = rc.Control(func(fd uintptr) { dfd = int(fd) })
	} else {
		return fallbackCopyFileToConn(ctx, dst, src)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
	}
	var total int64
	const chunk = 1 << 20 // 1MB per call
	for {
		sent, err := unix.Sendfile(dfd, int(src.Fd()), nil, chunk)
		total += int64(sent)
		if sent == 0 && err == nil {
			return total, nil
		}
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return total, err
		}
	}
}

// CopyFileToConnGeneric is used by the Linux path when direct fd access is unavailable.
func fallbackCopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
	}
	return io.Copy(dst, src)
}

// syscallConn abstracts the minimal methods of syscall.RawConn used here.
type syscallConn interface{ Control(func(fd uintptr)) error }
