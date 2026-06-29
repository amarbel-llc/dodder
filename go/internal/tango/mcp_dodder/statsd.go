package mcp_dodder

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// statsd emission is best-effort observability for the MCP server. It exists
// so the build-per-call cost of OpenRepo (FDR-0019 #278) is measurable: if the
// timer shows repo construction dominating MCP latency, switch OpenRepo to the
// lazy per-repo cache (the FDR "MCP repo cache" lever). Dodder ships with no
// stats infrastructure, so this is deliberately tiny and entirely opt-in.

var (
	statsdOnce sync.Once
	// statsdAddr is the resolved UDP target, or "" when disabled. Resolved
	// once from STATSD_HOST/STATSD_PORT; empty STATSD_HOST (end users with no
	// stats-me running, and the bats sandbox) leaves emit a no-op.
	statsdAddr string
)

func statsdTarget() string {
	statsdOnce.Do(func() {
		host := os.Getenv("STATSD_HOST")
		if host == "" {
			return
		}

		port := os.Getenv("STATSD_PORT")
		if port == "" {
			port = "8125"
		}

		statsdAddr = net.JoinHostPort(host, port)
	})

	return statsdAddr
}

// emitTiming sends a statsd timer (`name:<ms>|ms`) over UDP. It is
// best-effort: a no-op when stats-me is not configured (STATSD_HOST unset),
// and it swallows every error so it can never block or fail the caller. The
// stats-me VictoriaMetrics ingest exposes timers as `stats.timers.<name>.*`.
func emitTiming(name string, d time.Duration) {
	addr := statsdTarget()
	if addr == "" {
		return
	}

	// UDP "dial" sets up the socket without a handshake, so this is fast;
	// the timeout only guards a pathological resolver.
	conn, err := net.DialTimeout("udp", addr, 100*time.Millisecond)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	_, _ = fmt.Fprintf(conn, "%s:%d|ms", name, d.Milliseconds())
}
