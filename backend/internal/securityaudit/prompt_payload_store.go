package securityaudit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const promptAuditPayloadFormat = "sub2api_prompt_audit_payload/v1"

type promptAuditPayload struct {
	Format                  string `json:"format"`
	ScanText                string `json:"scan_text"`
	LatestUserInput         string `json:"latest_user_input"`
	PreviousAssistantOutput string `json:"previous_assistant_output,omitempty"`
}

func encodePromptAuditPayload(snapshot PromptSnapshot) (string, error) {
	payload := promptAuditPayload{
		Format: promptAuditPayloadFormat, ScanText: snapshot.ScanText,
		LatestUserInput:         snapshot.LatestUserInput,
		PreviousAssistantOutput: snapshot.PreviousAssistantOutput,
	}
	if payload.ScanText == "" || payload.LatestUserInput == "" {
		return "", fmt.Errorf("prompt audit payload input invalid")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode prompt audit payload: %w", err)
	}
	return string(raw), nil
}

func decodePromptAuditPayload(raw string) (promptAuditPayload, error) {
	var payload promptAuditPayload
	if json.Unmarshal([]byte(raw), &payload) == nil && payload.Format == promptAuditPayloadFormat {
		if payload.ScanText == "" || payload.LatestUserInput == "" {
			return promptAuditPayload{}, fmt.Errorf("prompt audit payload input invalid")
		}
		return payload, nil
	}
	// Compatibility with queued payloads written before the versioned envelope.
	latestUserInput := latestPriorityScanSegment(raw)
	if latestUserInput == "" {
		return promptAuditPayload{}, fmt.Errorf("prompt audit payload input invalid")
	}
	return promptAuditPayload{
		Format: promptAuditPayloadFormat, ScanText: latestUserInput,
		LatestUserInput: FullPromptFromScanText(latestUserInput),
	}, nil
}

type PayloadStore interface {
	Set(ctx context.Context, jobID int64, scanText string, ttl time.Duration) error
	Get(ctx context.Context, jobID int64) (string, error)
	Delete(ctx context.Context, jobID int64) error
	Ping(ctx context.Context) error
}

type RedisPayloadStore struct {
	client *redis.Client
}

func NewRedisPayloadStore(client *redis.Client) *RedisPayloadStore {
	return &RedisPayloadStore{client: client}
}

func (s *RedisPayloadStore) Set(ctx context.Context, jobID int64, scanText string, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit payload store unavailable")
	}
	if jobID <= 0 || scanText == "" {
		return fmt.Errorf("prompt audit payload input invalid")
	}
	if ttl <= 0 || ttl > DefaultPayloadTTL {
		ttl = DefaultPayloadTTL
	}
	return s.client.Set(ctx, payloadKey(jobID), scanText, ttl).Err()
}

func (s *RedisPayloadStore) Get(ctx context.Context, jobID int64) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("prompt audit payload store unavailable")
	}
	return s.client.Get(ctx, payloadKey(jobID)).Result()
}

func (s *RedisPayloadStore) Delete(ctx context.Context, jobID int64) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit payload store unavailable")
	}
	return s.client.Del(ctx, payloadKey(jobID)).Err()
}

func (s *RedisPayloadStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit payload store unavailable")
	}
	return s.client.Ping(ctx).Err()
}

func payloadKey(jobID int64) string {
	return PayloadKeyPrefix + strconv.FormatInt(jobID, 10)
}
