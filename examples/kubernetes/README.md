# Kubernetes Example

Browser-based terminal that attaches to a Kubernetes pod via `kubectl exec`,
powered by [guacd](https://guacamole.apache.org/) and
[guacamole-common-js](https://www.npmjs.com/package/guacamole-common-js).

guacd uses its built-in `kubernetes` protocol to connect to the Kubernetes API
server and run `exec` inside the specified pod.

## Prerequisites

- A running `guacd` instance (≥ 1.4.0) with Kubernetes protocol support.
- The guacd process must have access to the Kubernetes API server (in-cluster or
  via kubeconfig).

## Run

```bash
go run . -guacd localhost:4822 -addr :8084
```

| Flag | Default | Description |
|------|---------|-------------|
| `-guacd` | `localhost:4822` | guacd address (`host:port`) |
| `-addr` | `:8084` | HTTP listen address |

Open <http://localhost:8084> in your browser.

## Connection form

| Field | Default | Description |
|-------|---------|-------------|
| API Server Host | — | Kubernetes API server hostname or IP (**required**) |
| Port | — | API server HTTPS port (default `443` when empty) |
| Namespace | `default` | Pod namespace |
| Pod | — | Pod name (**required**) |
| Container | — | Container name (optional; first container used when empty) |
| Exec command | `bash` | Command to execute inside the container |
| Exec path | — | Directory in which to run the command (optional) |

## Notes

- Terminal dimensions are set from the browser viewport and update on window
  resize.
- guacd connects to the Kubernetes API server using the credentials and CA
  configured in its own environment (service-account token or kubeconfig).
  The example passes `hostname` / `port` as connection hints to guacd but does
  not perform any Kubernetes authentication itself.
