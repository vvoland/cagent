package desktop

import (
	"context"
)

func IsDockerDesktopRunning(ctx context.Context) bool {
	err := ClientBackend.Get(ctx, "/ping", nil)
	return err == nil
}
