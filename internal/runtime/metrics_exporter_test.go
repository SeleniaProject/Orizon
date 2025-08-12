package runtime

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStartMetricsServer_ServesMetrics(t *testing.T) {
	collectors := map[string]MetricFunc{
		"testCollector": func() map[string]float64 {
			return map[string]float64{"requests_total": 123, "latency_ms": 4.5}
		},
	}
	addr, stop, err := StartMetricsServer(":0", collectors)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = stop(context.Background()) }()

	cli := &http.Client{Timeout: 2 * time.Second}
	resp, err := cli.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %v", resp.Status)
	}
	// Read a few lines and ensure our metric names appear
	rd := bufio.NewReader(resp.Body)
	var got string
	for i := 0; i < 5; i++ {
		line, _, err := rd.ReadLine()
		if err != nil {
			break
		}
		got += string(line) + "\n"
	}
	if !strings.Contains(got, "testCollector_requests_total") {
		t.Fatalf("missing metric name, got: %q", got)
	}
}

func TestSanitizeMetricToken(t *testing.T) {
	in := " metric name (bad)!"
	out := sanitizeMetricToken(in)
	if strings.ContainsAny(out, " !()") {
		t.Fatalf("token not sanitized: %q", out)
	}
	if out == "" {
		t.Fatalf("empty token")
	}
}
