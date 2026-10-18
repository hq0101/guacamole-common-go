package io

import (
	"errors"
	"fmt"
	stdio "io"
	"net"

	"github.com/hq0101/guacamole-common-go/protocol"
)

// GuacamoleReader provides raw and parsed read access to a stream of
// Guacamole instructions.
type GuacamoleReader interface {
	Available() (bool, error)
	Read() ([]rune, error)
	ReadInstruction() (protocol.Instruction, error)
}

// GuacamoleWriter provides raw and parsed write access to a stream of
// Guacamole instructions.
type GuacamoleWriter interface {
	Write([]rune) error
	WriteRange([]rune, int, int) error
	WriteInstruction(protocol.Instruction) error
}

type lenReader interface {
	Len() int
}

type flusher interface {
	Flush() error
}

// ReaderGuacamoleReader wraps a standard Go io.Reader as a Guacamole
// instruction stream.
type ReaderGuacamoleReader struct {
	input   stdio.Reader
	parser  protocol.Parser
	pending []byte
	eof     bool
}

// NewReaderGuacamoleReader returns a Guacamole reader backed by input.
func NewReaderGuacamoleReader(input stdio.Reader) *ReaderGuacamoleReader {
	return &ReaderGuacamoleReader{input: input}
}

// Available reports whether instruction data appears to be immediately
// available for reading.
func (r *ReaderGuacamoleReader) Available() (bool, error) {
	if len(r.pending) > 0 {
		return true, nil
	}
	if input, ok := r.input.(lenReader); ok {
		return input.Len() > 0, nil
	}
	return false, nil
}

// Read reads exactly one complete Guacamole instruction and returns its raw
// Guacamole protocol characters.
func (r *ReaderGuacamoleReader) Read() ([]rune, error) {
	instruction, err := r.ReadInstruction()
	if err != nil {
		return nil, err
	}
	return instruction.ToCharArray(), nil
}

// ReadInstruction reads exactly one complete Guacamole instruction.
func (r *ReaderGuacamoleReader) ReadInstruction() (protocol.Instruction, error) {
	var buf [20480]byte

	for {
		if instruction, ok := r.parser.Next(); ok {
			return instruction, nil
		}

		for len(r.pending) > 0 {
			consumed := r.parser.Append(r.pending)
			if err := r.parser.Err(); err != nil {
				return protocol.Instruction{}, err
			}
			if consumed > 0 {
				r.pending = r.pending[consumed:]
			}
			if instruction, ok := r.parser.Next(); ok {
				return instruction, nil
			}
			if consumed == 0 {
				break
			}
		}

		if r.eof {
			return protocol.Instruction{}, stdio.EOF
		}

		n, err := r.input.Read(buf[:])
		if n > 0 {
			r.pending = append(r.pending, buf[:n]...)
			continue
		}
		if err != nil {
			if errors.Is(err, stdio.EOF) {
				r.eof = true
				continue
			}
			return protocol.Instruction{}, mapIOError(err)
		}
	}
}

// WriterGuacamoleWriter wraps a standard Go io.Writer as a Guacamole
// instruction stream.
type WriterGuacamoleWriter struct {
	output stdio.Writer
}

// NewWriterGuacamoleWriter returns a Guacamole writer backed by output.
func NewWriterGuacamoleWriter(output stdio.Writer) *WriterGuacamoleWriter {
	return &WriterGuacamoleWriter{output: output}
}

// Write writes all given Guacamole protocol characters and flushes the
// underlying writer if flushing is supported.
func (w *WriterGuacamoleWriter) Write(chunk []rune) error {
	return w.WriteRange(chunk, 0, len(chunk))
}

// WriteRange writes the given range of Guacamole protocol characters and
// flushes the underlying writer if flushing is supported.
func (w *WriterGuacamoleWriter) WriteRange(chunk []rune, off, length int) error {
	if off < 0 || length < 0 || off+length > len(chunk) {
		return fmt.Errorf("write range [%d:%d] out of bounds for %d runes", off, off+length, len(chunk))
	}
	data := []byte(string(chunk[off : off+length]))
	n, err := w.output.Write(data)
	if err != nil {
		return mapIOError(err)
	}
	if n != len(data) {
		return mapIOError(stdio.ErrShortWrite)
	}
	if output, ok := w.output.(flusher); ok {
		if err := output.Flush(); err != nil {
			return mapIOError(err)
		}
	}
	return nil
}

// WriteInstruction writes the given fully parsed instruction.
func (w *WriterGuacamoleWriter) WriteInstruction(instruction protocol.Instruction) error {
	return w.Write(instruction.ToCharArray())
}

func mapIOError(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &protocol.Error{Status: protocol.StatusUpstreamTimeout, Message: "Connection to guacd timed out."}
	}
	if errors.Is(err, net.ErrClosed) {
		return &protocol.Error{Status: protocol.StatusServerError, Message: "Connection to guacd is closed."}
	}
	return &protocol.Error{Status: protocol.StatusServerError, Message: err.Error()}
}
