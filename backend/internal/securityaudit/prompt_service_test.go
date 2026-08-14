package securityaudit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type staticSettingRepository struct {
	values map[string]string
}

type customPromptScannerFunc func(context.Context, ActiveEndpoint, string) (*NormalizedResult, error)

func (f customPromptScannerFunc) Scan(ctx context.Context, endpoint ActiveEndpoint, chunk string, _ []string) (*NormalizedResult, error) {
	return f(ctx, endpoint, chunk)
}

func (f customPromptScannerFunc) ScanWithPrompt(ctx context.Context, endpoint ActiveEndpoint, chunk string, _ []string, _ string, _ int) (*NormalizedResult, error) {
	return f(ctx, endpoint, chunk)
}

func (r staticSettingRepository) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (r staticSettingRepository) GetValue(context.Context, string) (string, error) {
	return "", service.ErrSettingNotFound
}
func (r staticSettingRepository) Set(context.Context, string, string) error { return nil }
func (r staticSettingRepository) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = r.values[key]
	}
	return result, nil
}
func (r staticSettingRepository) SetMultiple(context.Context, map[string]string) error { return nil }
func (r staticSettingRepository) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r staticSettingRepository) Delete(context.Context, string) error { return nil }

func TestPromptServiceHasExplicitIdempotentLifecycle(t *testing.T) {
	config := NewConfigManager(nil, staticSettingRepository{values: map[string]string{
		SettingKeyPromptAuditConfig: "",
		SettingKeyRiskControl:       "false",
	}}, nil, prefixEncryptor{}, testTotpKeyConfig())
	service := NewPromptService(
		config,
		NewPostgreSQLRepository(nil),
		NewRedisPayloadStore(nil),
		NewOpenAICompatibleScanner(),
		NewAtomicMetrics(),
	)

	require.Nil(t, service.cancel, "construction must not start background work")
	require.NoError(t, service.Start(context.Background()))
	require.NotNil(t, service.cancel)
	require.NoError(t, service.Start(context.Background()), "Start must be idempotent")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, service.Shutdown(ctx))
	require.Nil(t, service.cancel)
	require.NoError(t, service.Shutdown(ctx), "Shutdown must be idempotent")
}

func TestPromptServiceStartReportsDependencyFailureWithoutPanic(t *testing.T) {
	service := &PromptService{}
	require.Error(t, service.Start(context.Background()))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, service.Shutdown(ctx))
}

func TestPromptServiceBlockingLatestTurnOnlyUsesNarrowSnapshot(t *testing.T) {
	seen := make([]string, 0, 2)
	evaluator := newGuardEvaluator(PromptScannerFunc(func(_ context.Context, _ ActiveEndpoint, chunk string, _ []string) (*NormalizedResult, error) {
		seen = append(seen, chunk)
		return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
	}), nil, NewAtomicMetrics(), 2, 2)
	service := &PromptService{
		config: &fakeConfigStore{active: true, cfg: ActiveConfig{
			RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, BlockingLatestTurnOnly: true, AllGroups: true,
			Scanners: AllScannerIDs, Endpoints: []ActiveEndpoint{{ID: "guard-1", Enabled: true, TimeoutMS: 1000, InputLimit: 4096}},
		}},
		evaluator: evaluator,
	}
	decision, err := service.Evaluate(context.Background(), Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"system","content":"system instruction"},{"role":"user","content":"older user input"},{"role":"assistant","content":"previous output"},{"role":"user","content":"latest user input"}]}`)})
	require.NoError(t, err)
	require.Equal(t, DecisionAllow, decision.Kind)
	require.Equal(t, []string{formatSynchronousAuditInput("latest user input", "previous output")}, seen)
}

func TestPromptServiceReusesRepeatedBlockingDecision(t *testing.T) {
	var scannerCalls atomic.Int64
	scanner := PromptScannerFunc(func(_ context.Context, _ ActiveEndpoint, _ string, _ []string) (*NormalizedResult, error) {
		scannerCalls.Add(1)
		return &NormalizedResult{
			Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock,
			ScannerScores:   map[string]float64{"custom_prompt": 0.95},
			ScannerEvidence: map[string]string{"custom_prompt": "blocked once"},
		}, nil
	})
	groupID := int64(2)
	cfg := ActiveConfig{
		ConfigVersion: 30, RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true,
		BlockHTTPStatus: 403, BlockMessage: "blocked",
		Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 1000, InputLimit: 4096}},
	}
	config := &fakeConfigStore{cfg: cfg, active: true}
	repo := &fakeJobRepository{}
	service := &PromptService{
		config: config, evaluator: newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2),
		decisionCache: newPromptDecisionCache(time.Minute, time.Second, 32),
	}
	request := Request{
		RequestID: "first", UserID: 46, APIKeyID: 37, GroupID: &groupID,
		Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"user","content":"same prompt"}]}`),
	}

	first, err := service.Evaluate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, DecisionBlock, first.Kind)
	first.Result.ScannerEvidence["custom_prompt"] = "caller mutation"

	request.RequestID = "retry-with-new-request-id"
	second, err := service.Evaluate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, DecisionBlock, second.Kind)
	require.Equal(t, "blocked once", second.Result.ScannerEvidence["custom_prompt"])
	require.Equal(t, int64(1), scannerCalls.Load())
	require.Equal(t, 1, repo.recordBlockingCalls)

	config.cfg.ConfigVersion++
	request.RequestID = "after-config-change"
	_, err = service.Evaluate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, int64(2), scannerCalls.Load(), "a policy version change must bypass the cache")
}

func TestPromptServiceCoalescesConcurrentRepeatedAudit(t *testing.T) {
	var scannerCalls atomic.Int64
	started := make(chan struct{})
	release := make(chan struct{})
	scanner := PromptScannerFunc(func(_ context.Context, _ ActiveEndpoint, _ string, _ []string) (*NormalizedResult, error) {
		if scannerCalls.Add(1) == 1 {
			close(started)
		}
		<-release
		return &NormalizedResult{
			Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow,
			ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		}, nil
	})
	cfg := ActiveConfig{
		ConfigVersion: 30, RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true,
		Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 1000, InputLimit: 4096}},
	}
	service := &PromptService{
		config:        &fakeConfigStore{cfg: cfg, active: true},
		evaluator:     newGuardEvaluator(scanner, nil, NewAtomicMetrics(), 2, 2),
		decisionCache: newPromptDecisionCache(time.Minute, time.Second, 32),
	}
	request := Request{
		UserID: 46, APIKeyID: 37, Protocol: "openai_chat_completions",
		Body: []byte(`{"messages":[{"role":"user","content":"same concurrent prompt"}]}`),
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, requestID := range []string{"one", "two"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			copy := request
			copy.RequestID = requestID
			_, err := service.Evaluate(context.Background(), copy)
			errs <- err
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int64(1), scannerCalls.Load())
}

func TestPromptServiceCanceledWaiterRecordsCancellationAndCachesBackgroundResult(t *testing.T) {
	var scannerCalls atomic.Int64
	started := make(chan struct{})
	release := make(chan struct{})
	scanner := PromptScannerFunc(func(_ context.Context, _ ActiveEndpoint, _ string, _ []string) (*NormalizedResult, error) {
		if scannerCalls.Add(1) == 1 {
			close(started)
		}
		<-release
		return &NormalizedResult{
			Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow,
			ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		}, nil
	})
	repo := &fakeJobRepository{}
	cfg := ActiveConfig{
		ConfigVersion: 31, RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true,
		Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 1000, InputLimit: 4096}},
	}
	service := &PromptService{
		config:        &fakeConfigStore{cfg: cfg, active: true},
		evaluator:     newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2),
		background:    context.Background(),
		decisionCache: newPromptDecisionCache(time.Minute, time.Second, 32),
	}
	request := Request{
		RequestID: "canceled-first-attempt", UserID: 46, APIKeyID: 37,
		Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"user","content":"same prompt"}]}`),
	}

	type evaluationResult struct {
		decision *PromptDecision
		err      error
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan evaluationResult, 1)
	go func() {
		decision, err := service.Evaluate(ctx, request)
		resultCh <- evaluationResult{decision: decision, err: err}
	}()
	<-started
	cancel()
	canceled := <-resultCh
	require.Nil(t, canceled.decision)
	require.ErrorIs(t, canceled.err, context.Canceled)

	repo.mu.Lock()
	require.Len(t, repo.recordBlockingResults, 1)
	require.Equal(t, []string{"request_canceled"}, repo.recordBlockingResults[0].Categories)
	require.Equal(t, "request_context", repo.recordBlockingResults[0].ScannerBackend)
	repo.mu.Unlock()

	close(release)
	request.RequestID = "retry-after-cancel"
	reused, err := service.Evaluate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, DecisionAllow, reused.Kind)
	require.Equal(t, int64(1), scannerCalls.Load(), "the detached first audit must populate the decision cache")

	repo.mu.Lock()
	require.Len(t, repo.recordBlockingResults, 2)
	require.Equal(t, EventPass, repo.recordBlockingResults[1].Decision)
	repo.mu.Unlock()
}

func TestPromptServiceOversizeFollowupUsesConfidenceBandAndFullSnapshot(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		confidence    float64
		wantKind      DecisionKind
		wantFollowup  bool
		wantSyncInput string
	}{
		{name: "low confidence does not enqueue", input: "abcdefghij", confidence: 0.39, wantKind: DecisionAllow, wantSyncInput: "abchij"},
		{name: "flagged oversize enqueues full input", input: "abcdefghij", confidence: 0.50, wantKind: DecisionFlag, wantFollowup: true, wantSyncInput: "abchij"},
		{name: "blocked oversize does not enqueue", input: "abcdefghij", confidence: 0.70, wantKind: DecisionBlock, wantSyncInput: "abchij"},
		{name: "flagged short input does not enqueue", input: "abc", confidence: 0.50, wantKind: DecisionFlag, wantSyncInput: "abc"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var syncInputs []string
			scanner := customPromptScannerFunc(func(_ context.Context, _ ActiveEndpoint, chunk string) (*NormalizedResult, error) {
				syncInputs = append(syncInputs, chunk)
				return &NormalizedResult{
					ScannerScores:   map[string]float64{"custom_prompt": test.confidence},
					ScannerEvidence: map[string]string{"custom_prompt": "test"},
				}, nil
			})
			cfg := ActiveConfig{
				RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true,
				CustomPromptEnabled: true, CustomPromptFlagThreshold: 0.4, CustomPromptBlockThreshold: 0.7,
				BlockHTTPStatus: 403, BlockMessage: "blocked", WorkerCount: 1, QueueCapacity: 8,
				Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 1000, InputLimit: 6}},
			}
			config := &fakeConfigStore{cfg: cfg, active: true}
			repo := &fakeJobRepository{}
			payload := &fakePayloadStore{values: map[int64]string{}}
			service := &PromptService{
				config: config, repo: nil, payload: nil,
				enqueuer:   NewEnqueuer(config, repo, payload),
				evaluator:  newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2),
				background: context.Background(), enqueueSlots: make(chan struct{}, 4),
			}
			body, err := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "user", "content": test.input}}})
			require.NoError(t, err)

			decision, err := service.Evaluate(context.Background(), Request{RequestID: "followup-test", Protocol: "openai_chat_completions", Body: body})
			require.NoError(t, err)
			require.Equal(t, test.wantKind, decision.Kind)
			require.Equal(t, []string{test.wantSyncInput}, syncInputs)
			service.enqueueWG.Wait()

			if !test.wantFollowup {
				require.Empty(t, payload.values)
				return
			}
			require.Len(t, payload.values, 1)
			for _, raw := range payload.values {
				stored, decodeErr := decodePromptAuditPayload(raw)
				require.NoError(t, decodeErr)
				require.Equal(t, test.input, stored.ScanText)
				require.Equal(t, test.input, stored.LatestUserInput)
			}
		})
	}
}

func TestPromptServiceUnavailableFailsOpenRecordsAndEnqueuesFullFollowup(t *testing.T) {
	var scannerCalls atomic.Int64
	scanner := customPromptScannerFunc(func(context.Context, ActiveEndpoint, string) (*NormalizedResult, error) {
		scannerCalls.Add(1)
		return nil, &GuardError{Code: ErrorCodeUnavailable, Retryable: true}
	})
	cfg := ActiveConfig{
		RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true,
		CustomPromptEnabled: true, BlockingLatestTurnOnly: true, CustomPromptFlagThreshold: 0.4, CustomPromptBlockThreshold: 0.7,
		WorkerCount: 1, QueueCapacity: 8,
		Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 1000, InputLimit: 6}},
	}
	config := &fakeConfigStore{cfg: cfg, active: true}
	repo := &fakeJobRepository{}
	payload := &fakePayloadStore{values: map[int64]string{}}
	service := &PromptService{
		config:     config,
		enqueuer:   NewEnqueuer(config, repo, payload),
		evaluator:  newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2),
		background: context.Background(), enqueueSlots: make(chan struct{}, 4),
	}

	request := Request{
		RequestID: "followup-unavailable", Protocol: "openai_chat_completions",
		Body: []byte(`{"messages":[{"role":"user","content":"older input"},{"role":"assistant","content":"previous output"},{"role":"user","content":"abcdefghij"}]}`),
	}
	decision, err := service.Evaluate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, DecisionAllow, decision.Kind)
	require.True(t, decision.AllowNextStage)
	require.True(t, decision.AuditFailedOpen)
	require.Equal(t, "unavailable", decision.FailureCode)
	service.enqueueWG.Wait()
	request.RequestID = "followup-unavailable-retry"
	reused, err := service.Evaluate(context.Background(), request)
	require.NoError(t, err)
	require.True(t, reused.AuditFailedOpen)
	service.enqueueWG.Wait()
	require.Equal(t, int64(1), scannerCalls.Load())
	require.Equal(t, 1, repo.recordBlockingCalls)
	require.Equal(t, []string{"audit_unavailable"}, repo.recordBlockingResult.Categories)
	require.Empty(t, repo.recordBlockingSnapshot.ScanText)
	require.Len(t, payload.values, 1)
	for _, raw := range payload.values {
		stored, decodeErr := decodePromptAuditPayload(raw)
		require.NoError(t, decodeErr)
		require.Equal(t, "abcdefghij"+promptAuditPrioritySeparator+"previous output", stored.ScanText)
		require.Equal(t, "abcdefghij", stored.LatestUserInput)
		require.Equal(t, "previous output", stored.PreviousAssistantOutput)
	}
}

func TestPromptServiceFailOpenQueueFailureNeverBlocksRequest(t *testing.T) {
	scanner := customPromptScannerFunc(func(context.Context, ActiveEndpoint, string) (*NormalizedResult, error) {
		return nil, &GuardError{Code: ErrorCodeUnavailable, Retryable: true}
	})
	cfg := ActiveConfig{
		RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true,
		CustomPromptEnabled: true, WorkerCount: 1, QueueCapacity: 8,
		Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 1000, InputLimit: 6}},
	}
	config := &fakeConfigStore{cfg: cfg, active: true}
	repo := &fakeJobRepository{createErr: errors.New("database unavailable")}
	service := &PromptService{
		config: config, enqueuer: NewEnqueuer(config, repo, &fakePayloadStore{values: map[int64]string{}}),
		evaluator:  newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2),
		background: context.Background(), enqueueSlots: make(chan struct{}, 4),
	}
	decision, err := service.Evaluate(context.Background(), Request{
		RequestID: "queue-failure", Protocol: "openai_chat_completions",
		Body: []byte(`{"messages":[{"role":"user","content":"short"}]}`),
	})
	require.NoError(t, err)
	require.True(t, decision.AllowNextStage)
	require.True(t, decision.AuditFailedOpen)
	service.enqueueWG.Wait()
	require.Equal(t, 1, repo.recordBlockingCalls)
}

func TestPromptServiceRejectsInvalidDeleteConfirmationClaims(t *testing.T) {
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	filter := EventFilter{Decision: string(EventCritical), StartAt: &start, EndAt: &end}
	const snapshotMaxID int64 = 10
	filterHash := FilterHash(filter, snapshotMaxID)
	validClaims := deleteClaims{
		FilterHash: filterHash, SnapshotMaxID: snapshotMaxID, AdminID: 7,
		IssuedAt: now, ExpiresAt: now.Add(5 * time.Minute),
	}
	claimsToken := func(claims deleteClaims) string {
		raw, err := json.Marshal(claims)
		require.NoError(t, err)
		return string(raw)
	}
	validRequest := DeleteByFilterRequest{
		Filter: filter, SnapshotMaxID: snapshotMaxID, FilterHash: filterHash,
		ConfirmationToken: claimsToken(validClaims), Confirm: true,
	}

	tests := []struct {
		name    string
		request DeleteByFilterRequest
		adminID int64
	}{
		{name: "confirm false", request: func() DeleteByFilterRequest { value := validRequest; value.Confirm = false; return value }(), adminID: 7},
		{name: "malformed token", request: func() DeleteByFilterRequest {
			value := validRequest
			value.ConfirmationToken = "not-json"
			return value
		}(), adminID: 7},
		{name: "different administrator", request: validRequest, adminID: 8},
		{name: "filter hash mismatch", request: func() DeleteByFilterRequest {
			value := validRequest
			value.FilterHash = strings.Repeat("b", 64)
			return value
		}(), adminID: 7},
		{name: "snapshot mismatch", request: func() DeleteByFilterRequest { value := validRequest; value.SnapshotMaxID++; return value }(), adminID: 7},
		{name: "expired", request: func() DeleteByFilterRequest {
			value := validRequest
			claims := validClaims
			claims.ExpiresAt = now
			value.ConfirmationToken = claimsToken(claims)
			return value
		}(), adminID: 7},
	}

	service := &PromptService{config: &fakeConfigStore{}, clock: fixedClock{now: now}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.DeleteByFilter(context.Background(), test.request, test.adminID)
			require.Error(t, err)
			require.Nil(t, result)
		})
	}
}
