package main

import (
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestPeerIPUsesSuccessfulIPv4AfterAAAAUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
			accepted <- struct{}{}
		}
	}()

	dnsServer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer dnsServer.Close()
	var sawAAAA atomic.Bool
	go serveFallbackDNS(dnsServer, &sawAAAA)
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, dnsServer.LocalAddr().String())
	}}
	dialer := &net.Dialer{Resolver: resolver, Timeout: 2 * time.Second, FallbackDelay: 25 * time.Millisecond}
	peer := &peerRecorder{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := recordPeerDialContext(dialer.DialContext, peer)(ctx, "tcp", "fallback.test.:"+strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatalf("dial with unreachable AAAA and reachable A: %v", err)
	}
	defer conn.Close()
	<-accepted
	if !sawAAAA.Load() {
		t.Fatal("fake DNS did not return an AAAA candidate")
	}
	remote, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("remote endpoint type = %T", conn.RemoteAddr())
	}
	if got := peer.IP(); got != remote.IP.String() || got != "127.0.0.1" {
		t.Fatalf("recorded peer IP = %q; actual successful socket peer is %q", got, remote.IP)
	}
}

func serveFallbackDNS(conn *net.UDPConn, sawAAAA *atomic.Bool) {
	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		packet := append([]byte(nil), buf[:n]...)
		response, qtype, ok := fallbackDNSAnswer(packet)
		if !ok {
			continue
		}
		if qtype == 28 {
			sawAAAA.Store(true)
		}
		_, _ = conn.WriteToUDP(response, addr)
	}
}

func fallbackDNSAnswer(query []byte) ([]byte, uint16, bool) {
	if len(query) < 17 {
		return nil, 0, false
	}
	offset := 12
	for {
		if offset >= len(query) {
			return nil, 0, false
		}
		labelLen := int(query[offset])
		offset++
		if labelLen == 0 {
			break
		}
		if labelLen&0xc0 != 0 || offset+labelLen > len(query) {
			return nil, 0, false
		}
		offset += labelLen
	}
	if offset+4 > len(query) {
		return nil, 0, false
	}
	qtype := binary.BigEndian.Uint16(query[offset : offset+2])
	questionEnd := offset + 4
	var rdata []byte
	switch qtype {
	case 1:
		rdata = []byte{127, 0, 0, 1}
	case 28:
		if ip := net.ParseIP("2001:db8::1").To16(); ip != nil {
			rdata = ip
		}
	default:
		return nil, 0, false
	}
	response := make([]byte, 12, questionEnd+16+len(rdata))
	copy(response[:2], query[:2])
	response[2], response[3] = 0x81, 0x80 // response, recursion available
	response[5], response[7] = 1, 1       // one question and one answer
	response = append(response, query[12:questionEnd]...)
	response = append(response, 0xc0, 0x0c, byte(qtype>>8), byte(qtype), 0, 1, 0, 0, 0, 30, byte(len(rdata)>>8), byte(len(rdata)))
	response = append(response, rdata...)
	return response, qtype, true
}
