package sse_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/offen/restore-manager/internal/sse"
)

func TestBroadcasterSendsEventsToSubscribers(t *testing.T) {
	b := sse.NewBroadcaster()

	ch := b.Subscribe("token-1")
	defer b.Unsubscribe("token-1", ch)

	event := sse.Event{
		Step:    "downloading",
		Status:  "in_progress",
		Message: "Downloading backup (50%)",
		Percent: 50,
	}

	b.Send("token-1", event)

	select {
	case got := <-ch:
		if got.Step != "downloading" {
			t.Fatalf("step: got %q, want %q", got.Step, "downloading")
		}
		if got.Percent != 50 {
			t.Fatalf("percent: got %d, want 50", got.Percent)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBroadcasterIsolatesTokens(t *testing.T) {
	b := sse.NewBroadcaster()

	ch1 := b.Subscribe("token-1")
	defer b.Unsubscribe("token-1", ch1)

	ch2 := b.Subscribe("token-2")
	defer b.Unsubscribe("token-2", ch2)

	b.Send("token-1", sse.Event{Step: "downloading", Status: "in_progress", Message: "test"})

	select {
	case <-ch2:
		t.Fatal("token-2 should not receive token-1 events")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEventSerializesToJSON(t *testing.T) {
	event := sse.Event{
		Step:    "complete",
		Status:  "done",
		Message: "Restore complete",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded sse.Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Step != "complete" {
		t.Fatalf("step: got %q, want %q", decoded.Step, "complete")
	}
}

func TestEventVolumeFieldsSerialization(t *testing.T) {
	event := sse.Event{
		Step:        "downloading",
		Status:      "in_progress",
		Message:     "Downloading backup (34%)",
		Volume:      "postgres_data",
		VolumeIndex: 1,
		VolumeTotal: 2,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(data, &parsed)

	if parsed["volume"] != "postgres_data" {
		t.Fatalf("volume: got %v", parsed["volume"])
	}
	if int(parsed["volume_index"].(float64)) != 1 {
		t.Fatalf("volume_index: got %v", parsed["volume_index"])
	}
	if int(parsed["volume_total"].(float64)) != 2 {
		t.Fatalf("volume_total: got %v", parsed["volume_total"])
	}
}

func TestEventVolumeFieldsOmittedWhenZero(t *testing.T) {
	event := sse.Event{
		Step:    "downloading",
		Status:  "in_progress",
		Message: "Downloading",
	}

	data, _ := json.Marshal(event)
	var parsed map[string]any
	json.Unmarshal(data, &parsed)

	if _, ok := parsed["volume"]; ok {
		t.Fatal("volume field should be omitted when empty")
	}
	if _, ok := parsed["volume_index"]; ok {
		t.Fatal("volume_index field should be omitted when zero")
	}
	if _, ok := parsed["volume_total"]; ok {
		t.Fatal("volume_total field should be omitted when zero")
	}
}

func TestBroadcasterWaitBlocksUntilComplete(t *testing.T) {
	b := sse.NewBroadcaster()
	b.Subscribe("token-1")

	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		b.Wait(ctx, "token-1")
		close(done)
	}()

	b.Send("token-1", sse.Event{Step: "complete", Status: "done", Message: "done"})
	b.Complete("token-1")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after Complete")
	}
}

func TestBroadcasterBuffersEventsBeforeSubscribe(t *testing.T) {
	b := sse.NewBroadcaster()
	b.Register("token-1")

	b.Send("token-1", sse.Event{Step: "downloading", Status: "in_progress", Message: "50%"})
	b.Send("token-1", sse.Event{Step: "extracting", Status: "in_progress", Message: "extracting"})

	ch := b.Subscribe("token-1")
	defer b.Unsubscribe("token-1", ch)

	var events []sse.Event
	for i := 0; i < 2; i++ {
		select {
		case e := <-ch:
			events = append(events, e)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for buffered event %d", i)
		}
	}

	if events[0].Step != "downloading" {
		t.Fatalf("first buffered event: got %q, want %q", events[0].Step, "downloading")
	}
	if events[1].Step != "extracting" {
		t.Fatalf("second buffered event: got %q, want %q", events[1].Step, "extracting")
	}
}

func TestBroadcasterRegisterThenLiveEvents(t *testing.T) {
	b := sse.NewBroadcaster()
	b.Register("token-1")

	b.Send("token-1", sse.Event{Step: "downloading", Status: "in_progress", Message: "buffered"})

	ch := b.Subscribe("token-1")
	defer b.Unsubscribe("token-1", ch)

	b.Send("token-1", sse.Event{Step: "complete", Status: "done", Message: "live"})

	select {
	case e := <-ch:
		if e.Step != "downloading" {
			t.Fatalf("expected buffered event first, got %q", e.Step)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	select {
	case e := <-ch:
		if e.Step != "complete" {
			t.Fatalf("expected live event second, got %q", e.Step)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}
