//go:build linux

package main

import (
	"fmt"
	"strings"
	"syscall"
)

// tcpReceiveBufferBytes is set per measurement HTTP socket, never through a
// global sysctl. The target router's net.core.rmem_max is 212992 bytes. An
// explicit SO_RCVBUF also locks Linux receive autotuning for that socket, so
// this cap is intentionally tied to that deployment and must be reevaluated
// if the kernel limit or WAN RTT changes.
const tcpReceiveBufferBytes = 212992

func socketReadBufferControl(bindControl func(string, string, syscall.RawConn) error) func(string, string, syscall.RawConn) error {
	return func(network, address string, raw syscall.RawConn) error {
		if bindControl != nil {
			if err := bindControl(network, address, raw); err != nil {
				return err
			}
		}
		if !strings.HasPrefix(network, "tcp") {
			return nil
		}
		var socketErr error
		if err := raw.Control(func(fd uintptr) {
			socketErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, tcpReceiveBufferBytes)
		}); err != nil {
			return fmt.Errorf("set per-socket receive buffer: %w", err)
		}
		if socketErr != nil {
			return fmt.Errorf("set per-socket receive buffer to %d bytes: %w", tcpReceiveBufferBytes, socketErr)
		}
		return nil
	}
}
