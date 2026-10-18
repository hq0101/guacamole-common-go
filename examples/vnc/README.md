# VNC Example

Browser-based VNC client powered by [guacd](https://guacamole.apache.org/) and
[guacamole-common-js](https://www.npmjs.com/package/guacamole-common-js).

## Prerequisites

A running `guacd` instance accessible from this process.

## Run

```bash
go run . -guacd localhost:4822 -addr :8081
```

| Flag | Default | Description |
|------|---------|-------------|
| `-guacd` | `localhost:4822` | guacd address (`host:port`) |
| `-addr` | `:8081` | HTTP listen address |

Open <http://localhost:8081> in your browser.

## Connection form

| Field | Default | Description |
|-------|---------|-------------|
| Host | — | VNC server hostname or IP (**required**) |
| Port | `5900` | VNC TCP port |
| Password | — | VNC password (if set) |
| Read-only | ☐ | View-only mode — mouse and keyboard input disabled |

Display resolution and DPI are derived automatically from the browser viewport.

## Notes

- Resize the browser window to trigger a live VNC display resize (requires
  server-side resize support, e.g. TigerVNC with `randr` extension).
- Image formats advertised to guacd: `image/png`, `image/jpeg`, `image/webp`.
