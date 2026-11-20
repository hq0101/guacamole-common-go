# guacamole-common-go

English | [中文](README.zh-CN.md)

A Go port of Apache Guacamole's `guacamole-common` library, letting Go services proxy browser-based RDP/VNC/SSH/Telnet sessions through guacd.

## Features

- Full Guacamole protocol implementation: instruction encoding/decoding with Unicode codepoint length prefixes, streaming parser, and the complete connection handshake (protocol versions 1.0.0–1.5.0)
- WebSocket tunnel handler compatible with [guacamole-common-js](https://www.npmjs.com/package/guacamole-common-js): subprotocol negotiation, UUID frame, ping handling, and batched instruction forwarding
- HTTP tunnel handler implementing the `connect` / `read` / `write` long-polling protocol, with session tokens and automatic tunnel cleanup
- Socket layer for guacd connections: plaintext TCP, TLS, and a failover socket that surfaces upstream errors before the first sync
- Tunnels with UUID-based session tracking and exclusive reader/writer access
- Instruction filters for intercepting, rewriting, or dropping instructions in either direction
- API mirrors Java `guacamole-common` (including `Get*` aliases) for easy porting
- Runnable examples for RDP, VNC, SSH, Telnet, and Kubernetes, each with an embedded web UI

## Quick start

Requires a running [guacd](https://guacamole.apache.org/doc/gug/guacamole-architecture.html) (default port 4822).

```bash
go get github.com/hq0101/guacamole-common-go
```

Serve a WebSocket tunnel that the browser's `guacamole-common-js` client can connect to:

```go
package main

import (
	"context"
	"log"
	"net/http"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
	guacws "github.com/hq0101/guacamole-common-go/websocket"
)

func main() {
	connect := func(r *http.Request) (guacnet.Tunnel, error) {
		cfg := &protocol.Config{
			Protocol: "rdp",
			Params: map[string]string{
				"hostname":    "192.168.1.100",
				"port":        "3389",
				"username":    "alice",
				"password":    "secret",
				"ignore-cert": "true",
			},
		}
		return guacnet.Connect(context.Background(), "localhost:4822", cfg, protocol.NewClientInfo())
	}

	http.Handle("/ws", guacws.NewHandler(connect))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

On the browser side, create a `Guacamole.Client` with `new Guacamole.WebSocketTunnel("ws://localhost:8080/ws")` — see the embedded pages in `examples/*/static/index.html` for complete working frontends.

To use the HTTP long-polling tunnel instead of WebSocket, pass the same `connect` function to `servlet.NewHandler`:

```go
handler := servlet.NewHandler(connect) // import "github.com/hq0101/guacamole-common-go/servlet"
defer handler.Shutdown()
http.Handle("/tunnel", handler)
```

Connection parameters can also be taken from the incoming request (query string, headers, session) inside `connect`, as the examples do. Set `Config.ConnectionID` to join an existing guacd connection instead of starting a new one.

## Examples

Each example embeds its own web UI and accepts `-guacd host:port` and `-addr`:

| Example | Protocol | Default port |
|---|---|---|
| `examples/rdp` | RDP | `:8083` |
| `examples/ssh` | SSH | `:8082` |
| `examples/telnet` | Telnet | `:8084` |
| `examples/vnc` | VNC | `:8085` |
| `examples/kubernetes` | Kubernetes exec | `:8086` |

```bash
go run ./examples/rdp -guacd localhost:4822
```

Then open http://localhost:8083 and fill in the connection form. Connection parameters are entered in the browser form; the guacd address defaults to a LAN address, so pass `-guacd` explicitly.

## Packages

| Package | Purpose |
|---|---|
| `protocol` | Instructions, parser, connection config, client info, status codes, protocol versions |
| `io` | `GuacamoleReader` / `GuacamoleWriter` instruction streams and filters |
| `net` | guacd sockets (TCP/TLS/failover), handshake, UUID tunnels |
| `websocket` | WebSocket tunnel `http.Handler` for guacamole-common-js |
| `servlet` | HTTP long-polling tunnel handler with session management |

## License

Apache License 2.0 — see [LICENSE](LICENSE).