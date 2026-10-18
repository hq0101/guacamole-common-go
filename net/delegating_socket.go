package net

import guacio "github.com/hq0101/guacamole-common-go/io"

// DelegatingSocket forwards all calls to an underlying Socket.
type DelegatingSocket struct {
	Socket
}

// NewDelegatingSocket wraps socket such that all calls are delegated to it.
func NewDelegatingSocket(socket Socket) *DelegatingSocket {
	return &DelegatingSocket{Socket: socket}
}

// GetDelegateSocket returns the Socket wrapped by this DelegatingSocket.
func (s *DelegatingSocket) GetDelegateSocket() Socket {
	return s.Socket
}

// GetProtocol returns the protocol exposed by the wrapped socket.
func (s *DelegatingSocket) GetProtocol() string {
	return s.Socket.Protocol()
}

// GetReader returns a reader backed by the wrapped socket.
func (s *DelegatingSocket) GetReader() guacio.GuacamoleReader {
	return s.Socket.Reader()
}

// GetWriter returns a writer backed by the wrapped socket.
func (s *DelegatingSocket) GetWriter() guacio.GuacamoleWriter {
	return s.Socket.Writer()
}
