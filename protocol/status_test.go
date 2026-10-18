package protocol_test

import (
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestStatus_HTTPCode(t *testing.T) {
	cases := []struct {
		status protocol.Status
		want   int
	}{
		{protocol.StatusSuccess, 200},
		{protocol.StatusUnsupported, 501},
		{protocol.StatusServerError, 500},
		{protocol.StatusResourceNotFound, 404},
		{protocol.StatusResourceConflict, 409},
		{protocol.StatusClientBadRequest, 400},
		{protocol.StatusClientUnauthorized, 403},
		{protocol.StatusClientForbidden, 403},
		{protocol.StatusClientTimeout, 408},
		{protocol.StatusClientTooMany, 429},
	}
	for _, c := range cases {
		if got := c.status.HTTPCode(); got != c.want {
			t.Errorf("Status(%#x).HTTPCode() = %d, want %d", int(c.status), got, c.want)
		}
	}
}

func TestStatus_WebSocketCode(t *testing.T) {
	cases := []struct {
		status protocol.Status
		want   int
	}{
		{protocol.StatusSuccess, 1000},
		{protocol.StatusUnsupported, 1011},
		{protocol.StatusServerError, 1011},
		{protocol.StatusClientBadRequest, 1002},
		{protocol.StatusClientUnauthorized, 1008},
		{protocol.StatusClientTooMany, 1008},
	}
	for _, c := range cases {
		if got := c.status.WebSocketCode(); got != c.want {
			t.Errorf("Status(%#x).WebSocketCode() = %d, want %d", int(c.status), got, c.want)
		}
	}
}

func TestStatusFromCode(t *testing.T) {
	s, ok := protocol.StatusFromCode(0x0301)
	if !ok || s != protocol.StatusClientUnauthorized {
		t.Errorf("StatusFromCode(0x0301) = %v, %v", s, ok)
	}
	_, ok2 := protocol.StatusFromCode(0xFFFF)
	if ok2 {
		t.Error("StatusFromCode(0xFFFF) should return false")
	}
}

func TestStatus_JavaStyleAccessors(t *testing.T) {
	status := protocol.StatusClientOverrun

	if got := status.GetGuacamoleStatusCode(); got != 0x030D {
		t.Errorf("GetGuacamoleStatusCode() = %#x, want 0x030d", got)
	}
	if got := status.GetHttpStatusCode(); got != 413 {
		t.Errorf("GetHttpStatusCode() = %d, want 413", got)
	}
	if got := status.GetWebSocketCode(); got != 1009 {
		t.Errorf("GetWebSocketCode() = %d, want 1009", got)
	}
}

func TestFromGuacamoleStatusCodeMatchesJavaName(t *testing.T) {
	status, ok := protocol.FromGuacamoleStatusCode(0x0208)
	if !ok || status != protocol.StatusUpstreamUnavailable {
		t.Errorf("FromGuacamoleStatusCode(0x0208) = %v, %v; want StatusUpstreamUnavailable, true", status, ok)
	}

	if _, ok := protocol.FromGuacamoleStatusCode(0xFFFF); ok {
		t.Error("FromGuacamoleStatusCode(0xFFFF) ok = true, want false")
	}
}

func TestError_Methods(t *testing.T) {
	e := &protocol.Error{Status: protocol.StatusClientUnauthorized, Message: "bad creds"}
	if e.HTTPCode() != 403 {
		t.Errorf("HTTPCode() = %d, want 403", e.HTTPCode())
	}
	if e.WebSocketCode() != 1008 {
		t.Errorf("WebSocketCode() = %d, want 1008", e.WebSocketCode())
	}
	if e.Error() == "" {
		t.Error("Error() should not be empty")
	}
}

func TestError_JavaStyleAccessors(t *testing.T) {
	e := &protocol.Error{Status: protocol.StatusClientBadType, Message: "bad type"}

	if got := e.GetStatus(); got != protocol.StatusClientBadType {
		t.Errorf("GetStatus() = %v, want StatusClientBadType", got)
	}
	if got := e.GetHttpStatusCode(); got != 415 {
		t.Errorf("GetHttpStatusCode() = %d, want 415", got)
	}
	if got := e.GetWebSocketCode(); got != 1003 {
		t.Errorf("GetWebSocketCode() = %d, want 1003", got)
	}
}

func TestStatus_AllConstants(t *testing.T) {
	// Verify full set of constants defined in spec
	constants := []protocol.Status{
		protocol.StatusSuccess,
		protocol.StatusUnsupported,
		protocol.StatusServerError,
		protocol.StatusServerBusy,
		protocol.StatusUpstreamTimeout,
		protocol.StatusUpstreamError,
		protocol.StatusResourceNotFound,
		protocol.StatusResourceConflict,
		protocol.StatusResourceClosed,
		protocol.StatusUpstreamNotFound,
		protocol.StatusUpstreamUnavailable,
		protocol.StatusSessionConflict,
		protocol.StatusSessionTimeout,
		protocol.StatusSessionClosed,
		protocol.StatusClientBadRequest,
		protocol.StatusClientUnauthorized,
		protocol.StatusClientForbidden,
		protocol.StatusClientTimeout,
		protocol.StatusClientOverrun,
		protocol.StatusClientBadType,
		protocol.StatusClientTooMany,
	}
	if len(constants) != 21 {
		t.Errorf("expected 21 status constants, got %d", len(constants))
	}
}

func TestStatus_HTTPCode_Full(t *testing.T) {
	cases := []struct {
		s    protocol.Status
		want int
	}{
		{protocol.StatusServerBusy, 503},
		{protocol.StatusUpstreamError, 502},
		{protocol.StatusUpstreamNotFound, 502},
		{protocol.StatusUpstreamUnavailable, 502},
		{protocol.StatusResourceClosed, 404},
		{protocol.StatusSessionClosed, 404},
		{protocol.StatusClientUnauthorized, 403},
		{protocol.StatusClientForbidden, 403},
		{protocol.StatusClientOverrun, 413},
		{protocol.StatusClientBadType, 415},
	}
	for _, c := range cases {
		if got := c.s.HTTPCode(); got != c.want {
			t.Errorf("Status(%#x).HTTPCode() = %d, want %d", int(c.s), got, c.want)
		}
	}
}

func TestStatus_WebSocketCode_Full(t *testing.T) {
	cases := []struct {
		s    protocol.Status
		want int
	}{
		{protocol.StatusServerBusy, 1008},
		{protocol.StatusResourceNotFound, 1002},
		{protocol.StatusResourceConflict, 1008},
		{protocol.StatusResourceClosed, 1002},
		{protocol.StatusSessionConflict, 1008},
		{protocol.StatusSessionTimeout, 1002},
		{protocol.StatusSessionClosed, 1002},
		{protocol.StatusClientTimeout, 1002},
		{protocol.StatusClientOverrun, 1009},
		{protocol.StatusClientBadType, 1003},
		{protocol.StatusClientUnauthorized, 1008},
		{protocol.StatusClientForbidden, 1008},
		{protocol.StatusClientTooMany, 1008},
	}
	for _, c := range cases {
		if got := c.s.WebSocketCode(); got != c.want {
			t.Errorf("Status(%#x).WebSocketCode() = %d, want %d", int(c.s), got, c.want)
		}
	}
}
