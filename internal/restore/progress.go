package restore

import (
	"fmt"
	"io"

	"github.com/offen/restore-manager/internal/sse"
)

type ProgressWriter struct {
	inner       io.Writer
	total       int64
	written     int64
	broadcaster *sse.Broadcaster
	token       string
	lastPercent int
}

func NewProgressWriter(inner io.Writer, total int64, broadcaster *sse.Broadcaster, token string) *ProgressWriter {
	return &ProgressWriter{
		inner:       inner,
		total:       total,
		broadcaster: broadcaster,
		token:       token,
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.inner.Write(p)
	pw.written += int64(n)

	percent := int(pw.written * 100 / pw.total)
	if percent > 100 {
		percent = 100
	}

	if percent >= pw.lastPercent+5 || percent == 100 {
		pw.broadcaster.Send(pw.token, sse.Event{
			Step:    "downloading",
			Status:  "in_progress",
			Message: fmt.Sprintf("Downloading backup (%d%%)", percent),
			Percent: percent,
		})
		pw.lastPercent = percent
	}

	return n, err
}
