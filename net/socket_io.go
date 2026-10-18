package net

import (
	"fmt"

	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// SocketGuacamoleReader adapts a Socket to the Java-style GuacamoleReader API.
type SocketGuacamoleReader struct {
	socket Socket
}

// NewSocketReader returns a GuacamoleReader backed by socket.
func NewSocketReader(socket Socket) guacio.GuacamoleReader {
	return &SocketGuacamoleReader{socket: socket}
}

// Available reports whether the underlying socket is open.
func (r *SocketGuacamoleReader) Available() (bool, error) {
	return false, nil
}

// Read reads the next complete instruction as raw Guacamole protocol
// characters.
func (r *SocketGuacamoleReader) Read() ([]rune, error) {
	instruction, err := r.ReadInstruction()
	if err != nil {
		return nil, err
	}
	return instruction.ToCharArray(), nil
}

// ReadInstruction reads the next complete instruction from the socket.
func (r *SocketGuacamoleReader) ReadInstruction() (protocol.Instruction, error) {
	return r.socket.ReceiveInstruction()
}

// SocketGuacamoleWriter adapts a Socket to the Java-style GuacamoleWriter API.
type SocketGuacamoleWriter struct {
	socket Socket
}

// NewSocketWriter returns a GuacamoleWriter backed by socket.
func NewSocketWriter(socket Socket) guacio.GuacamoleWriter {
	return &SocketGuacamoleWriter{socket: socket}
}

// Write parses all complete instructions in chunk and sends them through the
// wrapped socket.
func (w *SocketGuacamoleWriter) Write(chunk []rune) error {
	return w.WriteRange(chunk, 0, len(chunk))
}

// WriteRange parses all complete instructions in the given range and sends
// them through the wrapped socket.
func (w *SocketGuacamoleWriter) WriteRange(chunk []rune, off, length int) error {
	if off < 0 || length < 0 || off+length > len(chunk) {
		return fmt.Errorf("write range [%d:%d] out of bounds for %d runes", off, off+length, len(chunk))
	}

	var parser protocol.Parser
	remaining := []byte(string(chunk[off : off+length]))
	pendingIncomplete := false

	for len(remaining) > 0 {
		consumed := parser.Append(remaining)
		if err := parser.Err(); err != nil {
			return err
		}
		if consumed > 0 {
			remaining = remaining[consumed:]
			pendingIncomplete = true
		}

		if instruction, ok := parser.Next(); ok {
			pendingIncomplete = false
			if err := w.WriteInstruction(instruction); err != nil {
				return err
			}
			continue
		}

		if consumed == 0 {
			break
		}
	}

	if len(remaining) > 0 || pendingIncomplete {
		return &protocol.Error{
			Status:  protocol.StatusServerError,
			Message: "Socket write() contained an incomplete instruction.",
		}
	}
	return nil
}

// WriteInstruction sends instruction through the wrapped socket.
func (w *SocketGuacamoleWriter) WriteInstruction(instruction protocol.Instruction) error {
	return w.socket.SendInstruction(instruction)
}

// Reader returns a GuacamoleReader backed by this socket.
func (s *InetSocket) Reader() guacio.GuacamoleReader {
	return NewSocketReader(s)
}

// GetReader returns a GuacamoleReader backed by this socket.
func (s *InetSocket) GetReader() guacio.GuacamoleReader {
	return s.Reader()
}

// Writer returns a GuacamoleWriter backed by this socket.
func (s *InetSocket) Writer() guacio.GuacamoleWriter {
	return NewSocketWriter(s)
}

// GetWriter returns a GuacamoleWriter backed by this socket.
func (s *InetSocket) GetWriter() guacio.GuacamoleWriter {
	return s.Writer()
}
