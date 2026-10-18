package protocol

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

type parserState int

const (
	stateLength parserState = iota
	stateContent
	stateComplete
	stateError
)

// Parser is a stateful Guacamole protocol parser.
// Call Append to feed raw bytes; call Next to retrieve complete instructions.
type Parser struct {
	state     parserState
	instr     Instruction
	opcodeSet bool
	elemRunes []rune
	elemLen   int
	lenDigits string
	raw       []byte
	err       error
}

// Append feeds bytes into the parser and returns the number consumed.
// Returns 0 if a complete instruction is pending retrieval via Next.
func (p *Parser) Append(buf []byte) (consumed int) {
	if p.state == stateComplete || p.state == stateError {
		return 0
	}
	defer func() {
		if consumed > 0 {
			p.raw = append(p.raw, buf[:consumed]...)
		}
	}()

	for consumed < len(buf) && p.state != stateComplete && p.state != stateError {
		switch p.state {
		case stateLength:
			if p.elementCount() >= InstructionMaxElements {
				p.setError("instruction contains too many elements")
				break
			}
			b := buf[consumed]
			if b >= '0' && b <= '9' {
				p.lenDigits += string(b)
				consumed++
			} else if b == '.' {
				n, err := strconv.Atoi(p.lenDigits)
				if err != nil {
					p.setError("element length is empty")
					break
				}
				if n < 0 {
					p.setError("element length is negative")
					break
				}
				if n > InstructionMaxLength {
					p.setError(fmt.Sprintf("instruction element length %d exceeds maximum %d", n, InstructionMaxLength))
					break
				}
				p.elemLen = n
				p.elemRunes = make([]rune, 0, n)
				p.lenDigits = ""
				p.state = stateContent
				consumed++
			} else {
				p.setError("non-numeric character in element length")
			}

		case stateContent:
			if len(p.elemRunes) < p.elemLen {
				if !utf8.FullRune(buf[consumed:]) {
					return consumed
				}
				r, size := utf8.DecodeRune(buf[consumed:])
				if r == utf8.RuneError && size == 1 {
					p.setError("invalid UTF-8 in instruction element")
					break
				}
				p.elemRunes = append(p.elemRunes, r)
				consumed += size
			} else {
				// element complete; read separator
				if consumed >= len(buf) {
					return consumed
				}
				sep := buf[consumed]
				consumed++
				elem := string(p.elemRunes)
				p.elemRunes = nil
				if !p.opcodeSet {
					p.instr.Opcode = elem
					p.opcodeSet = true
				} else {
					p.instr.Args = append(p.instr.Args, elem)
				}
				switch sep {
				case ',':
					p.state = stateLength
				case ';':
					p.state = stateComplete
				default:
					p.setError("element terminator of instruction was not ';' nor ','")
				}
			}
		}
	}
	return consumed
}

// Next retrieves a completed instruction and resets the parser for the next instruction.
// Returns (Instruction{}, false) if no complete instruction is available.
func (p *Parser) Next() (Instruction, bool) {
	if p.state != stateComplete {
		return Instruction{}, false
	}
	result := p.instr
	if len(p.raw) > 0 {
		result.raw = string(p.raw)
		result.rawOp = result.Opcode
		result.rawArgs = append([]string(nil), result.Args...)
	}
	p.instr = Instruction{}
	p.opcodeSet = false
	p.raw = nil
	p.state = stateLength
	return result, true
}

// HasNext reports whether a complete instruction is available, matching
// Java's GuacamoleParser.hasNext().
func (p *Parser) HasNext() bool {
	return p.state == stateComplete
}

// Remove is unsupported, matching Java's GuacamoleParser.remove().
func (p *Parser) Remove() error {
	return &Error{Status: StatusUnsupported, Message: "remove not supported"}
}

// Err returns the first parse error encountered by Append, if any.
func (p *Parser) Err() error {
	return p.err
}

func (p *Parser) setError(message string) {
	p.state = stateError
	if p.err == nil {
		p.err = &Error{Status: StatusServerError, Message: message}
	}
}

func (p *Parser) elementCount() int {
	if !p.opcodeSet {
		return 0
	}
	return 1 + len(p.instr.Args)
}
