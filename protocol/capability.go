package protocol

// ProtocolCapability describes a Guacamole protocol capability introduced in a
// specific protocol version.
type ProtocolCapability struct {
	minVersion ProtocolVersion
}

var (
	CapabilityArbitraryHandshakeOrder  = ProtocolCapability{minVersion: Version1_1_0}
	CapabilityMsgInstruction           = ProtocolCapability{minVersion: Version1_5_0}
	CapabilityNameHandshake            = ProtocolCapability{minVersion: Version1_5_0}
	CapabilityProtocolVersionDetection = ProtocolCapability{minVersion: Version1_1_0}
	CapabilityRequiredInstruction      = ProtocolCapability{minVersion: Version1_3_0}
	CapabilityTimezoneHandshake        = ProtocolCapability{minVersion: Version1_1_0}
)

var (
	ARBITRARY_HANDSHAKE_ORDER  = CapabilityArbitraryHandshakeOrder
	MSG_INSTRUCTION            = CapabilityMsgInstruction
	NAME_HANDSHAKE             = CapabilityNameHandshake
	PROTOCOL_VERSION_DETECTION = CapabilityProtocolVersionDetection
	REQUIRED_INSTRUCTION       = CapabilityRequiredInstruction
	TIMEZONE_HANDSHAKE         = CapabilityTimezoneHandshake
)

// IsSupported reports whether this capability is supported by the given
// Guacamole protocol version.
func (c ProtocolCapability) IsSupported(version ProtocolVersion) bool {
	return version.AtLeast(c.minVersion)
}

// AllProtocolCapabilities returns all known protocol capabilities in the same
// order as Java's GuacamoleProtocolCapability.values().
func AllProtocolCapabilities() []ProtocolCapability {
	return []ProtocolCapability{
		CapabilityArbitraryHandshakeOrder,
		CapabilityMsgInstruction,
		CapabilityNameHandshake,
		CapabilityProtocolVersionDetection,
		CapabilityRequiredInstruction,
		CapabilityTimezoneHandshake,
	}
}
