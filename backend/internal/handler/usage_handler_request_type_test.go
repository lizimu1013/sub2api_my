package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userUsageRepoCapture struct {
	service.UsageLogRepository
	listParams   pagination.PaginationParams
	listFilters  usagestats.UsageLogFilters
	statsFilters usagestats.UsageLogFilters
	trendFilters usagestats.UsageLogFilters
	groupFilters usagestats.UsageLogFilters
	listRows     []service.UsageLog
	stats        *usagestats.UsageStats
	modelStats   []usagestats.ModelStat
	groupStats   []usagestats.GroupStat
}

func (s *userUsageRepoCapture) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listParams = params
	s.listFilters = filters
	return s.listRows, &pagination.PaginationResult{
		Total:    int64(len(s.listRows)),
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

type userUsageSettingRepoStub struct {
	service.SettingRepository
	values map[string]string
}

func (s *userUsageSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s != nil {
		if value, ok := s.values[key]; ok {
			return value, nil
		}
	}
	return "", service.ErrSettingNotFound
}

func (s *userUsageRepoCapture) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.statsFilters = filters
	if s.stats != nil {
		return s.stats, nil
	}
	return &usagestats.UsageStats{}, nil
}

func (s *userUsageRepoCapture) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8, ipAddress ...string) ([]usagestats.TrendDataPoint, error) {
	s.trendFilters = usagestats.UsageLogFilters{
		UserID:      userID,
		APIKeyID:    apiKeyID,
		AccountID:   accountID,
		GroupID:     groupID,
		Model:       model,
		RequestType: requestType,
		Stream:      stream,
		BillingType: billingType,
	}
	if len(ipAddress) > 0 {
		s.trendFilters.IPAddress = ipAddress[0]
	}
	return []usagestats.TrendDataPoint{}, nil
}

func (s *userUsageRepoCapture) GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8, ipAddress ...string) ([]usagestats.ModelStat, error) {
	return s.modelStats, nil
}

func (s *userUsageRepoCapture) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8, ipAddress ...string) ([]usagestats.GroupStat, error) {
	s.groupFilters = usagestats.UsageLogFilters{
		UserID:      userID,
		APIKeyID:    apiKeyID,
		AccountID:   accountID,
		GroupID:     groupID,
		RequestType: requestType,
		Stream:      stream,
		BillingType: billingType,
	}
	if len(ipAddress) > 0 {
		s.groupFilters.IPAddress = ipAddress[0]
	}
	return s.groupStats, nil
}

func newUserUsageRequestTypeTestRouter(repo *userUsageRepoCapture) *gin.Engine {
	return newUserUsageRequestTypeTestRouterWithSettings(repo, nil)
}

func newUserUsageRequestTypeTestRouterWithSettings(repo *userUsageRepoCapture, settingValues map[string]string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	var settingSvc *service.SettingService
	if settingValues != nil {
		settingSvc = service.NewSettingService(&userUsageSettingRepoStub{values: settingValues}, &config.Config{})
	}
	handler := NewUsageHandler(usageSvc, nil, nil, settingSvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage", handler.List)
	router.GET("/usage/stats", handler.Stats)
	router.GET("/usage/dashboard/models", handler.DashboardModels)
	router.GET("/usage/dashboard/snapshot-v2", handler.DashboardSnapshotV2)
	return router
}

func TestUserUsageListRequestTypePriority(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?request_type=ws_v2&stream=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.listFilters.UserID)
	require.NotNil(t, repo.listFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeWSV2), *repo.listFilters.RequestType)
	require.Nil(t, repo.listFilters.Stream)
}

func TestUserUsageListAppliesVisibleDaysLimit(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouterWithSettings(repo, map[string]string{
		service.SettingKeyUserUsageVisibleDays: "0",
	})

	req := httptest.NewRequest(http.MethodGet, "/usage?start_date=2020-01-01&timezone=UTC", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, repo.listFilters.StartTime)
	expectedStart := timezone.StartOfDayInUserLocation(time.Now(), "UTC")
	require.WithinDuration(t, expectedStart, *repo.listFilters.StartTime, time.Second)
}

func TestUserUsageListInvalidRequestType(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?request_type=invalid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageListInvalidStream(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?stream=invalid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageListAdvancedFilters(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?group_id=7&model=gpt-5&billing_type=1&billing_mode=image&start_date=2026-03-01&end_date=2026-03-02", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.listFilters.UserID)
	require.Equal(t, int64(7), repo.listFilters.GroupID)
	require.Equal(t, "gpt-5", repo.listFilters.Model)
	require.Equal(t, usagestats.ModelSourceRequested, repo.listFilters.ModelFilterSource)
	require.NotNil(t, repo.listFilters.BillingType)
	require.Equal(t, int8(1), *repo.listFilters.BillingType)
	require.Equal(t, "image", repo.listFilters.BillingMode)
	require.NotNil(t, repo.listFilters.StartTime)
	require.NotNil(t, repo.listFilters.EndTime)
}

func TestUserUsageListInvalidBillingMode(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?billing_mode=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageListAllowsVideoBillingMode(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?billing_mode=video", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "video", repo.listFilters.BillingMode)
}

func TestUserUsageListOmitsTotalCostAndAdminCostFields(t *testing.T) {
	ipAddress := "203.0.113.10"
	upstreamModel := "upstream-private-model"
	billingTier := "internal-tier"
	channelID := int64(99)
	accountRateMultiplier := 1.7
	accountStatsCost := 0.12
	repo := &userUsageRepoCapture{
		listRows: []service.UsageLog{{
			ID:                    1,
			UserID:                42,
			APIKeyID:              7,
			AccountID:             5,
			RequestID:             "req_user_billing",
			Model:                 "gpt-5",
			InputCost:             0.01,
			OutputCost:            0.02,
			CacheCreationCost:     0.03,
			CacheReadCost:         0.04,
			TotalCost:             0.10,
			ActualCost:            0.08,
			RateMultiplier:        0.8,
			IPAddress:             &ipAddress,
			UpstreamModel:         &upstreamModel,
			BillingTier:           &billingTier,
			ChannelID:             &channelID,
			AccountRateMultiplier: &accountRateMultiplier,
			AccountStatsCost:      &accountStatsCost,
		}},
	}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, `"input_cost":0.01`)
	require.Contains(t, body, `"output_cost":0.02`)
	require.Contains(t, body, `"cache_creation_cost":0.03`)
	require.Contains(t, body, `"cache_read_cost":0.04`)
	require.NotContains(t, body, `"total_cost"`)
	require.Contains(t, body, `"actual_cost":0.08`)
	require.Contains(t, body, `"rate_multiplier":0.8`)
	require.Contains(t, body, `"ip_address":"203.0.113.10"`)
	require.NotContains(t, body, "upstream_endpoint")
	require.NotContains(t, body, "account_rate_multiplier")
	require.NotContains(t, body, "account_stats_cost")
	require.NotContains(t, body, "upstream_model")
	require.NotContains(t, body, "billing_tier")
	require.NotContains(t, body, "channel_id")
	require.NotContains(t, body, `"account":`)
}

func TestUserUsageStatsUsesScopedFilters(t *testing.T) {
	accountCost := 0.12
	repo := &userUsageRepoCapture{
		stats: &usagestats.UsageStats{
			TotalCost:        0.10,
			TotalActualCost:  0.08,
			TotalAccountCost: &accountCost,
			Endpoints: []usagestats.EndpointStat{{
				Endpoint:   "/v1/chat/completions",
				Cost:       0.10,
				ActualCost: 0.08,
			}},
			UpstreamEndpoints: []usagestats.EndpointStat{{
				Endpoint: "/v1/responses",
			}},
			EndpointPaths: []usagestats.EndpointStat{{
				Endpoint: "/v1/chat/completions -> /v1/responses",
			}},
		},
	}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/stats?group_id=9&request_type=sync&billing_mode=token&start_date=2026-03-01&end_date=2026-03-02", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.statsFilters.UserID)
	require.Equal(t, int64(9), repo.statsFilters.GroupID)
	require.Equal(t, usagestats.ModelSourceRequested, repo.statsFilters.ModelFilterSource)
	require.NotNil(t, repo.statsFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeSync), *repo.statsFilters.RequestType)
	require.Equal(t, "token", repo.statsFilters.BillingMode)
	require.NotContains(t, rec.Body.String(), `"total_cost"`)
	require.Contains(t, rec.Body.String(), `"total_actual_cost":0.08`)
	require.NotContains(t, rec.Body.String(), `"cost":`)
	require.NotContains(t, rec.Body.String(), "total_account_cost")
	require.NotContains(t, rec.Body.String(), "upstream_endpoints")
	require.NotContains(t, rec.Body.String(), "endpoint_paths")
}

func TestUserUsageDashboardModelsOmitsAccountCost(t *testing.T) {
	repo := &userUsageRepoCapture{
		modelStats: []usagestats.ModelStat{{
			Model:       "gpt-5",
			Requests:    2,
			TotalTokens: 30,
			Cost:        0.10,
			ActualCost:  0.08,
			AccountCost: 0.07,
		}},
	}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/models?start_date=2026-03-01&end_date=2026-03-02", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.NotContains(t, body, `"cost"`)
	require.Contains(t, body, `"actual_cost":0.08`)
	require.NotContains(t, body, "account_cost")
}

func TestUserUsageCostMappersOmitStandardCost(t *testing.T) {
	trend := userTrendDataFromUsageStats([]usagestats.TrendDataPoint{{Cost: 0.10, ActualCost: 0.08}})
	dashboard := userDashboardStatsFromUsageStats(&usagestats.UserDashboardStats{
		TotalCost:       1.25,
		TotalActualCost: 1.00,
		TodayCost:       0.50,
		TodayActualCost: 0.40,
	})

	payload, err := json.Marshal(struct {
		Trend     []userTrendDataPoint `json:"trend"`
		Dashboard *userDashboardStats  `json:"dashboard"`
	}{Trend: trend, Dashboard: dashboard})
	require.NoError(t, err)
	body := string(payload)
	require.NotContains(t, body, `"cost"`)
	require.NotContains(t, body, "total_cost")
	require.NotContains(t, body, "today_cost")
	require.Contains(t, body, `"actual_cost":0.08`)
	require.Contains(t, body, `"total_actual_cost":1`)
	require.Contains(t, body, `"today_actual_cost":0.4`)
}

func TestUserUsageDashboardModelsRejectsAdminModelSources(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/models?model_source=upstream&start_date=2026-03-01&end_date=2026-03-02", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageSnapshotUsesScopedFilters(t *testing.T) {
	repo := &userUsageRepoCapture{
		modelStats: []usagestats.ModelStat{{Model: "gpt-5", AccountCost: 0.07}},
		groupStats: []usagestats.GroupStat{{GroupID: 1, GroupName: "default", AccountCost: 0.06}},
	}
	router := newUserUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/snapshot-v2?include_trend=true&include_model_stats=true&include_group_stats=true&group_id=11&request_type=stream&start_date=2026-03-01&end_date=2026-03-02", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.trendFilters.UserID)
	require.Equal(t, int64(11), repo.trendFilters.GroupID)
	require.NotNil(t, repo.trendFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeStream), *repo.trendFilters.RequestType)
	require.Equal(t, int64(42), repo.groupFilters.UserID)
	require.Equal(t, int64(11), repo.groupFilters.GroupID)
	require.NotContains(t, rec.Body.String(), "account_cost")
}

func TestUserUsageSnapshotRejectsInvalidIncludeFlags(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo)

	for _, query := range []string{
		"include_trend=bad",
		"include_model_stats=bad",
		"include_group_stats=bad",
	} {
		req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/snapshot-v2?start_date=2026-03-01&end_date=2026-03-02&"+query, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, query)
	}
}
