package net

import (
	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// FilteredSocket wraps a Socket and applies read/write filters to instruction streams.
type FilteredSocket struct {
	Socket
	readFilter  protocol.Filter
	writeFilter protocol.Filter
}

// NewFilteredSocket creates a FilteredSocket. nil filters pass instructions through unchanged.
func NewFilteredSocket(socket Socket, readFilter, writeFilter protocol.Filter) *FilteredSocket {
	return &FilteredSocket{
		Socket:      socket,
		readFilter:  readFilter,
		writeFilter: writeFilter,
	}
}

func (s *FilteredSocket) Reader() guacio.GuacamoleReader {
	reader := s.Socket.Reader()
	if s.readFilter == nil {
		return reader
	}
	return guacio.NewFilteredGuacamoleReader(reader, s.readFilter)
}

func (s *FilteredSocket) GetReader() guacio.GuacamoleReader {
	return s.Reader()
}

func (s *FilteredSocket) Writer() guacio.GuacamoleWriter {
	writer := s.Socket.Writer()
	if s.writeFilter == nil {
		return writer
	}
	return guacio.NewFilteredGuacamoleWriter(writer, s.writeFilter)
}

func (s *FilteredSocket) GetWriter() guacio.GuacamoleWriter {
	return s.Writer()
}

func (s *FilteredSocket) GetProtocol() string {
	return s.Socket.Protocol()
}

func (s *FilteredSocket) ReceiveInstruction() (protocol.Instruction, error) {
	for {
		instr, err := s.Socket.ReceiveInstruction()
		if err != nil {
			return protocol.Instruction{}, err
		}
		if s.readFilter == nil {
			return instr, nil
		}
		filtered, err := s.readFilter(instr)
		if err == protocol.ErrDropInstruction {
			continue
		}
		if err != nil {
			return protocol.Instruction{}, err
		}
		return filtered, nil
	}
}

func (s *FilteredSocket) SendInstruction(instr protocol.Instruction) error {
	if s.writeFilter == nil {
		return s.Socket.SendInstruction(instr)
	}
	filtered, err := s.writeFilter(instr)
	if err == protocol.ErrDropInstruction {
		return nil
	}
	if err != nil {
		return err
	}
	return s.Socket.SendInstruction(filtered)
}
