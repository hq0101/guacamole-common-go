# SSH Example

Browser-based SSH terminal powered by [guacd](https://guacamole.apache.org/) and
[guacamole-common-js](https://www.npmjs.com/package/guacamole-common-js).

## Prerequisites

A running `guacd` instance accessible from this process.

## Run

```bash
go run . -guacd localhost:4822 -addr :8082
```

| Flag | Default | Description |
|------|---------|-------------|
| `-guacd` | `localhost:4822` | guacd address (`host:port`) |
| `-addr` | `:8082` | HTTP listen address |

Open <http://localhost:8082> in your browser.

## Connection form

| Field | Default | Description |
|-------|---------|-------------|
| Host | — | SSH server hostname or IP (**required**) |
| Port | `22` | SSH TCP port |
| Username | — | Login user |
| Password | — | Login password (leave empty when using a private key) |
| Private key | — | PEM-encoded private key (paste contents) |
| Passphrase | — | Passphrase for the private key (if encrypted) |
| Color scheme | `gray-black` | Terminal color scheme (guacd built-in name) |

## Notes

- Provide either **Password** or **Private key**, not both.
- Terminal dimensions are set from the browser viewport and update on window
  resize.
- Supported color scheme values include `black-white`, `white-black`,
  `gray-black`, `green-black` (guacd defaults).
