package halt

import (
	"sync/atomic"
	"testing"
)

func Test_NewShutdownSignal_RequestShutdownClosesSignalAndRunsTriggerOnce(t *testing.T) {
	t.Parallel()

	var triggerCalls atomic.Int64
	shutdownSignal, requestShutdown := NewShutdownSignal(func() {
		triggerCalls.Add(1)
	})

	requestShutdown()
	requestShutdown()

	if triggerCalls.Load() != 1 {
		t.Fatalf("expected trigger to run once, got %d", triggerCalls.Load())
	}

	select {
	case <-shutdownSignal:
	default:
		t.Fatal("expected shutdown signal channel to be closed")
	}
}

func Test_NewShutdownSignal_RequestShutdownClosesSignalWhenTriggerNil(t *testing.T) {
	t.Parallel()

	shutdownSignal, requestShutdown := NewShutdownSignal(nil)
	requestShutdown()

	select {
	case <-shutdownSignal:
	default:
		t.Fatal("expected shutdown signal channel to be closed")
	}
}
