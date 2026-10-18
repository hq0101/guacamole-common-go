package net

import (
	"context"

	"github.com/hq0101/guacamole-common-go/protocol"
)

// Connect dials guacd at addr, performs the full Guacamole handshake, and
// returns a ready-to-use SimpleTunnel. The caller is responsible for closing
// the tunnel when done.
func Connect(ctx context.Context, addr string, cfg *protocol.Config, info *protocol.ClientInfo) (*SimpleTunnel, error) {
	sock, err := Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	cs, err := NewConfiguredSocket(sock, cfg, info)
	if err != nil {
		sock.Close()
		return nil, err
	}
	return NewTunnel(cs), nil
}
