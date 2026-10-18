package net_test

import (
	"testing"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestSocketReaderWriterAdaptInstructions(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			protocol.NewInstruction("sync", "0"),
		},
	}
	var socket guacnet.Socket = s

	instr, err := socket.Reader().ReadInstruction()
	if err != nil {
		t.Fatalf("Reader().ReadInstruction() error = %v", err)
	}
	if instr.Opcode != "sync" || len(instr.Args) != 1 || instr.Args[0] != "0" {
		t.Fatalf("Reader().ReadInstruction() = %+v, want sync 0", instr)
	}

	if err := socket.Writer().WriteInstruction(protocol.NewInstruction("nop")); err != nil {
		t.Fatalf("Writer().WriteInstruction() error = %v", err)
	}
	if len(s.sendBuf) != 1 || s.sendBuf[0].Opcode != "nop" {
		t.Fatalf("sendBuf = %+v, want nop instruction", s.sendBuf)
	}
}

func TestSocketReaderAvailableReturnsFalseWhenUnknown(t *testing.T) {
	reader := guacnet.NewSocketReader(&mockSocket{})

	available, err := reader.Available()
	if err != nil {
		t.Fatalf("Available() error = %v", err)
	}
	if available {
		t.Fatal("Available() = true, want false when immediate availability is unknown")
	}
}

func TestFailoverSocketReaderReplaysBufferedInstructions(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			protocol.NewInstruction("sync", "0"),
			protocol.NewInstruction("nop"),
		},
	}
	fo, err := guacnet.NewFailoverSocket(s)
	if err != nil {
		t.Fatalf("NewFailoverSocket() error = %v", err)
	}

	reader := fo.Reader()
	first, err := reader.ReadInstruction()
	if err != nil {
		t.Fatalf("first ReadInstruction() error = %v", err)
	}
	second, err := reader.ReadInstruction()
	if err != nil {
		t.Fatalf("second ReadInstruction() error = %v", err)
	}
	if first.Opcode != "sync" || second.Opcode != "nop" {
		t.Fatalf("read opcodes = %q, %q; want sync, nop", first.Opcode, second.Opcode)
	}
}

func TestFilteredSocketReaderWriterUseFilters(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			protocol.NewInstruction("drop"),
			protocol.NewInstruction("keep"),
		},
	}
	readFilter := func(instr protocol.Instruction) (protocol.Instruction, error) {
		if instr.Opcode == "drop" {
			return protocol.Instruction{}, protocol.ErrDropInstruction
		}
		return instr, nil
	}
	writeFilter := func(instr protocol.Instruction) (protocol.Instruction, error) {
		return protocol.NewInstruction("mutated", instr.Opcode), nil
	}
	fs := guacnet.NewFilteredSocket(s, readFilter, writeFilter)

	instr, err := fs.Reader().ReadInstruction()
	if err != nil {
		t.Fatalf("Reader().ReadInstruction() error = %v", err)
	}
	if instr.Opcode != "keep" {
		t.Fatalf("filtered read opcode = %q, want keep", instr.Opcode)
	}

	if err := fs.Writer().WriteInstruction(protocol.NewInstruction("original")); err != nil {
		t.Fatalf("Writer().WriteInstruction() error = %v", err)
	}
	if len(s.sendBuf) != 1 || s.sendBuf[0].Opcode != "mutated" || s.sendBuf[0].Args[0] != "original" {
		t.Fatalf("sendBuf = %+v, want mutated original", s.sendBuf)
	}
}
