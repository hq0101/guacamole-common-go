package net

import (
	"fmt"
	"strconv"

	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// ConfiguredSocket wraps a Socket and performs the full Guacamole handshake.
type ConfiguredSocket struct {
	Socket
	config       *protocol.Config
	connectionID string
	protocolName string
	version      protocol.ProtocolVersion
}

// NewConfiguredSocket performs the Guacamole handshake over socket.
// info may be nil; default display dimensions (1024×768@96dpi) are used.
func NewConfiguredSocket(socket Socket, cfg *protocol.Config, info *protocol.ClientInfo) (*ConfiguredSocket, error) {
	cs := &ConfiguredSocket{Socket: socket, config: cfg}
	if err := cs.handshake(cfg, info); err != nil {
		return nil, err
	}
	return cs, nil
}

// ConnectionID returns the connection ID assigned by guacd after a successful handshake.
func (cs *ConfiguredSocket) ConnectionID() string { return cs.connectionID }

// GetConnectionID returns the connection ID assigned by guacd after a
// successful handshake.
func (cs *ConfiguredSocket) GetConnectionID() string { return cs.ConnectionID() }

// ProtocolVersion returns the negotiated Guacamole protocol version.
func (cs *ConfiguredSocket) ProtocolVersion() protocol.ProtocolVersion { return cs.version }

// GetProtocolVersion returns the negotiated Guacamole protocol version.
func (cs *ConfiguredSocket) GetProtocolVersion() protocol.ProtocolVersion {
	return cs.ProtocolVersion()
}

// GetConfiguration returns the configuration used to configure this socket.
func (cs *ConfiguredSocket) GetConfiguration() *protocol.Config { return cs.config }

// Protocol returns the configured protocol name for this Guacamole connection.
func (cs *ConfiguredSocket) Protocol() string { return cs.GetProtocol() }

// GetProtocol returns the configured protocol name for this Guacamole
// connection.
func (cs *ConfiguredSocket) GetProtocol() string {
	if cs.config != nil {
		return cs.config.GetProtocol()
	}
	return cs.protocolName
}

// GetReader returns the GuacamoleReader for this configured socket.
func (cs *ConfiguredSocket) GetReader() guacio.GuacamoleReader {
	return cs.Socket.Reader()
}

// GetWriter returns the GuacamoleWriter for this configured socket.
func (cs *ConfiguredSocket) GetWriter() guacio.GuacamoleWriter {
	return cs.Socket.Writer()
}

func (cs *ConfiguredSocket) handshake(cfg *protocol.Config, info *protocol.ClientInfo) error {
	cs.protocolName = cfg.Protocol

	// Step 1: send select
	selectArg := cfg.Protocol
	if cfg.ConnectionID != "" || cfg.ConnectionIDSet {
		selectArg = cfg.ConnectionID
	}
	if err := cs.SendInstruction(protocol.Instruction{Opcode: "select", Args: []string{selectArg}}); err != nil {
		return fmt.Errorf("handshake: sending select: %w", err)
	}

	// Step 2: receive args
	argsInstr, err := cs.ReceiveInstruction()
	if err != nil {
		return fmt.Errorf("handshake: receiving args: %w", err)
	}
	if err := handleHandshakeControl(argsInstr, "args"); err != nil {
		return err
	}
	if argsInstr.Opcode != "args" {
		return &protocol.Error{
			Status:  protocol.StatusServerError,
			Message: fmt.Sprintf("handshake: expected 'args', got %q", argsInstr.Opcode),
		}
	}

	// Step 3: parse server version from first arg if present
	serverVersion := protocol.Version1_0_0
	argKeys := argsInstr.Args
	if len(argKeys) > 0 {
		if v, ok := protocol.ParseVersion(argKeys[0]); ok {
			serverVersion = v
			// Do NOT remove VERSION from argKeys: Java guacamole-common
			// preserves the slot and returns the negotiated version in CONNECT.
		}
	}
	// Negotiate: min(clientVersion, serverVersion)
	clientVersion := protocol.VersionLatest
	if serverVersion.AtLeast(clientVersion) {
		cs.version = clientVersion
	} else {
		cs.version = serverVersion
	}

	// Steps 4-9: send capability instructions
	if err := cs.sendCapabilities(info); err != nil {
		return err
	}

	// Step 10: send connect with params in argKeys order
	var params map[string]string
	if cfg.Params != nil {
		params = cfg.Params
	}
	values := make([]string, len(argKeys))
	for i, key := range argKeys {
		if i == 0 {
			if _, ok := protocol.ParseVersion(key); ok {
				values[i] = cs.version.String()
				continue
			}
		}
		values[i] = params[key]
	}
	if err := cs.SendInstruction(protocol.Instruction{Opcode: "connect", Args: values}); err != nil {
		return fmt.Errorf("handshake: sending connect: %w", err)
	}

	// Step 11: receive ready or error
	resp, err := cs.ReceiveInstruction()
	if err != nil {
		return fmt.Errorf("handshake: receiving ready: %w", err)
	}
	if err := handleHandshakeControl(resp, "ready"); err != nil {
		return err
	}
	if resp.Opcode != "ready" {
		return &protocol.Error{
			Status:  protocol.StatusServerError,
			Message: fmt.Sprintf("handshake: expected 'ready', got %q", resp.Opcode),
		}
	}
	if len(resp.Args) == 0 {
		return &protocol.Error{
			Status:  protocol.StatusServerError,
			Message: "handshake: no connection ID received",
		}
	}
	cs.connectionID = resp.Args[0]
	return nil
}

func handleHandshakeControl(instr protocol.Instruction, expectedOpcode string) error {
	switch instr.Opcode {
	case "error":
		return errorFromInstruction(instr)
	case "disconnect":
		return &protocol.Error{
			Status:  protocol.StatusServerError,
			Message: fmt.Sprintf("handshake: server disconnected while waiting for %q", expectedOpcode),
		}
	default:
		return nil
	}
}

func errorFromInstruction(instr protocol.Instruction) error {
	message := "Internal error within guacd / protocol handling."
	status := protocol.StatusServerError

	if len(instr.Args) >= 1 {
		message = instr.Args[0]
	}
	if len(instr.Args) >= 2 {
		if code, err := strconv.Atoi(instr.Args[1]); err == nil {
			if parsed, ok := protocol.StatusFromCode(code); ok {
				status = parsed
			}
		}
	}

	return &protocol.Error{Status: status, Message: message}
}

func (cs *ConfiguredSocket) sendCapabilities(info *protocol.ClientInfo) error {
	// Step 4: send size
	w, h, dpi := 1024, 768, 96
	var audio, video, image []string
	if info != nil {
		w = info.GetOptimalScreenWidth()
		h = info.GetOptimalScreenHeight()
		dpi = info.GetOptimalResolution()
		audio = info.GetAudioMimetypes()
		video = info.GetVideoMimetypes()
		image = info.GetImageMimetypes()
	}
	if err := cs.SendInstruction(protocol.Instruction{
		Opcode: "size",
		Args:   []string{strconv.Itoa(w), strconv.Itoa(h), strconv.Itoa(dpi)},
	}); err != nil {
		return fmt.Errorf("handshake: sending size: %w", err)
	}

	// Steps 5-7: Java guacamole-common always sends these capability
	// instructions, even when the corresponding mimetype lists are empty.
	if err := cs.SendInstruction(protocol.Instruction{Opcode: "audio", Args: audio}); err != nil {
		return fmt.Errorf("handshake: sending audio: %w", err)
	}
	if err := cs.SendInstruction(protocol.Instruction{Opcode: "video", Args: video}); err != nil {
		return fmt.Errorf("handshake: sending video: %w", err)
	}
	if err := cs.SendInstruction(protocol.Instruction{Opcode: "image", Args: image}); err != nil {
		return fmt.Errorf("handshake: sending image: %w", err)
	}

	// Step 8: send timezone if protocol >= 1.1.0 and timezone is set.
	if info != nil && protocol.CapabilityTimezoneHandshake.IsSupported(cs.version) {
		if timezone, ok := info.TimezoneValue(); ok {
			if err := cs.SendInstruction(protocol.Instruction{Opcode: "timezone", Args: []string{timezone}}); err != nil {
				return fmt.Errorf("handshake: sending timezone: %w", err)
			}
		}
	}

	// Step 9: send name if protocol >= 1.5.0 and name is set.
	if info != nil && protocol.CapabilityNameHandshake.IsSupported(cs.version) {
		if name, ok := info.NameValue(); ok {
			if err := cs.SendInstruction(protocol.Instruction{Opcode: "name", Args: []string{name}}); err != nil {
				return fmt.Errorf("handshake: sending name: %w", err)
			}
		}
	}

	return nil
}
