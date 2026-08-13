package securityaudit

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

type GuardEvaluator struct {
	scanner PromptScanner
	repo    JobRepository
	metrics Metrics
	clock   Clock

	global       chan struct{}
	perNodeLimit int
	nodeMu       sync.Mutex
	nodes        map[string]chan struct{}
}

func NewGuardEvaluator(scanner PromptScanner, repo JobRepository, metrics Metrics) *GuardEvaluator {
	return newGuardEvaluator(scanner, repo, metrics, 64, 16)
}

func newGuardEvaluator(scanner PromptScanner, repo JobRepository, metrics Metrics, globalLimit, perNodeLimit int) *GuardEvaluator {
	if globalLimit < 1 {
		globalLimit = 64
	}
	if perNodeLimit < 1 {
		perNodeLimit = 16
	}
	return &GuardEvaluator{scanner: scanner, repo: repo, metrics: metrics, clock: realClock{},
		global: make(chan struct{}, globalLimit), perNodeLimit: perNodeLimit, nodes: map[string]chan struct{}{}}
}

func (g *GuardEvaluator) Evaluate(ctx context.Context, cfg ActiveConfig, snapshot PromptSnapshot) (*PromptDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, &GuardError{Code: ErrorCodeUnavailable, Retryable: true, Cause: err}
	}
	if g == nil || g.scanner == nil {
		if g != nil && g.metrics != nil {
			g.metrics.Observe(DecisionUnavailable, 0)
		}
		logGuardFailure(snapshot, cfg, DecisionUnavailable, ErrorCodeUnavailable, "", 0)
		return g.failOpen(ctx, cfg, snapshot, &GuardError{Code: ErrorCodeUnavailable}, 0), nil
	}
	start := g.clock.Now()
	baseFields := snapshotLogFields(snapshot)
	baseFields["config_version"] = cfg.ConfigVersion
	endpoints := cfg.EnabledEndpoints()
	if len(endpoints) == 0 {
		if g.metrics != nil {
			g.metrics.Observe(DecisionUnavailable, g.clock.Now().Sub(start))
		}
		logGuardFailure(snapshot, cfg, DecisionUnavailable, ErrorCodeUnavailable, "", g.clock.Now().Sub(start))
		return g.failOpen(ctx, cfg, snapshot, &GuardError{Code: ErrorCodeUnavailable}, g.clock.Now().Sub(start)), nil
	}
	select {
	case g.global <- struct{}{}:
		defer func() { <-g.global }()
	default:
		if g.metrics != nil {
			g.metrics.IncBulkheadFull()
			g.metrics.Observe(DecisionUnavailable, g.clock.Now().Sub(start))
		}
		logGuardFailure(snapshot, cfg, DecisionUnavailable, ErrorCodeUnavailable, "", g.clock.Now().Sub(start))
		return g.failOpen(ctx, cfg, snapshot, &GuardError{Code: ErrorCodeUnavailable}, g.clock.Now().Sub(start)), nil
	}
	timeout := time.Duration(endpoints[0].TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = DefaultTimeoutMS * time.Millisecond
	}
	evalCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	inputLimit := minimumInputLimit(endpoints)
	syncSource := strings.ReplaceAll(snapshot.ScanText, promptAuditPrioritySeparator, "\n\n")
	syncInput, truncated := headTailRunes(syncSource, inputLimit)
	if syncInput == "" {
		if g.metrics != nil {
			g.metrics.Observe(DecisionAllow, g.clock.Now().Sub(start))
		}
		return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true}, nil
	}
	chunks := []string{syncInput}
	LogInfo(EventEvaluationStarted, mergeLogFields(baseFields, map[string]any{"chunk_total": len(chunks), "status": "started"}))
	results := make([]*NormalizedResult, 0, len(chunks))
	for index, chunk := range chunks {
		chunkStarted := g.clock.Now()
		LogInfo(EventChunkStarted, mergeLogFields(baseFields, map[string]any{
			"chunk_index": index + 1, "chunk_total": len(chunks),
			"chunk_chars": len([]rune(chunk)), "input_chars": snapshot.PromptLength, "input_limit": inputLimit,
			"status": "started",
		}))
		result, err := g.scanChunk(evalCtx, cfg, endpoints, chunk)
		if err != nil {
			if ctx.Err() != nil {
				return nil, err
			}
			code := guardErrorCode(err)
			LogWarn(EventChunkFailed, mergeLogFields(baseFields, map[string]any{
				"chunk_index": index + 1, "chunk_total": len(chunks),
				"chunk_chars": len([]rune(chunk)), "input_chars": snapshot.PromptLength, "input_limit": inputLimit,
				"latency_ms": g.clock.Now().Sub(chunkStarted).Milliseconds(), "error_code": code, "status": "failed",
			}))
			kind := DecisionUnavailable
			if code == ErrorCodeInvalidResponse {
				kind = DecisionInvalid
			}
			if g.metrics != nil {
				g.metrics.Observe(kind, g.clock.Now().Sub(start))
				var guardErr *GuardError
				if errors.As(err, &guardErr) && guardErr.Timeout {
					g.metrics.IncTimeout()
				}
			}
			logGuardFailure(snapshot, cfg, kind, code, "", g.clock.Now().Sub(start))
			return g.failOpen(ctx, cfg, snapshot, err, g.clock.Now().Sub(start)), nil
		}
		result.ChunkTotal = len(chunks)
		result.ChunkIndex = index + 1
		results = append(results, result)
		LogInfo(EventChunkCompleted, mergeLogFields(baseFields, map[string]any{
			"chunk_index": index + 1, "chunk_total": len(chunks),
			"chunk_chars": len([]rune(chunk)), "input_chars": snapshot.PromptLength, "input_limit": inputLimit,
			"guard_endpoint_id": result.GuardEndpointID, "action": result.Action,
			"latency_ms": g.clock.Now().Sub(chunkStarted).Milliseconds(), "status": "completed",
		}))
		if result.Action == ActionBlock {
			break
		}
	}
	aggregated, err := AggregateResults(results, g.clock.Now().Sub(start))
	if err != nil {
		if g.metrics != nil {
			g.metrics.Observe(DecisionInvalid, g.clock.Now().Sub(start))
		}
		logGuardFailure(snapshot, cfg, DecisionInvalid, ErrorCodeInvalidResponse, "", g.clock.Now().Sub(start))
		return g.failOpen(ctx, cfg, snapshot, &GuardError{Code: ErrorCodeInvalidResponse, Cause: err}, g.clock.Now().Sub(start)), nil
	}
	aggregated.ChunkTotal = len(chunks)
	kind := DecisionAllow
	if aggregated.Action == ActionWarn {
		kind = DecisionFlag
	}
	if aggregated.Action == ActionBlock {
		kind = DecisionBlock
	}
	decision := &PromptDecision{
		Kind: kind, Result: aggregated, AllowNextStage: kind == DecisionAllow || kind == DecisionFlag,
		SyncTruncated: truncated,
	}
	if kind == DecisionBlock {
		decision.ErrorCode = ErrorCodeBlocked
	}
	if g.metrics != nil {
		g.metrics.Observe(kind, g.clock.Now().Sub(start))
	}
	LogInfo(EventChunksAggregated, mergeLogFields(baseFields, map[string]any{
		"decision":   kind,
		"risk_level": aggregated.RiskLevel, "action": aggregated.Action, "chunk_total": aggregated.ChunkTotal,
		"latency_ms": aggregated.LatencyMS, "guard_endpoint_id": aggregated.GuardEndpointID, "stage": snapshot.Stage,
		"status": "completed",
	}))
	if g.repo != nil {
		if _, recordErr := g.repo.RecordBlocking(ctx, snapshot.Redacted(), cfg.ConfigVersion, aggregated, cfg.StorePassEvents); recordErr != nil {
			if g.metrics != nil {
				g.metrics.IncRecordFailed()
			}
			LogWarn(EventResultRecordFailed, mergeLogFields(baseFields, map[string]any{
				"decision": kind, "error_code": "result_record_failed", "stage": snapshot.Stage,
				"status": "failed",
			}))
		}
	}
	if kind == DecisionBlock {
		LogWarn(EventGuardBlocked, mergeLogFields(baseFields, map[string]any{
			"guard_endpoint_id": aggregated.GuardEndpointID,
			"decision":          kind, "risk_level": aggregated.RiskLevel, "action": aggregated.Action, "chunk_total": aggregated.ChunkTotal,
			"latency_ms": aggregated.LatencyMS, "status": "blocked", "error_code": ErrorCodeBlocked,
			"stage": snapshot.Stage, "upstream_dispatched": false, "billing_preconsumed": false,
		}))
	} else {
		LogInfo(EventGuardAllowed, mergeLogFields(baseFields, map[string]any{
			"decision": kind, "risk_level": aggregated.RiskLevel, "action": aggregated.Action,
			"guard_endpoint_id": aggregated.GuardEndpointID, "chunk_total": aggregated.ChunkTotal,
			"latency_ms": aggregated.LatencyMS, "stage": snapshot.Stage, "status": "allowed",
		}))
	}
	return decision, nil
}

func (g *GuardEvaluator) failOpen(ctx context.Context, cfg ActiveConfig, snapshot PromptSnapshot, err error, latency time.Duration) *PromptDecision {
	failureCode, reason := failOpenFailure(err)
	result := &NormalizedResult{
		Decision: EventFlag, RiskLevel: RiskLow, Action: ActionWarn, Safety: "AuditUnavailable",
		Categories: []string{"audit_unavailable"}, MatchedScanners: []string{"audit_unavailable"},
		ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{"audit_unavailable": reason},
		ScannerBackend: "sync_fail_open", ScannerVersion: "1", PolicyID: "fail_open", PolicyVersion: 1,
		ChunkTotal: 1, LatencyMS: int(latency.Milliseconds()),
	}
	decision := &PromptDecision{
		Kind: DecisionAllow, Result: result, AllowNextStage: true,
		AuditFailedOpen: true, FailureCode: failureCode,
	}
	if g != nil && g.repo != nil {
		if _, recordErr := g.repo.RecordBlocking(ctx, snapshot.Redacted(), cfg.ConfigVersion, result, false); recordErr != nil {
			if g.metrics != nil {
				g.metrics.IncRecordFailed()
			}
			LogWarn(EventResultRecordFailed, mergeLogFields(snapshotLogFields(snapshot), map[string]any{
				"decision": DecisionAllow, "error_code": "result_record_failed", "stage": snapshot.Stage,
				"status": "failed", "audit_failed_open": true,
			}))
		}
	}
	LogWarn(EventGuardAllowed, mergeLogFields(snapshotLogFields(snapshot), map[string]any{
		"decision": DecisionAllow, "risk_level": RiskLow, "action": ActionWarn,
		"latency_ms": result.LatencyMS, "stage": snapshot.Stage, "status": "fail_open",
		"audit_failed_open": true, "failure_code": failureCode,
	}))
	return decision
}

func failOpenFailure(err error) (string, string) {
	var guardErr *GuardError
	if errors.As(err, &guardErr) {
		if guardErr.Timeout || errors.Is(guardErr.Cause, context.DeadlineExceeded) {
			return "timeout", "同步审核超时，已放行并进入异步补审"
		}
		if guardErr.HTTPStatus == 401 || guardErr.HTTPStatus == 403 {
			return "authentication_failed", "同步审核认证失败，已放行并进入异步补审"
		}
		if guardErr.Code == ErrorCodeInvalidResponse {
			return "invalid_response", "同步审核响应无效，已放行并进入异步补审"
		}
	}
	return "unavailable", "同步审核节点不可用，已放行并进入异步补审"
}

func headTailRunes(value string, limit int) (string, bool) {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value, false
	}
	head := limit / 2
	tail := limit - head
	return string(runes[:head]) + string(runes[len(runes)-tail:]), true
}

func logGuardFailure(snapshot PromptSnapshot, cfg ActiveConfig, kind DecisionKind, code, guardEndpointID string, latency time.Duration) {
	fields := snapshotLogFields(snapshot)
	fields["config_version"] = cfg.ConfigVersion
	LogWarn(EventGuardFailed, mergeLogFields(fields, map[string]any{
		"decision": kind, "guard_endpoint_id": guardEndpointID, "latency_ms": latency.Milliseconds(),
		"status": "failed", "error_code": code, "upstream_dispatched": false, "billing_preconsumed": false,
	}))
}

func (g *GuardEvaluator) scanChunk(ctx context.Context, cfg ActiveConfig, endpoints []ActiveEndpoint, chunk string) (*NormalizedResult, error) {
	var lastErr error
	for index, endpoint := range endpoints {
		attemptStarted := g.clock.Now()
		semaphore := g.nodeSemaphore(endpoint.ID)
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			return nil, &GuardError{Code: ErrorCodeUnavailable, Retryable: true, Timeout: errors.Is(ctx.Err(), context.DeadlineExceeded), Cause: ctx.Err()}
		default:
			if g.metrics != nil {
				g.metrics.IncBulkheadFull()
				g.observeEndpointBulkhead(endpoint.ID)
			}
			lastErr = &GuardError{Code: ErrorCodeUnavailable, Retryable: true}
			if index < len(endpoints)-1 && g.metrics != nil {
				g.metrics.IncFailover()
				g.observeEndpointFailover(endpoint.ID)
			}
			continue
		}
		result, err := callPromptScannerSafely(ctx, g.scanner, endpoint, chunk, cfg.Scanners, cfg.CustomPromptEnabled, cfg.CustomSystemPrompt, cfg.CustomPromptMaxTokens, cfg.CustomPromptFlagThreshold, cfg.CustomPromptBlockThreshold)
		<-semaphore
		if err == nil && result != nil {
			g.observeEndpoint(endpoint.ID, decisionKindForResult(result), g.clock.Now().Sub(attemptStarted))
			return result, nil
		}
		if err == nil {
			err = &GuardError{Code: ErrorCodeInvalidResponse, Retryable: false}
		}
		lastErr = err
		kind := DecisionUnavailable
		if guardErrorCode(err) == ErrorCodeInvalidResponse {
			kind = DecisionInvalid
		}
		g.observeEndpoint(endpoint.ID, kind, g.clock.Now().Sub(attemptStarted))
		var guardErr *GuardError
		if errors.As(err, &guardErr) && guardErr.Timeout {
			g.observeEndpointTimeout(endpoint.ID)
		}
		if guardErr == nil || !guardErr.Retryable {
			return nil, err
		}
		if index < len(endpoints)-1 && g.metrics != nil {
			g.metrics.IncFailover()
			g.observeEndpointFailover(endpoint.ID)
		}
	}
	if lastErr == nil {
		lastErr = &GuardError{Code: ErrorCodeUnavailable}
	}
	return nil, lastErr
}

func (g *GuardEvaluator) observeEndpoint(endpointID string, kind DecisionKind, latency time.Duration) {
	if endpointMetrics, ok := g.metrics.(EndpointMetrics); ok {
		endpointMetrics.ObserveEndpoint(endpointID, kind, latency)
	}
}

func (g *GuardEvaluator) observeEndpointFailover(endpointID string) {
	if endpointMetrics, ok := g.metrics.(EndpointMetrics); ok {
		endpointMetrics.IncEndpointFailover(endpointID)
	}
}

func (g *GuardEvaluator) observeEndpointBulkhead(endpointID string) {
	if endpointMetrics, ok := g.metrics.(EndpointMetrics); ok {
		endpointMetrics.IncEndpointBulkheadFull(endpointID)
	}
}

func (g *GuardEvaluator) observeEndpointTimeout(endpointID string) {
	if endpointMetrics, ok := g.metrics.(EndpointMetrics); ok {
		endpointMetrics.IncEndpointTimeout(endpointID)
	}
}

func callPromptScanner(ctx context.Context, scanner PromptScanner, endpoint ActiveEndpoint, chunk string, scanners []string, customPromptEnabled bool, systemPrompt string, maxTokens int, flagThreshold, blockThreshold float64) (result *NormalizedResult, err error) {
	if customPromptEnabled || endpoint.RequestMode == RequestModeModerations {
		customScanner, ok := scanner.(CustomPromptScanner)
		if !ok {
			return nil, &GuardError{Code: ErrorCodeUnavailable}
		}
		result, err = customScanner.ScanWithPrompt(ctx, endpoint, chunk, scanners, systemPrompt, maxTokens)
		if err == nil && result != nil && customPromptEnabled && endpoint.RequestMode != RequestModeModerations {
			applyCustomPromptThresholds(result, flagThreshold, blockThreshold)
		}
		return result, err
	}
	return scanner.Scan(ctx, endpoint, chunk, scanners)
}

func callPromptScannerSafely(ctx context.Context, scanner PromptScanner, endpoint ActiveEndpoint, chunk string, scanners []string, customPromptEnabled bool, systemPrompt string, maxTokens int, flagThreshold, blockThreshold float64) (result *NormalizedResult, err error) {
	defer func() {
		if recover() != nil {
			result = nil
			err = &GuardError{Code: ErrorCodeUnavailable, Retryable: false}
		}
	}()
	return callPromptScanner(ctx, scanner, endpoint, chunk, scanners, customPromptEnabled, systemPrompt, maxTokens, flagThreshold, blockThreshold)
}

func applyCustomPromptThresholds(result *NormalizedResult, flagThreshold, blockThreshold float64) {
	if result == nil {
		return
	}
	confidence, ok := result.ScannerScores["custom_prompt"]
	if !ok {
		return
	}
	result.Safety = "Safe"
	result.Categories = []string{}
	result.MatchedScanners = []string{}
	result.Decision, result.RiskLevel, result.Action = EventPass, RiskLow, ActionAllow
	if confidence >= blockThreshold {
		result.Safety = "Unsafe"
		result.Categories = []string{"custom_prompt"}
		result.MatchedScanners = []string{"custom_prompt"}
		result.Decision, result.RiskLevel, result.Action = EventCritical, RiskCritical, ActionBlock
		return
	}
	if confidence >= flagThreshold {
		result.Safety = "Controversial"
		result.Categories = []string{"custom_prompt"}
		result.MatchedScanners = []string{"custom_prompt"}
		result.Decision, result.RiskLevel, result.Action = EventFlag, RiskMedium, ActionWarn
	}
}

func (g *GuardEvaluator) nodeSemaphore(id string) chan struct{} {
	g.nodeMu.Lock()
	defer g.nodeMu.Unlock()
	semaphore := g.nodes[id]
	if semaphore == nil {
		semaphore = make(chan struct{}, g.perNodeLimit)
		g.nodes[id] = semaphore
	}
	return semaphore
}

func minimumInputLimit(endpoints []ActiveEndpoint) int {
	limit := DefaultInputLimit
	for index, endpoint := range endpoints {
		value := endpoint.InputLimit
		if value <= 0 {
			value = DefaultInputLimit
		}
		if index == 0 || value < limit {
			limit = value
		}
	}
	return limit
}

func guardErrorCode(err error) string {
	var guardErr *GuardError
	if errors.As(err, &guardErr) && guardErr.Code != "" {
		return guardErr.Code
	}
	return ErrorCodeUnavailable
}

func pointerLogID(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
