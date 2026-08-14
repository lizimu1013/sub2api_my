package securityaudit

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultPromptDecisionCacheTTL      = 5 * time.Minute
	defaultPromptFailOpenCacheTTL      = time.Minute
	defaultPromptDecisionCacheMaxItems = 8192
)

type promptDecisionCacheEntry struct {
	decision  *PromptDecision
	expiresAt time.Time
}

type promptDecisionLoadResult struct {
	decision *PromptDecision
	cacheHit bool
}

type promptDecisionCache struct {
	mu          sync.RWMutex
	items       map[string]promptDecisionCacheEntry
	flight      singleflight.Group
	successTTL  time.Duration
	failOpenTTL time.Duration
	maxItems    int
	now         func() time.Time
}

func newPromptDecisionCache(successTTL, failOpenTTL time.Duration, maxItems int) *promptDecisionCache {
	if successTTL <= 0 {
		successTTL = defaultPromptDecisionCacheTTL
	}
	if failOpenTTL <= 0 {
		failOpenTTL = defaultPromptFailOpenCacheTTL
	}
	if maxItems <= 0 {
		maxItems = defaultPromptDecisionCacheMaxItems
	}
	return &promptDecisionCache{
		items:       make(map[string]promptDecisionCacheEntry),
		successTTL:  successTTL,
		failOpenTTL: failOpenTTL,
		maxItems:    maxItems,
		now:         time.Now,
	}
}

func promptDecisionCacheKey(cfg ActiveConfig, snapshot PromptSnapshot) string {
	if snapshot.PromptHash == "" {
		return ""
	}
	groupID := int64(0)
	if snapshot.GroupID != nil {
		groupID = *snapshot.GroupID
	}
	parts := []string{
		strconv.FormatInt(cfg.ConfigVersion, 10),
		strconv.FormatInt(snapshot.UserID, 10),
		strconv.FormatInt(snapshot.APIKeyID, 10),
		strconv.FormatInt(groupID, 10),
		strings.TrimSpace(snapshot.Stage),
		snapshot.PromptHash,
	}
	return strings.Join(parts, ":")
}

func (c *promptDecisionCache) GetOrLoad(
	ctx context.Context,
	key string,
	load func() (*PromptDecision, error),
) (*PromptDecision, bool, error) {
	if load == nil {
		return nil, false, nil
	}
	if decision, ok := c.get(key); ok {
		return decision, true, nil
	}
	if c == nil || key == "" {
		decision, err := load()
		return clonePromptDecision(decision), false, err
	}

	loadStarted := false
	resultCh := c.flight.DoChan(key, func() (any, error) {
		if decision, ok := c.get(key); ok {
			return promptDecisionLoadResult{decision: decision, cacheHit: true}, nil
		}
		loadStarted = true
		decision, err := load()
		if err != nil {
			return nil, err
		}
		c.set(key, decision)
		return promptDecisionLoadResult{decision: clonePromptDecision(decision)}, nil
	})

	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return nil, false, result.Err
		}
		loaded, ok := result.Val.(promptDecisionLoadResult)
		if !ok {
			return nil, false, nil
		}
		reused := loaded.cacheHit || (result.Shared && !loadStarted)
		return clonePromptDecision(loaded.decision), reused, nil
	}
}

func (c *promptDecisionCache) get(key string) (*PromptDecision, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	now := c.now()
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !now.Before(entry.expiresAt) {
		c.mu.Lock()
		if current, exists := c.items[key]; exists && !now.Before(current.expiresAt) {
			delete(c.items, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	return clonePromptDecision(entry.decision), true
}

func (c *promptDecisionCache) set(key string, decision *PromptDecision) {
	if c == nil || key == "" || decision == nil || decision.FailureCode == "request_canceled" {
		return
	}
	ttl := c.successTTL
	if decision.AuditFailedOpen {
		ttl = c.failOpenTTL
	}
	if ttl <= 0 {
		return
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = make(map[string]promptDecisionCacheEntry)
	}
	if len(c.items) >= c.maxItems {
		checked := 0
		for cachedKey, entry := range c.items {
			if !now.Before(entry.expiresAt) {
				delete(c.items, cachedKey)
			}
			checked++
			if checked >= 64 {
				break
			}
		}
	}
	if len(c.items) >= c.maxItems {
		for cachedKey := range c.items {
			delete(c.items, cachedKey)
			break
		}
	}
	c.items[key] = promptDecisionCacheEntry{
		decision:  clonePromptDecision(decision),
		expiresAt: now.Add(ttl),
	}
}

func clonePromptDecision(decision *PromptDecision) *PromptDecision {
	if decision == nil {
		return nil
	}
	cloned := *decision
	if decision.FallbackGroupID != nil {
		id := *decision.FallbackGroupID
		cloned.FallbackGroupID = &id
	}
	cloned.Result = cloneNormalizedResult(decision.Result)
	return &cloned
}

func cloneNormalizedResult(result *NormalizedResult) *NormalizedResult {
	if result == nil {
		return nil
	}
	cloned := *result
	cloned.Categories = append([]string(nil), result.Categories...)
	cloned.MatchedScanners = append([]string(nil), result.MatchedScanners...)
	cloned.UnknownCategories = append([]string(nil), result.UnknownCategories...)
	cloned.ScannerScores = cloneStringFloatMap(result.ScannerScores)
	cloned.ScannerEvidence = cloneStringStringMap(result.ScannerEvidence)
	cloned.ScannerEvidenceChunks = cloneStringIntMap(result.ScannerEvidenceChunks)
	return &cloned
}

func cloneStringFloatMap(values map[string]float64) map[string]float64 {
	if values == nil {
		return nil
	}
	result := make(map[string]float64, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneStringStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneStringIntMap(values map[string]int) map[string]int {
	if values == nil {
		return nil
	}
	result := make(map[string]int, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
