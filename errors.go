package guacamole

import "github.com/hq0101/guacamole-common-go/protocol"

// GuacamoleException is the base Java-compatible Guacamole error type.
type GuacamoleException struct {
	err   *protocol.Error
	cause error
}

func newGuacamoleException(status protocol.Status, message string, causes []error) *GuacamoleException {
	var cause error
	if len(causes) > 0 {
		cause = causes[0]
	}
	if message == "" && cause != nil {
		message = cause.Error()
	}
	return &GuacamoleException{
		err:   &protocol.Error{Status: status, Message: message},
		cause: cause,
	}
}

// NewGuacamoleException returns a base Guacamole error.
func NewGuacamoleException(message string, cause ...error) *GuacamoleException {
	return newGuacamoleException(protocol.StatusServerError, message, cause)
}

// StatusToException returns the Java-compatible GuacamoleException subclass
// corresponding to status. Calling this with StatusSuccess panics, matching
// Java's GuacamoleStatus.SUCCESS.toException().
func StatusToException(status protocol.Status, message string, cause ...error) error {
	switch status {
	case protocol.StatusSuccess:
		panic("the Guacamole protocol SUCCESS status code cannot be converted into a GuacamoleException")
	case protocol.StatusUnsupported:
		return NewGuacamoleUnsupportedException(message, cause...)
	case protocol.StatusServerError:
		return NewGuacamoleServerException(message, cause...)
	case protocol.StatusServerBusy:
		return NewGuacamoleServerBusyException(message, cause...)
	case protocol.StatusUpstreamTimeout:
		return NewGuacamoleUpstreamTimeoutException(message, cause...)
	case protocol.StatusUpstreamError:
		return NewGuacamoleUpstreamException(message, cause...)
	case protocol.StatusResourceNotFound:
		return NewGuacamoleResourceNotFoundException(message, cause...)
	case protocol.StatusResourceConflict:
		return NewGuacamoleResourceConflictException(message, cause...)
	case protocol.StatusResourceClosed:
		return NewGuacamoleResourceClosedException(message, cause...)
	case protocol.StatusUpstreamNotFound:
		return NewGuacamoleUpstreamNotFoundException(message, cause...)
	case protocol.StatusUpstreamUnavailable:
		return NewGuacamoleUpstreamUnavailableException(message, cause...)
	case protocol.StatusSessionConflict:
		return NewGuacamoleSessionConflictException(message, cause...)
	case protocol.StatusSessionTimeout:
		return NewGuacamoleSessionTimeoutException(message, cause...)
	case protocol.StatusSessionClosed:
		return NewGuacamoleSessionClosedException(message, cause...)
	case protocol.StatusClientBadRequest:
		return NewGuacamoleClientException(message, cause...)
	case protocol.StatusClientUnauthorized:
		return NewGuacamoleUnauthorizedException(message, cause...)
	case protocol.StatusClientForbidden:
		return NewGuacamoleSecurityException(message, cause...)
	case protocol.StatusClientTimeout:
		return NewGuacamoleClientTimeoutException(message, cause...)
	case protocol.StatusClientOverrun:
		return NewGuacamoleClientOverrunException(message, cause...)
	case protocol.StatusClientBadType:
		return NewGuacamoleClientBadTypeException(message, cause...)
	case protocol.StatusClientTooMany:
		return NewGuacamoleClientTooManyException(message, cause...)
	default:
		return NewGuacamoleServerException(message, cause...)
	}
}

func (e *GuacamoleException) Error() string {
	return e.err.Error()
}

// Unwrap exposes both the protocol error and the optional underlying cause.
func (e *GuacamoleException) Unwrap() []error {
	if e.cause == nil {
		return []error{e.err}
	}
	return []error{e.err, e.cause}
}

// GetCause returns the underlying cause of this exception, if any.
func (e *GuacamoleException) GetCause() error {
	return e.cause
}

// GetStatus returns the Guacamole protocol status associated with this error.
func (e *GuacamoleException) GetStatus() protocol.Status {
	return e.err.Status
}

// GetHttpStatusCode returns the HTTP status code associated with this error.
func (e *GuacamoleException) GetHttpStatusCode() int {
	return e.err.GetHttpStatusCode()
}

// GetWebSocketCode returns the WebSocket close code associated with this error.
func (e *GuacamoleException) GetWebSocketCode() int {
	return e.err.GetWebSocketCode()
}

type GuacamoleClientException struct{ *GuacamoleException }
type GuacamoleClientBadTypeException struct{ *GuacamoleException }
type GuacamoleClientOverrunException struct{ *GuacamoleException }
type GuacamoleClientTimeoutException struct{ *GuacamoleException }
type GuacamoleClientTooManyException struct{ *GuacamoleException }
type GuacamoleConnectionClosedException struct{ *GuacamoleException }
type GuacamoleResourceClosedException struct{ *GuacamoleException }
type GuacamoleResourceConflictException struct{ *GuacamoleException }
type GuacamoleResourceNotFoundException struct{ *GuacamoleException }
type GuacamoleSecurityException struct{ *GuacamoleException }
type GuacamoleServerBusyException struct{ *GuacamoleException }
type GuacamoleServerException struct{ *GuacamoleException }
type GuacamoleSessionClosedException struct{ *GuacamoleException }
type GuacamoleSessionConflictException struct{ *GuacamoleException }
type GuacamoleSessionTimeoutException struct{ *GuacamoleException }
type GuacamoleUnauthorizedException struct{ *GuacamoleException }
type GuacamoleUnsupportedException struct{ *GuacamoleException }
type GuacamoleUpstreamException struct{ *GuacamoleException }
type GuacamoleUpstreamNotFoundException struct{ *GuacamoleException }
type GuacamoleUpstreamTimeoutException struct{ *GuacamoleException }
type GuacamoleUpstreamUnavailableException struct{ *GuacamoleException }

func NewGuacamoleClientException(message string, cause ...error) *GuacamoleClientException {
	return &GuacamoleClientException{newGuacamoleException(protocol.StatusClientBadRequest, message, cause)}
}

func NewGuacamoleClientBadTypeException(message string, cause ...error) *GuacamoleClientBadTypeException {
	return &GuacamoleClientBadTypeException{newGuacamoleException(protocol.StatusClientBadType, message, cause)}
}

func NewGuacamoleClientOverrunException(message string, cause ...error) *GuacamoleClientOverrunException {
	return &GuacamoleClientOverrunException{newGuacamoleException(protocol.StatusClientOverrun, message, cause)}
}

func NewGuacamoleClientTimeoutException(message string, cause ...error) *GuacamoleClientTimeoutException {
	return &GuacamoleClientTimeoutException{newGuacamoleException(protocol.StatusClientTimeout, message, cause)}
}

func NewGuacamoleClientTooManyException(message string, cause ...error) *GuacamoleClientTooManyException {
	return &GuacamoleClientTooManyException{newGuacamoleException(protocol.StatusClientTooMany, message, cause)}
}

func NewGuacamoleConnectionClosedException(message string, cause ...error) *GuacamoleConnectionClosedException {
	return &GuacamoleConnectionClosedException{newGuacamoleException(protocol.StatusServerError, message, cause)}
}

func NewGuacamoleResourceClosedException(message string, cause ...error) *GuacamoleResourceClosedException {
	return &GuacamoleResourceClosedException{newGuacamoleException(protocol.StatusResourceClosed, message, cause)}
}

func NewGuacamoleResourceConflictException(message string, cause ...error) *GuacamoleResourceConflictException {
	return &GuacamoleResourceConflictException{newGuacamoleException(protocol.StatusResourceConflict, message, cause)}
}

func NewGuacamoleResourceNotFoundException(message string, cause ...error) *GuacamoleResourceNotFoundException {
	return &GuacamoleResourceNotFoundException{newGuacamoleException(protocol.StatusResourceNotFound, message, cause)}
}

func NewGuacamoleSecurityException(message string, cause ...error) *GuacamoleSecurityException {
	return &GuacamoleSecurityException{newGuacamoleException(protocol.StatusClientForbidden, message, cause)}
}

func NewGuacamoleServerBusyException(message string, cause ...error) *GuacamoleServerBusyException {
	return &GuacamoleServerBusyException{newGuacamoleException(protocol.StatusServerBusy, message, cause)}
}

func NewGuacamoleServerException(message string, cause ...error) *GuacamoleServerException {
	return &GuacamoleServerException{newGuacamoleException(protocol.StatusServerError, message, cause)}
}

func NewGuacamoleSessionClosedException(message string, cause ...error) *GuacamoleSessionClosedException {
	return &GuacamoleSessionClosedException{newGuacamoleException(protocol.StatusSessionClosed, message, cause)}
}

func NewGuacamoleSessionConflictException(message string, cause ...error) *GuacamoleSessionConflictException {
	return &GuacamoleSessionConflictException{newGuacamoleException(protocol.StatusSessionConflict, message, cause)}
}

func NewGuacamoleSessionTimeoutException(message string, cause ...error) *GuacamoleSessionTimeoutException {
	return &GuacamoleSessionTimeoutException{newGuacamoleException(protocol.StatusSessionTimeout, message, cause)}
}

func NewGuacamoleUnauthorizedException(message string, cause ...error) *GuacamoleUnauthorizedException {
	return &GuacamoleUnauthorizedException{newGuacamoleException(protocol.StatusClientUnauthorized, message, cause)}
}

func NewGuacamoleUnsupportedException(message string, cause ...error) *GuacamoleUnsupportedException {
	return &GuacamoleUnsupportedException{newGuacamoleException(protocol.StatusUnsupported, message, cause)}
}

func NewGuacamoleUpstreamException(message string, cause ...error) *GuacamoleUpstreamException {
	return &GuacamoleUpstreamException{newGuacamoleException(protocol.StatusUpstreamError, message, cause)}
}

func NewGuacamoleUpstreamNotFoundException(message string, cause ...error) *GuacamoleUpstreamNotFoundException {
	return &GuacamoleUpstreamNotFoundException{newGuacamoleException(protocol.StatusUpstreamNotFound, message, cause)}
}

func NewGuacamoleUpstreamTimeoutException(message string, cause ...error) *GuacamoleUpstreamTimeoutException {
	return &GuacamoleUpstreamTimeoutException{newGuacamoleException(protocol.StatusUpstreamTimeout, message, cause)}
}

func NewGuacamoleUpstreamUnavailableException(message string, cause ...error) *GuacamoleUpstreamUnavailableException {
	return &GuacamoleUpstreamUnavailableException{newGuacamoleException(protocol.StatusUpstreamUnavailable, message, cause)}
}
