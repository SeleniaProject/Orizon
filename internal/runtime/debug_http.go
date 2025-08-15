package runtime

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// StartDebugHTTP starts a lightweight HTTP server that exposes diagnostic endpoints
// for the running ActorSystem. The server provides the following endpoints:
//
//	GET /actors                -> JSON of DebugSystemSnapshot
//	GET /actors/messages       -> JSON array of recent TraceEvent for a given actor id.
//	                              Query params: id=<actorID>&n=<count>
//
// It returns a shutdown function compatible with http.Server.Shutdown.
func StartDebugHTTP(as *ActorSystem, addr string) (func(ctx context.Context) error, error) {
	mux := http.NewServeMux()

	// System snapshot
	mux.HandleFunc("/actors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		// Avoid long handler time by using a short timeout context for any future blocking ops
		_ = r.Context()
		snap := as.GetSystemSnapshot()
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(snap)
	})

	// Recent messages for an actor
	mux.HandleFunc("/actors/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		idStr := q.Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		n := 100
		if nStr := q.Get("n"); nStr != "" {
			if v, err := strconv.Atoi(nStr); err == nil && v > 0 {
				n = v
			}
		}
		list := as.GetRecentMessages(ActorID(id64), n)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	// Actor graph (nodes and edges)
	mux.HandleFunc("/actors/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		g := as.BuildActorGraph()
		// limit filter
		if limStr := q.Get("limit"); limStr != "" {
			if lim, err := strconv.Atoi(limStr); err == nil && lim >= 0 {
				if lim < len(g.Nodes) {
					g.Nodes = g.Nodes[:lim]
				}
				if lim < len(g.Edges) {
					g.Edges = g.Edges[:lim]
				}
			}
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(g)
	})

	// Potential deadlocks
	mux.HandleFunc("/actors/deadlocks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		list := as.DetectPotentialDeadlocks()
		if mcs := q.Get("minCycle"); mcs != "" {
			if m, err := strconv.Atoi(mcs); err == nil && m > 1 {
				out := make([]DebugDeadlockReport, 0, len(list))
				for _, r := range list {
					if r.Size >= m {
						out = append(out, r)
					}
				}
				list = out
			}
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	// Actor system metrics (includes I/O related counters)
	mux.HandleFunc("/actors/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		stats := as.GetStatistics()
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(stats)
	})

	// Condensed I/O-only metrics for quick dashboards
	mux.HandleFunc("/actors/io", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		s := as.GetStatistics()
		ioOnly := map[string]any{
			"eventsReadable":   s.IOEventsReadable,
			"eventsWritable":   s.IOEventsWritable,
			"eventsErrors":     s.IOEventsErrors,
			"rateLimitedDrops": s.IORateLimitedDrops,
			"overflowDrops":    s.IOOverflowDrops,
			"pausesRead":       s.IOPausesRead,
			"pausesWrite":      s.IOPausesWrite,
			"resumesRead":      s.IOResumesRead,
			"resumesWrite":     s.IOResumesWrite,
		}
		// Optional since/until (RFC3339) filter will aggregate from recent ring buffer
		sinceStr := q.Get("since")
		untilStr := q.Get("until")
		if sinceStr != "" || untilStr != "" {
			var since time.Time
			var until time.Time
			var err error
			if sinceStr != "" {
				since, err = time.Parse(time.RFC3339, sinceStr)
				if err != nil {
					since = time.Time{}
				}
			}
			if untilStr != "" {
				until, err = time.Parse(time.RFC3339, untilStr)
				if err != nil {
					until = time.Time{}
				}
			}
			if until.IsZero() {
				until = time.Now()
			}
			// Aggregate from ring
			readable := uint64(0)
			writable := uint64(0)
			errs := uint64(0)
			as.ioEventsMu.Lock()
			for _, rec := range as.ioEventsLog {
				if (!since.IsZero() && rec.Timestamp.Before(since)) || rec.Timestamp.After(until) {
					continue
				}
				switch rec.Type {
				case 0: // Readable
					readable++
				case 1: // Writable
					writable++
				default:
					errs++
				}
			}
			as.ioEventsMu.Unlock()
			ioOnly["eventsReadableWindow"] = readable
			ioOnly["eventsWritableWindow"] = writable
			ioOnly["eventsErrorsWindow"] = errs
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(ioOnly)
	})

	// Mailbox stats endpoint: /actors/mailbox?id=<actorID>
	mux.HandleFunc("/actors/mailbox", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		idStr := q.Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if st, ok := as.GetMailboxStats(ActorID(id64)); ok {
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(st)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	// Per-actor I/O stats endpoint: /actors/io/actor?id=<actorID>
	mux.HandleFunc("/actors/io/actor", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		idStr := q.Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		as.mutex.RLock()
		act := as.actors[ActorID(id64)]
		as.mutex.RUnlock()
		if act == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		ioStats := map[string]any{
			"readable": act.Statistics.IOEventsReadable,
			"writable": act.Statistics.IOEventsWritable,
			"errors":   act.Statistics.IOEventsErrors,
		}
		// Optional time window aggregation
		sinceStr := q.Get("since")
		untilStr := q.Get("until")
		if sinceStr != "" || untilStr != "" {
			var since time.Time
			var until time.Time
			if sinceStr != "" {
				if t, e := time.Parse(time.RFC3339, sinceStr); e == nil {
					since = t
				}
			}
			if untilStr != "" {
				if t, e := time.Parse(time.RFC3339, untilStr); e == nil {
					until = t
				}
			}
			if until.IsZero() {
				until = time.Now()
			}
			readable := uint64(0)
			writable := uint64(0)
			errs := uint64(0)
			as.ioEventsMu.Lock()
			for _, rec := range as.ioEventsLog {
				if rec.Actor != ActorID(id64) {
					continue
				}
				if (!since.IsZero() && rec.Timestamp.Before(since)) || rec.Timestamp.After(until) {
					continue
				}
				switch rec.Type {
				case 0:
					readable++
				case 1:
					writable++
				default:
					errs++
				}
			}
			as.ioEventsMu.Unlock()
			ioStats["readableWindow"] = readable
			ioStats["writableWindow"] = writable
			ioStats["errorsWindow"] = errs
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(ioStats)
	})

	// Top I/O actors endpoint: /actors/io/top?n=<count>&since=<RFC3339>&until=<RFC3339>
	mux.HandleFunc("/actors/io/top", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		n := 10
		if ns := q.Get("n"); ns != "" {
			if v, err := strconv.Atoi(ns); err == nil && v > 0 {
				n = v
			}
		}
		sinceStr := q.Get("since")
		untilStr := q.Get("until")
		var since time.Time
		var until time.Time
		if sinceStr != "" {
			if t, e := time.Parse(time.RFC3339, sinceStr); e == nil {
				since = t
			}
		}
		if untilStr != "" {
			if t, e := time.Parse(time.RFC3339, untilStr); e == nil {
				until = t
			}
		}
		if until.IsZero() {
			until = time.Now()
		}
		// Aggregate counts per actor
		type agg struct{ R, W, E uint64 }
		tab := make(map[ActorID]*agg)
		as.ioEventsMu.Lock()
		for _, rec := range as.ioEventsLog {
			if (!since.IsZero() && rec.Timestamp.Before(since)) || rec.Timestamp.After(until) {
				continue
			}
			a := tab[rec.Actor]
			if a == nil {
				a = &agg{}
				tab[rec.Actor] = a
			}
			switch rec.Type {
			case 0:
				a.R++
			case 1:
				a.W++
			default:
				a.E++
			}
		}
		as.ioEventsMu.Unlock()
		// Build and sort results by total desc
		type row struct {
			Actor    ActorID `json:"actor"`
			Readable uint64  `json:"readable"`
			Writable uint64  `json:"writable"`
			Errors   uint64  `json:"errors"`
			Total    uint64  `json:"total"`
		}
		list := make([]row, 0, len(tab))
		for id, v := range tab {
			list = append(list, row{Actor: id, Readable: v.R, Writable: v.W, Errors: v.E, Total: v.R + v.W + v.E})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Total > list[j].Total })
		if n < len(list) {
			list = list[:n]
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	// Actor lookup by name: /actors/lookup?name=<actorName>
	mux.HandleFunc("/actors/lookup", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "missing name", http.StatusBadRequest)
			return
		}
		if id, ok := as.LookupActorID(name); ok {
			resp := map[string]any{"id": id, "name": name}
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	// Correlation events: /actors/correlation?id=<cid>&n=<count>
	mux.HandleFunc("/actors/correlation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		cid := q.Get("id")
		if cid == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		n := 100
		if nStr := q.Get("n"); nStr != "" {
			if v, err := strconv.Atoi(nStr); err == nil && v > 0 {
				n = v
			}
		}
		// Reserved filters (actor, since, until) could be applied here
		_ = q.Get("actor")
		_ = q.Get("since")
		_ = q.Get("until")
		list := as.GetCorrelationEvents(cid, n)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = server.ListenAndServe() }()
	return server.Shutdown, nil
}

// StartDebugHTTPOn starts the debug HTTP server on an explicit listener address and returns
// the shutdown function along with the bound address string (useful when addr uses :0).
func StartDebugHTTPOn(as *ActorSystem, addr string) (shutdown func(ctx context.Context) error, boundAddr string, err error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/actors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		snap := as.GetSystemSnapshot()
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(snap)
	})

	mux.HandleFunc("/actors/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		idStr := q.Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		n := 100
		if nStr := q.Get("n"); nStr != "" {
			if v, err := strconv.Atoi(nStr); err == nil && v > 0 {
				n = v
			}
		}
		list := as.GetRecentMessages(ActorID(id64), n)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	mux.HandleFunc("/actors/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		g := as.BuildActorGraph()
		if limStr := q.Get("limit"); limStr != "" {
			if lim, err := strconv.Atoi(limStr); err == nil && lim >= 0 {
				if lim < len(g.Nodes) {
					g.Nodes = g.Nodes[:lim]
				}
				if lim < len(g.Edges) {
					g.Edges = g.Edges[:lim]
				}
			}
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(g)
	})

	mux.HandleFunc("/actors/deadlocks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		list := as.DetectPotentialDeadlocks()
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	mux.HandleFunc("/actors/correlation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		q := r.URL.Query()
		cid := q.Get("id")
		if cid == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		n := 100
		if nStr := q.Get("n"); nStr != "" {
			if v, err := strconv.Atoi(nStr); err == nil && v > 0 {
				n = v
			}
		}
		list := as.GetCorrelationEvents(cid, n)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(list)
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = server.Serve(ln) }()
	return server.Shutdown, ln.Addr().String(), nil
}
