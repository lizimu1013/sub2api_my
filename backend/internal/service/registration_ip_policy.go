package service

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// NormalizeRegistrationIPBlacklist normalizes and validates registration IP blacklist items.
func NormalizeRegistrationIPBlacklist(raw []string) ([]string, error) {
	return normalizeRegistrationIPBlacklist(raw, true)
}

// ParseRegistrationIPBlacklist parses persisted JSON into normalized IP blacklist items.
// Invalid entries are ignored to keep old misconfigurations from breaking runtime reads.
func ParseRegistrationIPBlacklist(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	normalized, _ := normalizeRegistrationIPBlacklist(items, false)
	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

// IsRegistrationIPBlocked checks whether the client IP matches a registration IP blacklist item.
func IsRegistrationIPBlocked(clientIP string, blacklist []string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil || len(blacklist) == 0 {
		return false
	}
	for _, item := range blacklist {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			_, cidr, err := net.ParseCIDR(item)
			if err == nil && cidr != nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		blockedIP := net.ParseIP(item)
		if blockedIP != nil && blockedIP.Equal(ip) {
			return true
		}
	}
	return false
}

func normalizeRegistrationIPBlacklist(raw []string, strict bool) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		normalized, err := normalizeRegistrationIPBlacklistItem(item)
		if err != nil {
			if strict {
				return nil, err
			}
			continue
		}
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeRegistrationIPBlacklistItem(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.Contains(value, "/") {
		_, cidr, err := net.ParseCIDR(value)
		if err != nil || cidr == nil {
			return "", fmt.Errorf("invalid registration ip blacklist item: %q", raw)
		}
		return cidr.String(), nil
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return "", fmt.Errorf("invalid registration ip blacklist item: %q", raw)
	}
	return ip.String(), nil
}
