//go:build !windows

package socket

import (
	"net"
)

// DialUnix is a simple wrapper for `net.Dial("unix")`.
func DialUnix(path string) (net.Conn, error) {
	return net.DialUnix("unix", nil, &net.UnixAddr{Name: stripUnixScheme(path), Net: "unix"})
}
