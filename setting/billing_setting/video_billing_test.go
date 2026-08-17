package billing_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/require"
)

func TestValidateVideoPricesJSON(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
		"billing_setting.video_prices": `{}`,
	}))

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:  "valid three tiers",
			value: `{"video-model":{"480p":0.5,"720p":0.8,"1080p":1.2}}`,
		},
		{
			name:  "valid two tiers",
			value: `{"video-model":{"480p":0.5,"720p":0.8}}`,
		},
		{
			name:  "valid single tier",
			value: `{"video-model":{"720p":0.8}}`,
		},
		{
			name:    "empty tiers",
			value:   `{"video-model":{}}`,
			wantErr: true,
		},
		{
			name:    "zero price",
			value:   `{"video-model":{"480p":0,"720p":0.8,"1080p":1.2}}`,
			wantErr: true,
		},
		{
			name:    "negative price",
			value:   `{"video-model":{"480p":0.5,"720p":-0.8}}`,
			wantErr: true,
		},
		{
			name:    "unknown tier",
			value:   `{"video-model":{"480p":0.5,"720p":0.8,"1080p":1.2,"2k":2}}`,
			wantErr: true,
		},
		{name: "invalid json", value: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVideoPricesJSON(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateBillingModesRequiresVideoPrices(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
		"billing_setting.video_prices": `{}`,
	}))

	require.Error(t, ValidateBillingModesJSON(`{"video-model":"video"}`))
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.video_prices": `{"video-model":{"480p":0.5,"720p":0.8}}`,
	}))
	require.NoError(t, ValidateBillingModesJSON(`{"video-model":"video"}`))
}

func TestGetVideoPriceRejectsUnconfiguredTier(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"video-model":"video"}`,
		"billing_setting.video_prices": `{"video-model":{"480p":0.5,"720p":0.8}}`,
	}))

	prices, ok := GetVideoPrices("video-model")
	require.True(t, ok)
	require.Equal(t, 0.5, prices.P480)
	require.Equal(t, 0.8, prices.P720)
	require.Equal(t, 0.0, prices.P1080)

	price, ok := GetVideoPrice("video-model", VideoResolution720P)
	require.True(t, ok)
	require.Equal(t, 0.8, price)

	_, ok = GetVideoPrice("video-model", VideoResolution1080P)
	require.False(t, ok)
}

func TestPreferredPriceFallsBackToConfiguredTier(t *testing.T) {
	tests := []struct {
		name      string
		prices    VideoResolutionPrices
		preferred string
		wantRes   string
		wantPrice float64
		ok        bool
	}{
		{
			name:      "prefers 720p when priced",
			prices:    VideoResolutionPrices{P480: 0.31, P720: 0.52},
			preferred: VideoResolution720P,
			wantRes:   VideoResolution720P,
			wantPrice: 0.52,
			ok:        true,
		},
		{
			name:      "falls back to 480p when 720p is missing",
			prices:    VideoResolutionPrices{P480: 0.31},
			preferred: VideoResolution720P,
			wantRes:   VideoResolution480P,
			wantPrice: 0.31,
			ok:        true,
		},
		{
			name:      "falls back to 1080p when only that tier is priced",
			prices:    VideoResolutionPrices{P1080: 1.2},
			preferred: VideoResolution720P,
			wantRes:   VideoResolution1080P,
			wantPrice: 1.2,
			ok:        true,
		},
		{
			name:      "empty prices are not configured",
			preferred: VideoResolution720P,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, price, ok := tt.prices.PreferredPrice(tt.preferred)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.wantRes, resolution)
			require.Equal(t, tt.wantPrice, price)
		})
	}
}

func TestVideoResolutionRequestSize(t *testing.T) {
	require.Equal(t, "854x480", VideoResolutionRequestSize(VideoResolution480P))
	require.Equal(t, "720x1280", VideoResolutionRequestSize(VideoResolution720P))
	require.Equal(t, "1920x1080", VideoResolutionRequestSize(VideoResolution1080P))
	require.Equal(t, "720x1280", VideoResolutionRequestSize(""))
}
