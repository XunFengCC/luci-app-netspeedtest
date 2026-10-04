package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
)

type peerRecorder struct {
	mu sync.RWMutex
	ip string
}

func (p *peerRecorder) set(addr net.Addr) {
	if addr == nil {
		return
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return
	}
	p.mu.Lock()
	p.ip = strings.Trim(host, "[]")
	p.mu.Unlock()
}

func (p *peerRecorder) IP() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ip
}

type peerRoundTripper struct {
	base      http.RoundTripper
	userAgent string
}

func (t peerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.userAgent != "" {
		req = req.Clone(req.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(req)
}

// recordPeerDialContext records the remote endpoint only after DialContext
// returns the actual connected TCP socket. This follows Happy Eyeballs fallback
// to its successful address instead of reporting the first DNS answer.
func recordPeerDialContext(dial func(context.Context, string, string) (net.Conn, error), peer *peerRecorder) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err == nil && strings.HasPrefix(network, "tcp") {
			peer.set(conn.RemoteAddr())
		}
		return conn, err
	}
}

func newProviderClient(cfg config, userAgents ...string) (*http.Client, *peerRecorder, error) {
	peer := &peerRecorder{}
	userAgent := ""
	if len(userAgents) > 0 {
		userAgent = userAgents[0]
	}
	var localTCP net.Addr
	var localUDP net.Addr
	var control func(string, string, syscall.RawConn) error
	if cfg.source != "" {
		if ip := net.ParseIP(cfg.source); ip != nil {
			localTCP = &net.TCPAddr{IP: ip}
			localUDP = &net.UDPAddr{IP: ip}
		} else {
			if _, err := net.InterfaceByName(cfg.source); err != nil {
				return nil, nil, fmt.Errorf("source %q is neither an IP address nor an interface: %w", cfg.source, err)
			}
			var err error
			control, err = bindInterfaceControl(cfg.source)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		var local net.Addr
		if strings.HasPrefix(network, "tcp") {
			local = localTCP
		} else {
			local = localUDP
		}
		d := &net.Dialer{Timeout: 3 * time.Second, LocalAddr: local, Control: control}
		return d.DialContext(ctx, network, cfg.dns)
	}}
	proxyURL, err := parseLocalHTTPProxy(cfg.httpProxy)
	if err != nil {
		return nil, nil, err
	}
	socketControl := socketReadBufferControl(control)
	if proxyURL != nil {
		// The WAN socket belongs to the proxy process, so a receive-buffer
		// request here only affects the loopback hop and can be misleading.
		socketControl = control
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second, LocalAddr: localTCP, Resolver: resolver, Control: socketControl}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	transport.DialContext = recordPeerDialContext(dialer.DialContext, peer)
	transport.MaxIdleConns = maxConnections * 2
	transport.MaxIdleConnsPerHost = maxConnections
	transport.MaxConnsPerHost = maxConnections
	transport.IdleConnTimeout = 30 * time.Second
	transport.ResponseHeaderTimeout = 5 * time.Second
	transport.DisableCompression = true
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create HTTP cookie jar: %w", err)
	}
	client := &http.Client{Transport: peerRoundTripper{base: transport, userAgent: userAgent}, Jar: jar}
	return client, peer, nil
}

func absoluteURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid absolute URL %q", raw)
	}
	return u, nil
}

func responseIsHTML(resp *http.Response) bool {
	return strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html")
}
