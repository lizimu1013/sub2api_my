package service

import (
	"context"
	"sort"
	"strings"
	"time"
)

const (
	OpsRecentAccountStatusNormal             = "normal"
	OpsRecentAccountStatusRateLimited        = "rate_limited"
	OpsRecentAccountStatusError              = "error"
	OpsRecentAccountStatusOverloaded         = "overloaded"
	OpsRecentAccountStatusTempUnschedulable  = "temp_unschedulable"
	OpsRecentAccountStatusPaused             = "paused"
	OpsRecentAccountStatusDisabled           = "disabled"
	OpsRecentAccountStatusOther              = "other"
	defaultOpsRecentAccountStatusItemLimit   = 20
	maxOpsRecentAccountStatusItemLimit       = 100
	opsRecentAccountStatusDefaultWindowHours = 24
)

type OpsRecentAccountStatusFilter struct {
	StartTime time.Time
	EndTime   time.Time
	Platform  string
	GroupID   *int64
	Limit     int
}

type OpsRecentAccountStatusSummary struct {
	GeneratedAt time.Time `json:"generated_at"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Platform    string    `json:"platform"`
	GroupID     *int64    `json:"group_id,omitempty"`

	TotalCount             int64 `json:"total_count"`
	NormalCount            int64 `json:"normal_count"`
	RateLimitedCount       int64 `json:"rate_limited_count"`
	ErrorCount             int64 `json:"error_count"`
	OverloadedCount        int64 `json:"overloaded_count"`
	TempUnschedulableCount int64 `json:"temp_unschedulable_count"`
	PausedCount            int64 `json:"paused_count"`
	DisabledCount          int64 `json:"disabled_count"`
	OtherCount             int64 `json:"other_count"`

	Items []OpsRecentAccountStatusItem `json:"items"`
}

type OpsRecentAccountStatusItem struct {
	AccountID      int64      `json:"account_id"`
	AccountName    string     `json:"account_name"`
	Platform       string     `json:"platform"`
	GroupID        int64      `json:"group_id"`
	GroupName      string     `json:"group_name"`
	CreatedAt      time.Time  `json:"created_at"`
	Status         string     `json:"status"`
	StatusCategory string     `json:"status_category"`
	Schedulable    bool       `json:"schedulable"`
	RateLimitReset *time.Time `json:"rate_limit_reset_at,omitempty"`
	OverloadUntil  *time.Time `json:"overload_until,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

func (s *OpsService) GetRecentAccountStatusSummary(ctx context.Context, filter OpsRecentAccountStatusFilter) (*OpsRecentAccountStatusSummary, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}

	filter.Platform = strings.TrimSpace(filter.Platform)
	filter.Limit = normalizeOpsRecentAccountStatusLimit(filter.Limit)
	now := time.Now()
	if filter.EndTime.IsZero() {
		filter.EndTime = now
	}
	if filter.StartTime.IsZero() {
		filter.StartTime = filter.EndTime.Add(-opsRecentAccountStatusDefaultWindowHours * time.Hour)
	}
	if filter.StartTime.After(filter.EndTime) {
		filter.StartTime, filter.EndTime = filter.EndTime, filter.StartTime
	}

	accounts, err := s.listAllAccountsForOps(ctx, filter.Platform)
	if err != nil {
		return nil, err
	}

	filtered := make([]Account, 0, len(accounts))
	for _, acc := range accounts {
		if !accountCreatedInRange(acc.CreatedAt, filter.StartTime, filter.EndTime) {
			continue
		}
		if filter.GroupID != nil && *filter.GroupID > 0 && !accountHasGroup(acc, *filter.GroupID) {
			continue
		}
		filtered = append(filtered, acc)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	out := &OpsRecentAccountStatusSummary{
		GeneratedAt: now.UTC(),
		StartTime:   filter.StartTime,
		EndTime:     filter.EndTime,
		Platform:    filter.Platform,
		GroupID:     filter.GroupID,
		Items:       make([]OpsRecentAccountStatusItem, 0, minInt(len(filtered), filter.Limit)),
	}

	for _, acc := range filtered {
		category := classifyRecentAccountStatus(acc, now)
		out.TotalCount++
		switch category {
		case OpsRecentAccountStatusNormal:
			out.NormalCount++
		case OpsRecentAccountStatusRateLimited:
			out.RateLimitedCount++
		case OpsRecentAccountStatusError:
			out.ErrorCount++
		case OpsRecentAccountStatusOverloaded:
			out.OverloadedCount++
		case OpsRecentAccountStatusTempUnschedulable:
			out.TempUnschedulableCount++
		case OpsRecentAccountStatusPaused:
			out.PausedCount++
		case OpsRecentAccountStatusDisabled:
			out.DisabledCount++
		default:
			out.OtherCount++
		}

		if len(out.Items) < filter.Limit {
			groupID, groupName := displayGroupForRecentAccount(acc, filter.GroupID)
			out.Items = append(out.Items, OpsRecentAccountStatusItem{
				AccountID:      acc.ID,
				AccountName:    acc.Name,
				Platform:       acc.Platform,
				GroupID:        groupID,
				GroupName:      groupName,
				CreatedAt:      acc.CreatedAt,
				Status:         acc.Status,
				StatusCategory: category,
				Schedulable:    acc.Schedulable,
				RateLimitReset: acc.RateLimitResetAt,
				OverloadUntil:  acc.OverloadUntil,
				ErrorMessage:   acc.ErrorMessage,
			})
		}
	}

	return out, nil
}

func normalizeOpsRecentAccountStatusLimit(limit int) int {
	if limit <= 0 {
		return defaultOpsRecentAccountStatusItemLimit
	}
	if limit > maxOpsRecentAccountStatusItemLimit {
		return maxOpsRecentAccountStatusItemLimit
	}
	return limit
}

func accountCreatedInRange(createdAt, startTime, endTime time.Time) bool {
	if createdAt.IsZero() {
		return false
	}
	return !createdAt.Before(startTime) && createdAt.Before(endTime)
}

func accountHasGroup(acc Account, groupID int64) bool {
	if groupID <= 0 {
		return true
	}
	for _, grp := range acc.Groups {
		if grp != nil && grp.ID == groupID {
			return true
		}
	}
	return false
}

func classifyRecentAccountStatus(acc Account, now time.Time) string {
	isRateLimited := acc.RateLimitResetAt != nil && now.Before(*acc.RateLimitResetAt)
	isOverloaded := acc.OverloadUntil != nil && now.Before(*acc.OverloadUntil)
	isTempUnschedulable := acc.TempUnschedulableUntil != nil && now.Before(*acc.TempUnschedulableUntil)

	switch {
	case acc.Status == StatusError:
		return OpsRecentAccountStatusError
	case acc.Status == StatusActive && isRateLimited:
		return OpsRecentAccountStatusRateLimited
	case acc.Status == StatusActive && isOverloaded:
		return OpsRecentAccountStatusOverloaded
	case acc.Status == StatusActive && isTempUnschedulable:
		return OpsRecentAccountStatusTempUnschedulable
	case acc.Status == StatusActive && acc.Schedulable:
		return OpsRecentAccountStatusNormal
	case acc.Status == StatusActive:
		return OpsRecentAccountStatusPaused
	case acc.Status == StatusDisabled:
		return OpsRecentAccountStatusDisabled
	default:
		return OpsRecentAccountStatusOther
	}
}

func displayGroupForRecentAccount(acc Account, filterGroupID *int64) (int64, string) {
	if filterGroupID != nil && *filterGroupID > 0 {
		for _, grp := range acc.Groups {
			if grp != nil && grp.ID == *filterGroupID {
				return grp.ID, grp.Name
			}
		}
	}
	if len(acc.Groups) > 0 && acc.Groups[0] != nil {
		return acc.Groups[0].ID, acc.Groups[0].Name
	}
	return 0, ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
