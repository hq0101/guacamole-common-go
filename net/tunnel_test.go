package net_test

import (
	"errors"
	"testing"
	"time"

	guacio "github.com/hq0101/guacamole-common-go/io"
	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// mockSocket implements Socket for testing.
type mockSocket struct {
	sendBuf   []protocol.Instruction
	recvQueue []protocol.Instruction
	closed    bool
	proto     string
}

func (m *mockSocket) SendInstruction(i protocol.Instruction) error {
	if m.closed {
		return errors.New("closed")
	}
	m.sendBuf = append(m.sendBuf, i)
	return nil
}
func (m *mockSocket) ReceiveInstruction() (protocol.Instruction, error) {
	if len(m.recvQueue) == 0 {
		return protocol.Instruction{}, &protocol.Error{Status: protocol.StatusResourceClosed, Message: "no more"}
	}
	i := m.recvQueue[0]
	m.recvQueue = m.recvQueue[1:]
	return i, nil
}
func (m *mockSocket) Protocol() string { return m.proto }
func (m *mockSocket) GetProtocol() string {
	return m.Protocol()
}
func (m *mockSocket) Close() error { m.closed = true; return nil }
func (m *mockSocket) IsOpen() bool { return !m.closed }
func (m *mockSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(m)
}
func (m *mockSocket) GetReader() guacio.GuacamoleReader {
	return m.Reader()
}
func (m *mockSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(m)
}
func (m *mockSocket) GetWriter() guacio.GuacamoleWriter {
	return m.Writer()
}

func TestSimpleTunnel_UUID(t *testing.T) {
	s := &mockSocket{}
	t1 := guacnet.NewTunnel(s)
	t2 := guacnet.NewTunnel(s)
	if t1.UUID() == t2.UUID() {
		t.Error("two tunnels should have distinct UUIDs")
	}
	if t1.UUID().String() == "" {
		t.Error("UUID should not be empty")
	}
}

func TestSimpleTunnel_DelegatesToSocket(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{{Opcode: "sync", Args: []string{"0"}}},
	}
	tun := guacnet.NewTunnel(s)

	// Send
	if err := tun.SendInstruction(protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}); err != nil {
		t.Fatalf("SendInstruction: %v", err)
	}
	if len(s.sendBuf) != 1 || s.sendBuf[0].Opcode != "key" {
		t.Errorf("expected key instruction in send buffer")
	}

	// Receive
	got, err := tun.ReceiveInstruction()
	if err != nil || got.Opcode != "sync" {
		t.Errorf("ReceiveInstruction: err=%v opcode=%q", err, got.Opcode)
	}
}

func TestSimpleTunnel_AcquireReaderIsExclusive(t *testing.T) {
	tun := guacnet.NewTunnel(&mockSocket{})

	first := tun.AcquireReader()
	if first == nil {
		t.Fatal("AcquireReader() = nil")
	}

	acquired := make(chan struct{})
	go func() {
		reader := tun.AcquireReader()
		if reader == nil {
			t.Errorf("second AcquireReader() = nil")
		}
		close(acquired)
		tun.ReleaseReader()
	}()

	waitForQueued(t, tun.HasQueuedReaderThreads)
	select {
	case <-acquired:
		t.Fatal("second reader acquired before first was released")
	default:
	}

	tun.ReleaseReader()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second reader did not acquire after release")
	}
	if tun.HasQueuedReaderThreads() {
		t.Fatal("HasQueuedReaderThreads() = true after queued reader acquired")
	}
}

func TestSimpleTunnel_AcquireWriterIsExclusive(t *testing.T) {
	tun := guacnet.NewTunnel(&mockSocket{})

	first := tun.AcquireWriter()
	if first == nil {
		t.Fatal("AcquireWriter() = nil")
	}

	acquired := make(chan struct{})
	go func() {
		writer := tun.AcquireWriter()
		if writer == nil {
			t.Errorf("second AcquireWriter() = nil")
		}
		close(acquired)
		tun.ReleaseWriter()
	}()

	waitForQueued(t, tun.HasQueuedWriterThreads)
	select {
	case <-acquired:
		t.Fatal("second writer acquired before first was released")
	default:
	}

	tun.ReleaseWriter()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second writer did not acquire after release")
	}
	if tun.HasQueuedWriterThreads() {
		t.Fatal("HasQueuedWriterThreads() = true after queued writer acquired")
	}
}

func TestDelegatingTunnelDelegates(t *testing.T) {
	base := guacnet.NewTunnel(&mockSocket{})
	tun := guacnet.NewDelegatingTunnel(base)

	if tun.UUID() != base.UUID() {
		t.Fatalf("UUID() = %v, want %v", tun.UUID(), base.UUID())
	}
	if tun.GetSocket() != base.GetSocket() {
		t.Fatal("GetSocket() did not return delegate socket")
	}
	if tun.AcquireReader() == nil {
		t.Fatal("AcquireReader() = nil")
	}
	tun.ReleaseReader()
	if tun.AcquireWriter() == nil {
		t.Fatal("AcquireWriter() = nil")
	}
	tun.ReleaseWriter()
}

func TestInternalDataOpcode(t *testing.T) {
	if guacnet.InternalDataOpcode != "" {
		t.Errorf("InternalDataOpcode = %q, want empty string", guacnet.InternalDataOpcode)
	}
}

func waitForQueued(t *testing.T, queued func() bool) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if queued() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("queued state was not observed")
		case <-ticker.C:
		}
	}
}
