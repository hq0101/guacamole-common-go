package servlet

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// ConnectFunc establishes a Guacamole tunnel for the given HTTP request.
type ConnectFunc func(r *http.Request) (guacnet.Tunnel, error)

// Handler implements the Guacamole HTTP tunnel protocol.
type Handler struct {
	connect ConnectFunc
	tunnels *HTTPTunnelMap
}

// NewHandler returns a Handler implementing the Guacamole HTTP tunnel protocol.
func NewHandler(connect ConnectFunc) *Handler {
	h := &Handler{
		connect: connect,
		tunnels: NewHTTPTunnelMap(),
	}
	return h
}

// Shutdown stops the HTTP tunnel cleanup task associated with this handler.
func (h *Handler) Shutdown() {
	h.tunnels.Shutdown()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, POST, HEAD")
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.RawQuery

	if query == "" && !r.URL.ForceQuery {
		sendStatusError(w, protocol.StatusClientBadRequest, "No query string provided.")
		return
	}

	if query == "connect" {
		h.handleConnect(w, r)
		return
	}

	if r.Header.Get("Guacamole-Tunnel-Token") == "" {
		sendStatusError(w, protocol.StatusClientBadRequest, "The HTTP tunnel session token is required for all requests after connecting.")
		return
	}

	if strings.HasPrefix(query, "read:") {
		h.handleRead(w, r)
		return
	}

	if strings.HasPrefix(query, "write:") {
		h.handleWrite(w, r)
		return
	}

	sendStatusError(w, protocol.StatusClientBadRequest, "Invalid tunnel operation: "+query)
}

func (h *Handler) handleConnect(w http.ResponseWriter, r *http.Request) {
	tun, err := h.connect(r)
	if err != nil {
		sendError(w, err)
		return
	}
	if tun == nil {
		sendStatusError(w, protocol.StatusResourceNotFound, "No tunnel created.")
		return
	}

	token := generateToken()
	h.tunnels.Put(token, tun)

	w.Header().Set("Guacamole-Tunnel-Token", token)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, tun.UUID().String())
}

func (h *Handler) handleRead(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Guacamole-Tunnel-Token")
	if token == "" {
		sendStatusError(w, protocol.StatusClientBadRequest, "The HTTP tunnel session token is required for all requests after connecting.")
		return
	}
	entry := h.getEntry(token)
	if entry == nil {
		sendStatusError(w, protocol.StatusResourceNotFound, "No such tunnel.")
		return
	}

	if !entry.IsOpen() {
		sendStatusError(w, protocol.StatusResourceNotFound, "Tunnel is closed.")
		return
	}

	reader := entry.AcquireReader()
	defer entry.ReleaseReader()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	for {
		instr, err := reader.ReadInstruction()
		if err != nil {
			h.removeEntry(token)
			entry.Close() //nolint:errcheck
			break
		}
		w.Write([]byte(string(instr.ToCharArray()))) //nolint:errcheck
		if flusher != nil {
			flusher.Flush()
		}
		if entry.HasQueuedReaderThreads() {
			break
		}
		if instr.Opcode == "sync" {
			break
		}
	}
	// Write end-of-chunk marker
	w.Write([]byte("0.;")) //nolint:errcheck
	if flusher != nil {
		flusher.Flush()
	}
}

func (h *Handler) handleWrite(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Guacamole-Tunnel-Token")
	if token == "" {
		sendStatusError(w, protocol.StatusClientBadRequest, "The HTTP tunnel session token is required for all requests after connecting.")
		return
	}

	entry := h.getEntry(token)
	if entry == nil {
		sendStatusError(w, protocol.StatusResourceNotFound, "No such tunnel.")
		return
	}

	if !entry.IsOpen() {
		writeNoContent(w)
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		sendStatusError(w, protocol.StatusClientBadRequest, err.Error())
		return
	}

	writer := entry.AcquireWriter()
	defer entry.ReleaseWriter()

	if err := writer.Write([]rune(string(body))); err != nil {
		if isResourceClosed(err) {
			writeNoContent(w)
			return
		}
		sendError(w, err)
		return
	}
	writeNoContent(w)
}

func writeNoContent(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(http.StatusNoContent)
}

func isResourceClosed(err error) bool {
	var protoErr *protocol.Error
	return errors.As(err, &protoErr) && protoErr.Status == protocol.StatusResourceClosed
}

func (h *Handler) getEntry(token string) *HTTPTunnel {
	return h.tunnels.Get(token)
}

func (h *Handler) removeEntry(token string) {
	h.tunnels.Remove(token)
}

func generateToken() string {
	buf := make([]byte, 33)
	rand.Read(buf) //nolint:errcheck
	return base64.StdEncoding.EncodeToString(buf)
}

func sendError(w http.ResponseWriter, err error) {
	var protoErr *protocol.Error
	if !errors.As(err, &protoErr) {
		protoErr = &protocol.Error{Status: protocol.StatusServerError, Message: err.Error()}
	}
	message := protoErr.Message
	if !isClientErrorStatus(protoErr.Status) {
		message = "Internal server error."
	}
	w.Header().Set("Guacamole-Status-Code", strconv.Itoa(int(protoErr.Status)))
	w.Header().Set("Guacamole-Error-Message", message)
	http.Error(w, message, protoErr.HTTPCode())
}

func sendStatusError(w http.ResponseWriter, status protocol.Status, msg string) {
	sendError(w, &protocol.Error{Status: status, Message: msg})
}

func isClientErrorStatus(status protocol.Status) bool {
	switch status {
	case protocol.StatusResourceNotFound,
		protocol.StatusResourceConflict,
		protocol.StatusResourceClosed,
		protocol.StatusClientBadRequest,
		protocol.StatusClientUnauthorized,
		protocol.StatusClientForbidden,
		protocol.StatusClientTimeout,
		protocol.StatusClientOverrun,
		protocol.StatusClientBadType,
		protocol.StatusClientTooMany:
		return true
	default:
		return false
	}
}
