package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type lowBalanceSettingRepoStub struct {
	values  map[string]string
	updates map[string]string
}

func (s *lowBalanceSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *lowBalanceSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *lowBalanceSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *lowBalanceSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *lowBalanceSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.updates[key] = value
	}
	return nil
}

func (s *lowBalanceSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *lowBalanceSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingService_UpdateSettings_WritesLowBalanceDisplayRateThreshold(t *testing.T) {
	repo := &lowBalanceSettingRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		LowBalanceDisplayRateThreshold: 1.75,
	})

	require.NoError(t, err)
	require.Equal(t, "1.75000000", repo.updates[SettingKeyLowBalanceDisplayRateThreshold])
}

func TestSettingService_GetLowBalanceDisplayRateThreshold_DefaultsToTwo(t *testing.T) {
	svc := NewSettingService(&lowBalanceSettingRepoStub{values: map[string]string{}}, &config.Config{})

	require.Equal(t, LowBalanceDisplayRateThresholdDefault, svc.GetLowBalanceDisplayRateThreshold(context.Background()))
}

func TestSettingService_GetLowBalanceDisplayRateThreshold_ReadsConfiguredValue(t *testing.T) {
	svc := NewSettingService(&lowBalanceSettingRepoStub{values: map[string]string{
		SettingKeyLowBalanceDisplayRateThreshold: "1.75",
	}}, &config.Config{})

	require.Equal(t, 1.75, svc.GetLowBalanceDisplayRateThreshold(context.Background()))
}

func TestSettingService_GetLowBalanceDisplayRateThreshold_NormalizesNegativeToZero(t *testing.T) {
	svc := NewSettingService(&lowBalanceSettingRepoStub{values: map[string]string{
		SettingKeyLowBalanceDisplayRateThreshold: "-3",
	}}, &config.Config{})

	require.Zero(t, svc.GetLowBalanceDisplayRateThreshold(context.Background()))
}
