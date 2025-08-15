# Orizon Async I/O Poller Abstraction

This directory provides a cross-platform readiness-notification abstraction used by the Orizon runtime.

- Portable baseline: `goPoller` in `async_io.go`
- OS pollers: Linux `epoll`, BSD/macOS `kqueue`, Windows `WSAPoll` (default), optional Windows IOCP (behind build tag)

## Poller Interface
- `Start(ctx)` / `Stop()`
- `Register(conn, kinds, handler)`
- `Deregister(conn)`

Event kinds: `Readable`, `Writable`, `Error`.

## Platform Selection
- Linux: `NewOSPoller()` -> epoll
- BSD/macOS: `NewOSPoller()` -> kqueue
- Windows: `NewOSPoller()` selection priority
  1. IOCP if explicitly requested and compiled in: set `ORIZON_WIN_IOCP=1` and build with `-tags iocp`
  2. `WSAPoll` if `ORIZON_WIN_WSAPOLL=1`
  3. Portable goroutine-based poller if `ORIZON_WIN_PORTABLE=1`
  4. Default: portable goroutine-based poller (broad compatibility)

Note: IOCP implementation (`iocp_experimental_windows.go`) is experimental and only built with the `iocp` build tag. It associates sockets to a process-owned IOCP and uses zero-byte `WSARecv/WSASend` to drive completions.

## Behavioral Guarantees
- Register/Deregister are safe to call; after `Deregister`, no further handler invocations are delivered for that connection. Implementations guard with a disabled flag and wait for in-flight goroutines where applicable.
- Error/EOF: If the peer closes, a single `Error` event is delivered (EOF or OS-specific error). No further events follow for that connection.
- Writable throttling: To avoid CPU spikes when idle, all pollers throttle `Writable` notifications to at most one every 50ms per connection.
  - Implemented in: `async_io.go`, `iocp_poller_windows.go`, `epoll_poller_linux.go`, `kqueue_poller_bsd.go`.
- Backoff/adaptive polling: The portable poller dynamically adapts its internal tick to reduce CPU usage.

## Testing
Common conformance tests live in:
- `async_io_test.go` (readable/deregister, error on close, echo readiness)
- `async_io_abnormal_test.go` (concurrent deregister/stop)

Windows IOCP (when enabled with `-tags iocp`):
- `iocp_experimental_windows_test.go`
- Benchmarks: `iocp_experimental_windows_bench_test.go`

## Caveats
- Windows IOCP with Go's `net` package: associating runtime-managed sockets to a custom IOCP can conflict. The experimental IOCP path should be used with sockets opened by this runtime only.
- When switching pollers via environment variables, validate behavior in CI for parity.
