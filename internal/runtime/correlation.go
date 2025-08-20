package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
)

// NewCorrelationID returns a cryptographically-strong random correlation id.
// encoded as a 32-hex-character string. This id can be used to link related
// messages across actors for causal tracing and debugging.
func NewCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to zero bytes in the extremely unlikely event of failure.
		// This preserves determinism of the API without panicking.
	}

	dst := make([]byte, 32)
	hex.Encode(dst, b[:])

	return string(dst)
}

// SetCorrelationID sets the current correlation id on the actor context. It
// will be attached to messages sent via Tell/TellWithPriority. This method is
// safe to call from within Receive.
func (ctx *ActorContext) SetCorrelationID(id string) {
	if ctx != nil {
		ctx.Props["correlationID"] = id
	}
}

// GetCorrelationID returns the current correlation id on the actor context, or.
// an empty string when not set.
func (ctx *ActorContext) GetCorrelationID() string {
	if ctx == nil || ctx.Props == nil {
		return ""
	}

	if v, ok := ctx.Props["correlationID"].(string); ok {
		return v
	}

	return ""
}

// WithCorrelation sets the correlation id on the context and returns a restore.
// function that can be deferred to reset the previous value.
func (ctx *ActorContext) WithCorrelation(id string) (restore func()) {
	prev := ctx.GetCorrelationID()
	ctx.SetCorrelationID(id)

	return func() { ctx.SetCorrelationID(prev) }
}

// Tell sends a message to the receiver using the current correlation id from.
// the actor context, if any.
func (ctx *ActorContext) Tell(receiver ActorID, messageType MessageType, payload interface{}) error {
	if ctx == nil || ctx.System == nil {
		return nil
	}

	return ctx.System.SendMessageWithCorrelation(ctx.ActorID, receiver, messageType, payload, ctx.GetCorrelationID())
}

// TellWithPriority is like Tell but allows specifying an explicit priority.
func (ctx *ActorContext) TellWithPriority(receiver ActorID, messageType MessageType, payload interface{}, prio MessagePriority) error {
	if ctx == nil || ctx.System == nil {
		return nil
	}

	return ctx.System.SendMessageWithCorrelationPriority(ctx.ActorID, receiver, messageType, payload, prio, ctx.GetCorrelationID())
}

// SendMessageWithCorrelation sends a message with an explicit correlation id.
// This is useful for bridging non-actor code or cross-system tracing.
func (as *ActorSystem) SendMessageWithCorrelation(senderID, receiverID ActorID, messageType MessageType, payload interface{}, correlationID string) error {
	return as.SendMessageWithCorrelationPriority(senderID, receiverID, messageType, payload, NormalPriority, correlationID)
}

// SendMessageWithCorrelationPriority sends a message with priority and correlation id.
func (as *ActorSystem) SendMessageWithCorrelationPriority(senderID, receiverID ActorID, messageType MessageType, payload interface{}, prio MessagePriority, correlationID string) error {
	if !as.running {
		return fmt.Errorf("actor system is not running")
	}

	message := Message{
		ID:            MessageID(atomic.AddUint64(&globalMessageID, 1)),
		Type:          messageType,
		Sender:        senderID,
		Receiver:      receiverID,
		Payload:       payload,
		Priority:      prio,
		CorrelationID: correlationID,
	}

	return as.deliverMessage(message)
}
