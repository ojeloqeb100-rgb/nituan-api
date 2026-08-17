package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIsTaskProxyContentURLTreatsSignedLocalPathAsLocal(t *testing.T) {
	t.Parallel()

	assert.True(t, isTaskProxyContentURL("https://video.example/v1/videos/task_public/content?expires=1&sign=abc", "task_public"))
	assert.True(t, isTaskProxyContentURL("http://localhost:3000/v1/videos/task_public/content", "task_public"))
	assert.False(t, isTaskProxyContentURL("https://cdn.example/v1/videos/task_upstream/content?expires=1&sign=abc", "task_public"))
}

func TestVideoProxyExpiredSignatureIsGone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_public/content?expires=1&sign=abc", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "task_public"}}
	c.Set(videoContentSignedContextKey, true)

	VideoProxy(c)

	assert.Equal(t, http.StatusGone, w.Code)
	assert.Contains(t, w.Body.String(), "expired")
	assert.Empty(t, w.Header().Get("Location"))
	assert.NotContains(t, w.Body.String(), "migeapi.com")
}

func TestVideoProxyGoneMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)

	assert.Equal(t, http.StatusGone, w.Code)
	assert.Contains(t, w.Body.String(), "expired")
	assert.NotContains(t, w.Body.String(), "migeapi.com")
	assert.Empty(t, w.Header().Get("Location"))
}

func TestGetResultURLStillReadsHistoricalUpstream(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID: "task_public",
		PrivateData: model.TaskPrivateData{
			ResultURL: "http://localhost:3000/v1/videos/task_public/content?expires=1&sign=abc",
		},
		Data: json.RawMessage(`{"metadata":{"url":"https://cdn.example/v1/videos/task_upstream/content?expires=1787508561&sign=abc"}}`),
	}
	assert.Equal(t, "https://cdn.example/v1/videos/task_upstream/content?expires=1787508561&sign=abc", task.GetResultURL())
	assert.True(t, isTaskProxyContentURL(task.PrivateData.ResultURL, "task_public"))
}
