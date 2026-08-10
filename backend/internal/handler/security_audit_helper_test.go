package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCachesSecurityAuditCompletionSkipsWebSocketStages(t *testing.T) {
	require.True(t, cachesSecurityAuditCompletion("http"))
	require.True(t, cachesSecurityAuditCompletion(""))
	require.False(t, cachesSecurityAuditCompletion("first_turn"))
	require.False(t, cachesSecurityAuditCompletion("subsequent_turn"))
}

func TestRunSecurityAuditDoesNotSkipSubsequentWebSocketTurns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := &turnCountingEngine{mode: securityaudit.ModeAsync}
	coordinator := securityaudit.NewCoordinator(nil, engine)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	subject := middleware2.AuthSubject{UserID: 7, Concurrency: 1}
	first := runSecurityAudit(c, nil, coordinator, nil, nil, subject, "openai_responses", "gpt-test",
		[]byte(`{"type":"response.create","response":{"input":"benign"}}`), "first_turn")
	require.NotNil(t, first)
	require.True(t, first.AllowNextStage)
	require.Equal(t, int64(1), engine.enqueues.Load())
	_, cached := c.Get(securityAuditCompletedContextKey)
	require.False(t, cached, "WebSocket stages must not set the HTTP completion cache")

	// Even if an HTTP path previously cached completion on this Context, WS turns
	// must still audit every response.create payload.
	c.Set(securityAuditCompletedContextKey, true)

	second := runSecurityAudit(c, nil, coordinator, nil, nil, subject, "openai_responses", "gpt-test",
		[]byte(`{"type":"response.create","response":{"input":"malicious follow-up"}}`), "subsequent_turn")
	require.NotNil(t, second)
	require.Equal(t, int64(2), engine.enqueues.Load(), "subsequent WebSocket turns must be audited again")
}

func TestApplyPromptAuditFallbackChangesOnlyCurrentRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	oldID, targetID := int64(1), int64(2)
	oldGroup := &service.Group{ID: oldID, Name: "Original", Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	targetGroup := &service.Group{ID: targetID, Name: "Fallback", Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	apiKey := &service.APIKey{GroupID: &oldID, Group: oldGroup}
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeySubscription), &service.UserSubscription{})
	requestContext := service.WithResolvedTargetPlatform(c.Request.Context(), service.PlatformAnthropic)
	requestContext = context.WithValue(requestContext, ctxkey.ResolvedUpstreamModel, "old-upstream-model")
	c.Request = c.Request.WithContext(requestContext)
	decision := &securityaudit.Decision{
		Kind:   securityaudit.DecisionBlock,
		Prompt: &securityaudit.PromptDecision{Kind: securityaudit.DecisionBlock, FallbackGroupID: &targetID},
	}
	resolver := func(context.Context, int64) (*service.Group, error) { return targetGroup, nil }

	applyPromptAuditFallback(c, nil, decision, apiKey, resolver)

	require.True(t, decision.AllowNextStage)
	require.Equal(t, securityaudit.DecisionFlag, decision.Kind)
	require.Same(t, targetGroup, apiKey.Group)
	require.Equal(t, targetID, *apiKey.GroupID)
	groupFromContext, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
	require.True(t, ok)
	require.Same(t, targetGroup, groupFromContext)
	_, resolved := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	require.False(t, resolved)
	_, resolved = service.ResolvedUpstreamModelFromContext(c.Request.Context())
	require.False(t, resolved)
	subscription, exists := middleware2.GetSubscriptionFromContext(c)
	require.True(t, exists)
	require.Nil(t, subscription)
}

type turnCountingEngine struct {
	mode     securityaudit.Mode
	enqueues atomic.Int64
}

func (e *turnCountingEngine) EffectiveMode() securityaudit.Mode { return e.mode }
func (e *turnCountingEngine) Enqueue(context.Context, securityaudit.Request) error {
	e.enqueues.Add(1)
	return nil
}
func (e *turnCountingEngine) Evaluate(context.Context, securityaudit.Request) (*securityaudit.PromptDecision, error) {
	return &securityaudit.PromptDecision{Kind: securityaudit.DecisionAllow, AllowNextStage: true}, nil
}
