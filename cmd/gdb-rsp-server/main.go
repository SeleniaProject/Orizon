package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	dbg "github.com/orizon-lang/orizon/internal/debug"
	"github.com/orizon-lang/orizon/internal/debug/gdbserver"
)

func main() {
	var (
		addr    string
		dbgJSON string
	)
	flag.StringVar(&addr, "addr", ":9000", "listen address for RSP (tcp)")
	flag.StringVar(&dbgJSON, "debug-json", "", "path to ProgramDebugInfo JSON")
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

	srv := gdbserver.NewServer(info)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen failed:", err)
		os.Exit(1)
	}
	fmt.Println("RSP server listening on", ln.Addr().String())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	fmt.Println("RSP server stopped")
}
