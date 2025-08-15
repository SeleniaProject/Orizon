package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	dbg "github.com/orizon-lang/orizon/internal/debug"
	"github.com/orizon-lang/orizon/internal/debug/gdbserver"
	rt "github.com/orizon-lang/orizon/internal/runtime"
)

func main() {
	var (
		addr       string
		dbgJSON    string
		actorsJSON string
		actorsURL  string
		debugHTTP  string
		actorID    uint64
		actorN     int
	)
	flag.StringVar(&addr, "addr", ":9000", "listen address for RSP (tcp)")
	flag.StringVar(&dbgJSON, "debug-json", "", "path to ProgramDebugInfo JSON")
	flag.StringVar(&actorsJSON, "actors-json", "", "optional path to actors snapshot JSON for qXfer:actors:read")
	flag.StringVar(&actorsURL, "actors-url", "", "optional URL to fetch actors snapshot JSON for qXfer:actors:read")
	flag.StringVar(&debugHTTP, "debug-http", "", "optional address to serve actor runtime diagnostics (e.g. :8080)")
	flag.Uint64Var(&actorID, "actor-id", 0, "optional actor id for qXfer:actors-messages:read provider")
	flag.IntVar(&actorN, "actor-n", 100, "optional number of messages for qXfer:actors-messages:read provider")
	flag.Parse()

	if dbgJSON == "" {
		fmt.Fprintln(os.Stderr, "--debug-json is required")
		os.Exit(2)
	}
	b, err := os.ReadFile(dbgJSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read debug json failed:", err)
		os.Exit(1)
	}
	var info dbg.ProgramDebugInfo
	if err := json.Unmarshal(b, &info); err != nil {
		fmt.Fprintln(os.Stderr, "parse debug json failed:", err)
		os.Exit(1)
	}

	// Optional actors snapshot provider (static file or HTTP)
	if actorsJSON != "" {
		data, err := os.ReadFile(actorsJSON)
		if err == nil {
			local := make([]byte, len(data))
			copy(local, data)
			gdbserver.ActorsJSONProvider = func() []byte { return local }
		}
	} else if actorsURL != "" {
		gdbserver.ActorsJSONProvider = func() []byte {
			resp, err := http.Get(actorsURL)
			if err != nil || resp == nil || resp.Body == nil {
				return nil
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			return b
		}
	}

	// Optional messages provider: if actorsURL is provided and appears to end with /actors,
	// derive /actors/messages endpoint; otherwise, if debugHTTP is active, allow callers to
	// pass --actors-url=http://host:port/actors to enable message retrieval as well.
	gdbserver.ActorMessagesJSONProvider = func(id uint64, n int) []byte {
		base := actorsURL
		if base == "" && debugHTTP != "" {
			// If we are serving debug HTTP here, construct a default base
			base = "http://127.0.0.1" + debugHTTP + "/actors"
		}
		if base != "" {
			msgURL := base
			if strings.HasSuffix(strings.ToLower(msgURL), "/actors") {
				msgURL = strings.TrimSuffix(msgURL, "/actors") + "/actors/messages"
			} else if !strings.HasSuffix(strings.ToLower(msgURL), "/actors/messages") {
				msgURL = strings.TrimRight(msgURL, "/") + "/actors/messages"
			}
			query := msgURL + fmt.Sprintf("?id=%d&n=%d", id, n)
			resp, err := http.Get(query)
			if err == nil && resp != nil && resp.Body != nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				return b
			}
		}
		// Fallback: reuse actors snapshot if available
		if gdbserver.ActorsJSONProvider != nil {
			return gdbserver.ActorsJSONProvider()
		}
		return []byte("[]")
	}

	// Optional graph and deadlocks providers from HTTP endpoints (if actorsURL provided)
	gdbserver.ActorsGraphJSONProvider = func() []byte {
		base := actorsURL
		if base == "" && debugHTTP != "" {
			base = "http://127.0.0.1" + debugHTTP + "/actors"
		}
		if base == "" {
			return nil
		}
		url := base
		if strings.HasSuffix(strings.ToLower(url), "/actors") {
			url = strings.TrimSuffix(url, "/actors") + "/actors/graph"
		} else if !strings.HasSuffix(strings.ToLower(url), "/actors/graph") {
			url = strings.TrimRight(url, "/") + "/actors/graph"
		}
		resp, err := http.Get(url)
		if err == nil && resp != nil && resp.Body != nil {
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			return b
		}
		return nil
	}
	gdbserver.DeadlocksJSONProvider = func() []byte {
		base := actorsURL
		if base == "" && debugHTTP != "" {
			base = "http://127.0.0.1" + debugHTTP + "/actors"
		}

		// Correlation provider
		gdbserver.CorrelationJSONProvider = func(id string, n int) []byte {
			base := actorsURL
			if base == "" && debugHTTP != "" {
				base = "http://127.0.0.1" + debugHTTP + "/actors"
			}
			if base == "" {
				return nil
			}
			url := base
			if strings.HasSuffix(strings.ToLower(url), "/actors") {
				url = strings.TrimSuffix(url, "/actors") + "/actors/correlation"
			} else if !strings.HasSuffix(strings.ToLower(url), "/actors/correlation") {
				url = strings.TrimRight(url, "/") + "/actors/correlation"
			}
			query := url + fmt.Sprintf("?id=%s&n=%d", id, n)
			resp, err := http.Get(query)
			if err == nil && resp != nil && resp.Body != nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				return b
			}
			return nil
		}
		if base == "" {
			return nil
		}
		url := base
		if strings.HasSuffix(strings.ToLower(url), "/actors") {
			url = strings.TrimSuffix(url, "/actors") + "/actors/deadlocks"
		} else if !strings.HasSuffix(strings.ToLower(url), "/actors/deadlocks") {
			url = strings.TrimRight(url, "/") + "/actors/deadlocks"
		}
		resp, err := http.Get(url)
		if err == nil && resp != nil && resp.Body != nil {
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			return b
		}
		return nil
	}

	// Provide a simple locals provider when hosting runtime debug HTTP here
	gdbserver.LocalsJSONProvider = func(pc uint64) []byte {
		// If we have a debugHTTP runtime, ask its /actors endpoint for system snapshot and synthesize
		if debugHTTP != "" {
			// For demo purposes, return an empty list or minimal object; real runtime would inspect frames
			return []byte(`[]`)
		}
		return []byte(`[]`)
	}

	srv := gdbserver.NewServer(info)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen failed:", err)
		os.Exit(1)
	}
	fmt.Println("RSP server listening on", ln.Addr().String())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var shutdownHTTP func(ctx context.Context) error
	if debugHTTP != "" {
		// Launch a minimal runtime for diagnostics only. It is optional and independent from RSP flow.
		// This enables qXfer:actors:read via --actors-url=http://.../actors and messages via /actors/messages.
		sys, _ := rt.NewActorSystem(rt.DefaultActorSystemConfig)
		_ = sys.Start()
		sd, bound, _ := rt.StartDebugHTTPOn(sys, debugHTTP)
		// If actorsURL is empty, seed it using bound addr
		if actorsURL == "" {
			actorsURL = "http://" + bound + "/actors"
		}
		// Provide direct in-process providers for RSP when a runtime is hosted here
		gdbserver.ActorsJSONProvider = func() []byte {
			snap := sys.GetSystemSnapshot()
			b, _ := json.Marshal(snap)
			return b
		}
		gdbserver.ActorMessagesJSONProvider = func(id uint64, n int) []byte {
			if n <= 0 {
				n = 100
			}
			msgs := sys.GetRecentMessages(rt.ActorID(id), n)
			b, _ := json.Marshal(msgs)
			return b
		}
		shutdownHTTP = func(ctx context.Context) error {
			_ = sd(ctx)
			return sys.Stop()
		}
	}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Temporary errors: continue accepting
				continue
			}
			go func(conn net.Conn) {
				_ = srv.HandleConn(conn)
			}(c)
		}
	}()

	<-ctx.Done()
	_ = ln.Close()
	if shutdownHTTP != nil {
		_ = shutdownHTTP(context.Background())
	}
	fmt.Println("RSP server stopped")
}
