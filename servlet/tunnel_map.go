package servlet

import (
	"sync"
	"sync/atomic"
	"time"

	guacnet "github.com/hq0101/guacamole-common-go/net"
)

const (
	tunnelTimeout   = 15 * time.Second
	cleanupInterval = 15 * time.Second
)

// HTTPTunnel wraps a Guacamole tunnel and tracks the last time it was
// accessed by the HTTP tunnel.
type HTTPTunnel struct {
	*guacnet.DelegatingTunnel
	lastAccessedNano atomic.Int64
}

// NewHTTPTunnel wraps tunnel and tracks HTTP tunnel access time.
func NewHTTPTunnel(tunnel guacnet.Tunnel) *HTTPTunnel {
	return &HTTPTunnel{DelegatingTunnel: guacnet.NewDelegatingTunnel(tunnel)}
}

// Access marks this tunnel as recently accessed.
func (t *HTTPTunnel) Access() {
	t.lastAccessedNano.Store(time.Now().UnixNano())
}

// LastAccessedTime returns the last access time recorded for this tunnel.
func (t *HTTPTunnel) LastAccessedTime() time.Time {
	nano := t.lastAccessedNano.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// HTTPTunnelMap tracks active HTTP tunnels by tunnel session token.
type HTTPTunnelMap struct {
	mu       sync.RWMutex
	tunnels  map[string]*HTTPTunnel
	timeout  time.Duration
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	closed   atomic.Bool
}

// NewHTTPTunnelMap returns an HTTP tunnel map with Java-compatible timeout
// behavior.
func NewHTTPTunnelMap() *HTTPTunnelMap {
	return NewHTTPTunnelMapWithTimeout(tunnelTimeout, cleanupInterval)
}

// NewHTTPTunnelMapWithTimeout returns an HTTP tunnel map using custom cleanup
// timing. It is primarily useful for tests.
func NewHTTPTunnelMapWithTimeout(timeout, interval time.Duration) *HTTPTunnelMap {
	if interval <= 0 {
		interval = timeout
	}
	m := &HTTPTunnelMap{
		tunnels:  make(map[string]*HTTPTunnel),
		timeout:  timeout,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go m.cleanup()
	return m
}

// Get returns the tunnel associated with token and updates its last access
// time. If no such tunnel exists, nil is returned.
func (m *HTTPTunnelMap) Get(token string) *HTTPTunnel {
	m.mu.RLock()
	tunnel := m.tunnels[token]
	m.mu.RUnlock()
	if tunnel != nil {
		tunnel.Access()
	}
	return tunnel
}

// Put registers tunnel under token, wrapping it as an HTTP tunnel.
func (m *HTTPTunnelMap) Put(token string, tunnel guacnet.Tunnel) {
	httpTunnel := NewHTTPTunnel(tunnel)
	httpTunnel.Access()

	m.mu.Lock()
	m.tunnels[token] = httpTunnel
	m.mu.Unlock()
}

// Remove removes and returns the tunnel associated with token, if any.
func (m *HTTPTunnelMap) Remove(token string) *HTTPTunnel {
	m.mu.Lock()
	tunnel := m.tunnels[token]
	delete(m.tunnels, token)
	m.mu.Unlock()
	return tunnel
}

// Shutdown stops the background timeout task for this map.
func (m *HTTPTunnelMap) Shutdown() {
	if m.closed.CompareAndSwap(false, true) {
		close(m.stop)
		<-m.done
	}
}

func (m *HTTPTunnelMap) cleanup() {
	defer close(m.done)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.expireIdleTunnels()
		case <-m.stop:
			return
		}
	}
}

func (m *HTTPTunnelMap) expireIdleTunnels() {
	now := time.Now()
	var expired []*HTTPTunnel

	m.mu.Lock()
	for token, tunnel := range m.tunnels {
		if now.Sub(tunnel.LastAccessedTime()) >= m.timeout {
			delete(m.tunnels, token)
			expired = append(expired, tunnel)
		}
	}
	m.mu.Unlock()

	for _, tunnel := range expired {
		tunnel.Close() //nolint:errcheck
	}
}
