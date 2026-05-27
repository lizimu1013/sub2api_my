//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRegistrationEmailSuffixWhitelist(t *testing.T) {
	got, err := NormalizeRegistrationEmailSuffixWhitelist([]string{"example.com", "@EXAMPLE.COM", " @foo.bar ", "*.EDU.CN"})
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, got)
}

func TestNormalizeRegistrationEmailSuffixWhitelist_Invalid(t *testing.T) {
	for _, item := range []string{"@invalid_domain", "*.", "*", "*.@", "*.foo"} {
		t.Run(item, func(t *testing.T) {
			_, err := NormalizeRegistrationEmailSuffixWhitelist([]string{item})
			require.Error(t, err)
		})
	}
}

func TestParseRegistrationEmailSuffixWhitelist(t *testing.T) {
	got := ParseRegistrationEmailSuffixWhitelist(`["example.com","@foo.bar","*.EDU.CN","@invalid_domain","*.foo"]`)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, got)
}

func TestIsRegistrationEmailSuffixAllowed(t *testing.T) {
	require.True(t, IsRegistrationEmailSuffixAllowed("user@example.com", []string{"@example.com"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@sub.example.com", []string{"@example.com"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@qq.com", []string{"@qq.com"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@sub.qq.com", []string{"@qq.com"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("student@cs.edu.cn", []string{"*.edu.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("student@edu.cn", []string{"*.edu.cn"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("student@foo.cn", []string{"*.edu.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@a.com", []string{"@a.com", "*.b.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@school.b.cn", []string{"@a.com", "*.b.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@b.cn", []string{"@a.com", "*.b.cn"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@c.cn", []string{"@a.com", "*.b.cn"}))
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

func TestNormalizeRegistrationIdentityBlacklist(t *testing.T) {
	got, err := NormalizeRegistrationIdentityBlacklist([]string{
		" 12345 ",
		"LinuxDo:ABC",
		"linuxdo:abc",
		"oidc:user:with:colon",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"12345", "linuxdo:ABC", "oidc:user:with:colon"}, got)
}

func TestNormalizeRegistrationIdentityBlacklist_Invalid(t *testing.T) {
	for _, item := range []string{"linux do:123", "linuxdo:", ":123", "abc def"} {
		t.Run(item, func(t *testing.T) {
			_, err := NormalizeRegistrationIdentityBlacklist([]string{item})
			require.Error(t, err)
		})
	}
}

func TestParseRegistrationIdentityBlacklist(t *testing.T) {
	got := ParseRegistrationIdentityBlacklist(`["12345"," LinuxDo:ABC ","bad value",""]`)
	require.Equal(t, []string{"12345", "linuxdo:ABC"}, got)
}

func TestIsRegistrationIdentityBlocked(t *testing.T) {
	identity := PendingAuthIdentityKey{
		ProviderType:    "linuxdo",
		ProviderKey:     "linuxdo",
		ProviderSubject: "ABC123",
	}
	require.True(t, IsRegistrationIdentityBlocked(identity, []string{"ABC123"}))
	require.True(t, IsRegistrationIdentityBlocked(identity, []string{"linuxdo:abc123"}))
	require.False(t, IsRegistrationIdentityBlocked(identity, []string{"oidc:abc123"}))
	require.False(t, IsRegistrationIdentityBlocked(identity, []string{"other"}))
}

func TestNormalizeRegistrationIPBlacklist(t *testing.T) {
	got, err := NormalizeRegistrationIPBlacklist([]string{
		" 192.0.2.1 ",
		"192.0.2.0/24",
		"2001:db8::1",
		"192.0.2.1",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1", "192.0.2.0/24", "2001:db8::1"}, got)
}

func TestNormalizeRegistrationIPBlacklist_Invalid(t *testing.T) {
	for _, item := range []string{"not-an-ip", "192.0.2.0/99", "192.0.2.1/"} {
		t.Run(item, func(t *testing.T) {
			_, err := NormalizeRegistrationIPBlacklist([]string{item})
			require.Error(t, err)
		})
	}
}

func TestParseRegistrationIPBlacklist(t *testing.T) {
	got := ParseRegistrationIPBlacklist(`["192.0.2.1","198.51.100.0/24","bad"]`)
	require.Equal(t, []string{"192.0.2.1", "198.51.100.0/24"}, got)
}

func TestIsRegistrationIPBlocked(t *testing.T) {
	blacklist := []string{"192.0.2.1", "198.51.100.0/24"}
	require.True(t, IsRegistrationIPBlocked("192.0.2.1", blacklist))
	require.True(t, IsRegistrationIPBlocked("198.51.100.8", blacklist))
	require.False(t, IsRegistrationIPBlocked("203.0.113.8", blacklist))
	require.False(t, IsRegistrationIPBlocked("not-an-ip", blacklist))
}
