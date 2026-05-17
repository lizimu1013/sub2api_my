//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type opsRecentAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (s *opsRecentAccountRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	filtered := make([]Account, 0, len(s.accounts))
	for _, acc := range s.accounts {
		if platform != "" && acc.Platform != platform {
			continue
		}
		filtered = append(filtered, acc)
	}

	pageSize := params.Limit()
	offset := params.Offset()
	if offset >= len(filtered) {
		return []Account{}, &pagination.PaginationResult{Total: int64(len(filtered)), Page: params.Page, PageSize: pageSize, Pages: 1}, nil
	}
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], &pagination.PaginationResult{Total: int64(len(filtered)), Page: params.Page, PageSize: pageSize, Pages: 1}, nil
}

func TestOpsServiceGetRecentAccountStatusSummary_DefaultCountsAndItems(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(30 * time.Minute)
	overloadUntil := now.Add(20 * time.Minute)
	tempUntil := now.Add(10 * time.Minute)
	group := &Group{ID: 11, Name: "Pro", Platform: PlatformOpenAI}

	svc := &OpsService{
		accountRepo: &opsRecentAccountRepoStub{accounts: []Account{
			{ID: 1, Name: "normal", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, CreatedAt: now.Add(-time.Hour), Groups: []*Group{group}},
			{ID: 2, Name: "limited", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, CreatedAt: now.Add(-2 * time.Hour), RateLimitResetAt: &resetAt, Groups: []*Group{group}},
			{ID: 3, Name: "error", Platform: PlatformOpenAI, Status: StatusError, Schedulable: true, CreatedAt: now.Add(-3 * time.Hour), ErrorMessage: "invalid token", Groups: []*Group{group}},
			{ID: 4, Name: "overload", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, CreatedAt: now.Add(-4 * time.Hour), OverloadUntil: &overloadUntil, Groups: []*Group{group}},
			{ID: 5, Name: "temp", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, CreatedAt: now.Add(-5 * time.Hour), TempUnschedulableUntil: &tempUntil, Groups: []*Group{group}},
			{ID: 6, Name: "paused", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: false, CreatedAt: now.Add(-6 * time.Hour), Groups: []*Group{group}},
			{ID: 7, Name: "disabled", Platform: PlatformOpenAI, Status: StatusDisabled, Schedulable: false, CreatedAt: now.Add(-7 * time.Hour), Groups: []*Group{group}},
			{ID: 8, Name: "old", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, CreatedAt: now.Add(-48 * time.Hour), Groups: []*Group{group}},
		}},
	}

	got, err := svc.GetRecentAccountStatusSummary(context.Background(), OpsRecentAccountStatusFilter{
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now.Add(time.Hour),
		Platform:  PlatformOpenAI,
		GroupID:   int64Ptr(11),
		Limit:     3,
	})
	require.NoError(t, err)
	require.Equal(t, int64(7), got.TotalCount)
	require.Equal(t, int64(1), got.NormalCount)
	require.Equal(t, int64(1), got.RateLimitedCount)
	require.Equal(t, int64(1), got.ErrorCount)
	require.Equal(t, int64(1), got.OverloadedCount)
	require.Equal(t, int64(1), got.TempUnschedulableCount)
	require.Equal(t, int64(1), got.PausedCount)
	require.Equal(t, int64(1), got.DisabledCount)
	require.Len(t, got.Items, 3)
	require.Equal(t, int64(1), got.Items[0].AccountID)
	require.Equal(t, OpsRecentAccountStatusNormal, got.Items[0].StatusCategory)
	require.Equal(t, int64(11), got.Items[0].GroupID)
}

func TestClassifyRecentAccountStatus_ErrorPrecedesRateLimit(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(time.Hour)
	got := classifyRecentAccountStatus(Account{
		Status:           StatusError,
		Schedulable:      true,
		RateLimitResetAt: &resetAt,
	}, now)
	require.Equal(t, OpsRecentAccountStatusError, got)
}
