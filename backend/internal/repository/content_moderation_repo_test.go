package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildContentModerationLogWhere_BlockedIncludesAllBlockActions(t *testing.T) {
	where, args := buildContentModerationLogWhere(service.ContentModerationLogFilter{Result: "blocked"})

	require.Empty(t, args)
	sql := strings.Join(where, " AND ")
	require.Contains(t, sql, "l.action IN ('block', 'keyword_block', 'hash_block')")
	require.NotContains(t, sql, "l.action = 'block'")
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesHashBlock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND action <> 'hash_block'")).
		WithArgs(int64(1001), since, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, false)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCreateLog_InsertsMatchedKeyword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	now := time.Now()
	userID := int64(1001)
	apiKeyID := int64(2002)
	groupID := int64(3003)
	latency := 123
	queueDelay := 7
	log := &service.ContentModerationLog{
		RequestID:         "req-1",
		UserID:            &userID,
		UserEmail:         "user@example.com",
		APIKeyID:          &apiKeyID,
		APIKeyName:        "main key",
		GroupID:           &groupID,
		GroupName:         "GPT",
		Endpoint:          "/v1/responses",
		Provider:          "openai",
		Model:             "gpt-5.4-mini",
		Mode:              service.ContentModerationModePreBlock,
		Action:            service.ContentModerationActionKeywordBlock,
		Flagged:           true,
		HighestCategory:   "keyword",
		HighestScore:      1,
		CategoryScores:    map[string]float64{"keyword": 1},
		ThresholdSnapshot: map[string]float64{"sexual": 0.65},
		MatchedKeyword:    "病毒",
		InputExcerpt:      "full admin input with 病毒 near the end",
		UpstreamLatencyMS: &latency,
		Error:             "",
		ViolationCount:    3,
		QueueDelayMS:      &queueDelay,
	}

	mock.ExpectQuery(regexp.QuoteMeta("matched_keyword, input_excerpt")).
		WithArgs(
			log.RequestID, userID, log.UserEmail, apiKeyID, log.APIKeyName, groupID, log.GroupName,
			log.Endpoint, log.Provider, log.Model, log.Mode, log.Action, log.Flagged, log.HighestCategory, log.HighestScore,
			jsonTextArg(`{"keyword":1}`), jsonTextArg(`{"sexual":0.65}`), log.MatchedKeyword, log.InputExcerpt, latency, log.Error,
			log.ViolationCount, log.AutoBanned, log.EmailSent, queueDelay,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(42), now))

	err = repo.CreateLog(context.Background(), log)

	require.NoError(t, err)
	require.Equal(t, int64(42), log.ID)
	require.Equal(t, now, log.CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

type jsonTextArg string

func (a jsonTextArg) Match(value driver.Value) bool {
	got, ok := value.(string)
	return ok && got == string(a)
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesCyberPolicyWhenRequested(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND ($3::bool IS FALSE OR action <> 'cyber_policy')")).
		WithArgs(int64(1001), since, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, true)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}
