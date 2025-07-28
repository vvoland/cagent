package server

import (
	"net"

	winio "github.com/Microsoft/go-winio"
)

func listenNamedPipe(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}
