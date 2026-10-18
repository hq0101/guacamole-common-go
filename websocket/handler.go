package websocket

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"

	gorilla "github.com/gorilla/websocket"
	guacio "github.com/hq0101/guacamole-common-go/io"
	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

var upgrader = gorilla.Upgrader{
	CheckOrigin:  func(r *http.Request) bool { return true },
	Subprotocols: []string{"guacamole"},
}

const bufferSize = 8192

// ConnectFunc is called by the handler for each new WebSocket connection.
// The returned Tunnel must already be connected and handshaked.
type ConnectFunc func(r *http.Request) (guacnet.Tunnel, error)

// NewHandler returns an http.Handler that serves a Guacamole WebSocket tunnel.
func NewHandler(connect ConnectFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Negotiate subprotocol
		var respHeader http.Header
		for _, sub := range gorilla.Subprotocols(r) {
			if sub == "guacamole" {
				respHeader = http.Header{"Sec-Websocket-Protocol": []string{"guacamole"}}
				break
			}
		}
		ws, err := upgrader.Upgrade(w, r, respHeader)
		if err != nil {
			return
		}
		defer ws.Close()

		// wsMu serialises all WebSocket writes; gorilla is not concurrent-safe.
		var wsMu sync.Mutex
		closeWithStatus := func(status protocol.Status) {
			wm := gorilla.FormatCloseMessage(status.WebSocketCode(), strconv.Itoa(int(status)))
			wsMu.Lock()
			defer wsMu.Unlock()
			ws.WriteMessage(gorilla.CloseMessage, wm) //nolint:errcheck
		}

		tun, err := connect(r)
		if err != nil {
			closeWithStatus(statusFromError(err))
			return
		}
		if tun == nil {
			closeWithStatus(protocol.StatusResourceNotFound)
			return
		}
		defer tun.Close()

		wsWrite := func(data []byte) error {
			wsMu.Lock()
			defer wsMu.Unlock()
			return ws.WriteMessage(gorilla.TextMessage, data)
		}

		// Send UUID frame (internal opcode = "")
		uuidFrame := protocol.Instruction{Opcode: "", Args: []string{tun.UUID().String()}}
		if err := wsWrite(uuidFrame.Encode()); err != nil {
			return
		}

		// goroutine: Tunnel → WebSocket
		// When guacd closes the connection the goroutine exits and closes ws,
		// which unblocks ws.ReadMessage() in the main loop below.
		go func() {
			buf := make([]byte, 0, bufferSize)
			reader := tun.AcquireReader()
			defer tun.ReleaseReader()
			for {
				message, err := reader.Read()
				if err != nil {
					log.Printf("[guac] tunnel read error (closing ws): %v", err)
					closeWithStatus(statusFromTunnelReadError(err))
					ws.Close()
					return
				}

				buf = append(buf, []byte(string(message))...)

				available, err := reader.Available()
				if err != nil {
					log.Printf("[guac] tunnel availability error (closing ws): %v", err)
					closeWithStatus(statusFromTunnelReadError(err))
					ws.Close()
					return
				}

				if !available || len(buf) >= bufferSize {
					if err := wsWrite(buf); err != nil {
						log.Printf("[guac] ws write error: %v", err)
						return
					}
					buf = buf[:0]
				}
			}
		}()

		// Main loop: WebSocket → Tunnel
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				log.Printf("[guac] ws read error: %v", err)
				return
			}

			writer := tun.AcquireWriter()
			filtered := guacio.NewFilteredGuacamoleWriter(writer, func(instr protocol.Instruction) (protocol.Instruction, error) {
				// Intercept tunnel-internal instructions. Java guacamole-common
				// responds to ["ping", value] with the same ping value and
				// drops all internal instructions instead of forwarding them.
				if instr.Opcode == guacnet.InternalDataOpcode {
					if len(instr.Args) >= 2 && instr.Args[0] == "ping" {
						ping := protocol.Instruction{Opcode: "", Args: []string{"ping", instr.Args[1]}}
						wsWrite(ping.Encode()) //nolint:errcheck
					}
					return protocol.Instruction{}, protocol.ErrDropInstruction
				}
				return instr, nil
			})
			err = filtered.Write([]rune(string(msg)))
			tun.ReleaseWriter()
			if err != nil {
				if isResourceClosed(err) {
					log.Printf("[guac] tunnel write closed: %v", err)
					continue
				}
				log.Printf("[guac] tunnel write error: %v", err)
			}
		}
	})
}

func statusFromError(err error) protocol.Status {
	var protoErr *protocol.Error
	if errors.As(err, &protoErr) {
		return protoErr.Status
	}
	return protocol.StatusServerError
}

func statusFromTunnelReadError(err error) protocol.Status {
	status := statusFromError(err)
	if status == protocol.StatusResourceClosed {
		return protocol.StatusSuccess
	}
	return status
}

func isResourceClosed(err error) bool {
	var protoErr *protocol.Error
	return errors.As(err, &protoErr) && protoErr.Status == protocol.StatusResourceClosed
}
