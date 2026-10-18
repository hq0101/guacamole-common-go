package protocol_test

import (
	"strings"
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestParser_SingleInstruction(t *testing.T) {
	p := &protocol.Parser{}
	data := []byte("6.select,3.rdp;")
	n := p.Append(data)
	if n != len(data) {
		t.Errorf("Append() consumed %d bytes, want %d", n, len(data))
	}
	instr, ok := p.Next()
	if !ok {
		t.Fatal("Next() = false after complete instruction")
	}
	if instr.Opcode != "select" || len(instr.Args) != 1 || instr.Args[0] != "rdp" {
		t.Errorf("unexpected instruction: %+v", instr)
	}
	_, ok2 := p.Next()
	if ok2 {
		t.Error("second Next() should return false")
	}
}

func TestParser_NoArgs(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("3.nop;"))
	instr, ok := p.Next()
	if !ok || instr.Opcode != "nop" || len(instr.Args) != 0 {
		t.Errorf("unexpected: ok=%v instr=%+v", ok, instr)
	}
}

func TestParser_Incremental(t *testing.T) {
	p := &protocol.Parser{}
	// Feed partial data
	p.Append([]byte("6.sele"))
	if _, ok := p.Next(); ok {
		t.Error("Next() should return false before instruction is complete")
	}
	// Feed remainder
	p.Append([]byte("ct,3.rdp;"))
	instr, ok := p.Next()
	if !ok || instr.Opcode != "select" || instr.Args[0] != "rdp" {
		t.Errorf("unexpected: ok=%v instr=%+v", ok, instr)
	}
}

func TestParser_MultipleCalls(t *testing.T) {
	p := &protocol.Parser{}
	data := []byte("3.nop;4.sync,1.0;")

	// First instruction
	n1 := p.Append(data)
	instr1, ok1 := p.Next()
	if !ok1 || instr1.Opcode != "nop" {
		t.Fatalf("first instr: ok=%v opcode=%q", ok1, instr1.Opcode)
	}

	// Second instruction from remaining bytes
	n2 := p.Append(data[n1:])
	_ = n2
	instr2, ok2 := p.Next()
	if !ok2 || instr2.Opcode != "sync" || instr2.Args[0] != "0" {
		t.Fatalf("second instr: ok=%v opcode=%q args=%v", ok2, instr2.Opcode, instr2.Args)
	}
}

func TestParser_PendingReturnZero(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("3.nop;"))
	// instruction is ready but Next() not called — Append should return 0
	n := p.Append([]byte("extra"))
	if n != 0 {
		t.Errorf("Append with pending instruction: got %d, want 0", n)
	}
}

func TestParser_JavaStyleHasNextAndRemove(t *testing.T) {
	p := &protocol.Parser{}
	if p.HasNext() {
		t.Fatal("HasNext() = true before complete instruction")
	}

	p.Append([]byte("3.nop;"))
	if !p.HasNext() {
		t.Fatal("HasNext() = false after complete instruction")
	}

	if err := p.Remove(); err == nil {
		t.Fatal("Remove() error = nil, want unsupported error")
	}

	if _, ok := p.Next(); !ok {
		t.Fatal("Next() = false after HasNext()")
	}
	if p.HasNext() {
		t.Fatal("HasNext() = true after Next() consumed instruction")
	}
}

func TestParser_EmptyArg(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("4.test,0.;"))
	instr, ok := p.Next()
	if !ok || instr.Opcode != "test" || len(instr.Args) != 1 || instr.Args[0] != "" {
		t.Errorf("unexpected: ok=%v instr=%+v", ok, instr)
	}
}

func TestParser_UnicodeContent(t *testing.T) {
	p := &protocol.Parser{}
	// "你好" = 2 codepoints
	p.Append([]byte("4.echo,2.你好;"))
	instr, ok := p.Next()
	if !ok || instr.Opcode != "echo" || instr.Args[0] != "你好" {
		t.Errorf("unexpected: ok=%v instr=%+v", ok, instr)
	}
}

func TestParser_AcceptsLengthPrefixWithLeadingZeros(t *testing.T) {
	p := &protocol.Parser{}
	n := p.Append([]byte("000006.select,000003.rdp;"))

	if err := p.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
	if n != len("000006.select,000003.rdp;") {
		t.Fatalf("Append() consumed %d bytes, want full instruction", n)
	}
	instr, ok := p.Next()
	if !ok {
		t.Fatal("Next() = false after complete instruction")
	}
	if instr.Opcode != "select" || len(instr.Args) != 1 || instr.Args[0] != "rdp" {
		t.Errorf("instruction = %+v, want select [rdp]", instr)
	}
}

func TestParser_PreservesRawInstructionString(t *testing.T) {
	p := &protocol.Parser{}
	raw := "000006.select,000003.rdp;"
	p.Append([]byte(raw))
	instr, ok := p.Next()
	if !ok {
		t.Fatal("Next() = false after complete instruction")
	}

	if got := instr.String(); got != raw {
		t.Errorf("String() = %q, want raw %q", got, raw)
	}
	if got := string(instr.Encode()); got != raw {
		t.Errorf("Encode() = %q, want raw %q", got, raw)
	}
	if got := string(instr.ToCharArray()); got != raw {
		t.Errorf("ToCharArray() = %q, want raw %q", got, raw)
	}
}

func TestParser_RawInstructionInvalidatedByMutation(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("000006.select,000003.rdp;"))
	instr, ok := p.Next()
	if !ok {
		t.Fatal("Next() = false after complete instruction")
	}

	instr.Args[0] = "ssh"
	if got, want := instr.String(), "6.select,3.ssh;"; got != want {
		t.Errorf("String() after mutation = %q, want %q", got, want)
	}
}

func TestParser_UnicodeContentSplitWithinUTF8Rune(t *testing.T) {
	p := &protocol.Parser{}
	data := []byte("4.echo,2.你好;")

	// Split after the first byte of "你", forcing the parser to carry a
	// partial UTF-8 sequence across Append calls.
	prefix := []byte("4.echo,2.")
	split := len(prefix) + 1
	consumed := p.Append(data[:split])
	if consumed != len(prefix) {
		t.Fatalf("first Append consumed %d bytes, want %d", consumed, len(prefix))
	}
	if err := p.Err(); err != nil {
		t.Fatalf("Err() after partial UTF-8 = %v, want nil", err)
	}

	if consumed := p.Append(data[consumed:]); consumed != len(data)-len(prefix) {
		t.Fatalf("second Append consumed %d bytes, want %d", consumed, len(data)-len(prefix))
	}
	instr, ok := p.Next()
	if !ok {
		t.Fatal("Next() = false after completing split UTF-8 instruction")
	}
	if instr.Opcode != "echo" || len(instr.Args) != 1 || instr.Args[0] != "你好" {
		t.Errorf("instruction = %+v, want echo [你好]", instr)
	}
}

func TestParser_EmptyOpcode(t *testing.T) {
	p := &protocol.Parser{}
	// "0." = empty string opcode, "3.foo" = arg "foo"
	p.Append([]byte("0.,3.foo;"))
	instr, ok := p.Next()
	if !ok {
		t.Fatal("Next() = false")
	}
	if instr.Opcode != "" {
		t.Errorf("Opcode = %q, want empty string", instr.Opcode)
	}
	if len(instr.Args) != 1 || instr.Args[0] != "foo" {
		t.Errorf("Args = %v, want [foo]", instr.Args)
	}
}

func TestParser_ErrNonNumericLength(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("x.foo;"))

	err := p.Err()
	if err == nil {
		t.Fatal("Err() = nil, want parse error")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("Err() type = %T, want *protocol.Error", err)
	}
	if protoErr.Status != protocol.StatusServerError {
		t.Errorf("Status = %v, want StatusServerError", protoErr.Status)
	}
}

func TestParser_ErrElementTooLong(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("8193."))

	if err := p.Err(); err == nil {
		t.Fatal("Err() = nil, want maximum length parse error")
	}
}

func TestParser_ErrTooManyElements(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte(strings.Repeat("0.,", protocol.InstructionMaxElements) + "0.;"))

	if err := p.Err(); err == nil {
		t.Fatal("Err() = nil, want too many elements parse error")
	}
}

func TestParser_ErrInvalidTerminator(t *testing.T) {
	p := &protocol.Parser{}
	p.Append([]byte("3.foo:"))

	if err := p.Err(); err == nil {
		t.Fatal("Err() = nil, want invalid terminator parse error")
	}
}
