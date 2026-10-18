package websocket_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	guacio "github.com/hq0101/guacamole-common-go/io"
	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
	guacws "github.com/hq0101/guacamole-common-go/websocket"
)

// pipeSocket simulates a connected guacd (sends/receives instructions via channels).
type pipeSocket struct {
	toClient   chan protocol.Instruction
	fromClient chan protocol.Instruction
	open       atomic.Bool
}

func newPipeSocket() *pipeSocket {
	ps := &pipeSocket{
		toClient:   make(chan protocol.Instruction, 10),
		fromClient: make(chan protocol.Instruction, 10),
	}
	ps.open.Store(true)
	return ps
}
func (p *pipeSocket) SendInstruction(i protocol.Instruction) error {
	p.fromClient <- i
	return nil
}
func (p *pipeSocket) ReceiveInstruction() (protocol.Instruction, error) {
	i, ok := <-p.toClient
	if !ok {
		return protocol.Instruction{}, &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"}
	}
	return i, nil
}
func (p *pipeSocket) Protocol() string { return "rdp" }
func (p *pipeSocket) IsOpen() bool     { return p.open.Load() }
func (p *pipeSocket) Close() error {
	if p.open.Swap(false) {
		close(p.toClient)
	}
	return nil
}
func (p *pipeSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(p)
}
func (p *pipeSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(p)
}

type failingReadSocket struct {
	open atomic.Bool
	err  error
}

func newFailingReadSocket(err error) *failingReadSocket {
	s := &failingReadSocket{err: err}
	s.open.Store(true)
	return s
}

func (s *failingReadSocket) SendInstruction(i protocol.Instruction) error { return nil }
func (s *failingReadSocket) ReceiveInstruction() (protocol.Instruction, error) {
	return protocol.Instruction{}, s.err
}
func (s *failingReadSocket) Protocol() string { return "rdp" }
func (s *failingReadSocket) IsOpen() bool     { return s.open.Load() }
func (s *failingReadSocket) Close() error {
	s.open.Store(false)
	return nil
}
func (s *failingReadSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(s)
}
func (s *failingReadSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(s)
}

type resourceClosedWriteSocket struct {
	toClient chan protocol.Instruction
	open     atomic.Bool
}

func newResourceClosedWriteSocket() *resourceClosedWriteSocket {
	s := &resourceClosedWriteSocket{toClient: make(chan protocol.Instruction, 10)}
	s.open.Store(true)
	return s
}

func (s *resourceClosedWriteSocket) SendInstruction(protocol.Instruction) error {
	return &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"}
}

func (s *resourceClosedWriteSocket) ReceiveInstruction() (protocol.Instruction, error) {
	i, ok := <-s.toClient
	if !ok {
		return protocol.Instruction{}, &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"}
	}
	return i, nil
}

func (s *resourceClosedWriteSocket) Protocol() string { return "rdp" }
func (s *resourceClosedWriteSocket) IsOpen() bool     { return s.open.Load() }
func (s *resourceClosedWriteSocket) Close() error {
	if s.open.Swap(false) {
		close(s.toClient)
	}
	return nil
}
func (s *resourceClosedWriteSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(s)
}
func (s *resourceClosedWriteSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(s)
}

type trackedTunnel struct {
	guacnet.Tunnel
	acquireReader atomic.Int32
	releaseReader atomic.Int32
	acquireWriter atomic.Int32
	releaseWriter atomic.Int32
}

func (t *trackedTunnel) AcquireReader() guacio.GuacamoleReader {
	t.acquireReader.Add(1)
	return t.Tunnel.AcquireReader()
}

func (t *trackedTunnel) ReleaseReader() {
	t.releaseReader.Add(1)
	t.Tunnel.ReleaseReader()
}

func (t *trackedTunnel) AcquireWriter() guacio.GuacamoleWriter {
	t.acquireWriter.Add(1)
	return t.Tunnel.AcquireWriter()
}

func (t *trackedTunnel) ReleaseWriter() {
	t.releaseWriter.Add(1)
	t.Tunnel.ReleaseWriter()
}

type scriptedSocket struct {
	reader *scriptedReader
	open   atomic.Bool
}

func newScriptedSocket(instructions ...protocol.Instruction) *scriptedSocket {
	socket := &scriptedSocket{reader: &scriptedReader{instructions: instructions}}
	socket.open.Store(true)
	return socket
}

func (s *scriptedSocket) SendInstruction(protocol.Instruction) error {
	return nil
}

func (s *scriptedSocket) ReceiveInstruction() (protocol.Instruction, error) {
	return s.reader.ReadInstruction()
}

func (s *scriptedSocket) Protocol() string { return "rdp" }

func (s *scriptedSocket) Close() error {
	s.open.Store(false)
	return nil
}

func (s *scriptedSocket) IsOpen() bool { return s.open.Load() }

func (s *scriptedSocket) Reader() guacio.GuacamoleReader {
	return s.reader
}

func (s *scriptedSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(s)
}

type scriptedReader struct {
	instructions []protocol.Instruction
	pos          int
}

func (r *scriptedReader) Available() (bool, error) {
	return r.pos < len(r.instructions), nil
}

func (r *scriptedReader) Read() ([]rune, error) {
	instr, err := r.ReadInstruction()
	if err != nil {
		return nil, err
	}
	return instr.ToCharArray(), nil
}

func (r *scriptedReader) ReadInstruction() (protocol.Instruction, error) {
	if r.pos >= len(r.instructions) {
		return protocol.Instruction{}, &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"}
	}
	instr := r.instructions[r.pos]
	r.pos++
	return instr, nil
}

func TestHandler_UUIDFrame(t *testing.T) {
	ps := newPipeSocket()
	tun := guacnet.NewTunnel(ps)
	uuidStr := tun.UUID().String()

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer ps.Close()

	wsURL := "ws" + srv.URL[4:] + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// First message from server should be the UUID frame: "0.,<len>.<uuid>;"
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}
	p := &protocol.Parser{}
	p.Append(msg)
	frame, ok := p.Next()
	if !ok || frame.Opcode != "" {
		t.Fatalf("expected internal opcode '', got %q", frame.Opcode)
	}
	if len(frame.Args) == 0 || frame.Args[0] != uuidStr {
		t.Errorf("UUID arg = %q, want %q", frame.Args[0], uuidStr)
	}
}

func TestHandler_OutboundMessagesAreBufferedUntilReaderUnavailable(t *testing.T) {
	socket := newScriptedSocket(
		protocol.NewInstruction("blob", "first"),
		protocol.NewInstruction("sync", "0"),
	)
	tun := guacnet.NewTunnel(socket)

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read buffered frame: %v", err)
	}

	got := string(msg)
	if !strings.Contains(got, "4.blob") || !strings.Contains(got, "4.sync") {
		t.Fatalf("buffered frame = %q, want blob and sync instructions together", got)
	}
}

func TestHandler_UsesTunnelAcquireReaderAndWriter(t *testing.T) {
	ps := newPipeSocket()
	base := guacnet.NewTunnel(ps)
	tun := &trackedTunnel{Tunnel: base}

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer ps.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	waitForAtomicValue(t, &tun.acquireReader, 1)

	key := protocol.NewInstruction("key", "65", "1")
	if err := conn.WriteMessage(websocket.TextMessage, key.Encode()); err != nil {
		t.Fatalf("write key: %v", err)
	}

	select {
	case got := <-ps.fromClient:
		if got.Opcode != "key" {
			t.Fatalf("forwarded opcode = %q, want key", got.Opcode)
		}
	case <-time.After(time.Second):
		t.Fatal("key instruction was not forwarded")
	}

	waitForAtomicValue(t, &tun.acquireWriter, 1)
	waitForAtomicValue(t, &tun.releaseWriter, 1)
}

func TestHandler_ConnectErrorClosesWithGuacamoleStatus(t *testing.T) {
	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return nil, &protocol.Error{Status: protocol.StatusClientUnauthorized, Message: "denied"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, _, err = conn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("ReadMessage() error = %T %v, want *websocket.CloseError", err, err)
	}
	if closeErr.Code != protocol.StatusClientUnauthorized.WebSocketCode() {
		t.Errorf("close code = %d, want %d", closeErr.Code, protocol.StatusClientUnauthorized.WebSocketCode())
	}
	if closeErr.Text != "769" {
		t.Errorf("close reason = %q, want 769", closeErr.Text)
	}
}

func TestHandler_WrappedConnectErrorClosesWithGuacamoleStatus(t *testing.T) {
	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return nil, fmt.Errorf("connect failed: %w", &protocol.Error{Status: protocol.StatusClientUnauthorized, Message: "denied"})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, _, err = conn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("ReadMessage() error = %T %v, want *websocket.CloseError", err, err)
	}
	if closeErr.Code != protocol.StatusClientUnauthorized.WebSocketCode() {
		t.Errorf("close code = %d, want %d", closeErr.Code, protocol.StatusClientUnauthorized.WebSocketCode())
	}
	if closeErr.Text != "769" {
		t.Errorf("close reason = %q, want 769", closeErr.Text)
	}
}

func TestHandler_TunnelReadErrorClosesWithGuacamoleStatus(t *testing.T) {
	socketErr := &protocol.Error{Status: protocol.StatusClientTimeout, Message: "client timed out"}
	tun := guacnet.NewTunnel(newFailingReadSocket(socketErr))

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// First frame is the UUID internal instruction.
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	_, _, err = conn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("ReadMessage() error = %T %v, want *websocket.CloseError", err, err)
	}
	if closeErr.Code != protocol.StatusClientTimeout.WebSocketCode() {
		t.Errorf("close code = %d, want %d", closeErr.Code, protocol.StatusClientTimeout.WebSocketCode())
	}
	if closeErr.Text != "776" {
		t.Errorf("close reason = %q, want 776", closeErr.Text)
	}
}

func TestHandler_TunnelResourceClosedClosesSuccessfully(t *testing.T) {
	socketErr := &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"}
	tun := guacnet.NewTunnel(newFailingReadSocket(socketErr))

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	_, _, err = conn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("ReadMessage() error = %T %v, want *websocket.CloseError", err, err)
	}
	if closeErr.Code != protocol.StatusSuccess.WebSocketCode() {
		t.Errorf("close code = %d, want %d", closeErr.Code, protocol.StatusSuccess.WebSocketCode())
	}
	if closeErr.Text != "0" {
		t.Errorf("close reason = %q, want 0", closeErr.Text)
	}
}

func TestHandler_InboundResourceClosedWriteDoesNotCloseWebSocket(t *testing.T) {
	socket := newResourceClosedWriteSocket()
	tun := guacnet.NewTunnel(socket)

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer socket.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	if err := conn.WriteMessage(websocket.TextMessage, key.Encode()); err != nil {
		t.Fatalf("write key: %v", err)
	}

	socket.toClient <- protocol.Instruction{Opcode: "sync", Args: []string{"0"}}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read sync after write ResourceClosed: %v", err)
	}
	p := &protocol.Parser{}
	p.Append(msg)
	instr, ok := p.Next()
	if !ok || instr.Opcode != "sync" {
		t.Fatalf("got opcode=%q args=%v, want sync", instr.Opcode, instr.Args)
	}
}

func TestHandler_PingPong(t *testing.T) {
	ps := newPipeSocket()
	tun := guacnet.NewTunnel(ps)

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer ps.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Consume UUID frame
	conn.ReadMessage() //nolint:errcheck

	// Send Java-compatible ping: opcode="" args=["ping", timestamp]
	ping := protocol.Instruction{Opcode: "", Args: []string{"ping", "12345"}}
	conn.WriteMessage(websocket.TextMessage, ping.Encode()) //nolint:errcheck

	// Expect ping response back: opcode="" args=["ping", same timestamp]
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ping response: %v", err)
	}
	p := &protocol.Parser{}
	p.Append(msg)
	pingResponse, ok := p.Next()
	if !ok || pingResponse.Opcode != "" || len(pingResponse.Args) != 2 ||
		pingResponse.Args[0] != "ping" || pingResponse.Args[1] != "12345" {
		t.Errorf("expected ping response, got opcode=%q args=%v", pingResponse.Opcode, pingResponse.Args)
	}

	// Verify ping was NOT forwarded to guacd
	select {
	case i := <-ps.fromClient:
		t.Errorf("ping should not be forwarded, got %+v", i)
	default:
	}
}

func TestHandler_InvalidInboundMessageDoesNotCloseWebSocket(t *testing.T) {
	ps := newPipeSocket()
	tun := guacnet.NewTunnel(ps)

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer ps.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("x.foo;")); err != nil {
		t.Fatalf("write invalid message: %v", err)
	}
	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	if err := conn.WriteMessage(websocket.TextMessage, key.Encode()); err != nil {
		t.Fatalf("write key after invalid message: %v", err)
	}

	select {
	case got := <-ps.fromClient:
		if got.Opcode != "key" {
			t.Fatalf("forwarded opcode = %q, want key", got.Opcode)
		}
	case <-time.After(time.Second):
		t.Fatal("valid instruction after invalid message was not forwarded")
	}
}

func TestHandler_InboundInstructionCannotSpanWebSocketMessages(t *testing.T) {
	ps := newPipeSocket()
	tun := guacnet.NewTunnel(ps)

	handler := guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer ps.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[4:]+"/", http.Header{
		"Sec-WebSocket-Protocol": []string{"guacamole"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read uuid frame: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("3.fo")); err != nil {
		t.Fatalf("write partial message: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("o;")); err != nil {
		t.Fatalf("write second partial message: %v", err)
	}

	select {
	case got := <-ps.fromClient:
		t.Fatalf("split WebSocket messages forwarded %+v, want no forwarded instructions", got)
	case <-time.After(100 * time.Millisecond):
	}
}

func waitForAtomicValue(t *testing.T, value *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if value.Load() >= want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("atomic value = %d, want at least %d", value.Load(), want)
		case <-ticker.C:
		}
	}
}
