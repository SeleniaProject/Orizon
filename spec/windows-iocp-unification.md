# Windows IOCP Unification Plan (Design Draft)

Status: Draft
Owner: Runtime/AsyncIO
Scope: internal/runtime/asyncio

## Goals

- Unify Windows async I/O pollers under one coherent abstraction with cross-platform API consistency.
- Make Register idempotent, Deregister robust, and Writable notifications throttled across strategies.
- Provide a safe and maintainable path to adopt IOCP without breaking existing users.

## Current Implementations

- Portable poller (goroutine-based probing) — default, cross-platform.
- WSAPoll-based poller (windowsPoller) — pure Go, no C/C++ deps.
- IOCP experimental poller (iocpPoller, build tag `iocp`) — zero-byte WSARecv/WSASend, CancelIoEx.

## Selection Strategy (Windows)

- Environment-driven selection (see `internal/runtime/asyncio/poller_factory_windows.go`):
  1) IOCP if `-tags iocp` and `ORIZON_WIN_IOCP=1`.
  2) WSAPoll if `ORIZON_WIN_WSAPOLL=1`.
  3) Portable (default) otherwise.

## Configuration

- Writable throttling interval (all pollers):
  - Env: `ORIZON_WIN_WRITABLE_INTERVAL_MS` (integer, milliseconds)
  - Default: 50ms
  - Clamp: [5ms, 5000ms]
  - Applied via shared helper `getWritableInterval()` in `internal/runtime/asyncio/writable_throttle.go`.

## API Guarantees

- Register(conn, kinds, handler) is idempotent.
  - Re-registering the same conn updates `kinds` and `handler` in-place.
- Deregister(conn) is robust and idempotent.
  - Safe after `conn.Close()` (by-conn fallback lookup on Windows).
  - Safe to call multiple times, no error, no further events after completion.
- Writable notifications are throttled.
  - Prevent flood; platform-specific mechanisms may differ but frequency is bounded.
- Shutdown semantics
  - Stop() cancels watchers, cancels pending I/O (Windows), and waits boundedly for completion.

## IOCP Behavior

- Readable: zero-byte `WSARecv` enqueued; completion indicates readiness/EOF.
- Writable: periodic zero-byte `WSASend` probes, delivered as `Writable` on completion.
- Cancellation: `CancelIoEx(handle, nil)` used during Deregister/Stop to abort in-flight operations.
- Completion loop: `GetQueuedCompletionStatus` drives dispatch; disabled regs are ignored.

## WSAPoll Behavior

- Poll set managed internally; `wake()` rebuilds FD sets upon Register/Deregister.
- Readable: deadline+Peek or non-destructive probe to detect data/EOF consistently.
- Writable: time-based throttling with atomic last-fire timestamp.

## Event Semantics

- Readable: delivered when at least 1 byte can be read; EOF is reported as `Error{Err: io.EOF}` consistently.
- Writable: delivered periodically under throttling; not guaranteed per-byte flow control signal.
- Error: network errors surfaced; on Windows, common errno mapped (e.g., `ERROR_NETNAME_DELETED` → EOF).

## Deregister Robustness (Windows)

- Key extraction path (via `SyscallConn`) preferred; if unavailable or already closed, fallback by-conn map scan.
- Disable → stop writable ticker/watchers → `CancelIoEx` → remove from maps → wait closed/watchDone with timeout.
- Double Deregister is a no-op; post-Deregister event delivery is suppressed by `reg.disabled`.

## Testing Strategy

- Cross-platform tests:
  - Register idempotency updates handler/kinds.
  - Deregister idempotent and safe after close.
  - No events after Deregister under traffic.
  - Writable throttling frequency bounds.
- Windows-specific (optionally `-tags iocp`):
  - Zero-byte recv/send path coverage.
  - CancelIoEx cancellation behavior during Deregister/Stop.
  - Writable throttling honors env on WSAPoll and IOCP.
  - No events after Deregister under active traffic (WSAPoll).

## Migration & Unification Roadmap

1) Solidify API consistency and tests (DONE/ONGOING).
2) Document selection & guarantees (DONE/ONGOING).
3) Abstract zero-byte notification strategy behind a small internal interface:
   - `type winNotifier interface { armReadable(*reg); armWritable(*reg); cancel(*reg) }`
4) Implement `winNotifier` for IOCP and WSAPoll; select at Start() time.
5) Gradually retire duplicate paths by delegating to the notifier while keeping maps/handlers unified.
6) Telemetry & perf validation on high-conn-count workloads.

## Proposed winNotifier Wiring (No-Behavior-Change Step)

Objective: Introduce a minimal wiring layer so both WSAPoll and IOCP can share arming/cancel hooks without changing observable behavior.

Design:

- Selection at Start():
  - In `windowsPoller.Start()`, set `p.notifier` according to effective Windows mode.
    - IOCP mode (enabled via build tag + env): `p.notifier = iocpNotifier{}`
    - Otherwise: `p.notifier = wsapollNotifier{}`
  - Current implementations are no-op, so this change is safe.

- Hooks in Register/Deregister:
  - `Register`: after storing/refreshing the registration, call:
    - `notifier.armReadable(&winRegLite{sock: s})` if `Readable` requested.
    - `notifier.armWritable(&winRegLite{sock: s})` if `Writable` requested.
  - `Deregister`: before or immediately after removing from maps, call:
    - `notifier.cancel(&winRegLite{sock: s})`.
  - Both are already invoked in the WSAPoll poller with no-op notifiers (kept as-is).

- Dispatch path unchanged:
  - WSAPoll keeps its `WSAPoll`-driven readiness; IOCP experimental keeps its `GetQueuedCompletionStatus` loop. The notifier only prepares/cancels OS-specific I/O.

Migration Steps:

1) Land the no-op wiring (already present for WSAPoll; select notifier at Start()).
2) Implement `iocpNotifier` to arm zero-byte `WSARecv/WSASend` and to `CancelIoEx` on cancel.
3) Switch IOCP path to rely on notifier for arming/cancel while keeping completion loop intact.
4) Add stress/race tests and CI gates for IOCP mode.

Risks & Mitigations:

- Risk: Conflicts with Go netpoller when arming I/O on sockets managed elsewhere.
  - Mitigation: Limit IOCP mode to explicitly requested scenarios; test with runtime-owned sockets.
- Risk: Deregister vs completion race.
  - Mitigation: Preserve `reg.disabled` checks in dispatch paths; keep cancel ordering as documented.

## Risks & Mitigations

- Conflict with Go runtime netpoller when associating existing sockets to IOCP.
  - Limit IOCP mode to explicitly requested/controlled scenarios (build tag + env).
- Race on Deregister vs completion dispatch.
  - Use `reg.disabled` checks in dispatch paths; bounded waits on watchers.
- Writable storm.
  - Enforce throttling intervals; allow tuning via future config if needed.

## Open Items

- Configurable throttling intervals per registration.
- Unified error classification across platforms.
- Benchmarks for WSAPoll vs IOCP under diverse workloads.
