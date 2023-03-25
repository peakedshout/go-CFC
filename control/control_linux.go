//go:build linux

package control

import "syscall"

func NetControl(network string, address string, c syscall.RawConn) error {
	return c.Control(func(fd uintptr) {
		//syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
}
func NetFdControl(fd uintptr) {
	syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
