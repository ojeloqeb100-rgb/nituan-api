package sora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVideoResolution(t *testing.T) {
	tests := []struct {
		value string
		want  string
		ok    bool
	}{
		{"480P", billing_setting.VideoResolution480P, true},
		{" 854 X 480 ", billing_setting.VideoResolution480P, true},
		{"480*832", billing_setting.VideoResolution480P, true},
		{"1280×720", billing_setting.VideoResolution720P, true},
		{"720 x 1280", billing_setting.VideoResolution720P, true},
		{"1080P", billing_setting.VideoResolution1080P, true},
		{"1024x1792", billing_setting.VideoResolution1080P, true},
		{"2k", "", false},
		{"2048x1080", "", false},
		{"hd", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, ok := NormalizeVideoResolution(tt.value)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func loadVideoBillingConfig(t *testing.T, pricesJSON string) {
	t.Helper()
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
		"billing_setting.video_prices": pricesJSON,
	}))
}

func TestValidateVideoBillingRequest(t *testing.T) {
	loadVideoBillingConfig(t, `{"video-model":{"480p":0.5,"720p":0.8}}`)

	tests := []struct {
		name    string
		seconds string
		size    string
		wantErr bool
	}{
		{name: "defaults are accepted"},
		{name: "numeric seconds", seconds: "10", size: "1280x720"},
		{name: "priced 480p", seconds: "4", size: "854x480"},
		{name: "zero seconds", seconds: "0", size: "720p", wantErr: true},
		{name: "negative seconds", seconds: "-1", size: "720p", wantErr: true},
		{name: "non numeric seconds", seconds: "abc", size: "720p", wantErr: true},
		{name: "duration too long", seconds: "3601", size: "720p", wantErr: true},
		{name: "unknown size", seconds: "4", size: "2k", wantErr: true},
		{name: "unpriced 1080p", seconds: "4", size: "1920x1080", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Set("task_request", relaycommon.TaskSubmitReq{
				Model:   "video-model",
				Seconds: tt.seconds,
				Size:    tt.size,
			})
			taskErr := validateVideoBillingRequest(ctx, &relaycommon.RelayInfo{
				OriginModelName: "video-model",
				TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
			})
			if tt.wantErr {
				require.NotNil(t, taskErr)
				require.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
				return
			}
			require.Nil(t, taskErr)
		})
	}

	t.Run("default size fails when 720p is not priced", func(t *testing.T) {
		loadVideoBillingConfig(t, `{"video-model":{"480p":0.5}}`)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Set("task_request", relaycommon.TaskSubmitReq{
			Model: "video-model",
		})
		taskErr := validateVideoBillingRequest(ctx, &relaycommon.RelayInfo{
			OriginModelName: "video-model",
			TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		})
		require.NotNil(t, taskErr)
		require.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	})
}

func TestEstimatePerCallPriceRejectsUnpricedResolution(t *testing.T) {
	loadVideoBillingConfig(t, `{"video-model":{"480p":0.5,"720p":0.8}}`)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "video-model",
		Size:  "1920x1080",
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	price, taskErr := (&TaskAdaptor{}).EstimatePerCallPrice(ctx, info)
	require.NotNil(t, taskErr)
	require.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	require.Equal(t, 0.0, price)
}

func TestEstimatePerCallPriceReusesVideoSnapshot(t *testing.T) {
	loadVideoBillingConfig(t, `{"video-model":{"480p":0.5,"720p":0.8,"1080p":1.2}}`)

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}
	info.Action = constant.TaskActionRemix
	info.BillingMode = billing_setting.BillingModeVideo
	info.PriceData = hosttypes.PriceData{UsePrice: true, ModelPrice: 0.65}

	price, taskErr := (&TaskAdaptor{}).EstimatePerCallPrice(nil, info)
	require.Nil(t, taskErr)
	require.Equal(t, 0.65, price)
}

func TestEstimateBillingIncludesVideoSeconds(t *testing.T) {
	loadVideoBillingConfig(t, `{"video-model":{"480p":0.5,"720p":0.8}}`)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Seconds: "10",
		Size:    "1280x720",
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	ratios := (&TaskAdaptor{}).EstimateBilling(ctx, info)
	require.Equal(t, map[string]float64{"seconds": 10}, ratios)
}

func TestSoraBuildRequestBodyMapsOpenAICompatibleFields(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		wantDuration float64
		wantRatio    string
	}{
		{
			name:         "numeric seconds and landscape 720 become duration and 16:9",
			payload:      `{"model":"m","prompt":"p","seconds":4,"size":"1280x720"}`,
			wantDuration: 4,
			wantRatio:    "16:9",
		},
		{
			name:         "string seconds and portrait 720 become duration and 9:16",
			payload:      `{"model":"m","prompt":"p","seconds":"8","size":"720x1280"}`,
			wantDuration: 8,
			wantRatio:    "9:16",
		},
		{
			name:         "480 landscape maps to 16:9",
			payload:      `{"model":"m","prompt":"p","seconds":4,"size":"854x480"}`,
			wantDuration: 4,
			wantRatio:    "16:9",
		},
		{
			name:         "1080 portrait maps to 9:16",
			payload:      `{"model":"m","prompt":"p","seconds":4,"size":"1080x1920"}`,
			wantDuration: 4,
			wantRatio:    "9:16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader([]byte(tt.payload)))
			c.Request.Header.Set("Content-Type", "application/json")
			defer common.CleanupBodyStorage(c)

			var req relaycommon.TaskSubmitReq
			require.NoError(t, common.Unmarshal([]byte(tt.payload), &req))
			c.Set("task_request", req)

			adaptor := &TaskAdaptor{}
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       constant.ChannelTypeOpenAI,
					UpstreamModelName: "upstream-model",
				},
			}
			adaptor.Init(info)

			body, err := adaptor.BuildRequestBody(c, info)
			require.NoError(t, err)
			sent, err := io.ReadAll(body)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, common.Unmarshal(sent, &got))
			require.Equal(t, "upstream-model", got["model"])
			require.Equal(t, "p", got["prompt"])
			require.Equal(t, tt.wantDuration, got["duration"])
			require.Equal(t, tt.wantRatio, got["aspect_ratio"])
			_, hasSeconds := got["seconds"]
			_, hasSize := got["size"]
			require.False(t, hasSeconds)
			require.False(t, hasSize)
		})
	}
}

func TestSoraBuildRequestBodyKeepsOfficialSoraSeconds(t *testing.T) {
	payload := `{"model":"m","prompt":"p","seconds":4,"size":"720x1280"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeSora,
			UpstreamModelName: "upstream-model",
		},
	}
	adaptor.Init(info)

	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	sent, err := io.ReadAll(body)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(sent, &got))
	require.Equal(t, "4", got["seconds"])
	require.Equal(t, "720x1280", got["size"])
	_, hasDuration := got["duration"]
	_, hasRatio := got["aspect_ratio"]
	require.False(t, hasDuration)
	require.False(t, hasRatio)
}

func TestSoraBuildRequestBodyRejectsUnknownSize(t *testing.T) {
	payload := `{"model":"m","prompt":"p","seconds":4,"size":"2k"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "m", Prompt: "p", Seconds: "4", Size: "2k"})

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "upstream-model",
		},
	}
	adaptor.Init(info)

	_, err := adaptor.BuildRequestBody(c, info)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported video size")
}

func TestSoraDoResponseExtractsWrappedTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":[{"task_id":"up_123"}]}`))),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2.0-mini",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}

	taskID, raw, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "up_123", taskID)
	require.Contains(t, string(raw), "up_123")

	var got map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "task_public", got["id"])
	require.Equal(t, "task_public", got["task_id"])
}

func TestSoraDoResponseExtractsOpenAITaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"video_abc","object":"video","status":"queued"}`))),
	}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}

	taskID, _, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "video_abc", taskID)

	var got map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "task_public", got["id"])
	require.Equal(t, "queued", got["status"])
}

func TestSoraParseTaskResultMapsWrappedCompletedVideo(t *testing.T) {
	body := []byte(`{"data":{"status":"completed","progress":100,"result":{"videos":[{"url":["https://cdn.example/v.mp4"]}]}}}`)
	got, err := (&TaskAdaptor{}).ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), got.Status)
	require.Equal(t, "https://cdn.example/v.mp4", got.Url)
}

func TestSoraParseTaskResultReadsMetadataURL(t *testing.T) {
	body := []byte(`{"id":"task_upstream","object":"video","status":"completed","progress":100,"metadata":{"url":"https://cdn.example/v1/videos/task_upstream/content?expires=1&sign=abc"}}`)
	got, err := (&TaskAdaptor{}).ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), got.Status)
	require.Equal(t, "https://cdn.example/v1/videos/task_upstream/content?expires=1&sign=abc", got.Url)
}

func TestSoraParseTaskResultMapsSucceededStatus(t *testing.T) {
	body := []byte(`{"id":"video_abc","status":"succeeded"}`)
	got, err := (&TaskAdaptor{}).ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), got.Status)
}

func TestSoraConvertToOpenAIVideoMapsWrappedResult(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_public",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Data:     []byte(`{"data":{"status":"completed","result":{"videos":[{"url":["https://cdn.example/v.mp4"]}]}}}`),
	}
	raw, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(raw, &got))
	require.Equal(t, "task_public", got["id"])
	require.Equal(t, "completed", got["status"])
	metadata, ok := got["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "/v1/videos/task_public/content", metadata["url"])
	require.NotContains(t, metadata["url"], "cdn.example")
	require.NotContains(t, metadata["url"], "localhost")
}

func TestSoraConvertToOpenAIVideoRewritesSignedMetadataURL(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Status: model.TaskStatusSuccess,
		Data:   []byte(`{"id":"task_upstream","object":"video","status":"completed","metadata":{"url":"https://cdn.example/v1/videos/task_upstream/content?expires=1&sign=abc"}}`),
	}
	raw, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(raw, &got))
	require.Equal(t, "task_public", got["id"])
	metadata, ok := got["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "/v1/videos/task_public/content", metadata["url"])
	require.NotContains(t, string(raw), "cdn.example")
}

func TestSoraBuildRequestBodyReturnsReplayablePassThroughBody(t *testing.T) {
	payload := []byte("opaque-sora-request-body")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/octet-stream")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	replayable, ok := body.(common.ReplayableBody)
	require.True(t, ok)

	sent, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, sent)
	assert.EqualValues(t, len(payload), replayable.Size())

	replayBody, err := replayable.NewReader()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)
}
