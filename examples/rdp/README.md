# RDP Example

Browser-based RDP client powered by [guacd](https://guacamole.apache.org/) and
[guacamole-common-js](https://www.npmjs.com/package/guacamole-common-js).

## Prerequisites

A running `guacd` instance accessible from this process.

## Run

```bash
go run . -guacd localhost:4822 -addr :8083
```

| Flag | Default | Description |
|------|---------|-------------|
| `-guacd` | `localhost:4822` | guacd address (`host:port`) |
| `-addr` | `:8083` | HTTP listen address |

Open <http://localhost:8083> in your browser.

## Connection form

| Field | Default | Description |
|-------|---------|-------------|
| Host | — | RDP server hostname or IP (**required**) |
| Port | `3389` | RDP TCP port |
| Username | `Administrator` | Windows login user |
| Password | — | Windows login password |
| Domain | — | Active Directory domain (optional) |
| Security | `any` | `any` / `nla` / `nla-ext` / `tls` / `rdp` |
| Color depth | `24` | `8` / `16` / `24` / `32` bit |
| Ignore TLS cert | ✓ | Skip certificate validation (self-signed certs) |

Display resolution and DPI are derived automatically from the browser viewport.

## Notes

- Wallpaper and window theming are enabled by default (`enable-wallpaper`,
  `enable-theming`). Remove those parameters from `main.go` to disable them for
  lower-bandwidth connections.
- Resize the browser window to trigger a live RDP display resize.
