package concurrency

import (
	"testing"
	"time"
)

func TestDeadlockDetector_FindsSimpleCycle(t *testing.T) {
	d := NewDeadlockDetector()
	m1 := NewMonitoredMutex(1, d)
	m2 := NewMonitoredMutex(2, d)

	done := make(chan struct{}, 2)
	go func() {
		m1.Lock(10)
		time.Sleep(10 * time.Millisecond)
		m2.Lock(10)
		done <- struct{}{}
	}()
	go func() {
		m2.Lock(20)
		time.Sleep(10 * time.Millisecond)
		m1.Lock(20)
		done <- struct{}{}
	}()

	if err := d.WaitUntilDeadlock(500*time.Millisecond, 5*time.Millisecond); err != nil {
		t.Fatalf("expected deadlock within timeout: %v", err)
	}
}
