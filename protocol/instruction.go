package protocol

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	InstructionMaxLength   = 8192
	InstructionMaxElements = 128
	InstructionMaxDigits   = 5

	INSTRUCTION_MAX_LENGTH   = InstructionMaxLength
	INSTRUCTION_MAX_ELEMENTS = InstructionMaxElements
	INSTRUCTION_MAX_DIGITS   = InstructionMaxDigits
)

// Instruction is the fundamental unit of the Guacamole protocol.
type Instruction struct {
	Opcode string
	Args   []string

	raw     string
	rawOp   string
	rawArgs []string
}

// NewInstruction creates a new Guacamole instruction with the given opcode and
// argument values.
func NewInstruction(opcode string, args ...string) Instruction {
	copiedArgs := append([]string(nil), args...)
	return Instruction{Opcode: opcode, Args: copiedArgs}
}

// GetOpcode returns the opcode associated with this instruction.
func (i Instruction) GetOpcode() string {
	return i.Opcode
}

// GetArgs returns a copy of all argument values specified for this instruction.
func (i Instruction) GetArgs() []string {
	return append([]string(nil), i.Args...)
}

// Encode returns the wire-format encoding of the instruction.
// Length prefixes use Unicode codepoint count, not byte count.
func (i Instruction) Encode() []byte {
	return []byte(i.String())
}

func (i Instruction) encodeString() string {
	var b strings.Builder
	writeElem := func(s string) {
		b.WriteString(strconv.Itoa(utf8.RuneCountInString(s)))
		b.WriteByte('.')
		b.WriteString(s)
	}
	writeElem(i.Opcode)
	for _, arg := range i.Args {
		b.WriteByte(',')
		writeElem(arg)
	}
	b.WriteByte(';')
	return b.String()
}

// String returns this instruction in the form it would be sent over the
// Guacamole protocol.
func (i Instruction) String() string {
	if i.raw != "" && i.matchesRawInstruction() {
		return i.raw
	}
	return i.encodeString()
}

// ToCharArray returns this instruction as a slice of runes in Guacamole
// protocol form.
func (i Instruction) ToCharArray() []rune {
	return []rune(i.String())
}

func (i Instruction) matchesRawInstruction() bool {
	if i.Opcode != i.rawOp || len(i.Args) != len(i.rawArgs) {
		return false
	}
	for idx, arg := range i.Args {
		if arg != i.rawArgs[idx] {
			return false
		}
	}
	return true
}
