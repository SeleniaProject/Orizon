//go:build darwin
// +build darwin

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

// CopyFileToConn uses sendfile on Darwin for zero-copy file->socket transfer.
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	// obtain destination socket fd via RawConn
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
		return fallbackCopyFileToConnDarwin(ctx, dst, src)
	}

	// deadline
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
	}

	// Determine file size and current offset
	st, err := src.Stat()
	if err != nil {
		return 0, err
	}
	var off int64 = 0
	var total int64
	// Loop until all bytes sent
	for total < st.Size() {
		toSend := int(st.Size() - total)
		if toSend > 1<<20 {
			toSend = 1 << 20
		} // 1MB chunks
		// sendfile(out, in, *offset, count) → returns bytes written
		n, err := unix.Sendfile(dfd, int(src.Fd()), &off, toSend)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return total, err
		}
		if n == 0 {
			break
		}
		total += int64(n)
	}
	return total, nil
}

func fallbackCopyFileToConnDarwin(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
	}
	return io.Copy(dst, src)
}
