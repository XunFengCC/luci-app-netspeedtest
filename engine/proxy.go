package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Proxy configuration deliberately accepts only a loopback HTTP listener. The
// helper owns the proxy lifecycle; allowing a URL or remote host here could
// route measurements through an unreviewed third party or leak credentials.
func parseLocalHTTPProxy(value string) (*url.URL, error) {
	if value == "" {
		return nil, nil
	}
	host, portText, err := net.SplitHostPort(value)
	if err != nil || host != "127.0.0.1" {
		return nil, fmt.Errorf("--http-proxy must be exactly 127.0.0.1:PORT")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("--http-proxy port must be between 1 and 65535")
	}
	return &url.URL{Scheme: "http", Host: net.JoinHostPort(host, portText)}, nil
}

func validateProxyOptions(cfg config) error {
	if _, err := parseLocalHTTPProxy(cfg.httpProxy); err != nil {
		return err
	}
	if cfg.httpProxy != "" && cfg.source != "" {
		return fmt.Errorf("--source cannot be combined with --http-proxy; the local proxy owns outbound routing")
	}
	if cfg.proxyName != "" && cfg.httpProxy == "" {
		return fmt.Errorf("--proxy-name requires --http-proxy")
	}
	if utf8.RuneCountInString(cfg.proxyName) > 128 || strings.ContainsAny(cfg.proxyName, "\r\n\x00") {
		return fmt.Errorf("--proxy-name must be at most 128 characters and contain no control characters")
	}
	return nil
}

func connectionMode(cfg config) string {
	if cfg.httpProxy != "" {
		return "proxy"
	}
	return "direct"
}
