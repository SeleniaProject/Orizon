package remote

import (
    "fmt"
    "sync"
    "time"
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
    Discover  Discovery
    started   bool
    mutex     sync.RWMutex
    // Retry/backoff configuration
    RetryMaxAttempts int
    RetryInitialMs   int
    RetryMaxBackoffMs int
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
    if rs.Discover != nil { _ = rs.Discover.Register(nodeName, rs.Address) }
    if rs.RetryMaxAttempts == 0 { rs.RetryMaxAttempts = 6 }
    if rs.RetryInitialMs == 0 { rs.RetryInitialMs = 50 }
    if rs.RetryMaxBackoffMs == 0 { rs.RetryMaxBackoffMs = 800 }
    return nil
}

// Stop terminates the transport.
func (rs *RemoteSystem) Stop() error {
    rs.mutex.Lock(); defer rs.mutex.Unlock()
    if !rs.started { return nil }
    rs.started = false
    if rs.Discover != nil { rs.Discover.Unregister(rs.NodeName) }
    if rs.Trans != nil { return rs.Trans.Stop() }
    return nil
}

// Send delivers a message to a remote node.
func (rs *RemoteSystem) Send(remoteAddrOrNode, receiverName string, msgType uint32, payload interface{}) error {
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
        ReceiverNode:  remoteAddrOrNode,
        ReceiverName:  receiverName,
        MessageType:   msgType,
        PayloadBytes:  b,
        TimestampUnix: NowUnix(),
    }
    // If a discovery is available, allow passing node name instead of address
    target := remoteAddrOrNode
    if rs.Discover != nil {
        if addr, ok := rs.Discover.Resolve(remoteAddrOrNode); ok { target = addr }
    }
    return rs.sendWithRetry(target, env)
}

// SendWithRetry allows specifying custom attempts/backoff for a single send.
func (rs *RemoteSystem) SendWithRetry(remoteAddrOrNode, receiverName string, msgType uint32, payload interface{}, attempts int, initialMs int) error {
    rs.mutex.RLock(); codec := rs.Default; node := rs.NodeName; rs.mutex.RUnlock()
    var b []byte
    var err error
    if pb, ok := payload.([]byte); ok {
        b, err = codec.Marshal(pb)
    } else {
        b, err = codec.Marshal(payload)
    }
    if err != nil { return err }
    env := Envelope{ SenderNode: node, ReceiverNode: remoteAddrOrNode, ReceiverName: receiverName, MessageType: msgType, PayloadBytes: b, TimestampUnix: NowUnix() }
    target := remoteAddrOrNode
    if rs.Discover != nil { if addr, ok := rs.Discover.Resolve(remoteAddrOrNode); ok { target = addr } }
    // temporarily override
    prevA, prevI := rs.RetryMaxAttempts, rs.RetryInitialMs
    if attempts > 0 { rs.RetryMaxAttempts = attempts }
    if initialMs > 0 { rs.RetryInitialMs = initialMs }
    defer func(){ rs.RetryMaxAttempts, rs.RetryInitialMs = prevA, prevI }()
    return rs.sendWithRetry(target, env)
}

// sendWithRetry attempts to send with exponential backoff up to configured attempts.
func (rs *RemoteSystem) sendWithRetry(target string, env Envelope) error {
    attempts := rs.RetryMaxAttempts
    if attempts <= 0 { attempts = 1 }
    backoff := rs.RetryInitialMs
    if backoff <= 0 { backoff = 50 }
    max := rs.RetryMaxBackoffMs
    if max <= 0 { max = 800 }
    var lastErr error
    for i := 0; i < attempts; i++ {
        if err := rs.Trans.Send(target, env); err == nil {
            return nil
        } else {
            lastErr = err
        }
        // Attempt discovery refresh between retries
        if rs.Discover != nil {
            if addr, ok := rs.Discover.Resolve(env.ReceiverNode); ok { target = addr }
        }
        if i < attempts-1 {
            d := backoff
            if d > max { d = max }
            time.Sleep(time.Duration(d) * time.Millisecond)
            backoff <<= 1
            if backoff > max { backoff = max }
        }
    }
    return lastErr
}


// SendWithRetry sends with simple exponential backoff retry when transport or resolution fails.
// attempts <= 1 means single try; baseDelay defines the initial sleep duration.
func (rs *RemoteSystem) SendWithRetry(remoteAddrOrNode, receiverName string, msgType uint32, payload interface{}, attempts int, baseDelayMs int) error {
    if attempts < 1 { attempts = 1 }
    delay := baseDelayMs
    var lastErr error
    for i := 0; i < attempts; i++ {
        if err := rs.Send(remoteAddrOrNode, receiverName, msgType, payload); err == nil {
            return nil
        } else {
            lastErr = err
        }
        if i < attempts-1 {
            // exponential backoff with cap
            if delay <= 0 { delay = 10 }
            if delay > 2000 { delay = 2000 }
            // simple jitter-less sleep
            time.Sleep(time.Duration(delay) * time.Millisecond)
            if delay < 2000 { delay = delay * 2 }
        }
    }
    return lastErr
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


