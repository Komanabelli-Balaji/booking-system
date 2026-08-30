package ws

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
	}

	hub.subscribe(client, "test-movie")
	hub.Broadcast("test-movie")

	select {
	case msg := <-client.send:
		var result map[string]string
		if err := json.Unmarshal(msg, &result); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if result["type"] != "seats_changed" || result["movie_id"] != "test-movie" {
			t.Errorf("Unexpected message: %v", result)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	hub.Broadcast("another-movie")

	select {
	case <-client.send:
		t.Fatal("Should not have received a message")
	case <-time.After(100 * time.Millisecond):
	}
}
