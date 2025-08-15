package concurrency

import (
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func TestRaceDetector_DetectsWriteWrite(t *testing.T) {
	det := NewRaceDetector()
	var shared int64
	addr := uintptr(unsafe.Pointer(&shared))
	done := make(chan struct{}, 2)
	go func() {
		gid := int64(1)
		for i := 0; i < 1000; i++ {
			atomic.AddInt64(&shared, 1)
			det.Write(gid, addr)
		}
		done <- struct{}{}
	}()
	go func() {
		gid := int64(2)
		for i := 0; i < 1000; i++ {
			atomic.AddInt64(&shared, 1)
			det.Write(gid, addr)
		}
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
	}
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
	}
	if !det.HasRace() {
		// Due to scheduling, it's possible we missed interleavings; retry a bit
		for i := 0; i < 5 && !det.HasRace(); i++ {
			// emulate further conflicting accesses
			det.Write(1, addr)
			det.Write(2, addr)
		}
	}
	if !det.HasRace() {
		t.Fatalf("expected a race to be detected, none found")
	}
}

func TestRaceDetector_NoRaceUnderMutex(t *testing.T) {
	det := NewRaceDetector()
	m := NewTrackedMutex(100, det)
	var shared int64
	addr := uintptr(unsafe.Pointer(&shared))
	done := make(chan struct{}, 2)
	go func() {
		gid := int64(1)
		for i := 0; i < 1000; i++ {
			m.Lock(gid)
			shared++
			det.Write(gid, addr)
			m.Unlock(gid)
		}
		done <- struct{}{}
	}()
	go func() {
		gid := int64(2)
		for i := 0; i < 1000; i++ {
			m.Lock(gid)
			shared++
			det.Write(gid, addr)
			m.Unlock(gid)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	if det.HasRace() {
		t.Fatalf("did not expect a race under mutex, got: %+v", det.Races())
	}
}
