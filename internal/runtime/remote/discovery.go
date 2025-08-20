package remote

import "sync"

// Discovery provides node name -> address resolution for remote messaging.
type Discovery interface {
	Register(nodeName, address string) error
	Unregister(nodeName string)
	Resolve(nodeName string) (string, bool)
	Members() map[string]string
}

// StaticDiscovery is a simple in-memory discovery useful for tests or single-process clusters.
type StaticDiscovery struct {
	nodes map[string]string
	mu    sync.RWMutex
}

func NewStaticDiscovery() *StaticDiscovery { return &StaticDiscovery{nodes: make(map[string]string)} }

func (d *StaticDiscovery) Register(nodeName, address string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nodes[nodeName] = address

	return nil
}

func (d *StaticDiscovery) Unregister(nodeName string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.nodes, nodeName)
}

func (d *StaticDiscovery) Resolve(nodeName string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	addr, ok := d.nodes[nodeName]

	return addr, ok
}

func (d *StaticDiscovery) Members() map[string]string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make(map[string]string, len(d.nodes))
	for k, v := range d.nodes {
		out[k] = v
	}

	return out
}
