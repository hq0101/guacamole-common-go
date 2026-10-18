package net_test

import (
	"testing"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestJavaStyleNetAliasesAndConstructors(t *testing.T) {
	socket := &mockSocket{}

	var guacSocket guacnet.GuacamoleSocket = socket
	if guacSocket != socket {
		t.Fatal("GuacamoleSocket alias did not preserve socket value")
	}
	if guacSocket.GetProtocol() != socket.Protocol() {
		t.Fatal("GuacamoleSocket.GetProtocol() did not delegate protocol")
	}
	if guacSocket.GetReader() == nil {
		t.Fatal("GuacamoleSocket.GetReader() returned nil")
	}
	if guacSocket.GetWriter() == nil {
		t.Fatal("GuacamoleSocket.GetWriter() returned nil")
	}

	tunnel := guacnet.NewSimpleGuacamoleTunnel(socket)
	var _ *guacnet.SimpleGuacamoleTunnel = tunnel
	if tunnel.GetSocket() != socket {
		t.Fatal("NewSimpleGuacamoleTunnel() wrapped wrong socket")
	}
	if tunnel.GetUUID() != tunnel.UUID() {
		t.Fatal("GetUUID() did not match UUID()")
	}

	delegating := guacnet.NewDelegatingGuacamoleTunnel(tunnel)
	var _ *guacnet.DelegatingGuacamoleTunnel = delegating
	if delegating.UUID() != tunnel.UUID() {
		t.Fatal("NewDelegatingGuacamoleTunnel() did not delegate UUID")
	}
	if delegating.GetUUID() != tunnel.GetUUID() {
		t.Fatal("NewDelegatingGuacamoleTunnel() did not delegate GetUUID")
	}

	filtered := guacnet.NewFilteredGuacamoleSocket(socket, nil, nil)
	var _ *guacnet.FilteredGuacamoleSocket = filtered
	if filtered.Protocol() != socket.Protocol() {
		t.Fatal("NewFilteredGuacamoleSocket() did not delegate socket")
	}
	if filtered.GetReader() == nil {
		t.Fatal("NewFilteredGuacamoleSocket().GetReader() returned nil")
	}
	if filtered.GetWriter() == nil {
		t.Fatal("NewFilteredGuacamoleSocket().GetWriter() returned nil")
	}

	delegatingSocket := guacnet.NewDelegatingGuacamoleSocket(socket)
	var _ *guacnet.DelegatingGuacamoleSocket = delegatingSocket
	if delegatingSocket.GetDelegateSocket() != socket {
		t.Fatal("NewDelegatingGuacamoleSocket() wrapped wrong socket")
	}
	if delegatingSocket.Protocol() != socket.Protocol() {
		t.Fatal("NewDelegatingGuacamoleSocket() did not delegate protocol")
	}
	if err := delegatingSocket.SendInstruction(protocol.NewInstruction("nop")); err != nil {
		t.Fatalf("SendInstruction() error = %v", err)
	}
	if len(socket.sendBuf) != 1 || socket.sendBuf[0].Opcode != "nop" {
		t.Fatalf("delegated send buffer = %+v, want nop instruction", socket.sendBuf)
	}

	var _ *guacnet.InetGuacamoleSocket
	var _ *guacnet.SSLGuacamoleSocket
	var _ *guacnet.FailoverGuacamoleSocket
	var _ *guacnet.ConfiguredGuacamoleSocket
}
