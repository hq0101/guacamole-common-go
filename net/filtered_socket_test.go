package net_test

import (
	"errors"
	"testing"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestFilteredSocket_ReadFilter(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			{Opcode: "drop-me", Args: []string{}},
			{Opcode: "keep-me", Args: []string{}},
		},
	}
	dropFilter := protocol.Filter(func(instr protocol.Instruction) (protocol.Instruction, error) {
		if instr.Opcode == "drop-me" {
			return protocol.Instruction{}, protocol.ErrDropInstruction
		}
		return instr, nil
	})
	fs := guacnet.NewFilteredSocket(s, dropFilter, nil)

	got, err := fs.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction: %v", err)
	}
	if got.Opcode != "keep-me" {
		t.Errorf("expected keep-me, got %q", got.Opcode)
	}
}

func TestFilteredSocket_WriteFilter(t *testing.T) {
	s := &mockSocket{}
	mutateFilter := protocol.Filter(func(instr protocol.Instruction) (protocol.Instruction, error) {
		instr.Opcode = "mutated"
		return instr, nil
	})
	fs := guacnet.NewFilteredSocket(s, nil, mutateFilter)

	fs.SendInstruction(protocol.Instruction{Opcode: "original"})
	if len(s.sendBuf) != 1 || s.sendBuf[0].Opcode != "mutated" {
		t.Errorf("expected mutated instruction, got %v", s.sendBuf)
	}
}

func TestFilteredSocket_WriteFilterDrop(t *testing.T) {
	s := &mockSocket{}
	dropAll := protocol.Filter(func(instr protocol.Instruction) (protocol.Instruction, error) {
		return protocol.Instruction{}, protocol.ErrDropInstruction
	})
	fs := guacnet.NewFilteredSocket(s, nil, dropAll)
	fs.SendInstruction(protocol.Instruction{Opcode: "anything"})
	if len(s.sendBuf) != 0 {
		t.Error("dropped instruction should not reach underlying socket")
	}
}

func TestFilteredSocket_FilterError(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{{Opcode: "bad"}},
	}
	errFilter := protocol.Filter(func(instr protocol.Instruction) (protocol.Instruction, error) {
		return protocol.Instruction{}, errors.New("filter error")
	})
	fs := guacnet.NewFilteredSocket(s, errFilter, nil)
	_, err := fs.ReceiveInstruction()
	if err == nil || err.Error() != "filter error" {
		t.Errorf("expected filter error, got %v", err)
	}
}
