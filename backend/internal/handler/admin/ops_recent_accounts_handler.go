package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetRecentAccountStatus returns status statistics for accounts created in the selected window.
// GET /api/v1/admin/ops/recent-accounts/status
func (h *OpsHandler) GetRecentAccountStatus(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	startTime, endTime, err := parseOpsRecentAccountStatusTimeRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var groupID *int64
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}

	limit := 20
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			response.BadRequest(c, "Invalid limit")
			return
		}
		limit = n
	}

	data, err := h.opsService.GetRecentAccountStatusSummary(c.Request.Context(), service.OpsRecentAccountStatusFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Platform:  strings.TrimSpace(c.Query("platform")),
		GroupID:   groupID,
		Limit:     limit,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, data)
}

func parseOpsRecentAccountStatusTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	if strings.TrimSpace(c.Query("start_time")) != "" ||
		strings.TrimSpace(c.Query("end_time")) != "" ||
		strings.TrimSpace(c.Query("time_range")) != "" {
		return parseOpsTimeRange(c, "24h")
	}

	now := time.Now()
	year, month, day := now.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	return start, start.AddDate(0, 0, 1), nil
}
