//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRegistrationEmailSuffixWhitelist(t *testing.T) {
	got, err := NormalizeRegistrationEmailSuffixWhitelist([]string{"example.com", "@EXAMPLE.COM", " @foo.bar "})
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, got)
}

func TestNormalizeRegistrationEmailSuffixWhitelist_Invalid(t *testing.T) {
	_, err := NormalizeRegistrationEmailSuffixWhitelist([]string{"@invalid_domain"})
	require.Error(t, err)
}

func TestParseRegistrationEmailSuffixWhitelist(t *testing.T) {
	got := ParseRegistrationEmailSuffixWhitelist(`["example.com","@foo.bar","@invalid_domain"]`)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, got)
}

func TestIsRegistrationEmailSuffixAllowed(t *testing.T) {
	require.True(t, IsRegistrationEmailSuffixAllowed("user@example.com", []string{"@example.com"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@sub.example.com", []string{"@example.com"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@any.com", []string{}))
}

func TestNormalizeRegistrationEmailBlacklist(t *testing.T) {
	got, err := NormalizeRegistrationEmailBlacklist([]string{
		"User@Example.COM",
		"example.com",
		"@FOO.bar",
		"user@example.com",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"user@example.com", "@example.com", "@foo.bar"}, got)
}

func TestNormalizeRegistrationEmailBlacklist_Invalid(t *testing.T) {
	_, err := NormalizeRegistrationEmailBlacklist([]string{"bad@@example.com"})
	require.Error(t, err)
}

func TestParseRegistrationEmailBlacklist(t *testing.T) {
	got := ParseRegistrationEmailBlacklist(`["User@Example.COM","@foo.bar","bad@@example.com"]`)
	require.Equal(t, []string{"user@example.com", "@foo.bar"}, got)
}

func TestIsRegistrationEmailBlocked(t *testing.T) {
	blacklist := []string{"blocked@example.com", "@evil.com"}
	require.True(t, IsRegistrationEmailBlocked("Blocked@Example.com", blacklist))
	require.True(t, IsRegistrationEmailBlocked("user@evil.com", blacklist))
	require.False(t, IsRegistrationEmailBlocked("user@sub.evil.com", blacklist))
	require.False(t, IsRegistrationEmailBlocked("user@example.com", blacklist))
}
