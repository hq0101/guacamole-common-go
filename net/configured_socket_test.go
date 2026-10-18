package net_test

import (
	"context"
	"net"
	"reflect"
	"testing"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
)

// simulateGuacd performs the guacd handshake on conn.
func simulateGuacd(t *testing.T, conn net.Conn, serverVersion string, argKeys []string, connectionID string) {
	t.Helper()
	defer conn.Close()

	sock := guacnet.NewInetSocketFromConn(conn)

	// Expect select
	sel, err := sock.ReceiveInstruction()
	if err != nil || sel.Opcode != "select" {
		t.Errorf("expected select, got %v %v", sel.Opcode, err)
		return
	}

	// Send args (with optional version as first arg)
	args := argKeys
	if serverVersion != "" {
		args = append([]string{serverVersion}, argKeys...)
	}
	sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: args})

	// Consume capability instructions until connect
	for {
		instr, err := sock.ReceiveInstruction()
		if err != nil {
			return
		}
		if instr.Opcode == "connect" {
			break
		}
	}

	sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{connectionID}})
}

func TestConfiguredSocket_BasicHandshake(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	argKeys := []string{"hostname", "port", "username", "password"}
	go simulateGuacd(t, serverConn, "", argKeys, "$conn-123")

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{
		Protocol: "rdp",
		Params: map[string]string{
			"hostname": "192.168.1.1",
			"port":     "3389",
		},
	}
	cs, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}
	if cs.ConnectionID() != "$conn-123" {
		t.Errorf("ConnectionID() = %q, want $conn-123", cs.ConnectionID())
	}
}

func TestConfiguredSocket_ProtocolReturnsConfiguredProtocol(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go simulateGuacd(t, serverConn, "", []string{"hostname"}, "$protocol-conn")

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "ssh", Params: map[string]string{"hostname": "host"}}
	cs, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}
	if got := cs.Protocol(); got != "ssh" {
		t.Errorf("Protocol() = %q, want ssh", got)
	}
}

func TestConfiguredSocket_JavaStyleAccessors(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go simulateGuacd(t, serverConn, "VERSION_1_3_0", []string{"hostname"}, "$java-accessors")

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := protocol.NewConfig()
	cfg.SetProtocol("rdp")
	cfg.SetParameter("hostname", "host")

	cs, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}
	if cs.GetConfiguration() != cfg {
		t.Fatal("GetConfiguration() did not return original config")
	}
	if cs.GetConnectionID() != "$java-accessors" {
		t.Fatalf("GetConnectionID() = %q, want $java-accessors", cs.GetConnectionID())
	}
	if cs.GetProtocolVersion() != protocol.Version1_3_0 {
		t.Fatalf("GetProtocolVersion() = %v, want VERSION_1_3_0", cs.GetProtocolVersion())
	}

	cfg.SetProtocol("ssh")
	if cs.GetProtocol() != "ssh" {
		t.Fatalf("GetProtocol() = %q after config mutation, want ssh", cs.GetProtocol())
	}
}

func TestConfiguredSocket_ExplicitEmptyConnectionIDIsSelected(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	selectArgs := make(chan []string, 1)
	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sel, err := sock.ReceiveInstruction()
		if err != nil {
			t.Errorf("ReceiveInstruction(select): %v", err)
			return
		}
		if sel.Opcode != "select" {
			t.Errorf("select opcode = %q, want select", sel.Opcode)
			return
		}
		selectArgs <- append([]string(nil), sel.Args...)
		sock.SendInstruction(protocol.Instruction{Opcode: "args"})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{"$empty-select"}})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp"}
	cfg.SetConnectionID("")
	if _, err := guacnet.NewConfiguredSocket(inet, cfg, nil); err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}

	args := <-selectArgs
	if len(args) != 1 {
		t.Fatalf("select args = %v, want one empty arg", args)
	}
	if args[0] != "" {
		t.Errorf("select arg = %q, want empty string", args[0])
	}
}

func TestConfiguredSocket_VersionNegotiation(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go simulateGuacd(t, serverConn, "VERSION_1_3_0", []string{"hostname"}, "$v13-conn")

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	cs, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err != nil {
		t.Fatalf("NewConfiguredSocket with VERSION: %v", err)
	}
	// Negotiated version = min(1.5.0, 1.3.0) = 1.3.0
	if cs.ProtocolVersion() != protocol.Version1_3_0 {
		t.Errorf("ProtocolVersion() = %v, want 1.3.0", cs.ProtocolVersion())
	}
}

func TestConfiguredSocket_ConnectSendsNegotiatedVersion(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	connectArgs := make(chan []string, 1)
	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"VERSION_1_3_0", "hostname"}})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			if instr.Opcode == "connect" {
				connectArgs <- instr.Args
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{"$version-conn"}})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	if _, err := guacnet.NewConfiguredSocket(inet, cfg, nil); err != nil {
		t.Fatalf("NewConfiguredSocket with VERSION: %v", err)
	}

	args := <-connectArgs
	if len(args) != 2 {
		t.Fatalf("connect args = %v, want 2 args", args)
	}
	if args[0] != "VERSION_1_3_0" {
		t.Errorf("connect VERSION arg = %q, want VERSION_1_3_0", args[0])
	}
	if args[1] != "host" {
		t.Errorf("connect hostname arg = %q, want host", args[1])
	}
}

func TestConfiguredSocket_DefaultClientInfoMatchesJavaHandshake(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	received := make(chan protocol.Instruction, 8)
	go func() {
		defer serverConn.Close()
		defer close(received)
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"hostname"}})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			received <- instr
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{"$default-info"}})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	if _, err := guacnet.NewConfiguredSocket(inet, cfg, nil); err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}

	var got []protocol.Instruction
	for instr := range received {
		got = append(got, instr)
	}
	if len(got) != 5 {
		t.Fatalf("received %d handshake instructions, want 5: %v", len(got), got)
	}
	wantOpcodes := []string{"size", "audio", "video", "image", "connect"}
	for i, want := range wantOpcodes {
		if got[i].Opcode != want {
			t.Fatalf("instruction %d opcode = %q, want %q; all=%v", i, got[i].Opcode, want, got)
		}
	}
	if args := got[0].Args; len(args) != 3 || args[0] != "1024" || args[1] != "768" || args[2] != "96" {
		t.Errorf("size args = %v, want [1024 768 96]", got[0].Args)
	}
	for _, idx := range []int{1, 2, 3} {
		if len(got[idx].Args) != 0 {
			t.Errorf("%s args = %v, want none", got[idx].Opcode, got[idx].Args)
		}
	}
}

func TestConfiguredSocket_ExplicitZeroClientInfoDimensionsAreSent(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	sizeArgs := make(chan []string, 1)
	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args"})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			if instr.Opcode == "size" {
				sizeArgs <- append([]string(nil), instr.Args...)
			}
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{"$zero-size"}})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp"}
	info := protocol.NewClientInfo()
	info.SetOptimalScreenWidth(0)
	info.SetOptimalScreenHeight(0)
	info.SetOptimalResolution(0)
	if _, err := guacnet.NewConfiguredSocket(inet, cfg, info); err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}

	args := <-sizeArgs
	if want := []string{"0", "0", "0"}; !reflect.DeepEqual(args, want) {
		t.Errorf("size args = %v, want %v", args, want)
	}
}

func TestConfiguredSocket_ErrorResponse(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"hostname"}})
		for {
			instr, _ := sock.ReceiveInstruction()
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{
			Opcode: "error",
			Args:   []string{"Invalid credentials", "769"}, // 0x0301
		})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "bad"}}
	_, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err == nil {
		t.Fatal("expected error from guacd error response")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("expected *protocol.Error, got %T: %v", err, err)
	}
	if protoErr.Status != protocol.StatusClientUnauthorized {
		t.Errorf("Status = %v, want StatusClientUnauthorized", protoErr.Status)
	}
}

func TestConfiguredSocket_ErrorBeforeArgsUsesGuacamoleStatus(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{
			Opcode: "error",
			Args:   []string{"Denied", "769"}, // 0x0301
		})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp"}
	_, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err == nil {
		t.Fatal("expected error from guacd error response")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("expected *protocol.Error, got %T: %v", err, err)
	}
	if protoErr.Status != protocol.StatusClientUnauthorized {
		t.Errorf("Status = %v, want StatusClientUnauthorized", protoErr.Status)
	}
	if protoErr.Message != "Denied" {
		t.Errorf("Message = %q, want Denied", protoErr.Message)
	}
}

func TestConfiguredSocket_DisconnectBeforeReadyIsServerError(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"hostname"}})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "disconnect"})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	_, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err == nil {
		t.Fatal("expected disconnect error")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("expected *protocol.Error, got %T: %v", err, err)
	}
	if protoErr.Status != protocol.StatusServerError {
		t.Errorf("Status = %v, want StatusServerError", protoErr.Status)
	}
}

func TestConfiguredSocket_ReadyWithoutConnectionIDIsError(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"hostname"}})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready"})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	_, err := guacnet.NewConfiguredSocket(inet, cfg, nil)
	if err == nil {
		t.Fatal("NewConfiguredSocket() error = nil, want missing connection ID error")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("error type = %T, want *protocol.Error", err)
	}
	if protoErr.Status != protocol.StatusServerError {
		t.Errorf("Status = %v, want StatusServerError", protoErr.Status)
	}
}

func TestConfiguredSocket_WithClientInfo(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	received := make(chan []string, 10)
	go func() {
		defer serverConn.Close()
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"VERSION_1_5_0", "hostname"}})
		for {
			instr, _ := sock.ReceiveInstruction()
			received <- append([]string{instr.Opcode}, instr.Args...)
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{"$id"}})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	info := &protocol.ClientInfo{
		OptimalWidth:      1920,
		OptimalHeight:     1080,
		OptimalResolution: 96,
		Timezone:          "Asia/Shanghai",
		Name:              "alice",
		AudioMimetypes:    []string{"audio/L16"},
	}
	if _, err := guacnet.NewConfiguredSocket(inet, cfg, info); err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}

	// Collect received opcodes
	close(received)
	var opcodes []string
	for parts := range received {
		opcodes = append(opcodes, parts[0])
	}
	// Must contain size, audio, timezone, name (protocol 1.5.0)
	must := []string{"size", "audio", "timezone", "name", "connect"}
	for _, want := range must {
		found := false
		for _, got := range opcodes {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q instruction, opcodes: %v", want, opcodes)
		}
	}
}

func TestConfiguredSocket_ExplicitEmptyClientInfoValuesAreSent(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	received := make(chan protocol.Instruction, 10)
	go func() {
		defer serverConn.Close()
		defer close(received)
		sock := guacnet.NewInetSocketFromConn(serverConn)
		sock.ReceiveInstruction() // select
		sock.SendInstruction(protocol.Instruction{Opcode: "args", Args: []string{"VERSION_1_5_0", "hostname"}})
		for {
			instr, err := sock.ReceiveInstruction()
			if err != nil {
				return
			}
			received <- instr
			if instr.Opcode == "connect" {
				break
			}
		}
		sock.SendInstruction(protocol.Instruction{Opcode: "ready", Args: []string{"$id"}})
	}()

	inet := guacnet.NewInetSocketFromConn(clientConn)
	cfg := &protocol.Config{Protocol: "rdp", Params: map[string]string{"hostname": "host"}}
	info := &protocol.ClientInfo{}
	info.SetTimezone("")
	info.SetName("")
	if _, err := guacnet.NewConfiguredSocket(inet, cfg, info); err != nil {
		t.Fatalf("NewConfiguredSocket: %v", err)
	}

	var sawTimezone, sawName bool
	for instr := range received {
		switch instr.Opcode {
		case "timezone":
			sawTimezone = true
			if len(instr.Args) != 1 || instr.Args[0] != "" {
				t.Errorf("timezone args = %v, want empty string arg", instr.Args)
			}
		case "name":
			sawName = true
			if len(instr.Args) != 1 || instr.Args[0] != "" {
				t.Errorf("name args = %v, want empty string arg", instr.Args)
			}
		}
	}
	if !sawTimezone {
		t.Error("explicit empty timezone was not sent")
	}
	if !sawName {
		t.Error("explicit empty name was not sent")
	}
}

func TestConnect_Integration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		simulateGuacd(t, conn, "VERSION_1_5_0", []string{"hostname", "port"}, "$int-conn")
	}()

	tun, err := guacnet.Connect(context.Background(), ln.Addr().String(), &protocol.Config{
		Protocol: "rdp",
		Params:   map[string]string{"hostname": "host", "port": "3389"},
	}, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer tun.Close()
	if tun.UUID().String() == "" {
		t.Error("UUID should not be empty")
	}
}
