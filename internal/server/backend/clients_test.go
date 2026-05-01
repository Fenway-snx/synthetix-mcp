package backend

import (
	"testing"
)

func TestCloseHandlesNilReceiver(t *testing.T) {
	var clients *Clients

	if err := clients.Close(); err != nil {
		t.Fatalf("expected nil clients Close to succeed, got %v", err)
	}
}
