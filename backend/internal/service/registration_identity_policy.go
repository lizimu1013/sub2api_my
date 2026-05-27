package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const maxRegistrationIdentityBlacklistItemLength = 512

// NormalizeRegistrationIdentityBlacklist normalizes and validates OAuth registration identity blacklist items.
func NormalizeRegistrationIdentityBlacklist(raw []string) ([]string, error) {
	return normalizeRegistrationIdentityBlacklist(raw, true)
}

// ParseRegistrationIdentityBlacklist parses persisted JSON into normalized identity blacklist items.
// Invalid entries are ignored to keep old misconfigurations from breaking runtime reads.
func ParseRegistrationIdentityBlacklist(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	normalized, _ := normalizeRegistrationIdentityBlacklist(items, false)
	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

// IsRegistrationIdentityBlocked checks whether a provider identity is blocked from creating a new account.
// Items can be a raw provider subject ("12345") or provider-scoped ("linuxdo:12345", "oidc:abc").
func IsRegistrationIdentityBlocked(identity PendingAuthIdentityKey, blacklist []string) bool {
	subject := strings.TrimSpace(identity.ProviderSubject)
	if subject == "" || len(blacklist) == 0 {
		return false
	}
	providerType := strings.ToLower(strings.TrimSpace(identity.ProviderType))
	providerKey := strings.ToLower(strings.TrimSpace(identity.ProviderKey))
	for _, item := range blacklist {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if provider, blockedSubject, ok := strings.Cut(item, ":"); ok {
			provider = strings.ToLower(strings.TrimSpace(provider))
			blockedSubject = strings.TrimSpace(blockedSubject)
			if blockedSubject == "" {
				continue
			}
			if provider != providerType && provider != providerKey {
				continue
			}
			if blockedSubject == subject || strings.EqualFold(blockedSubject, subject) {
				return true
			}
			continue
		}
		if item == subject || strings.EqualFold(item, subject) {
			return true
		}
	}
	return false
}

func (s *AuthService) validateRegistrationIdentityPolicy(ctx context.Context, identity PendingAuthIdentityKey) error {
	if s == nil || s.settingService == nil {
		return nil
	}
	if IsRegistrationIdentityBlocked(identity, s.settingService.GetRegistrationIdentityBlacklist(ctx)) {
		return ErrRegistrationIDBlocked
	}
	return nil
}

func normalizeRegistrationIdentityBlacklist(raw []string, strict bool) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		normalized, err := normalizeRegistrationIdentityBlacklistItem(item)
		if err != nil {
			if strict {
				return nil, err
			}
			continue
		}
		if normalized == "" {
			continue
		}
		seenKey := strings.ToLower(normalized)
		if _, ok := seen[seenKey]; ok {
			continue
		}
		seen[seenKey] = struct{}{}
		out = append(out, normalized)
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeRegistrationIdentityBlacklistItem(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if len(value) > maxRegistrationIdentityBlacklistItemLength {
		return "", fmt.Errorf("registration identity blacklist item is too long: %q", raw)
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", fmt.Errorf("invalid registration identity blacklist item: %q", raw)
		}
	}
	if provider, subject, ok := strings.Cut(value, ":"); ok {
		provider = strings.ToLower(strings.TrimSpace(provider))
		subject = strings.TrimSpace(subject)
		if provider == "" || subject == "" || strings.Contains(provider, ":") {
			return "", fmt.Errorf("invalid registration identity blacklist item: %q", raw)
		}
		return provider + ":" + subject, nil
	}
	return value, nil
}
