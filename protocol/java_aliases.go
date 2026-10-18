package protocol

type GuacamoleConfiguration = Config
type GuacamoleClientInformation = ClientInfo
type GuacamoleInstruction = Instruction
type GuacamoleParser = Parser
type GuacamoleStatus = Status
type GuacamoleProtocolVersion = ProtocolVersion
type GuacamoleProtocolCapability = ProtocolCapability
type GuacamoleFilter = Filter

// NewGuacamoleConfiguration returns a Java-style blank configuration.
func NewGuacamoleConfiguration() *GuacamoleConfiguration {
	return NewConfig()
}

// NewGuacamoleConfigurationFrom returns an independent Java-style
// configuration copy.
func NewGuacamoleConfigurationFrom(config *GuacamoleConfiguration) *GuacamoleConfiguration {
	return NewConfigFrom(config)
}

// NewGuacamoleClientInformation returns Java-style client information with
// guacamole-common defaults.
func NewGuacamoleClientInformation() *GuacamoleClientInformation {
	return NewClientInfo()
}

// NewGuacamoleInstruction returns a Java-style protocol instruction.
func NewGuacamoleInstruction(opcode string, args ...string) GuacamoleInstruction {
	return NewInstruction(opcode, args...)
}

// NewGuacamoleParser returns a Java-style protocol parser.
func NewGuacamoleParser() *GuacamoleParser {
	return &Parser{}
}
