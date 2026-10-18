package guacamole

import (
	"errors"
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestTypedGuacamoleErrorsExposeProtocolStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want protocol.Status
	}{
		{"base", NewGuacamoleException("base"), protocol.StatusServerError},
		{"client", NewGuacamoleClientException("client"), protocol.StatusClientBadRequest},
		{"bad type", NewGuacamoleClientBadTypeException("bad type"), protocol.StatusClientBadType},
		{"overrun", NewGuacamoleClientOverrunException("overrun"), protocol.StatusClientOverrun},
		{"timeout", NewGuacamoleClientTimeoutException("timeout"), protocol.StatusClientTimeout},
		{"too many", NewGuacamoleClientTooManyException("too many"), protocol.StatusClientTooMany},
		{"closed", NewGuacamoleConnectionClosedException("closed"), protocol.StatusServerError},
		{"resource closed", NewGuacamoleResourceClosedException("resource closed"), protocol.StatusResourceClosed},
		{"resource conflict", NewGuacamoleResourceConflictException("resource conflict"), protocol.StatusResourceConflict},
		{"resource not found", NewGuacamoleResourceNotFoundException("resource not found"), protocol.StatusResourceNotFound},
		{"security", NewGuacamoleSecurityException("security"), protocol.StatusClientForbidden},
		{"server busy", NewGuacamoleServerBusyException("server busy"), protocol.StatusServerBusy},
		{"server", NewGuacamoleServerException("server"), protocol.StatusServerError},
		{"session closed", NewGuacamoleSessionClosedException("session closed"), protocol.StatusSessionClosed},
		{"session conflict", NewGuacamoleSessionConflictException("session conflict"), protocol.StatusSessionConflict},
		{"session timeout", NewGuacamoleSessionTimeoutException("session timeout"), protocol.StatusSessionTimeout},
		{"unauthorized", NewGuacamoleUnauthorizedException("unauthorized"), protocol.StatusClientUnauthorized},
		{"unsupported", NewGuacamoleUnsupportedException("unsupported"), protocol.StatusUnsupported},
		{"upstream", NewGuacamoleUpstreamException("upstream"), protocol.StatusUpstreamError},
		{"upstream not found", NewGuacamoleUpstreamNotFoundException("upstream not found"), protocol.StatusUpstreamNotFound},
		{"upstream timeout", NewGuacamoleUpstreamTimeoutException("upstream timeout"), protocol.StatusUpstreamTimeout},
		{"upstream unavailable", NewGuacamoleUpstreamUnavailableException("upstream unavailable"), protocol.StatusUpstreamUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var protoErr *protocol.Error
			if !errors.As(tt.err, &protoErr) {
				t.Fatalf("errors.As(*protocol.Error) = false")
			}
			if protoErr.Status != tt.want {
				t.Fatalf("Status = %v, want %v", protoErr.Status, tt.want)
			}

			statusProvider, ok := tt.err.(interface{ GetStatus() protocol.Status })
			if !ok {
				t.Fatalf("%T does not expose GetStatus()", tt.err)
			}
			if statusProvider.GetStatus() != tt.want {
				t.Fatalf("GetStatus() = %v, want %v", statusProvider.GetStatus(), tt.want)
			}
		})
	}
}

func TestTypedGuacamoleErrorPreservesCauseAndConcreteType(t *testing.T) {
	cause := errors.New("socket failed")
	err := NewGuacamoleUnauthorizedException("denied", cause)

	var unauthorized *GuacamoleUnauthorizedException
	if !errors.As(err, &unauthorized) {
		t.Fatal("errors.As(*GuacamoleUnauthorizedException) = false")
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(cause) = false")
	}
	if err.GetCause() != cause {
		t.Fatalf("GetCause() = %v, want %v", err.GetCause(), cause)
	}
	if err.GetHttpStatusCode() != protocol.StatusClientUnauthorized.HTTPCode() {
		t.Fatalf("GetHttpStatusCode() = %d, want %d", err.GetHttpStatusCode(), protocol.StatusClientUnauthorized.HTTPCode())
	}
	if err.GetWebSocketCode() != protocol.StatusClientUnauthorized.WebSocketCode() {
		t.Fatalf("GetWebSocketCode() = %d, want %d", err.GetWebSocketCode(), protocol.StatusClientUnauthorized.WebSocketCode())
	}
}

func TestStatusToExceptionMatchesJavaStatusMappings(t *testing.T) {
	tests := []struct {
		name   string
		status protocol.Status
		want   func(error) bool
	}{
		{"unsupported", protocol.StatusUnsupported, func(err error) bool {
			var target *GuacamoleUnsupportedException
			return errors.As(err, &target)
		}},
		{"server busy", protocol.StatusServerBusy, func(err error) bool {
			var target *GuacamoleServerBusyException
			return errors.As(err, &target)
		}},
		{"upstream timeout", protocol.StatusUpstreamTimeout, func(err error) bool {
			var target *GuacamoleUpstreamTimeoutException
			return errors.As(err, &target)
		}},
		{"resource not found", protocol.StatusResourceNotFound, func(err error) bool {
			var target *GuacamoleResourceNotFoundException
			return errors.As(err, &target)
		}},
		{"client bad request", protocol.StatusClientBadRequest, func(err error) bool {
			var target *GuacamoleClientException
			return errors.As(err, &target)
		}},
		{"client forbidden", protocol.StatusClientForbidden, func(err error) bool {
			var target *GuacamoleSecurityException
			return errors.As(err, &target)
		}},
		{"client bad type", protocol.StatusClientBadType, func(err error) bool {
			var target *GuacamoleClientBadTypeException
			return errors.As(err, &target)
		}},
		{"session closed", protocol.StatusSessionClosed, func(err error) bool {
			var target *GuacamoleSessionClosedException
			return errors.As(err, &target)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StatusToException(tt.status, "mapped")
			if err == nil {
				t.Fatal("StatusToException() = nil")
			}
			if !tt.want(err) {
				t.Fatalf("StatusToException(%v) type = %T, want Java-compatible typed exception", tt.status, err)
			}
			statusProvider, ok := err.(interface{ GetStatus() protocol.Status })
			if !ok {
				t.Fatalf("%T does not expose GetStatus()", err)
			}
			if statusProvider.GetStatus() != tt.status {
				t.Fatalf("GetStatus() = %v, want %v", statusProvider.GetStatus(), tt.status)
			}
		})
	}
}

func TestStatusToExceptionPanicsForSuccess(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("StatusToException(StatusSuccess) did not panic")
		}
	}()

	_ = StatusToException(protocol.StatusSuccess, "success")
}
