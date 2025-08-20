package channels

import (
	"context"
	"testing"
	"time"
)

func TestChannel_Basic(t *testing.T) {
	ch := New[int](2)
	if err := ch.Send(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	if !ch.TrySend(2) {
		t.Fatal("trysend failed")
	}

	if !ch.TrySend(3) { /* likely full */
	}

	if v, ok := ch.TryRecv(); !ok || v != 1 {
		t.Fatalf("got %v %v", v, ok)
	}

	v, ok, err := ch.Recv(context.Background())
	if err != nil || !ok || v != 2 {
		t.Fatalf("recv: %v %v %v", v, ok, err)
	}
}

func TestSelectRecv_Multiple(t *testing.T) {
	a := New[int](0)
	b := New[int](0)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() { _ = a.Send(context.Background(), 10) }()

	v, idx, ok, err := SelectRecv(ctx, a, b)
	if err != nil || !ok {
		t.Fatalf("select: %v %v", ok, err)
	}

	if !(idx == 0 && v == 10) {
		t.Fatalf("unexpected %d %d", idx, v)
	}
}

func TestChannel_Close(t *testing.T) {
	ch := New[int](1)
	ch.Close()

	if ch.TrySend(1) {
		t.Fatal("send on closed should fail")
	}

	if _, ok := ch.TryRecv(); ok { /* may be false */
	}
}
