//go:build linux

package main

import (
	"fmt"
	"syscall"
)

func bindInterfaceControl(name string) (func(string, string, syscall.RawConn) error, error) {
	return func(_, _ string, raw syscall.RawConn) error {
		var socketErr error
		if err := raw.Control(func(fd uintptr) {
			socketErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, name)
		}); err != nil {
			return err
		}
		if socketErr != nil {
			return fmt.Errorf("bind socket to interface %q: %w", name, socketErr)
		}
		return nil
	}, nil
}
