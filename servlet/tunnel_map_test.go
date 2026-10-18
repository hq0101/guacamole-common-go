package servlet_test

import (
	stdio "io"
	"sync/atomic"
	"testing"
	"time"

	guacio "github.com/hq0101/guacamole-common-go/io"
	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
	"github.com/hq0101/guacamole-common-go/servlet"
)

type closeTrackedSocket struct {
	closed atomic.Bool
}

func (s *closeTrackedSocket) SendInstruction(protocol.Instruction) error { return nil }

func (s *closeTrackedSocket) ReceiveInstruction() (protocol.Instruction, error) {
	return protocol.Instruction{}, stdio.EOF
}

func (s *closeTrackedSocket) Protocol() string { return "rdp" }

func (s *closeTrackedSocket) Close() error {
	s.closed.Store(true)
	return nil
}

func (s *closeTrackedSocket) IsOpen() bool { return !s.closed.Load() }

func (s *closeTrackedSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(s)
}

func (s *closeTrackedSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(s)
}

func TestHTTPTunnelMap_GetUpdatesLastAccessedTime(t *testing.T) {
	tunnelMap := servlet.NewHTTPTunnelMapWithTimeout(time.Hour, time.Hour)
	defer tunnelMap.Shutdown()

	tunnelMap.Put("token", guacnet.NewTunnel(newPipeSocket()))

	tunnel := tunnelMap.Get("token")
	if tunnel == nil {
		t.Fatal("Get() = nil, want tunnel")
	}
	first := tunnel.LastAccessedTime()
	if first.IsZero() {
		t.Fatal("LastAccessedTime() is zero after Get()")
	}

	time.Sleep(time.Millisecond)
	tunnel = tunnelMap.Get("token")
	if tunnel == nil {
		t.Fatal("second Get() = nil, want tunnel")
	}
	if !tunnel.LastAccessedTime().After(first) {
		t.Fatalf("LastAccessedTime() = %v, want after %v", tunnel.LastAccessedTime(), first)
	}
}

func TestHTTPTunnelMap_RemoveReturnsWrappedTunnel(t *testing.T) {
	tunnelMap := servlet.NewHTTPTunnelMapWithTimeout(time.Hour, time.Hour)
	defer tunnelMap.Shutdown()

	socket := &closeTrackedSocket{}
	base := guacnet.NewTunnel(socket)
	tunnelMap.Put("token", base)

	removed := tunnelMap.Remove("token")
	if removed == nil {
		t.Fatal("Remove() = nil, want tunnel")
	}
	if removed.GetSocket() != base.GetSocket() {
		t.Fatal("Remove() returned a tunnel wrapping the wrong socket")
	}
	if tunnelMap.Get("token") != nil {
		t.Fatal("Get() after Remove() returned tunnel, want nil")
	}
	if socket.closed.Load() {
		t.Fatal("Remove() closed tunnel, want only removal")
	}
}

func TestHTTPTunnelMap_ExpiresIdleTunnel(t *testing.T) {
	tunnelMap := servlet.NewHTTPTunnelMapWithTimeout(5*time.Millisecond, time.Millisecond)
	defer tunnelMap.Shutdown()

	socket := &closeTrackedSocket{}
	tunnelMap.Put("token", guacnet.NewTunnel(socket))

	waitForCondition(t, func() bool {
		return socket.closed.Load() && tunnelMap.Get("token") == nil
	})
}

func TestHTTPTunnelMap_ShutdownStopsExpiration(t *testing.T) {
	tunnelMap := servlet.NewHTTPTunnelMapWithTimeout(5*time.Millisecond, time.Millisecond)

	socket := &closeTrackedSocket{}
	tunnelMap.Put("token", guacnet.NewTunnel(socket))
	tunnelMap.Shutdown()

	time.Sleep(20 * time.Millisecond)
	if socket.closed.Load() {
		t.Fatal("tunnel closed after Shutdown(), want cleanup stopped")
	}
	if tunnelMap.Get("token") == nil {
		t.Fatal("Get() after Shutdown() = nil, want tunnel")
	}
}

func waitForCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if condition() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("condition was not satisfied")
		case <-ticker.C:
		}
	}
}
