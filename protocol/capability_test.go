package protocol_test

import (
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestProtocolCapability_IsSupportedMatchesJavaMinimumVersions(t *testing.T) {
	cases := []struct {
		name       string
		capability protocol.ProtocolCapability
		version    protocol.ProtocolVersion
		want       bool
	}{
		{"arbitrary handshake order before 1.1.0", protocol.CapabilityArbitraryHandshakeOrder, protocol.Version1_0_0, false},
		{"arbitrary handshake order at 1.1.0", protocol.CapabilityArbitraryHandshakeOrder, protocol.Version1_1_0, true},
		{"protocol version detection at 1.1.0", protocol.CapabilityProtocolVersionDetection, protocol.Version1_1_0, true},
		{"timezone handshake before 1.1.0", protocol.CapabilityTimezoneHandshake, protocol.Version1_0_0, false},
		{"timezone handshake after 1.1.0", protocol.CapabilityTimezoneHandshake, protocol.Version1_3_0, true},
		{"required instruction before 1.3.0", protocol.CapabilityRequiredInstruction, protocol.Version1_1_0, false},
		{"required instruction at 1.3.0", protocol.CapabilityRequiredInstruction, protocol.Version1_3_0, true},
		{"msg instruction before 1.5.0", protocol.CapabilityMsgInstruction, protocol.Version1_3_0, false},
		{"msg instruction at 1.5.0", protocol.CapabilityMsgInstruction, protocol.Version1_5_0, true},
		{"name handshake before 1.5.0", protocol.CapabilityNameHandshake, protocol.Version1_3_0, false},
		{"name handshake at 1.5.0", protocol.CapabilityNameHandshake, protocol.Version1_5_0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.capability.IsSupported(c.version); got != c.want {
				t.Errorf("IsSupported(%v) = %v, want %v", c.version, got, c.want)
			}
		})
	}
}

func TestProtocolCapability_AllProtocolCapabilitiesMatchesJavaEnum(t *testing.T) {
	capabilities := protocol.AllProtocolCapabilities()

	if len(capabilities) != 6 {
		t.Fatalf("AllProtocolCapabilities() returned %d capabilities, want 6", len(capabilities))
	}

	wantOrder := []protocol.ProtocolCapability{
		protocol.CapabilityArbitraryHandshakeOrder,
		protocol.CapabilityMsgInstruction,
		protocol.CapabilityNameHandshake,
		protocol.CapabilityProtocolVersionDetection,
		protocol.CapabilityRequiredInstruction,
		protocol.CapabilityTimezoneHandshake,
	}
	for i, want := range wantOrder {
		if capabilities[i] != want {
			t.Errorf("capabilities[%d] = %v, want %v", i, capabilities[i], want)
		}
	}
}
