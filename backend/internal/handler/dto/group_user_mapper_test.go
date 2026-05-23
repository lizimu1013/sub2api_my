package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupFromServiceUser_UsesDisplayMultiplier(t *testing.T) {
	src := &service.Group{
		ID:                    1,
		Name:                  "user-visible",
		RateMultiplier:        8.5,
		DisplayRateMultiplier: 1.25,
		Status:                service.StatusActive,
	}

	out := GroupFromServiceUser(src)
	require.NotNil(t, out)
	require.Equal(t, 1.25, out.RateMultiplier)
	require.Equal(t, 1.25, out.DisplayRateMultiplier)
}

func TestAPIKeyFromServiceUser_UsesDisplayMultiplierForNestedGroup(t *testing.T) {
	src := &service.APIKey{
		ID:     1,
		UserID: 2,
		Key:    "sk-user-display-rate",
		Name:   "User Display Rate",
		Status: service.StatusActive,
		Group: &service.Group{
			ID:                    10,
			Name:                  "group",
			RateMultiplier:        12,
			DisplayRateMultiplier: 2,
			Status:                service.StatusActive,
		},
	}

	out := APIKeyFromServiceUser(src)
	require.NotNil(t, out)
	require.NotNil(t, out.Group)
	require.Equal(t, 2.0, out.Group.RateMultiplier)
	require.Equal(t, 2.0, out.Group.DisplayRateMultiplier)
}
