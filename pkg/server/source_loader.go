package server

import (
	"context"
	"time"

	"github.com/docker/cagent/pkg/config"
)

type sourceLoader struct {
	inner           config.Source
	refreshInterval time.Duration
	lastLoaded      time.Time
}

func newSourceLoader(inner config.Source, refreshInterval time.Duration) *sourceLoader {
	return &sourceLoader{
		inner:           inner,
		refreshInterval: refreshInterval,
	}
}

func (sl *sourceLoader) Name() string {
	return sl.inner.Name()
}

func (sl *sourceLoader) ParentDir() string {
	return sl.inner.ParentDir()
}

func (sl *sourceLoader) Read(ctx context.Context) ([]byte, error) {
	if sl.refreshInterval == 0 {
		return sl.inner.Read(ctx)
	}

	if time.Since(sl.lastLoaded) < sl.refreshInterval {
		return sl.inner.Read(ctx)
	}

	data, err := sl.inner.Read(ctx)
	if err != nil {
		return nil, err
	}

	sl.lastLoaded = time.Now()
	return data, nil
}
