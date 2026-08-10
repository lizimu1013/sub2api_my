package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const securityAuditCompletedContextKey = "sub2api.security_audit.completed"

type securityAuditGroupResolver func(context.Context, int64) (*service.Group, error)

// cachesSecurityAuditCompletion reports whether a successful audit may be
// reused for the rest of the gin request. WebSocket turns share one Context
// across many response.create frames and must be audited independently.
func cachesSecurityAuditCompletion(stage string) bool {
	switch strings.TrimSpace(stage) {
	case "", "http":
		return true
	default:
		return false
	}
}

func (h *GatewayHandler) checkSecurityAudit(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte) *securityaudit.Decision {
	if h == nil {
		return nil
	}
	return runSecurityAuditWithResolver(c, reqLog, h.securityAuditCoordinator, h.contentModerationService, apiKey, subject, apiKeyGroupResolver(h.apiKeyService), protocol, model, body, "http")
}

func (h *OpenAIGatewayHandler) checkSecurityAudit(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte) *securityaudit.Decision {
	if h == nil {
		return nil
	}
	return runSecurityAuditWithResolver(c, reqLog, h.securityAuditCoordinator, h.contentModerationService, apiKey, subject, apiKeyGroupResolver(h.apiKeyService), protocol, model, body, "http")
}

func (h *OpenAIGatewayHandler) checkSecurityAuditStage(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte, stage string) *securityaudit.Decision {
	if h == nil {
		return nil
	}
	return runSecurityAuditWithResolver(c, reqLog, h.securityAuditCoordinator, h.contentModerationService, apiKey, subject, apiKeyGroupResolver(h.apiKeyService), protocol, model, body, stage)
}

func runSecurityAudit(c *gin.Context, reqLog *zap.Logger, coordinator *securityaudit.Coordinator, legacy *service.ContentModerationService, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte, stage string) *securityaudit.Decision {
	return runSecurityAuditWithResolver(c, reqLog, coordinator, legacy, apiKey, subject, nil, protocol, model, body, stage)
}

func apiKeyGroupResolver(apiKeys *service.APIKeyService) securityAuditGroupResolver {
	if apiKeys == nil {
		return nil
	}
	return apiKeys.ResolveGroupByID
}

func runSecurityAuditWithResolver(c *gin.Context, reqLog *zap.Logger, coordinator *securityaudit.Coordinator, legacy *service.ContentModerationService, apiKey *service.APIKey, subject middleware2.AuthSubject, resolveGroup securityAuditGroupResolver, protocol, model string, body []byte, stage string) *securityaudit.Decision {
	if c == nil || c.Request == nil {
		return nil
	}
	cacheCompletion := cachesSecurityAuditCompletion(stage)
	if cacheCompletion {
		if completed, exists := c.Get(securityAuditCompletedContextKey); exists && completed == true {
			return nil
		}
	}
	if coordinator == nil {
		legacyDecision := runContentModeration(c, reqLog, legacy, apiKey, subject, protocol, model, body)
		if legacyDecision == nil {
			return nil
		}
		decision := securityaudit.Decision{Kind: securityaudit.DecisionAllow, HTTPStatus: http.StatusOK, AllowNextStage: true}
		decision.Legacy = &securityaudit.LegacyDecision{
			Allowed: legacyDecision.Allowed, Blocked: legacyDecision.Blocked, Flagged: legacyDecision.Flagged,
			Message: legacyDecision.Message, StatusCode: legacyDecision.StatusCode,
			ErrorCode: "content_policy_violation", Action: legacyDecision.Action,
		}
		if legacyDecision.Blocked {
			decision.Kind, decision.HTTPStatus, decision.ErrorCode, decision.ClientMessage, decision.AllowNextStage = securityaudit.DecisionBlock, contentModerationStatus(legacyDecision), "content_policy_violation", legacyDecision.Message, false
		}
		if decision.AllowNextStage && cacheCompletion {
			c.Set(securityAuditCompletedContextKey, true)
		}
		return &decision
	}
	request := buildSecurityAuditRequest(c, apiKey, subject, protocol, model, body, stage)
	if reqLog != nil {
		reqLog.Info("security_audit.gateway_check_start",
			zap.String("request_id", request.RequestID), zap.Int64("user_id", request.UserID),
			zap.Int64("api_key_id", request.APIKeyID), zap.Int64p("group_id", request.GroupID),
			zap.String("endpoint", request.Endpoint), zap.String("provider", request.Provider),
			zap.String("protocol", request.Protocol), zap.String("model", request.Model), zap.String("stage", request.Stage),
			zap.Int("body_bytes", len(body)))
	}
	decision := coordinator.Check(c.Request.Context(), request)
	applyPromptAuditFallback(c, reqLog, &decision, apiKey, resolveGroup)
	if decision.AllowNextStage && cacheCompletion {
		c.Set(securityAuditCompletedContextKey, true)
	}
	if reqLog != nil {
		reqLog.Info("security_audit.gateway_check_done",
			zap.String("request_id", request.RequestID), zap.String("decision", string(decision.Kind)),
			zap.String("error_code", decision.ErrorCode), zap.Bool("allow_next_stage", decision.AllowNextStage),
			zap.String("stage", request.Stage))
	}
	return &decision
}

func applyPromptAuditFallback(c *gin.Context, reqLog *zap.Logger, decision *securityaudit.Decision, apiKey *service.APIKey, resolveGroup securityAuditGroupResolver) {
	if decision == nil || decision.Kind != securityaudit.DecisionBlock || decision.Legacy != nil && decision.Legacy.Blocked ||
		decision.Prompt == nil || decision.Prompt.FallbackGroupID == nil {
		return
	}
	groupID := *decision.Prompt.FallbackGroupID
	if resolveGroup == nil {
		if reqLog != nil {
			reqLog.Warn("security_audit.fallback_unavailable", zap.Int64("fallback_group_id", groupID), zap.String("reason", "group_resolver_unavailable"))
		}
		return
	}
	group, err := resolveGroup(c.Request.Context(), groupID)
	if err != nil || !service.IsGroupContextValid(group) || !group.IsActive() {
		if reqLog != nil {
			reqLog.Warn("security_audit.fallback_unavailable", zap.Int64("fallback_group_id", groupID), zap.String("reason", "group_unavailable"), zap.Error(err))
		}
		return
	}
	if apiKey == nil {
		apiKey, _ = middleware2.GetAPIKeyFromContext(c)
	}
	if apiKey == nil {
		if reqLog != nil {
			reqLog.Warn("security_audit.fallback_unavailable", zap.Int64("fallback_group_id", groupID), zap.String("reason", "api_key_unavailable"))
		}
		return
	}
	replacement := cloneAPIKeyWithGroup(apiKey, group)
	*apiKey = *replacement
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	// The original subscription was derived from the original group and must not
	// leak into billing or routing after a current-request group switch.
	c.Set(string(middleware2.ContextKeySubscription), (*service.UserSubscription)(nil))
	requestContext := context.WithValue(c.Request.Context(), ctxkey.Group, group)
	// A composite request may already have resolved its provider/model before
	// the audit ran. Those values belong to the original group and must not
	// leak into the fallback route; the scheduler will resolve them again for
	// the replacement group when needed.
	requestContext = context.WithValue(requestContext, ctxkey.ResolvedTargetPlatform, "")
	requestContext = context.WithValue(requestContext, ctxkey.ResolvedUpstreamModel, "")
	requestContext = context.WithValue(requestContext, ctxkey.RequestedPublicModel, "")
	requestContext = context.WithValue(requestContext, ctxkey.CompositeRouteSource, "")
	c.Request = c.Request.WithContext(requestContext)
	decision.Kind = securityaudit.DecisionFlag
	decision.HTTPStatus = http.StatusOK
	decision.ErrorCode = ""
	decision.ClientMessage = ""
	decision.AllowNextStage = true
	decision.Prompt.AllowNextStage = true
	if reqLog != nil {
		reqLog.Warn("security_audit.fallback_applied", zap.Int64("fallback_group_id", groupID), zap.String("group", group.Name), zap.Bool("current_request_only", true))
	}
}

func buildSecurityAuditRequest(c *gin.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte, stage string) securityaudit.Request {
	legacy := buildContentModerationInput(c, apiKey, subject, protocol, model, body)
	request := securityaudit.Request{
		RequestID: legacy.RequestID, UserID: legacy.UserID, UserEmail: legacy.UserEmail,
		APIKeyID: legacy.APIKeyID, APIKeyName: legacy.APIKeyName, GroupID: cloneSecurityAuditGroupID(legacy.GroupID),
		GroupName: legacy.GroupName, Provider: legacy.Provider, Endpoint: legacy.Endpoint,
		Protocol: legacy.Protocol, Model: legacy.Model, Body: body, Stage: strings.TrimSpace(stage),
	}
	if apiKey != nil && apiKey.User != nil {
		request.Username = apiKey.User.Username
		if request.UserEmail == "" {
			request.UserEmail = apiKey.User.Email
		}
	}
	if request.Stage == "" {
		request.Stage = "http"
	}
	return request
}

func securityAuditStatus(decision *securityaudit.Decision) int {
	if decision == nil || decision.HTTPStatus < 400 || decision.HTTPStatus > 599 {
		return http.StatusForbidden
	}
	return decision.HTTPStatus
}

func securityAuditErrorCode(decision *securityaudit.Decision) string {
	if decision == nil || strings.TrimSpace(decision.ErrorCode) == "" {
		return "content_policy_violation"
	}
	return decision.ErrorCode
}

func securityAuditMessage(decision *securityaudit.Decision) string {
	if decision == nil {
		return "Request blocked by content policy"
	}
	if decision.Legacy != nil && decision.Legacy.Blocked && strings.TrimSpace(decision.Legacy.Message) != "" {
		return decision.Legacy.Message
	}
	if strings.TrimSpace(decision.ClientMessage) != "" {
		return decision.ClientMessage
	}
	return "Request blocked by content policy"
}

func cloneSecurityAuditGroupID(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
