package net

import (
	"context"
	"crypto/tls"
	stdnet "net"
	"strconv"

	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

type GuacamoleSocket interface {
	Socket
	GetProtocol() string
	GetReader() guacio.GuacamoleReader
	GetWriter() guacio.GuacamoleWriter
}

type DelegatingGuacamoleSocket = DelegatingSocket
type InetGuacamoleSocket = InetSocket
type SSLGuacamoleSocket = TLSSocket
type FilteredGuacamoleSocket = FilteredSocket
type FailoverGuacamoleSocket = FailoverSocket
type ConfiguredGuacamoleSocket = ConfiguredSocket

type GuacamoleTunnel = Tunnel
type SimpleGuacamoleTunnel = SimpleTunnel
type DelegatingGuacamoleTunnel = DelegatingTunnel

// NewInetGuacamoleSocket connects to guacd using Java-style host/port
// arguments.
func NewInetGuacamoleSocket(ctx context.Context, hostname string, port int) (*InetGuacamoleSocket, error) {
	return Dial(ctx, stdnet.JoinHostPort(hostname, strconv.Itoa(port)))
}

// NewDelegatingGuacamoleSocket returns a Java-style delegating socket wrapper.
func NewDelegatingGuacamoleSocket(socket Socket) *DelegatingGuacamoleSocket {
	return NewDelegatingSocket(socket)
}

// NewSSLGuacamoleSocket connects to guacd over TLS using Java-style host/port
// arguments.
func NewSSLGuacamoleSocket(ctx context.Context, hostname string, port int, cfg *tls.Config) (*SSLGuacamoleSocket, error) {
	return DialTLS(ctx, stdnet.JoinHostPort(hostname, strconv.Itoa(port)), cfg)
}

// NewFilteredGuacamoleSocket returns a Java-style filtered socket wrapper.
func NewFilteredGuacamoleSocket(socket Socket, readFilter, writeFilter protocol.Filter) *FilteredGuacamoleSocket {
	return NewFilteredSocket(socket, readFilter, writeFilter)
}

// NewFailoverGuacamoleSocket returns a Java-style failover socket wrapper.
func NewFailoverGuacamoleSocket(socket Socket) (*FailoverGuacamoleSocket, error) {
	return NewFailoverSocket(socket)
}

// NewFailoverGuacamoleSocketWithLimit returns a Java-style failover socket
// wrapper using a custom instruction queue limit.
func NewFailoverGuacamoleSocketWithLimit(socket Socket, instructionQueueLimit int) (*FailoverGuacamoleSocket, error) {
	return NewFailoverSocketWithLimit(socket, instructionQueueLimit)
}

// NewConfiguredGuacamoleSocket returns a Java-style configured socket wrapper.
func NewConfiguredGuacamoleSocket(socket Socket, cfg *protocol.Config, info *protocol.ClientInfo) (*ConfiguredGuacamoleSocket, error) {
	return NewConfiguredSocket(socket, cfg, info)
}

// NewSimpleGuacamoleTunnel returns a Java-style simple tunnel wrapper.
func NewSimpleGuacamoleTunnel(socket Socket) *SimpleGuacamoleTunnel {
	return NewTunnel(socket)
}

// NewDelegatingGuacamoleTunnel returns a Java-style delegating tunnel wrapper.
func NewDelegatingGuacamoleTunnel(tunnel Tunnel) *DelegatingGuacamoleTunnel {
	return NewDelegatingTunnel(tunnel)
}
