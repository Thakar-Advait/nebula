package overlay

import (
	"io"
	"net/netip"

	"github.com/slackhq/nebula/routing"
)

type Device interface {
	io.ReadWriteCloser
	Activate() error
	// [DNS] Add RevertDns() error method
	RevertDns() error
	Networks() []netip.Prefix
	Name() string
	RoutesFor(netip.Addr) routing.Gateways
	NewMultiQueueReader() (io.ReadWriteCloser, error)
}
