package relay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertLocalSignedContentURL(t *testing.T, got, taskID, host string) {
	t.Helper()
	require.Contains(t, got, "/v1/videos/"+taskID+"/content?")
	require.Contains(t, got, "expires=")
	require.Contains(t, got, "sign=")
	require.NotContains(t, got, "migeapi.com")
	if host != "" {
		require.Contains(t, got, host)
	}
	parsed, err := url.Parse(got)
	if !strings.Contains(got, "://") {
		parsed, err = url.Parse("https://example.invalid" + got)
	}
	require.NoError(t, err)
	require.True(t, taskcommon.VerifyVideoContentSign(taskID, parsed.Query().Get("expires"), parsed.Query().Get("sign")))
}

func TestApplyOriginTaskRemixBillingLegacySecondsAndSize(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantSeconds float64
		wantSize    float64
		wantRes     string
		wantErr     bool
	}{
		{
			name:        "numeric seconds 10 is billed as 10",
			data:        `{"seconds":10,"size":"720x1280"}`,
			wantSeconds: 10,
			wantSize:    1,
			wantRes:     billing_setting.VideoResolution720P,
		},
		{
			name:        "string seconds 10 still works",
			data:        `{"seconds":"10","size":"720x1280"}`,
			wantSeconds: 10,
			wantSize:    1,
			wantRes:     billing_setting.VideoResolution720P,
		},
		{
			name:    "missing seconds and duration is rejected",
			data:    `{}`,
			wantErr: true,
		},
		{
			name:    "unparseable seconds is rejected",
			data:    `{"seconds":"abc"}`,
			wantErr: true,
		},
		{
			name:        "unparseable seconds falls back to duration",
			data:        `{"seconds":"abc","duration":8}`,
			wantSeconds: 8,
			wantSize:    1,
		},
		{
			name:    "fractional seconds is rejected",
			data:    `{"seconds":4.5}`,
			wantErr: true,
		},
		{
			name:    "seconds 0 is rejected",
			data:    `{"seconds":0}`,
			wantErr: true,
		},
		{
			name:    "seconds -1 is rejected",
			data:    `{"seconds":-1}`,
			wantErr: true,
		},
		{
			name:    "seconds 3601 is rejected",
			data:    `{"seconds":3601}`,
			wantErr: true,
		},
		{
			name:    "duration 0 is rejected when seconds is absent",
			data:    `{"duration":0}`,
			wantErr: true,
		},
		{
			name:    "duration 3601 is rejected when seconds is absent",
			data:    `{"duration":3601}`,
			wantErr: true,
		},
		{
			name:        "seconds 1 is accepted",
			data:        `{"seconds":1}`,
			wantSeconds: 1,
			wantSize:    1,
		},
		{
			name:        "seconds 3600 is accepted",
			data:        `{"seconds":3600}`,
			wantSeconds: 3600,
			wantSize:    1,
		},
		{
			name:        "numeric duration is used when seconds is absent",
			data:        `{"duration":10}`,
			wantSeconds: 10,
			wantSize:    1,
		},
		{
			name:        "1920x1080 does not get legacy 1080P size markup",
			data:        `{"seconds":4,"size":"1920x1080"}`,
			wantSeconds: 4,
			wantSize:    1,
			wantRes:     billing_setting.VideoResolution1080P,
		},
		{
			name:        "1080p does not get legacy 1080P size markup",
			data:        `{"seconds":4,"size":"1080p"}`,
			wantSeconds: 4,
			wantSize:    1,
			wantRes:     billing_setting.VideoResolution1080P,
		},
		{
			name:        "1080x1920 does not get legacy 1080P size markup",
			data:        `{"seconds":4,"size":"1080x1920"}`,
			wantSeconds: 4,
			wantSize:    1,
			wantRes:     billing_setting.VideoResolution1080P,
		},
		{
			name:        "1792x1024 still gets legacy size markup",
			data:        `{"seconds":4,"size":"1792x1024"}`,
			wantSeconds: 4,
			wantSize:    1.666667,
			wantRes:     billing_setting.VideoResolution1080P,
		},
		{
			name:        "1024x1792 still gets legacy size markup",
			data:        `{"seconds":4,"size":"1024x1792"}`,
			wantSeconds: 4,
			wantSize:    1.666667,
			wantRes:     billing_setting.VideoResolution1080P,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			taskErr := applyOriginTaskRemixBilling(info, &model.Task{Data: []byte(tt.data)})
			if tt.wantErr {
				require.NotNil(t, taskErr)
				assert.Equal(t, "invalid_seconds", taskErr.Code)
				assert.Nil(t, info.PriceData.OtherRatios())
				return
			}
			require.Nil(t, taskErr)
			ratios := info.PriceData.OtherRatios()
			require.NotNil(t, ratios)
			assert.Equal(t, tt.wantSeconds, ratios["seconds"])
			assert.Equal(t, tt.wantSize, ratios["size"])
			assert.Equal(t, tt.wantRes, info.VideoResolution)
		})
	}
}

func TestParseTaskDurationValueAcceptsJSONNumber(t *testing.T) {
	seconds, ok := parseTaskDurationValue(json.Number("10"))
	require.True(t, ok)
	assert.Equal(t, 10, seconds)

	seconds, ok = parseTaskDurationValue(json.Number("10.0"))
	require.True(t, ok)
	assert.Equal(t, 10, seconds)

	_, ok = parseTaskDurationValue(json.Number("4.5"))
	assert.False(t, ok)
}

func TestApplyOriginTaskRemixBillingReusesVideoSnapshot(t *testing.T) {
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	origin := &model.Task{
		Data: []byte(`{"seconds":4,"size":"1920x1080"}`),
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.65,
				PerCallBilling:  true,
				BillingMode:     billing_setting.BillingModeVideo,
				VideoResolution: billing_setting.VideoResolution720P,
				OtherRatios:     map[string]float64{"seconds": 10},
			},
		},
	}

	taskErr := applyOriginTaskRemixBilling(info, origin)
	require.Nil(t, taskErr)
	assert.Equal(t, 0.65, info.PriceData.ModelPrice)
	assert.True(t, info.PriceData.UsePrice)
	assert.Equal(t, billing_setting.BillingModeVideo, info.BillingMode)
	assert.Equal(t, billing_setting.VideoResolution720P, info.VideoResolution)
	assert.Equal(t, map[string]float64{"seconds": 10}, info.PriceData.OtherRatios())
}

func TestTaskModel2DtoHidesUpstreamVideoURL(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Platform:   "openai",
		Action:     constant.TaskActionTextGenerate,
		Status:     model.TaskStatusSuccess,
		FailReason: "https://api.migeapi.com/signed/v.mp4",
		Data:       []byte(`{"data":{"result":{"videos":[{"url":"https://api.migeapi.com/signed/v.mp4"}]}}}`),
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://api.migeapi.com/signed/v.mp4",
		},
	}

	got := TaskModel2Dto(task)
	require.NotNil(t, got)
	assert.Empty(t, got.FailReason)
	assertLocalSignedContentURL(t, got.ResultURL, "task_public", "")
	assert.NotContains(t, string(got.Data), "migeapi.com")
	assert.NotContains(t, got.FailReason, "migeapi.com")
	assert.NotContains(t, got.ResultURL, "migeapi.com")

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "/v1/videos/task_public/content")
	assert.NotContains(t, string(raw), "api.migeapi.com")
}

func TestTaskModel2DtoRedactsSignedContentURL(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_public",
		Platform: "openai",
		Action:   constant.TaskActionTextGenerate,
		Status:   model.TaskStatusSuccess,
		Data:     []byte(`{"metadata":{"url":"https://cdn.example/v1/videos/task_upstream/content?expires=1787508561&sign=abc"}}`),
		PrivateData: model.TaskPrivateData{
			ResultURL: "http://localhost:3000/v1/videos/task_public/content",
		},
	}

	got := TaskModel2Dto(task)
	require.NotNil(t, got)
	assertLocalSignedContentURL(t, got.ResultURL, "task_public", "")
	assert.NotContains(t, string(got.Data), "cdn.example")
	assert.NotContains(t, string(got.Data), "sign=abc")
}

func TestTaskModel2DtoHistoricalSuccessRewritesDetails(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_old",
		Platform: "openai",
		Action:   constant.TaskActionTextGenerate,
		Status:   model.TaskStatusSuccess,
		Data:     []byte(`{"metadata":{"url":"https://api.migeapi.com/old.mp4"}}`),
	}

	got := TaskModel2Dto(task)
	require.NotNil(t, got)
	assert.Empty(t, got.FailReason)
	assertLocalSignedContentURL(t, got.ResultURL, "task_old", "")
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "/v1/videos/task_old/content")
	assert.NotContains(t, string(raw), "api.migeapi.com")
}

func TestTaskModel2DtoForRequestUsesRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/task/", nil)
	c.Request.Host = "video.example.com"
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	task := &model.Task{
		TaskID:   "task_public",
		Platform: "openai",
		Action:   constant.TaskActionTextGenerate,
		Status:   model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://api.migeapi.com/signed/v.mp4",
		},
	}

	got := TaskModel2DtoForRequest(c, task)
	require.NotNil(t, got)
	assert.Empty(t, got.FailReason)
	assertLocalSignedContentURL(t, got.ResultURL, "task_public", "https://video.example.com")
	assert.NotContains(t, got.ResultURL, "migeapi.com")
	assert.NotContains(t, got.ResultURL, "127.0.0.1")
}

func TestSanitizeUserVideoAPIResponseStripsUpstreamPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_public", nil)
	c.Request.Host = "video.example.com"

	task := &model.Task{
		TaskID: "task_public",
		Action: constant.TaskActionTextGenerate,
		Status: model.TaskStatusSuccess,
	}
	raw := []byte(`{"id":"task_upstream","cost":1.2,"credits":3,"data":{"url":"https://api.migeapi.com/v.mp4"},"metadata":{"url":"https://api.migeapi.com/v.mp4"},"object":"video","status":"completed"}`)
	got := sanitizeUserVideoAPIResponse(c, raw, task)
	require.NotContains(t, string(got), "migeapi.com")
	require.NotContains(t, string(got), `"cost"`)
	require.NotContains(t, string(got), `"credits"`)
	require.Contains(t, string(got), `"task_public"`)
	require.Contains(t, string(got), "/v1/videos/task_public/content")
}

func TestTaskModel2DtoInProgressHasNoDetailsLink(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Action: constant.TaskActionTextGenerate,
		Status: model.TaskStatusInProgress,
	}

	got := TaskModel2Dto(task)
	require.NotNil(t, got)
	assert.Empty(t, got.FailReason)
	assert.Empty(t, got.ResultURL)
}

func TestTaskModel2DtoKeepsFailureReason(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Action:     constant.TaskActionTextGenerate,
		Status:     model.TaskStatusFailure,
		FailReason: "upstream rejected the prompt",
	}

	got := TaskModel2Dto(task)
	require.NotNil(t, got)
	assert.Empty(t, got.ResultURL)
	assert.Equal(t, "upstream rejected the prompt", got.FailReason)
}
