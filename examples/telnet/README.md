# Telnet Example

Browser-based Telnet terminal powered by [guacd](https://guacamole.apache.org/) and
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
| Host | — | Telnet server hostname or IP (**required**) |
| Port | `23` | Telnet TCP port |
| Username | — | Login username sent to the login prompt (optional) |
| Password | — | Login password sent to the password prompt (optional) |

## Notes

- Telnet is an unencrypted protocol. Use only on trusted networks.
- Terminal dimensions are set from the browser viewport and update on window
  resize.
- Username and password are forwarded to guacd which handles the login sequence
  automatically; leave them empty if the server provides its own interactive
  login.
