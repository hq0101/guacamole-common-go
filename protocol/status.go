package protocol

import "fmt"

// Status represents a Guacamole protocol status code.
type Status int

const (
	StatusSuccess             Status = 0x0000
	StatusUnsupported         Status = 0x0100
	StatusServerError         Status = 0x0200
	StatusServerBusy          Status = 0x0201
	StatusUpstreamTimeout     Status = 0x0202
	StatusUpstreamError       Status = 0x0203
	StatusResourceNotFound    Status = 0x0204
	StatusResourceConflict    Status = 0x0205
	StatusResourceClosed      Status = 0x0206
	StatusUpstreamNotFound    Status = 0x0207
	StatusUpstreamUnavailable Status = 0x0208
	StatusSessionConflict     Status = 0x0209
	StatusSessionTimeout      Status = 0x020A
	StatusSessionClosed       Status = 0x020B
	StatusClientBadRequest    Status = 0x0300
	StatusClientUnauthorized  Status = 0x0301
	StatusClientForbidden     Status = 0x0303
	StatusClientTimeout       Status = 0x0308
	StatusClientOverrun       Status = 0x030D
	StatusClientBadType       Status = 0x030F
	StatusClientTooMany       Status = 0x031D
)

const (
	SUCCESS              = StatusSuccess
	UNSUPPORTED          = StatusUnsupported
	SERVER_ERROR         = StatusServerError
	SERVER_BUSY          = StatusServerBusy
	UPSTREAM_TIMEOUT     = StatusUpstreamTimeout
	UPSTREAM_ERROR       = StatusUpstreamError
	RESOURCE_NOT_FOUND   = StatusResourceNotFound
	RESOURCE_CONFLICT    = StatusResourceConflict
	RESOURCE_CLOSED      = StatusResourceClosed
	UPSTREAM_NOT_FOUND   = StatusUpstreamNotFound
	UPSTREAM_UNAVAILABLE = StatusUpstreamUnavailable
	SESSION_CONFLICT     = StatusSessionConflict
	SESSION_TIMEOUT      = StatusSessionTimeout
	SESSION_CLOSED       = StatusSessionClosed
	CLIENT_BAD_REQUEST   = StatusClientBadRequest
	CLIENT_UNAUTHORIZED  = StatusClientUnauthorized
	CLIENT_FORBIDDEN     = StatusClientForbidden
	CLIENT_TIMEOUT       = StatusClientTimeout
	CLIENT_OVERRUN       = StatusClientOverrun
	CLIENT_BAD_TYPE      = StatusClientBadType
	CLIENT_TOO_MANY      = StatusClientTooMany
)

// HTTPCode returns the equivalent HTTP status code for this Guacamole status.
func (s Status) HTTPCode() int {
	switch s {
	case StatusSuccess:
		return 200
	case StatusUnsupported:
		return 501
	case StatusServerError:
		return 500
	case StatusServerBusy:
		return 503
	case StatusUpstreamTimeout:
		return 504
	case StatusUpstreamError, StatusUpstreamNotFound, StatusUpstreamUnavailable:
		return 502
	case StatusResourceNotFound:
		return 404
	case StatusResourceConflict, StatusSessionConflict:
		return 409
	case StatusResourceClosed, StatusSessionClosed:
		return 404
	case StatusSessionTimeout, StatusClientTimeout:
		return 408
	case StatusClientBadRequest:
		return 400
	case StatusClientUnauthorized, StatusClientForbidden:
		return 403 // deliberate: 401 has HTTP-auth implications, Guacamole maps both to 403
	case StatusClientOverrun:
		return 413
	case StatusClientBadType:
		return 415
	case StatusClientTooMany:
		return 429
	default:
		return 500
	}
}

// GetHttpStatusCode returns the equivalent HTTP status code, matching Java's
// GuacamoleStatus method name.
func (s Status) GetHttpStatusCode() int {
	return s.HTTPCode()
}

// WebSocketCode returns the equivalent WebSocket close code.
func (s Status) WebSocketCode() int {
	switch s {
	case StatusSuccess:
		return 1000
	case StatusUnsupported, StatusServerError:
		return 1011
	case StatusServerBusy, StatusResourceConflict, StatusSessionConflict,
		StatusClientUnauthorized, StatusClientForbidden, StatusClientTooMany:
		return 1008
	case StatusUpstreamTimeout, StatusUpstreamError, StatusUpstreamNotFound,
		StatusUpstreamUnavailable:
		return 1011
	case StatusResourceNotFound, StatusResourceClosed,
		StatusSessionTimeout, StatusSessionClosed,
		StatusClientBadRequest, StatusClientTimeout:
		return 1002
	case StatusClientOverrun:
		return 1009
	case StatusClientBadType:
		return 1003
	default:
		return 1011
	}
}

// GetWebSocketCode returns the equivalent WebSocket close code, matching
// Java's GuacamoleStatus method name.
func (s Status) GetWebSocketCode() int {
	return s.WebSocketCode()
}

// GetGuacamoleStatusCode returns the numeric Guacamole protocol status code.
func (s Status) GetGuacamoleStatusCode() int {
	return int(s)
}

// StatusFromCode returns the Status for a given numeric code.
func StatusFromCode(code int) (Status, bool) {
	s := Status(code)
	switch s {
	case StatusSuccess, StatusUnsupported,
		StatusServerError, StatusServerBusy, StatusUpstreamTimeout,
		StatusUpstreamError, StatusResourceNotFound, StatusResourceConflict,
		StatusResourceClosed, StatusUpstreamNotFound, StatusUpstreamUnavailable,
		StatusSessionConflict, StatusSessionTimeout, StatusSessionClosed,
		StatusClientBadRequest, StatusClientUnauthorized, StatusClientForbidden,
		StatusClientTimeout, StatusClientOverrun, StatusClientBadType,
		StatusClientTooMany:
		return s, true
	}
	return 0, false
}

// FromGuacamoleStatusCode returns the Status for a numeric Guacamole protocol
// status code, matching Java's GuacamoleStatus method name.
func FromGuacamoleStatusCode(code int) (Status, bool) {
	return StatusFromCode(code)
}

// Error represents a Guacamole protocol error.
type Error struct {
	Status  Status
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("guacamole error %#06x: %s", int(e.Status), e.Message)
}

func (e *Error) HTTPCode() int      { return e.Status.HTTPCode() }
func (e *Error) WebSocketCode() int { return e.Status.WebSocketCode() }

// GetStatus returns the Guacamole status associated with this error.
func (e *Error) GetStatus() Status { return e.Status }

// GetHttpStatusCode returns the HTTP status associated with this error.
func (e *Error) GetHttpStatusCode() int { return e.HTTPCode() }

// GetWebSocketCode returns the WebSocket close code associated with this error.
func (e *Error) GetWebSocketCode() int { return e.WebSocketCode() }
