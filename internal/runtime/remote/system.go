package remote

import (
    "fmt"
    "sync"
)

// LocalDispatcher is the subset of ActorSystem needed for local delivery.
type LocalDispatcher interface {
    // Deliver(receiverName string, msgType uint32, payload interface{}) error
    SendMessage(senderID uint64, receiverID uint64, msgType uint32, payload interface{}) error
    LookupActorID(name string) (uint64, bool)
}

// RemoteSystem wires a local actor system to a Transport to enable node-to-node messaging.
type RemoteSystem struct {
    NodeName  string
    Address   string
    Codecs    map[string]Codec
    Default   Codec
    Trans     Transport
    Local     LocalDispatcher
    Resolver  NameResolver
    started   bool
    mutex     sync.RWMutex
}

// NameResolver resolves actor names to IDs on the local node.
type NameResolver interface {
    Lookup(name string) (uint64, bool)
}

// Start initializes transport and begins receiving.
func (rs *RemoteSystem) Start(nodeName, addr string) error {
    rs.mutex.Lock(); defer rs.mutex.Unlock()
    if rs.started { return fmt.Errorf("remote already started") }
    if rs.Trans == nil || rs.Local == nil || rs.Default == nil {
        return fmt.Errorf("remote components not configured")
    }
    rs.NodeName = nodeName
    handler := func(env Envelope) error { return rs.receive(env) }
    if err := rs.Trans.Start(addr, handler); err != nil { return err }
    rs.Address = rs.Trans.Address()
    rs.started = true
    return nil
}

// Stop terminates the transport.
func (rs *RemoteSystem) Stop() error {
    rs.mutex.Lock(); defer rs.mutex.Unlock()
    if !rs.started { return nil }
    rs.started = false
    if rs.Trans != nil { return rs.Trans.Stop() }
    return nil
}

// Send delivers a message to a remote node.
func (rs *RemoteSystem) Send(remoteAddr, receiverName string, msgType uint32, payload interface{}) error {
    rs.mutex.RLock(); codec := rs.Default; node := rs.NodeName; rs.mutex.RUnlock()
    // For byte payloads, wrap in JSON so Unmarshal to []byte works symmetrically
    var b []byte
    var err error
    if pb, ok := payload.([]byte); ok {
        b, err = codec.Marshal(pb)
    } else {
        b, err = codec.Marshal(payload)
    }
    if err != nil { return err }
    env := Envelope{
        SenderNode:    node,
        ReceiverNode:  remoteAddr,
        ReceiverName:  receiverName,
        MessageType:   msgType,
        PayloadBytes:  b,
        TimestampUnix: NowUnix(),
    }
    return rs.Trans.Send(remoteAddr, env)
}

// receive handles an incoming envelope and dispatches to a local actor by name.
func (rs *RemoteSystem) receive(env Envelope) error {
    // Resolve actor by name then unmarshal and forward
    if rs.Resolver == nil || rs.Local == nil { return fmt.Errorf("remote not bound to local system") }
    id, ok := rs.Resolver.Lookup(env.ReceiverName)
    if !ok {
        return fmt.Errorf("local actor not found: %s", env.ReceiverName)
    }
    var payload interface{}
    // Attempt to decode with default codec into raw bytes; fall back to raw envelope payload on error.
    var raw []byte
    if err := rs.Default.Unmarshal(env.PayloadBytes, &raw); err == nil {
        payload = raw
    } else {
        payload = env.PayloadBytes
    }
    return rs.Local.SendMessage(0, id, env.MessageType, payload)
}


