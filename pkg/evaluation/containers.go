package evaluation

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

// containerTracker tracks running containers for cleanup on cancellation.
type containerTracker struct {
	mu         sync.Mutex
	containers map[string]struct{}
}

func newContainerTracker() *containerTracker {
	return &containerTracker{containers: make(map[string]struct{})}
}

func (ct *containerTracker) add(id string) {
	ct.mu.Lock()
	ct.containers[id] = struct{}{}
	ct.mu.Unlock()
}

func (ct *containerTracker) remove(id string) {
	ct.mu.Lock()
	delete(ct.containers, id)
	ct.mu.Unlock()
}

func (ct *containerTracker) killAll() {
	ct.mu.Lock()
	var ids []string
	for id := range ct.containers {
		ids = append(ids, id)
	}
	ct.mu.Unlock()

	for _, id := range ids {
		killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = exec.CommandContext(killCtx, "docker", "kill", id).Run()
		cancel()
	}
}
