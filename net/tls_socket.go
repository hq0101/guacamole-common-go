package net

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/hq0101/guacamole-common-go/protocol"
)

// TLSSocket is a TLS TCP Socket connecting to guacd.
type TLSSocket struct {
	*InetSocket
}

// DialTLS connects to guacd at addr over TLS.
func DialTLS(ctx context.Context, addr string, cfg *tls.Config) (*TLSSocket, error) {
	d := tls.Dialer{
		NetDialer: nil,
		Config:    cfg,
	}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, &protocol.Error{
			Status:  protocol.StatusUpstreamUnavailable,
			Message: fmt.Sprintf("failed to connect to guacd (TLS) at %s: %v", addr, err),
		}
	}
	return &TLSSocket{NewInetSocketFromConn(conn)}, nil
}
