package protocol_test

import (
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestJavaStyleProtocolAliasesAndConstructors(t *testing.T) {
	config := protocol.NewGuacamoleConfiguration()
	var _ *protocol.GuacamoleConfiguration = config
	config.SetProtocol("ssh")
	config.SetConnectionID("$connection")
	config.SetParameter("hostname", "guacd.example")

	clone := protocol.NewGuacamoleConfigurationFrom(config)
	var _ *protocol.GuacamoleConfiguration = clone
	config.SetProtocol("rdp")
	config.SetParameter("hostname", "changed.example")
	if clone.GetProtocol() != "ssh" {
		t.Fatalf("cloned GetProtocol() = %q, want ssh", clone.GetProtocol())
	}
	if clone.GetParameter("hostname") != "guacd.example" {
		t.Fatalf("cloned hostname = %q, want guacd.example", clone.GetParameter("hostname"))
	}

	info := protocol.NewGuacamoleClientInformation()
	var _ *protocol.GuacamoleClientInformation = info
	if info.GetOptimalScreenWidth() != 1024 {
		t.Fatalf("GetOptimalScreenWidth() = %d, want 1024", info.GetOptimalScreenWidth())
	}

	instruction := protocol.NewGuacamoleInstruction("sync", "0")
	var _ protocol.GuacamoleInstruction = instruction
	if instruction.GetOpcode() != "sync" {
		t.Fatalf("GetOpcode() = %q, want sync", instruction.GetOpcode())
	}

	parser := protocol.NewGuacamoleParser()
	var _ *protocol.GuacamoleParser = parser
	encoded := []byte(instruction.String())
	if consumed := parser.Append(encoded); consumed != len(encoded) {
		t.Fatalf("Append() consumed %d bytes, want %d", consumed, len(encoded))
	}
	if !parser.HasNext() {
		t.Fatal("HasNext() = false, want true")
	}

	var version protocol.GuacamoleProtocolVersion = protocol.VERSION_1_5_0
	if version.GetMinor() != 5 {
		t.Fatalf("GetMinor() = %d, want 5", version.GetMinor())
	}

	var status protocol.GuacamoleStatus = protocol.SUCCESS
	if status.GetGuacamoleStatusCode() != 0 {
		t.Fatalf("GetGuacamoleStatusCode() = %d, want 0", status.GetGuacamoleStatusCode())
	}
	if protocol.CLIENT_BAD_TYPE.GetHttpStatusCode() != 415 {
		t.Fatalf("CLIENT_BAD_TYPE.GetHttpStatusCode() = %d, want 415", protocol.CLIENT_BAD_TYPE.GetHttpStatusCode())
	}

	var capability protocol.GuacamoleProtocolCapability = protocol.CapabilityNameHandshake
	if !capability.IsSupported(protocol.VERSION_1_5_0) {
		t.Fatal("CapabilityNameHandshake should support VERSION_1_5_0")
	}
	if !protocol.NAME_HANDSHAKE.IsSupported(protocol.VERSION_1_5_0) {
		t.Fatal("NAME_HANDSHAKE should support VERSION_1_5_0")
	}
	if protocol.REQUIRED_INSTRUCTION.IsSupported(protocol.VERSION_1_1_0) {
		t.Fatal("REQUIRED_INSTRUCTION should not support VERSION_1_1_0")
	}

	if protocol.INSTRUCTION_MAX_LENGTH != protocol.InstructionMaxLength {
		t.Fatalf("INSTRUCTION_MAX_LENGTH = %d, want %d", protocol.INSTRUCTION_MAX_LENGTH, protocol.InstructionMaxLength)
	}
	if protocol.INSTRUCTION_MAX_DIGITS != protocol.InstructionMaxDigits {
		t.Fatalf("INSTRUCTION_MAX_DIGITS = %d, want %d", protocol.INSTRUCTION_MAX_DIGITS, protocol.InstructionMaxDigits)
	}
	if protocol.INSTRUCTION_MAX_ELEMENTS != protocol.InstructionMaxElements {
		t.Fatalf("INSTRUCTION_MAX_ELEMENTS = %d, want %d", protocol.INSTRUCTION_MAX_ELEMENTS, protocol.InstructionMaxElements)
	}

	var filter protocol.GuacamoleFilter = func(instruction protocol.Instruction) (protocol.Instruction, error) {
		return protocol.NewInstruction("echo", instruction.GetOpcode()), nil
	}
	filtered, err := filter(protocol.NewInstruction("ping"))
	if err != nil {
		t.Fatalf("filter error = %v", err)
	}
	if filtered.GetOpcode() != "echo" {
		t.Fatalf("filtered opcode = %q, want echo", filtered.GetOpcode())
	}
}
