package net

import (
	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// InternalDataOpcode is the opcode used for internal tunnel messages
// (UUID transmission and ping/pong). Not forwarded to guacd.
const InternalDataOpcode = ""

// Socket is the basic transport abstraction for communicating with guacd.
type Socket interface {
	SendInstruction(protocol.Instruction) error
	ReceiveInstruction() (protocol.Instruction, error)
	Reader() guacio.GuacamoleReader
	Writer() guacio.GuacamoleWriter
	// Protocol returns the negotiated protocol name after handshake, or "" before.
	Protocol() string
	Close() error
	IsOpen() bool
}
