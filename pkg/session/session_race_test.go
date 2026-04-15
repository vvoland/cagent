package session

import (
	"sync"
	"testing"

	"github.com/docker/docker-agent/pkg/chat"
)

func TestAddMessageUsageRecordConcurrent(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			s.AddMessageUsageRecord("agent", "model", 0.1, &chat.Usage{InputTokens: 10, OutputTokens: 5})
		})
	}
	wg.Wait()
	if got := len(s.MessageUsageHistory); got != 100 {
		t.Errorf("expected 100 records, got %d", got)
	}
}
