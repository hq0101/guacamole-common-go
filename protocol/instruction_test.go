package protocol_test

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestInstruction_Encode_ASCII(t *testing.T) {
	instr := protocol.Instruction{Opcode: "select", Args: []string{"rdp"}}
	got := string(instr.Encode())
	if got != "6.select,3.rdp;" {
		t.Errorf("Encode() = %q, want %q", got, "6.select,3.rdp;")
	}
}

func TestInstruction_JavaStyleConstructorAndAccessors(t *testing.T) {
	args := []string{"rdp", "host"}
	instr := protocol.NewInstruction("select", args...)
	args[0] = "ssh"

	if got := instr.GetOpcode(); got != "select" {
		t.Errorf("GetOpcode() = %q, want select", got)
	}

	gotArgs := instr.GetArgs()
	if want := []string{"rdp", "host"}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("GetArgs() = %v, want %v", gotArgs, want)
	}

	gotArgs[0] = "vnc"
	if got := instr.GetArgs()[0]; got != "rdp" {
		t.Errorf("GetArgs() exposed mutable internal args, first arg now %q", got)
	}
}

func TestInstruction_StringMatchesJavaToString(t *testing.T) {
	instr := protocol.NewInstruction("echo", "你好", "")

	if got, want := instr.String(), "4.echo,2.你好,0.;"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got, want := string(instr.ToCharArray()), "4.echo,2.你好,0.;"; got != want {
		t.Errorf("ToCharArray() = %q, want %q", got, want)
	}
}

func TestInstruction_Encode_NoArgs(t *testing.T) {
	instr := protocol.Instruction{Opcode: "nop", Args: []string{}}
	got := string(instr.Encode())
	if got != "3.nop;" {
		t.Errorf("Encode() = %q, want %q", got, "3.nop;")
	}
}

func TestInstruction_Encode_Unicode(t *testing.T) {
	// "你好" = 2 个 Unicode 码点，但 6 字节 UTF-8
	instr := protocol.Instruction{Opcode: "echo", Args: []string{"你好"}}
	got := string(instr.Encode())
	want := "4.echo,2.你好;"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
	if utf8.RuneCountInString("你好") != 2 {
		t.Error("rune count of '你好' should be 2")
	}
}

func TestInstruction_Encode_MultipleArgs(t *testing.T) {
	instr := protocol.Instruction{Opcode: "connect", Args: []string{"host", "3389", ""}}
	got := string(instr.Encode())
	want := "7.connect,4.host,4.3389,0.;"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}

func TestInstruction_Constants(t *testing.T) {
	if protocol.InstructionMaxLength != 8192 {
		t.Errorf("InstructionMaxLength = %d, want 8192", protocol.InstructionMaxLength)
	}
	if protocol.InstructionMaxElements != 128 {
		t.Errorf("InstructionMaxElements = %d, want 128", protocol.InstructionMaxElements)
	}
	if protocol.InstructionMaxDigits != 5 {
		t.Errorf("InstructionMaxDigits = %d, want 5", protocol.InstructionMaxDigits)
	}
}
