package protocol

import "errors"

// ErrDropInstruction is returned by a Filter to silently discard an instruction.
var ErrDropInstruction = errors.New("instruction dropped")

// Filter intercepts the instruction stream.
// Return the (possibly modified) instruction to pass it through.
// Return ErrDropInstruction to silently discard.
// Return any other error to propagate it and terminate the stream.
type Filter func(Instruction) (Instruction, error)
