package handler

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func waitForGroupFirstTokenDelay(ctx context.Context, group *service.Group) bool {
	if group == nil || group.FirstTokenDelayMS <= 0 {
		return true
	}
	timer := time.NewTimer(time.Duration(group.FirstTokenDelayMS) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
