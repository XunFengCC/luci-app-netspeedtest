//go:build !linux

package main

import (
	"errors"
	"syscall"
)

func bindInterfaceControl(name string) (func(string, string, syscall.RawConn) error, error) {
	return nil, errors.New("binding to a network interface is only supported on Linux")
}
