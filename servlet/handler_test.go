package servlet_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	guacio "github.com/hq0101/guacamole-common-go/io"
	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
	"github.com/hq0101/guacamole-common-go/servlet"
)

// pipeSocket is an in-memory Socket implementation for testing.
// toClient simulates instructions sent from guacd to the client.
// fromClient captures instructions sent from the client to guacd.
type pipeSocket struct {
	toClient   chan protocol.Instruction
	fromClient chan protocol.Instruction
	closed     bool
}

func newPipeSocket() *pipeSocket {
	return &pipeSocket{
		toClient:   make(chan protocol.Instruction, 16),
		fromClient: make(chan protocol.Instruction, 16),
	}
}

func (ps *pipeSocket) SendInstruction(i protocol.Instruction) error {
	ps.fromClient <- i
	return nil
}

func (ps *pipeSocket) ReceiveInstruction() (protocol.Instruction, error) {
	instr, ok := <-ps.toClient
	if !ok {
		return protocol.Instruction{}, io.EOF
	}
	return instr, nil
}

func (ps *pipeSocket) Protocol() string { return "rdp" }

func (ps *pipeSocket) Close() error {
	if !ps.closed {
		ps.closed = true
	}
	return nil
}

func (ps *pipeSocket) IsOpen() bool { return !ps.closed }
func (ps *pipeSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(ps)
}
func (ps *pipeSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(ps)
}

type closedWriteSocket struct {
	closed bool
}

func (s *closedWriteSocket) SendInstruction(protocol.Instruction) error {
	return &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"}
}

func (s *closedWriteSocket) ReceiveInstruction() (protocol.Instruction, error) {
	return protocol.Instruction{}, io.EOF
}

func (s *closedWriteSocket) Protocol() string { return "rdp" }

func (s *closedWriteSocket) Close() error {
	s.closed = true
	return nil
}

func (s *closedWriteSocket) IsOpen() bool { return !s.closed }
func (s *closedWriteSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(s)
}
func (s *closedWriteSocket) Writer() guacio.GuacamoleWriter {
	return guacnet.NewSocketWriter(s)
}

type wrappedClosedWriteSocket struct {
	closedWriteSocket
}

func (s *wrappedClosedWriteSocket) SendInstruction(protocol.Instruction) error {
	return fmt.Errorf("wrapped: %w", &protocol.Error{Status: protocol.StatusResourceClosed, Message: "closed"})
}

type queuedReadSocket struct {
	firstReady    chan struct{}
	releaseFirst  chan struct{}
	releaseSecond chan struct{}
	calls         atomic.Int32
	open          atomic.Bool
}

func newQueuedReadSocket() *queuedReadSocket {
	s := &queuedReadSocket{
		firstReady:    make(chan struct{}),
		releaseFirst:  make(chan struct{}),
		releaseSecond: make(chan struct{}),
	}
	s.open.Store(true)
	return s
}

func (s *queuedReadSocket) SendInstruction(protocol.Instruction) error { return nil }

func (s *queuedReadSocket) ReceiveInstruction() (protocol.Instruction, error) {
	switch s.calls.Add(1) {
	case 1:
		close(s.firstReady)
		<-s.releaseFirst
		return protocol.Instruction{Opcode: "blob", Args: []string{"first"}}, nil
	case 2:
		<-s.releaseSecond
		return protocol.Instruction{Opcode: "sync", Args: []string{"0"}}, nil
	default:
		return protocol.Instruction{}, io.EOF
	}
}

func (s *queuedReadSocket) Protocol() string { return "rdp" }
func (s *queuedReadSocket) Close() error {
	s.open.Store(false)
	return nil
}
func (s *queuedReadSocket) IsOpen() bool { return s.open.Load() }
func (s *queuedReadSocket) Reader() guacio.GuacamoleReader {
	return guacnet.NewSocketReader(s)
}
func (s *queuedReadSocket) Writer() guacio.GuacamoleWriter {
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

func newTestHandler(t *testing.T) (*httptest.Server, *pipeSocket) {
	ps := newPipeSocket()
	tun := guacnet.NewTunnel(ps)
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(h)
	t.Cleanup(func() {
		srv.Close()
		ps.Close()
	})
	return srv, ps
}

func TestServlet_NewHandlerReturnsShutdownCapableHandler(t *testing.T) {
	handler := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return nil, nil
	})
	var _ http.Handler = handler

	handler.Shutdown()
	handler.Shutdown()
}

func TestServlet_Connect(t *testing.T) {
	srv, _ := newTestHandler(t)

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	if token == "" {
		t.Error("Guacamole-Tunnel-Token header missing")
	}
	if cache := resp.Header.Get("Cache-Control"); cache != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cache)
	}
	body, _ := io.ReadAll(resp.Body)
	// Body should contain the UUID
	if len(body) == 0 {
		t.Error("connect response body should contain UUID")
	}
}

func TestServlet_UnsupportedMethodIsRejected(t *testing.T) {
	srv, _ := newTestHandler(t)

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/?connect", nil)
	if err != nil {
		t.Fatalf("new PUT request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Tunnel-Token") != "" {
		t.Errorf("Guacamole-Tunnel-Token = %q, want empty", resp.Header.Get("Guacamole-Tunnel-Token"))
	}
	if resp.Header.Get("Guacamole-Status-Code") != "" {
		t.Errorf("Guacamole-Status-Code = %q, want empty", resp.Header.Get("Guacamole-Status-Code"))
	}
}

func TestServlet_ConnectNilTunnelReturnsNotFound(t *testing.T) {
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return nil, nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Status-Code") != "516" {
		t.Errorf("Guacamole-Status-Code = %q, want 516", resp.Header.Get("Guacamole-Status-Code"))
	}
	if resp.Header.Get("Guacamole-Error-Message") != "No tunnel created." {
		t.Errorf("Guacamole-Error-Message = %q, want No tunnel created.", resp.Header.Get("Guacamole-Error-Message"))
	}
}

func TestServlet_ConnectServerErrorMasksMessage(t *testing.T) {
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return nil, &protocol.Error{Status: protocol.StatusUpstreamError, Message: "backend host leaked"}
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Status-Code") != "515" {
		t.Errorf("Guacamole-Status-Code = %q, want 515", resp.Header.Get("Guacamole-Status-Code"))
	}
	if resp.Header.Get("Guacamole-Error-Message") != "Internal server error." {
		t.Errorf("Guacamole-Error-Message = %q, want Internal server error.", resp.Header.Get("Guacamole-Error-Message"))
	}
}

func TestServlet_ConnectWrappedGuacamoleErrorPreservesStatus(t *testing.T) {
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return nil, fmt.Errorf("connect failed: %w", &protocol.Error{Status: protocol.StatusClientUnauthorized, Message: "Denied"})
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Status-Code") != "769" {
		t.Errorf("Guacamole-Status-Code = %q, want 769", resp.Header.Get("Guacamole-Status-Code"))
	}
	if resp.Header.Get("Guacamole-Error-Message") != "Denied" {
		t.Errorf("Guacamole-Error-Message = %q, want Denied", resp.Header.Get("Guacamole-Error-Message"))
	}
}

func TestServlet_ReadWrite(t *testing.T) {
	srv, ps := newTestHandler(t)

	// Connect to get the UUID and token
	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	// Queue instructions in guacd-side before reading
	ps.toClient <- protocol.Instruction{Opcode: "sync", Args: []string{"0"}}
	ps.toClient <- protocol.Instruction{Opcode: "done", Args: []string{}}
	close(ps.toClient)

	// Read from tunnel with the Java-compatible tunnel session token header.
	readReq, err := http.NewRequest("GET", srv.URL+"/?read:0", nil)
	if err != nil {
		t.Fatalf("new read request: %v", err)
	}
	readReq.Header.Set("Guacamole-Tunnel-Token", token)

	readResp, err := http.DefaultClient.Do(readReq)
	if err != nil {
		t.Fatalf("GET read: %v", err)
	}
	defer readResp.Body.Close()
	body, _ := io.ReadAll(readResp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "sync") {
		t.Errorf("read response = %q, expected to contain sync", bodyStr)
	}
	if !strings.HasSuffix(bodyStr, "0.;") {
		t.Errorf("read response should end with 0.; (end marker), got: %q", bodyStr)
	}

	// Write to tunnel with the Java-compatible tunnel session token header.
	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Content-Type", "application/octet-stream")
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)

	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()

	if writeResp.StatusCode != http.StatusNoContent {
		t.Errorf("write status = %d, want 204", writeResp.StatusCode)
	}
	if contentType := writeResp.Header.Get("Content-Type"); contentType != "application/octet-stream" {
		t.Errorf("write Content-Type = %q, want application/octet-stream", contentType)
	}
	if cache := writeResp.Header.Get("Cache-Control"); cache != "no-cache" {
		t.Errorf("write Cache-Control = %q, want no-cache", cache)
	}

	select {
	case got := <-ps.fromClient:
		if got.Opcode != "key" {
			t.Errorf("forwarded opcode = %q, want key", got.Opcode)
		}
	default:
		t.Error("key instruction not forwarded to guacd")
	}
}

func TestServlet_ReadWriteUseTunnelAcquireRelease(t *testing.T) {
	ps := newPipeSocket()
	base := guacnet.NewTunnel(ps)
	tun := &trackedTunnel{Tunnel: base}
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	defer ps.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	ps.toClient <- protocol.NewInstruction("sync", "0")
	readReq, err := http.NewRequest("GET", srv.URL+"/?read:0", nil)
	if err != nil {
		t.Fatalf("new read request: %v", err)
	}
	readReq.Header.Set("Guacamole-Tunnel-Token", token)
	readResp, err := http.DefaultClient.Do(readReq)
	if err != nil {
		t.Fatalf("GET read: %v", err)
	}
	io.Copy(io.Discard, readResp.Body) //nolint:errcheck
	readResp.Body.Close()

	if tun.acquireReader.Load() != 1 || tun.releaseReader.Load() != 1 {
		t.Fatalf("reader acquire/release = %d/%d, want 1/1", tun.acquireReader.Load(), tun.releaseReader.Load())
	}

	key := protocol.NewInstruction("key", "65", "1")
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)
	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()

	if tun.acquireWriter.Load() != 1 || tun.releaseWriter.Load() != 1 {
		t.Fatalf("writer acquire/release = %d/%d, want 1/1", tun.acquireWriter.Load(), tun.releaseWriter.Load())
	}
}

func TestServlet_ReadMissingUUID(t *testing.T) {
	srv, _ := newTestHandler(t)

	// Connect first
	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	resp.Body.Close()

	// Read without tunnel session token header - should get 400
	readResp, err := http.Get(srv.URL + "/?read:0")
	if err != nil {
		t.Fatalf("GET read: %v", err)
	}
	readResp.Body.Close()

	if readResp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", readResp.StatusCode)
	}
	if readResp.Header.Get("Guacamole-Status-Code") == "" {
		t.Error("Guacamole-Status-Code header missing on error response")
	}
}

func TestServlet_ReadEOFDeregistersTunnel(t *testing.T) {
	srv, ps := newTestHandler(t)

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	close(ps.toClient)

	readReq, err := http.NewRequest("GET", srv.URL+"/?read:0", nil)
	if err != nil {
		t.Fatalf("new read request: %v", err)
	}
	readReq.Header.Set("Guacamole-Tunnel-Token", token)
	readResp, err := http.DefaultClient.Do(readReq)
	if err != nil {
		t.Fatalf("GET read: %v", err)
	}
	body, _ := io.ReadAll(readResp.Body)
	readResp.Body.Close()
	if string(body) != "0.;" {
		t.Errorf("read EOF body = %q, want 0.;", body)
	}
	if !ps.closed {
		t.Error("tunnel should be closed after read EOF")
	}

	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)
	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()
	if writeResp.StatusCode != http.StatusNotFound {
		t.Errorf("write after read EOF status = %d, want 404", writeResp.StatusCode)
	}
}

func TestServlet_ReadClosedTunnelReturnsNotFound(t *testing.T) {
	srv, ps := newTestHandler(t)

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	ps.Close()
	close(ps.toClient)

	readReq, err := http.NewRequest("GET", srv.URL+"/?read:0", nil)
	if err != nil {
		t.Fatalf("new read request: %v", err)
	}
	readReq.Header.Set("Guacamole-Tunnel-Token", token)
	readResp, err := http.DefaultClient.Do(readReq)
	if err != nil {
		t.Fatalf("GET read: %v", err)
	}
	defer readResp.Body.Close()

	if readResp.StatusCode != http.StatusNotFound {
		t.Errorf("read closed tunnel status = %d, want 404", readResp.StatusCode)
	}
	if readResp.Header.Get("Guacamole-Status-Code") != "516" {
		t.Errorf("Guacamole-Status-Code = %q, want 516", readResp.Header.Get("Guacamole-Status-Code"))
	}
	if readResp.Header.Get("Guacamole-Error-Message") != "Tunnel is closed." {
		t.Errorf("Guacamole-Error-Message = %q, want Tunnel is closed.", readResp.Header.Get("Guacamole-Error-Message"))
	}
}

func TestServlet_ReadYieldsWhenAnotherReaderIsQueued(t *testing.T) {
	socket := newQueuedReadSocket()
	tun := guacnet.NewTunnel(socket)
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	defer socket.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	read := func(done chan<- string) {
		req, err := http.NewRequest("GET", srv.URL+"/?read:0", nil)
		if err != nil {
			done <- "ERR: " + err.Error()
			return
		}
		req.Header.Set("Guacamole-Tunnel-Token", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			done <- "ERR: " + err.Error()
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		done <- string(body)
	}

	firstDone := make(chan string, 1)
	secondDone := make(chan string, 1)
	go read(firstDone)

	<-socket.firstReady
	go read(secondDone)
	time.Sleep(50 * time.Millisecond)
	close(socket.releaseFirst)

	select {
	case body := <-firstDone:
		if !strings.Contains(body, "blob") {
			t.Fatalf("first read body = %q, want blob instruction", body)
		}
		if strings.Contains(body, "sync") {
			t.Fatalf("first read body = %q, want queued reader to receive later sync", body)
		}
	case <-time.After(500 * time.Millisecond):
		close(socket.releaseSecond)
		t.Fatal("first read did not return after another reader queued")
	}

	close(socket.releaseSecond)
	select {
	case body := <-secondDone:
		if !strings.Contains(body, "sync") {
			t.Fatalf("second read body = %q, want sync instruction", body)
		}
	case <-time.After(time.Second):
		t.Fatal("second read did not receive remaining instruction")
	}
}

func TestServlet_WriteResourceClosedReturnsNoContent(t *testing.T) {
	socket := &closedWriteSocket{}
	tun := guacnet.NewTunnel(socket)
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)

	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()

	if writeResp.StatusCode != http.StatusNoContent {
		t.Errorf("write status = %d, want 204", writeResp.StatusCode)
	}
	if writeResp.Header.Get("Guacamole-Status-Code") != "" {
		t.Errorf("Guacamole-Status-Code = %q, want empty success response", writeResp.Header.Get("Guacamole-Status-Code"))
	}
}

func TestServlet_WriteWrappedResourceClosedReturnsNoContent(t *testing.T) {
	socket := &wrappedClosedWriteSocket{}
	tun := guacnet.NewTunnel(socket)
	h := servlet.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		return tun, nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)

	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	defer writeResp.Body.Close()

	if writeResp.StatusCode != http.StatusNoContent {
		t.Errorf("write status = %d, want 204", writeResp.StatusCode)
	}
	if writeResp.Header.Get("Guacamole-Status-Code") != "" {
		t.Errorf("Guacamole-Status-Code = %q, want empty success response", writeResp.Header.Get("Guacamole-Status-Code"))
	}
}

func TestServlet_WriteClosedTunnelDoesNotForwardInstructions(t *testing.T) {
	srv, ps := newTestHandler(t)

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	ps.Close()

	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)

	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()

	if writeResp.StatusCode != http.StatusNoContent {
		t.Errorf("write closed tunnel status = %d, want 204", writeResp.StatusCode)
	}
	select {
	case got := <-ps.fromClient:
		t.Fatalf("closed tunnel write forwarded %+v, want no forwarded instructions", got)
	default:
	}
}

func TestServlet_WriteIncompleteInstructionReturnsServerError(t *testing.T) {
	srv, ps := newTestHandler(t)

	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	token := resp.Header.Get("Guacamole-Tunnel-Token")
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", strings.NewReader("3.foo"))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-Token", token)

	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()

	if writeResp.StatusCode != http.StatusInternalServerError {
		t.Errorf("write incomplete status = %d, want 500", writeResp.StatusCode)
	}
	if writeResp.Header.Get("Guacamole-Status-Code") != "512" {
		t.Errorf("Guacamole-Status-Code = %q, want 512", writeResp.Header.Get("Guacamole-Status-Code"))
	}
	if writeResp.Header.Get("Guacamole-Error-Message") != "Internal server error." {
		t.Errorf("Guacamole-Error-Message = %q, want Internal server error.", writeResp.Header.Get("Guacamole-Error-Message"))
	}
	select {
	case got := <-ps.fromClient:
		t.Fatalf("incomplete write forwarded %+v, want no forwarded instructions", got)
	default:
	}
}

func TestServlet_WriteInvalidToken(t *testing.T) {
	srv, ps := newTestHandler(t)

	// Connect
	resp, err := http.Post(srv.URL+"/?connect", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST connect: %v", err)
	}
	uuidBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	tunnelUUID := string(uuidBytes)

	// Write with wrong token
	key := protocol.Instruction{Opcode: "key", Args: []string{"65", "1"}}
	writeReq, err := http.NewRequest("POST", srv.URL+"/?write:0", bytes.NewReader(key.Encode()))
	if err != nil {
		t.Fatalf("new write request: %v", err)
	}
	writeReq.Header.Set("Guacamole-Tunnel-UUID", tunnelUUID)
	writeReq.Header.Set("Guacamole-Tunnel-Token", "wrong-token")

	writeResp, err := http.DefaultClient.Do(writeReq)
	if err != nil {
		t.Fatalf("POST write: %v", err)
	}
	writeResp.Body.Close()

	// Should be forbidden (403)
	if writeResp.StatusCode == http.StatusNoContent {
		t.Error("write with invalid token should not succeed")
	}
	if writeResp.Header.Get("Guacamole-Status-Code") == "" {
		t.Error("Guacamole-Status-Code header missing on error response")
	}
	if writeResp.Header.Get("Guacamole-Error-Message") != "No such tunnel." {
		t.Errorf("Guacamole-Error-Message = %q, want No such tunnel.", writeResp.Header.Get("Guacamole-Error-Message"))
	}

	// Nothing should have been forwarded
	select {
	case <-ps.fromClient:
		t.Error("instruction should not have been forwarded with invalid token")
	default:
		// expected: nothing forwarded
	}
}

func TestServlet_NoQueryStringReturnsClientError(t *testing.T) {
	srv, _ := newTestHandler(t)

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Status-Code") == "" {
		t.Error("Guacamole-Status-Code header missing on bad request")
	}
	if resp.Header.Get("Guacamole-Error-Message") != "No query string provided." {
		t.Errorf("Guacamole-Error-Message = %q, want No query string provided.", resp.Header.Get("Guacamole-Error-Message"))
	}
}

func TestServlet_InvalidOperationWithoutTokenRequiresToken(t *testing.T) {
	srv, _ := newTestHandler(t)

	resp, err := http.Get(srv.URL + "/?bogus")
	if err != nil {
		t.Fatalf("GET bogus: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Error-Message") != "The HTTP tunnel session token is required for all requests after connecting." {
		t.Errorf("Guacamole-Error-Message = %q, want token required message", resp.Header.Get("Guacamole-Error-Message"))
	}
}

func TestServlet_InvalidOperationWithTokenReportsInvalidOperation(t *testing.T) {
	srv, _ := newTestHandler(t)

	req, err := http.NewRequest("GET", srv.URL+"/?bogus", nil)
	if err != nil {
		t.Fatalf("new bogus request: %v", err)
	}
	req.Header.Set("Guacamole-Tunnel-Token", "token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET bogus with token: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if resp.Header.Get("Guacamole-Error-Message") != "Invalid tunnel operation: bogus" {
		t.Errorf("Guacamole-Error-Message = %q, want invalid operation message", resp.Header.Get("Guacamole-Error-Message"))
	}
}
