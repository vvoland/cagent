//go:build !windows

package server

import (
	"fmt"
	"net"
	"runtime"
)

func listenNamedPipe(string) (net.Listener, error) {
	return nil, fmt.Errorf("named pipes not supported on %s", runtime.GOOS)
}
