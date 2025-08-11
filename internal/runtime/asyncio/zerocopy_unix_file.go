//go:build linux
// +build linux

package asyncio

import (
    "context"
    "net"
    "os"
    "time"

    "golang.org/x/sys/unix"
)

// CopyFileToConn uses sendfile on Linux for zero-copy file->socket transfer.
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
    var n int64
    // obtain dst fd
    var dfd int
    if sc, ok := dst.(interface{ SyscallConn() (syscallConn, error) }); ok {
        rc, err := sc.SyscallConn()
        if err != nil { return 0, err }
        _ = rc.Control(func(fd uintptr) { dfd = int(fd) })
    } else {
        // fallback to generic path
        return CopyFileToConnGeneric(ctx, dst, src)
    }
    if deadline, ok := ctx.Deadline(); ok {
        _ = dst.SetWriteDeadline(deadline)
        defer dst.SetWriteDeadline(time.Time{})
    }
    // loop until EOF
    for {
        // determine remaining by fstat
        st, err := src.Stat()
        if err != nil { return n, err }
        // sendfile from current offset; kernel updates file offset
        sent, err := unix.Sendfile(dfd, int(src.Fd()), nil, int(st.Size()))
        n += int64(sent)
        if err != nil {
            if err == unix.EAGAIN { continue }
            if err == nil && int64(sent) == st.Size() { return n, nil }
            return n, err
        }
        if sent == 0 { // EOF
            return n, nil
        }
        // continue until fully sent
    }
}

// CopyFileToConnGeneric is used by the Linux path when direct fd access is unavailable.
func CopyFileToConnGeneric(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
    return CopyFileToConn(ctx, dst, src)
}

// syscallConn abstracts the minimal methods of syscall.RawConn used here.
type syscallConn interface{
    Control(func(fd uintptr)) error
}


