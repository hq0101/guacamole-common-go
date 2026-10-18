package net

import (
	"strconv"

	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

const failoverBufferMax = 128 * 1024 // 128 KB

// FailoverSocket wraps a Socket and detects upstream errors before the first sync.
// On construction it eagerly buffers initial instructions; an upstream error instruction
// causes NewFailoverSocket to return an error immediately. After the first sync instruction
// or 128 KB buffered, it transitions to passthrough mode.
type FailoverSocket struct {
	Socket
	buffered []protocol.Instruction
	pos      int // index into buffered for replay
}

// NewFailoverSocket creates a FailoverSocket and pre-fetches the initial instruction stream.
func NewFailoverSocket(socket Socket) (*FailoverSocket, error) {
	return NewFailoverSocketWithLimit(socket, failoverBufferMax)
}

// NewFailoverSocketWithLimit creates a FailoverSocket using a custom maximum
// amount of buffered instruction data while searching for upstream errors.
func NewFailoverSocketWithLimit(socket Socket, instructionQueueLimit int) (*FailoverSocket, error) {
	fo := &FailoverSocket{Socket: socket}
	buffered, err := fo.loadBuffer(instructionQueueLimit)
	if err != nil {
		return nil, err
	}
	fo.buffered = buffered
	return fo, nil
}

func (fo *FailoverSocket) loadBuffer(instructionQueueLimit int) ([]protocol.Instruction, error) {
	var buf []protocol.Instruction
	totalBytes := 0
	for {
		instr, err := fo.Socket.ReceiveInstruction()
		if err != nil {
			return nil, err
		}
		buf = append(buf, instr)
		if instr.Opcode == "error" {
			if upstreamErr, ok := parseUpstreamErrorInstruction(instr); ok {
				return nil, upstreamErr
			}
			return buf, nil
		}
		totalBytes += len(instr.Encode())
		if instr.Opcode == "sync" || totalBytes >= instructionQueueLimit {
			return buf, nil
		}
	}
}

func (fo *FailoverSocket) ReceiveInstruction() (protocol.Instruction, error) {
	if fo.pos < len(fo.buffered) {
		instr := fo.buffered[fo.pos]
		fo.pos++
		return instr, nil
	}
	return fo.Socket.ReceiveInstruction()
}

func (fo *FailoverSocket) Reader() guacio.GuacamoleReader {
	return &failoverSocketReader{socket: fo}
}

func (fo *FailoverSocket) GetReader() guacio.GuacamoleReader {
	return fo.Reader()
}

func (fo *FailoverSocket) GetWriter() guacio.GuacamoleWriter {
	return fo.Socket.Writer()
}

func (fo *FailoverSocket) GetProtocol() string {
	return fo.Socket.Protocol()
}

type failoverSocketReader struct {
	socket *FailoverSocket
}

func (r *failoverSocketReader) Available() (bool, error) {
	if r.socket.pos < len(r.socket.buffered) {
		return true, nil
	}
	return r.socket.Socket.Reader().Available()
}

func (r *failoverSocketReader) Read() ([]rune, error) {
	instruction, err := r.ReadInstruction()
	if err != nil {
		return nil, err
	}
	return instruction.ToCharArray(), nil
}

func (r *failoverSocketReader) ReadInstruction() (protocol.Instruction, error) {
	return r.socket.ReceiveInstruction()
}

func parseUpstreamErrorInstruction(instr protocol.Instruction) (*protocol.Error, bool) {
	if len(instr.Args) < 2 {
		return nil, false
	}
	code, err := strconv.Atoi(instr.Args[1])
	if err != nil {
		return nil, false
	}
	status, ok := protocol.StatusFromCode(code)
	if !ok {
		return nil, false
	}
	switch status {
	case protocol.StatusUpstreamError,
		protocol.StatusUpstreamNotFound,
		protocol.StatusUpstreamTimeout,
		protocol.StatusUpstreamUnavailable:
	default:
		return nil, false
	}

	msg := instr.Args[0]
	return &protocol.Error{Status: status, Message: msg}, true
}
