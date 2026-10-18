package net_test

import (
	"testing"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestFailoverSocket_PassesThrough(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			{Opcode: "sync", Args: []string{"0"}},
			{Opcode: "nop", Args: []string{}},
		},
	}
	fo, err := guacnet.NewFailoverSocket(s)
	if err != nil {
		t.Fatalf("NewFailoverSocket: %v", err)
	}
	// sync triggers passthrough; subsequent nop should be readable
	instr, err := fo.ReceiveInstruction()
	if err != nil || instr.Opcode != "sync" {
		t.Errorf("first read: err=%v opcode=%q", err, instr.Opcode)
	}
	instr2, err2 := fo.ReceiveInstruction()
	if err2 != nil || instr2.Opcode != "nop" {
		t.Errorf("second read: err=%v opcode=%q", err2, instr2.Opcode)
	}
}

func TestFailoverSocket_NonUpstreamErrorIsReplayed(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			{Opcode: "error", Args: []string{"bad credentials", "769"}},
			{Opcode: "nop"},
		},
	}
	fo, err := guacnet.NewFailoverSocket(s)
	if err != nil {
		t.Fatalf("NewFailoverSocket: %v", err)
	}

	instr, err := fo.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction: %v", err)
	}
	if instr.Opcode != "error" || len(instr.Args) != 2 || instr.Args[1] != "769" {
		t.Fatalf("first replayed instruction = %+v, want client error instruction", instr)
	}

	instr, err = fo.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction after replayed error: %v", err)
	}
	if instr.Opcode != "nop" {
		t.Errorf("next instruction opcode = %q, want nop", instr.Opcode)
	}
}

func TestFailoverSocket_UpstreamErrorInstructionFailsFast(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			{Opcode: "error", Args: []string{"upstream failed", "515"}},
		},
	}
	_, err := guacnet.NewFailoverSocket(s)
	if err == nil {
		t.Fatal("expected error from upstream error instruction")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("expected *protocol.Error, got %T", err)
	}
	if protoErr.Status != protocol.StatusUpstreamError {
		t.Errorf("Status = %v, want StatusUpstreamError", protoErr.Status)
	}
}

func TestFailoverSocket_CustomQueueLimitStopsBeforeUpstreamError(t *testing.T) {
	s := &mockSocket{
		recvQueue: []protocol.Instruction{
			{Opcode: "blob", Args: []string{"abcdefghij"}},
			{Opcode: "error", Args: []string{"upstream failed", "515"}},
		},
	}

	fo, err := guacnet.NewFailoverSocketWithLimit(s, 1)
	if err != nil {
		t.Fatalf("NewFailoverSocketWithLimit: %v", err)
	}

	instr, err := fo.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction first: %v", err)
	}
	if instr.Opcode != "blob" {
		t.Fatalf("first opcode = %q, want blob", instr.Opcode)
	}

	instr, err = fo.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction second: %v", err)
	}
	if instr.Opcode != "error" || len(instr.Args) != 2 || instr.Args[1] != "515" {
		t.Fatalf("second instruction = %+v, want replayed upstream error instruction", instr)
	}
}
