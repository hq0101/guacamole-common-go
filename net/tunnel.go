package net

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	guacio "github.com/hq0101/guacamole-common-go/io"
)

// Tunnel is a Socket with an associated UUID, used by HTTP/WebSocket layers
// for session tracking.
type Tunnel interface {
	Socket
	UUID() uuid.UUID
	GetUUID() uuid.UUID
	GetSocket() Socket
	AcquireReader() guacio.GuacamoleReader
	ReleaseReader()
	HasQueuedReaderThreads() bool
	AcquireWriter() guacio.GuacamoleWriter
	ReleaseWriter()
	HasQueuedWriterThreads() bool
}

// SimpleTunnel wraps a Socket and assigns it a UUID.
type SimpleTunnel struct {
	Socket
	id         uuid.UUID
	readerLock tunnelLock
	writerLock tunnelLock
}

// NewTunnel wraps socket in a SimpleTunnel with a new random UUID.
func NewTunnel(socket Socket) *SimpleTunnel {
	return &SimpleTunnel{Socket: socket, id: uuid.New()}
}

func (t *SimpleTunnel) UUID() uuid.UUID    { return t.id }
func (t *SimpleTunnel) GetUUID() uuid.UUID { return t.UUID() }
func (t *SimpleTunnel) GetSocket() Socket  { return t.Socket }

// AcquireReader acquires exclusive read access to the tunnel instruction
// stream.
func (t *SimpleTunnel) AcquireReader() guacio.GuacamoleReader {
	t.readerLock.Lock()
	return t.Socket.Reader()
}

// ReleaseReader releases exclusive read access to the tunnel instruction
// stream.
func (t *SimpleTunnel) ReleaseReader() {
	t.readerLock.Unlock()
}

// HasQueuedReaderThreads reports whether a reader is waiting for the read
// lock.
func (t *SimpleTunnel) HasQueuedReaderThreads() bool {
	return t.readerLock.HasQueued()
}

// AcquireWriter acquires exclusive write access to the tunnel instruction
// stream.
func (t *SimpleTunnel) AcquireWriter() guacio.GuacamoleWriter {
	t.writerLock.Lock()
	return t.Socket.Writer()
}

// ReleaseWriter releases exclusive write access to the tunnel instruction
// stream.
func (t *SimpleTunnel) ReleaseWriter() {
	t.writerLock.Unlock()
}

// HasQueuedWriterThreads reports whether a writer is waiting for the write
// lock.
func (t *SimpleTunnel) HasQueuedWriterThreads() bool {
	return t.writerLock.HasQueued()
}

// DelegatingTunnel forwards all calls to an underlying Tunnel.
type DelegatingTunnel struct {
	Tunnel
}

// NewDelegatingTunnel wraps tunnel such that all calls are delegated to it.
func NewDelegatingTunnel(tunnel Tunnel) *DelegatingTunnel {
	return &DelegatingTunnel{Tunnel: tunnel}
}

type tunnelLock struct {
	mu     sync.Mutex
	queued atomic.Int32
}

func (l *tunnelLock) Lock() {
	if l.mu.TryLock() {
		return
	}
	l.queued.Add(1)
	l.mu.Lock()
	l.queued.Add(-1)
}

func (l *tunnelLock) Unlock() {
	l.mu.Unlock()
}

func (l *tunnelLock) HasQueued() bool {
	return l.queued.Load() > 0
}
