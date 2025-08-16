package remote

import (
	"fmt"
	"sync"
)

// InMemoryTransport is an in-process transport useful for tests and single-process clusters.
type InMemoryTransport struct {
	addr    string
	handler Handler
	mutex   sync.RWMutex
}

var (
	registryMutex sync.RWMutex
	registry      = map[string]*InMemoryTransport{}
)

func (t *InMemoryTransport) Start(address string, handler Handler) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.addr != "" {
		return fmt.Errorf("transport already started")
	}
	registryMutex.Lock()
	defer registryMutex.Unlock()
	if _, exists := registry[address]; exists {
		return fmt.Errorf("address already in use: %s", address)
	}
	t.addr = address
	t.handler = handler
	registry[address] = t
	return nil
}

func (t *InMemoryTransport) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.addr == "" {
		return nil
	}
	registryMutex.Lock()
	delete(registry, t.addr)
	registryMutex.Unlock()
	t.addr = ""
	t.handler = nil
	return nil
}

func (t *InMemoryTransport) Address() string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.addr
}

func (t *InMemoryTransport) Send(to string, env Envelope) error {
	registryMutex.RLock()
	dst := registry[to]
	registryMutex.RUnlock()
	if dst == nil {
		return fmt.Errorf("destination not found: %s", to)
	}
	dst.mutex.RLock()
	handler := dst.handler
	dst.mutex.RUnlock()
	if handler == nil {
		return fmt.Errorf("destination has no handler: %s", to)
	}
	return handler(env)
}
