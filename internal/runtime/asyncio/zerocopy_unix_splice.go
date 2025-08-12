//go:build linux
// +build linux

package asyncio

import (
	"context"
	"io"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// SpliceConnToConn performs zero-copy transfer from src to dst using splice(2) via a pipe.
// length <= 0 copies until EOF. Returns bytes transferred.
func SpliceConnToConn(ctx context.Context, dst net.Conn, src net.Conn, length int64) (int64, error) {
	// get fds
	var sfd, dfd int
	if sc, ok := src.(interface {
		SyscallConn() (syscall.RawConn, error)
	}); ok {
		rc, err := sc.SyscallConn()
		if err != nil {
			return 0, err
		}
		_ = rc.Control(func(fd uintptr) { sfd = int(fd) })
	} else {
		return 0, io.ErrUnexpectedEOF
	}
	if sc, ok := dst.(interface {
		SyscallConn() (syscall.RawConn, error)
	}); ok {
		rc, err := sc.SyscallConn()
		if err != nil {
			return 0, err
		}
		_ = rc.Control(func(fd uintptr) { dfd = int(fd) })
	} else {
		return 0, io.ErrUnexpectedEOF
	}

	// Create pipe
	p := make([]int, 2)
	if err := unix.Pipe(p); err != nil {
		return 0, err
	}
	pr, pw := p[0], p[1]
	defer unix.Close(pr)
	defer unix.Close(pw)

	// optional deadline
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		_ = src.SetReadDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
		defer src.SetReadDeadline(time.Time{})
	}

	var transferred int64
	const chunk = 1 << 20 // 1MB per splice
	for length <= 0 || transferred < length {
		toRead := chunk
		if length > 0 {
			remaining := length - transferred
			if remaining < toRead {
				toRead = remaining
			}
		}
		// src -> pipe
		n1, err := unix.Splice(sfd, nil, pw, nil, int(toRead), unix.SPLICE_F_MOVE)
		if n1 == 0 && err == nil { // EOF
			break
		}
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return transferred, err
		}
		// pipe -> dst
		off := 0
		for off < n1 {
			n2, err2 := unix.Splice(pr, nil, dfd, nil, n1-off, unix.SPLICE_F_MOVE)
			if err2 != nil {
				if err2 == unix.EAGAIN || err2 == unix.EINTR {
					continue
				}
				return transferred, err2
			}
			if n2 == 0 {
				break
			}
			off += n2
		}
		transferred += int64(n1)
	}
	return transferred, nil
}
