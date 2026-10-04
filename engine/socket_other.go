//go:build !linux

package main

import "syscall"

// Per-socket SO_RCVBUF tuning is limited to the Linux router target. This
// keeps development-host behavior unchanged and leaves interface binding's
// platform-specific validation in bindInterfaceControl.
func socketReadBufferControl(bindControl func(string, string, syscall.RawConn) error) func(string, string, syscall.RawConn) error {
	return bindControl
}
