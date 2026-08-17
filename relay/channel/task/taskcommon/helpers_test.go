package taskcommon

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProxyURLIsHostRelative(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/v1/videos/task_public/content", BuildProxyURL("task_public"))
	assert.Empty(t, BuildProxyURL(" "))
	assert.NotContains(t, BuildProxyURL("task_public"), "localhost")
	assert.NotContains(t, BuildProxyURL("task_public"), "127.0.0.1")
}

func TestBuildPublicProxyURLUsesRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_public", nil)
	c.Request.Host = "video.example.com"
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	got := BuildPublicProxyURL(c, "task_public")
	assert.Equal(t, "https://video.example.com/v1/videos/task_public/content", got)
	assert.NotContains(t, got, "127.0.0.1")
	assert.NotContains(t, got, "migeapi.com")
}

func TestBuildPublicProxyURLUsesLocalhostFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_public", nil)
	c.Request.Host = "localhost:3000"

	got := BuildPublicProxyURL(c, "task_public")
	assert.Equal(t, "http://localhost:3000/v1/videos/task_public/content", got)
	assert.NotContains(t, got, "127.0.0.1")
}

func TestBuildPublicProxyURLFallsBackToServerAddress(t *testing.T) {
	previous := system_setting.ServerAddress
	system_setting.ServerAddress = "https://api.customer.example/"
	t.Cleanup(func() { system_setting.ServerAddress = previous })

	got := BuildPublicProxyURL(nil, "task_public")
	require.Equal(t, "https://api.customer.example/v1/videos/task_public/content", got)
}

func TestBuildPublicSignedProxyURLUsesRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_public", nil)
	c.Request.Host = "video.example.com"
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	expires := time.Now().Add(VideoContentTTL).Unix()
	got := BuildPublicSignedProxyURL(c, "task_public", expires)
	require.True(t, strings.HasPrefix(got, "https://video.example.com/v1/videos/task_public/content?"))
	require.Contains(t, got, "expires=")
	require.Contains(t, got, "sign=")
	require.NotContains(t, got, "migeapi.com")
	require.NotContains(t, got, "127.0.0.1")

	parsed, err := url.Parse(got)
	require.NoError(t, err)
	require.True(t, VerifyVideoContentSign("task_public", parsed.Query().Get("expires"), parsed.Query().Get("sign")))
}

func TestVerifyVideoContentSignRejectsTamperedValue(t *testing.T) {
	t.Parallel()

	expires := "4102444800"
	sign := SignVideoContent("task_public", expires)
	require.True(t, VerifyVideoContentSign("task_public", expires, sign))
	require.False(t, VerifyVideoContentSign("task_public", expires, "00"+sign[2:]))
	require.False(t, VerifyVideoContentSign("task_other", expires, sign))
}
