package restore_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/offen/restore-manager/internal/restore"
	"github.com/offen/restore-manager/internal/sse"
)

func TestProgressWriterEmitsAtFivePercentIntervals(t *testing.T) {
	b := sse.NewBroadcaster()
	ch := b.Subscribe("test-token")
	defer b.Unsubscribe("test-token", ch)

	var buf bytes.Buffer
	pw := restore.NewProgressWriter(&buf, 100, b, "test-token")

	pw.Write(bytes.Repeat([]byte("x"), 5))
	expectEvent(t, ch, 5)

	pw.Write(bytes.Repeat([]byte("x"), 5))
	expectEvent(t, ch, 10)

	pw.Write(bytes.Repeat([]byte("x"), 3))
	expectNoEvent(t, ch)

	pw.Write(bytes.Repeat([]byte("x"), 2))
	expectEvent(t, ch, 15)

	if buf.Len() != 15 {
		t.Fatalf("inner writer: got %d bytes, want 15", buf.Len())
	}
}

func TestProgressWriterEmitsAtHundredPercent(t *testing.T) {
	b := sse.NewBroadcaster()
	ch := b.Subscribe("test-token")
	defer b.Unsubscribe("test-token", ch)

	var buf bytes.Buffer
	pw := restore.NewProgressWriter(&buf, 20, b, "test-token")

	pw.Write(bytes.Repeat([]byte("x"), 20))

	var lastPercent int
	for {
		select {
		case ev := <-ch:
			lastPercent = ev.Percent
		case <-time.After(50 * time.Millisecond):
			if lastPercent != 100 {
				t.Fatalf("last percent: got %d, want 100", lastPercent)
			}
			return
		}
	}
}

func expectEvent(t *testing.T, ch chan sse.Event, wantPercent int) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.Percent != wantPercent {
			t.Fatalf("percent: got %d, want %d", ev.Percent, wantPercent)
		}
		if ev.Step != "downloading" {
			t.Fatalf("step: got %q, want %q", ev.Step, "downloading")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for event at %d%%", wantPercent)
	}
}

func expectNoEvent(t *testing.T, ch chan sse.Event) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}
