// Package remote provides distributed actor transport for Orizon runtime.
package remote

import "time"

// Envelope is a transport-level message wrapper for remote delivery.
type Envelope struct {
	SenderNode    string            `json:"senderNode"`
	ReceiverNode  string            `json:"receiverNode"`
	ReceiverName  string            `json:"receiverName"`
	ReceiverID    uint64            `json:"receiverId"`
	MessageType   uint32            `json:"messageType"`
	PayloadBytes  []byte            `json:"payload"`
	Headers       map[string]string `json:"headers,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty"`
	TimestampUnix int64             `json:"timestampUnix"`
}

// Handler is invoked by a Transport upon message arrival.
type Handler func(Envelope) error

// Transport abstracts a bidirectional messaging transport.
type Transport interface {
	Start(address string, handler Handler) error
	Stop() error
	Address() string
	Send(to string, env Envelope) error
}

// Codec defines payload serialization for remote transport.
type Codec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	ContentType() string
}

// NowUnix returns current time in unix nano for stamping envelopes.
func NowUnix() int64 { return time.Now().UnixNano() }
