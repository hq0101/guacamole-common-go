package net_test

import (
	"context"
	"net"
	"testing"
	"time"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestInetSocket_SendReceive(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()

	sock := guacnet.NewInetSocketFromConn(clientConn)
	defer sock.Close()

	// Server side: write an instruction
	want := protocol.Instruction{Opcode: "sync", Args: []string{"0"}}
	go func() {
		serverConn.Write(want.Encode())
	}()

	// Client side: receive it
	got, err := sock.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction: %v", err)
	}
	if got.Opcode != want.Opcode || got.Args[0] != want.Args[0] {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestInetSocket_IsOpen(t *testing.T) {
	_, clientConn := net.Pipe()
	sock := guacnet.NewInetSocketFromConn(clientConn)
	if !sock.IsOpen() {
		t.Error("newly created socket should be open")
	}
	sock.Close()
	if sock.IsOpen() {
		t.Error("closed socket should not be open")
	}
}

func TestInetSocket_ReceiveInstructionReturnsParserError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()

	sock := guacnet.NewInetSocketFromConn(clientConn)
	defer sock.Close()

	if err := clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}

	go func() {
		_, _ = serverConn.Write([]byte("x.foo;"))
	}()

	_, err := sock.ReceiveInstruction()
	if err == nil {
		t.Fatal("ReceiveInstruction() error = nil, want parser error")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("ReceiveInstruction() error type = %T, want *protocol.Error", err)
	}
	if protoErr.Status != protocol.StatusServerError {
		t.Errorf("Status = %v, want StatusServerError", protoErr.Status)
	}
}

func TestInetSocket_ReceiveInstructionSplitUTF8Rune(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()

	sock := guacnet.NewInetSocketFromConn(clientConn)
	defer sock.Close()

	data := []byte("4.echo,2.你好;")
	split := len([]byte("4.echo,2.")) + 1
	go func() {
		_, _ = serverConn.Write(data[:split])
		time.Sleep(10 * time.Millisecond)
		_, _ = serverConn.Write(data[split:])
	}()

	got, err := sock.ReceiveInstruction()
	if err != nil {
		t.Fatalf("ReceiveInstruction: %v", err)
	}
	if got.Opcode != "echo" || len(got.Args) != 1 || got.Args[0] != "你好" {
		t.Errorf("instruction = %+v, want echo [你好]", got)
	}
}

func TestDial_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	_, err := guacnet.Dial(ctx, "127.0.0.1:19999")
	if err == nil {
		t.Error("expected error dialing cancelled context")
	}
}
