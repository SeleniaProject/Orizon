//go:build windows
// +build windows

package asyncio

import (
	"context"
	"errors"
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

// CopyFileToConn on Windows attempts zero-copy via TransmitFile when possible.
// It falls back to io.Copy if the socket handle cannot be obtained or the call fails.
// Note: When TransmitFile succeeds, the returned byte count reflects the number of bytes
// sent from the current file offset to the end of file (best-effort using file size and offset).
func CopyFileToConn(ctx context.Context, dst net.Conn, src *os.File) (int64, error) {
	if dst == nil || src == nil {
		return 0, errors.New("nil dst or src")
	}

	// Apply best-effort deadline derived from context; cleared on return.
	if deadline, ok := ctx.Deadline(); ok {
		_ = dst.SetWriteDeadline(deadline)
		defer dst.SetWriteDeadline(time.Time{})
	}

	// Obtain underlying SOCKET handle from dst if available.
	var s uintptr
	if sc, ok := dst.(interface {
		SyscallConn() (syscall.RawConn, error)
	}); ok {
		if rc, err := sc.SyscallConn(); err == nil {
			_ = rc.Control(func(fd uintptr) { s = fd })
		}
	}

	// Try TransmitFile only if we have a SOCKET handle and the proc is resolvable.
	if s != 0 && procTransmitFile.Find() == nil {
		// Perform the call:
		// BOOL TransmitFile(
		//   SOCKET hSocket,
		//   HANDLE hFile,
		//   DWORD nNumberOfBytesToWrite,        // 0 => entire file from current fp
		//   DWORD nNumberOfBytesPerSend,        // 0 => system default chunking
		//   LPOVERLAPPED lpOverlapped,          // NULL => synchronous completion
		//   LPTRANSMIT_FILE_BUFFERS lpBuffers,  // NULL => no headers/trailers
		//   DWORD dwFlags                        // 0 => default
		// )
		r1, _, callErr := procTransmitFile.Call(s, src.Fd(), 0, 0, 0, 0, 0)
		if r1 != 0 {
			// Compute best-effort number of bytes sent: size - current offset
			// We avoid moving the file pointer here; use Seek(0, Current) to read position.
			var sent int64
			if st, statErr := src.Stat(); statErr == nil {
				if cur, offErr := src.Seek(0, io.SeekCurrent); offErr == nil {
					if st.Size() >= cur {
						sent = st.Size() - cur
					} else {
						sent = 0
					}
				}
			}
			return sent, nil
		}
		// If the call failed (including ERROR_NOT_SUPPORTED), fall through to the portable path.
		_ = callErr
	}

	// Portable fallback path.
	return io.Copy(dst, src)
}
