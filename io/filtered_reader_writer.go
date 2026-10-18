package io

import (
	"fmt"

	"github.com/hq0101/guacamole-common-go/protocol"
)

// FilteredGuacamoleReader wraps a GuacamoleReader, applying a filter to each
// instruction read. Instructions dropped by the filter are skipped.
type FilteredGuacamoleReader struct {
	reader GuacamoleReader
	filter protocol.Filter
}

// NewFilteredGuacamoleReader returns a Guacamole reader which filters all
// instructions produced by reader.
func NewFilteredGuacamoleReader(reader GuacamoleReader, filter protocol.Filter) *FilteredGuacamoleReader {
	return &FilteredGuacamoleReader{reader: reader, filter: filter}
}

// Available delegates to the wrapped reader.
func (r *FilteredGuacamoleReader) Available() (bool, error) {
	return r.reader.Available()
}

// Read reads the next instruction accepted by the filter and returns its raw
// Guacamole protocol characters.
func (r *FilteredGuacamoleReader) Read() ([]rune, error) {
	instruction, err := r.ReadInstruction()
	if err != nil {
		return nil, err
	}
	return instruction.ToCharArray(), nil
}

// ReadInstruction reads and filters instructions until one is accepted.
func (r *FilteredGuacamoleReader) ReadInstruction() (protocol.Instruction, error) {
	for {
		instruction, err := r.reader.ReadInstruction()
		if err != nil {
			return protocol.Instruction{}, err
		}
		if r.filter == nil {
			return instruction, nil
		}

		filtered, err := r.filter(instruction)
		if err == protocol.ErrDropInstruction {
			continue
		}
		if err != nil {
			return protocol.Instruction{}, err
		}
		return filtered, nil
	}
}

// FilteredGuacamoleWriter wraps a GuacamoleWriter, applying a filter to each
// instruction written. Instructions dropped by the filter are not written.
type FilteredGuacamoleWriter struct {
	writer GuacamoleWriter
	filter protocol.Filter
}

// NewFilteredGuacamoleWriter returns a Guacamole writer which filters all
// instructions written to writer.
func NewFilteredGuacamoleWriter(writer GuacamoleWriter, filter protocol.Filter) *FilteredGuacamoleWriter {
	return &FilteredGuacamoleWriter{writer: writer, filter: filter}
}

// Write parses all instructions in chunk and writes each instruction accepted
// by the filter.
func (w *FilteredGuacamoleWriter) Write(chunk []rune) error {
	return w.WriteRange(chunk, 0, len(chunk))
}

// WriteRange parses all instructions in the given range and writes each
// instruction accepted by the filter.
func (w *FilteredGuacamoleWriter) WriteRange(chunk []rune, off, length int) error {
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
		return filteredIncompleteInstructionError()
	}
	return nil
}

// WriteInstruction applies the filter and writes the resulting instruction.
func (w *FilteredGuacamoleWriter) WriteInstruction(instruction protocol.Instruction) error {
	if w.filter == nil {
		return w.writer.WriteInstruction(instruction)
	}

	filtered, err := w.filter(instruction)
	if err == protocol.ErrDropInstruction {
		return nil
	}
	if err != nil {
		return err
	}
	return w.writer.WriteInstruction(filtered)
}

func filteredIncompleteInstructionError() error {
	return &protocol.Error{
		Status:  protocol.StatusServerError,
		Message: "Filtered write() contained an incomplete instruction.",
	}
}
