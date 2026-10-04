//go:build linux

package main

import (
	"net"
	"syscall"
	"testing"
)

func TestSocketReadBufferControlSetsPerConnectionReceiveBuffer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	dialer := &net.Dialer{Control: socketReadBufferControl(nil)}
	client, err := dialer.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if conn := <-accepted; conn != nil {
		defer conn.Close()
	}
	tcpConn, ok := client.(*net.TCPConn)
	if !ok {
		t.Fatalf("dial returned %T, want TCPConn", client)
	}
	raw, err := tcpConn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var actual int
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		actual, socketErr = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	}); err != nil {
		t.Fatal(err)
	}
	if socketErr != nil {
		t.Fatal(socketErr)
	}
	if actual < tcpReceiveBufferBytes || actual > tcpReceiveBufferBytes*2 {
		t.Fatalf("SO_RCVBUF = %d, want kernel-applied value in [%d,%d]", actual, tcpReceiveBufferBytes, tcpReceiveBufferBytes*2)
	}
}
