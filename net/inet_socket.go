package net

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hq0101/guacamole-common-go/protocol"
)

const dialTimeout = 15 * time.Second

// InetSocket is a plaintext TCP Socket connecting to guacd.
type InetSocket struct {
	conn    net.Conn
	reader  *bufio.Reader
	mu      sync.Mutex
	parser  protocol.Parser
	pending []byte // unparsed bytes left after a stateComplete cut-off
	open    atomic.Bool
	proto   string
}

// Dial connects to guacd at addr over plaintext TCP.
func Dial(ctx context.Context, addr string) (*InetSocket, error) {
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, &protocol.Error{
			Status:  protocol.StatusUpstreamUnavailable,
			Message: fmt.Sprintf("failed to connect to guacd at %s: %v", addr, err),
		}
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}
	return NewInetSocketFromConn(conn), nil
}

// NewInetSocketFromConn creates an InetSocket from an existing connection.
// Primarily for testing with net.Pipe().
func NewInetSocketFromConn(conn net.Conn) *InetSocket {
	s := &InetSocket{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
	s.open.Store(true)
	return s
}

func (s *InetSocket) SendInstruction(instr protocol.Instruction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.conn.Write(instr.Encode())
	if err != nil {
		s.open.Store(false)
		return &protocol.Error{Status: protocol.StatusResourceClosed, Message: err.Error()}
	}
	return nil
}

// ReceiveInstruction reads the next complete Guacamole instruction from guacd.
// It must be called from a single goroutine at a time; concurrent reads are not
// safe (matching the read-lock contract of Java's InetGuacamoleSocket).
func (s *InetSocket) ReceiveInstruction() (protocol.Instruction, error) {
	for {
		// Drain any bytes saved from a previous read before consuming the next instruction.
		for len(s.pending) > 0 {
			consumed := s.parser.Append(s.pending)
			if err := s.parser.Err(); err != nil {
				s.open.Store(false)
				return protocol.Instruction{}, err
			}
			s.pending = s.pending[consumed:]
			instr, ok := s.parser.Next()
			if ok {
				return instr, nil
			}
			if consumed == 0 {
				// Pending bytes contain an incomplete UTF-8 sequence or element.
				// Read more and append to pending before parsing further.
				break
			}
		}

		if len(s.pending) > 0 {
			buf := make([]byte, 4096)
			n, err := s.reader.Read(buf)
			if err != nil {
				s.open.Store(false)
				return protocol.Instruction{}, &protocol.Error{
					Status:  protocol.StatusResourceClosed,
					Message: err.Error(),
				}
			}
			s.pending = append(s.pending, buf[:n]...)
			continue
		}

		instr, ok := s.parser.Next()
		if ok {
			return instr, nil
		}

		buf := make([]byte, 4096)
		n, err := s.reader.Read(buf)
		if err != nil {
			s.open.Store(false)
			return protocol.Instruction{}, &protocol.Error{
				Status:  protocol.StatusResourceClosed,
				Message: err.Error(),
			}
		}
		data := buf[:n]
		consumed := s.parser.Append(data)
		if err := s.parser.Err(); err != nil {
			s.open.Store(false)
			return protocol.Instruction{}, err
		}
		// Save any bytes that were not consumed (parser stopped at stateComplete).
		if consumed < len(data) {
			tail := make([]byte, len(data)-consumed)
			copy(tail, data[consumed:])
			s.pending = append(s.pending, tail...)
		}
	}
}

func (s *InetSocket) Protocol() string    { return s.proto }
func (s *InetSocket) GetProtocol() string { return s.Protocol() }
func (s *InetSocket) IsOpen() bool        { return s.open.Load() }
func (s *InetSocket) Close() error {
	s.open.Store(false)
	return s.conn.Close()
}
