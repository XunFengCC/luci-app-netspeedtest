package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/librespeed/speedtest-cli/defs"
)

func TestParseLocalHTTPProxyRequiresLiteralLoopback(t *testing.T) {
	for _, value := range []string{"localhost:8080", "127.0.0.2:8080", "http://127.0.0.1:8080", "127.0.0.1:0", "127.0.0.1:65536"} {
		if _, err := parseLocalHTTPProxy(value); err == nil {
			t.Errorf("accepted proxy address %q", value)
		}
	}
	if _, err := parseLocalHTTPProxy("127.0.0.1:8080"); err != nil {
		t.Fatalf("valid loopback proxy rejected: %v", err)
	}
	for _, cfg := range []config{{proxyName: "label"}, {httpProxy: "127.0.0.1:8080", source: "eth0"}, {httpProxy: "127.0.0.1:8080", proxyName: "a\nsecret"}} {
		if err := validateProxyOptions(cfg); err == nil {
			t.Errorf("accepted invalid proxy options: %#v", cfg)
		}
	}
}

func TestHTTPProxyConnectRoutesTLSAndKeepsPeerSemantics(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests.Add(1)
		if r.URL.Path == "/empty.php" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer target.Close()
	targetPort := portOf(t, target.Listener.Addr())
	proxyAddr, closeProxy := startConnectProxy(t, targetPort, false)
	defer closeProxy()
	client, peer, err := newProviderClient(config{httpProxy: proxyAddr, proxyName: "local-mihomo", dns: "127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(peerRoundTripper).base.(*http.Transport)
	transport.TLSClientConfig = target.Client().Transport.(*http.Transport).TLSClientConfig
	p := provider{ID: "tunnel", Name: "test", Country: "Test", CountryCC: "ZZ", BaseURL: "https://example.com:" + strconv.Itoa(targetPort), PingURL: "empty.php"}
	_, _, _, _, err = p.server(client).PingAndJitterContext(context.Background(), 3)
	if err != nil {
		t.Fatalf("ping through CONNECT: %v", err)
	}
	info := p.info(peer.IP(), true)
	if info.PeerIP != "" || info.TransportPeerIP != "127.0.0.1" {
		t.Fatalf("proxy peer semantics = %#v", info)
	}
	if peer.IP() != "127.0.0.1" {
		t.Fatalf("connected transport peer = %q, want loopback proxy", peer.IP())
	}
	if targetRequests.Load() != 3 {
		t.Fatalf("target received %d requests, want three warm ping probes", targetRequests.Load())
	}
	// The target hostname deliberately has no DNS entry and resolver points at
	// a closed local port. Success demonstrates CONNECT leaves target resolution
	// to the configured proxy instead of the engine's direct resolver.
}

func TestProxyCONNECTFailureNeverFallsBackDirect(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	proxyAddr, closeProxy := startConnectProxy(t, portOf(t, target.Listener.Addr()), true)
	defer closeProxy()
	client, _, err := newProviderClient(config{httpProxy: proxyAddr, dns: "127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(peerRoundTripper).base.(*http.Transport)
	transport.TLSClientConfig = target.Client().Transport.(*http.Transport).TLSClientConfig
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://no-resolve.invalid:"+strconv.Itoa(portOf(t, target.Listener.Addr()))+"/", nil)
	if _, err := client.Do(req); err == nil {
		t.Fatal("request succeeded despite CONNECT rejection")
	}
	if targetRequests.Load() != 0 {
		t.Fatalf("target was reached after proxy rejection: %d requests", targetRequests.Load())
	}
}

func TestProxyRequestCancellationInterruptsConnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, e := listener.Accept()
		if e == nil {
			accepted <- c
		}
	}()
	client, _, err := newProviderClient(config{httpProxy: listener.Addr().String(), dns: "127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://no-resolve.invalid:443/", nil)
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("request unexpectedly survived stalled proxy CONNECT")
	}
	select {
	case conn := <-accepted:
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_, _ = bufio.NewReader(conn).ReadString('\n')
	case <-time.After(time.Second):
		t.Fatal("proxy did not receive CONNECT")
	}
}

func startConnectProxy(t *testing.T, targetPort int, reject bool) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					return
				}
				if req.Method != http.MethodConnect || !strings.HasSuffix(req.Host, ":"+strconv.Itoa(targetPort)) {
					_, _ = io.WriteString(conn, "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n")
					return
				}
				if reject {
					_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
					return
				}
				upstream, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(targetPort)), time.Second)
				if err != nil {
					_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
					return
				}
				defer upstream.Close()
				_, _ = io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
				_ = conn.SetDeadline(time.Time{})
				go func() { _, _ = io.Copy(upstream, conn); _ = upstream.Close() }()
				_, _ = io.Copy(conn, upstream)
			}()
		}
	}()
	return listener.Addr().String(), func() {
		_ = listener.Close()
		<-done
	}
}

func portOf(t *testing.T, addr net.Addr) int {
	t.Helper()
	_, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

var _ defs.ProgressCallback = func(float64, time.Duration) {}
var _ = fmt.Sprintf
