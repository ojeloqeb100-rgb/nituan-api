package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistTaskVideoWritesLocalFile(t *testing.T) {
	t.Setenv("VIDEO_CACHE_DIR", t.TempDir())

	task := &model.Task{TaskID: "task_local_write"}
	require.NoError(t, PersistTaskVideo(task, strings.NewReader("video-bytes"), "video/mp4"))
	require.True(t, HasUsableLocalVideo(task))
	path, ok := SafeLocalVideoPath(task)
	require.True(t, ok)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("video-bytes"), got)
	assert.Greater(t, task.PrivateData.LocalExpiresAt, time.Now().Unix())
	assert.LessOrEqual(t, task.PrivateData.LocalExpiresAt, time.Now().Add(taskcommon.VideoContentTTL+time.Minute).Unix())
}

func TestExpiredLocalVideoIsNotUsable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIDEO_CACHE_DIR", dir)

	path := filepath.Join(dir, "task_expired.mp4")
	require.NoError(t, os.WriteFile(path, []byte("stale"), 0o600))
	task := &model.Task{
		TaskID: "task_expired",
		PrivateData: model.TaskPrivateData{
			LocalPath:      path,
			LocalExpiresAt: time.Now().Add(-time.Minute).Unix(),
		},
	}
	assert.True(t, TaskLocalVideoExpired(task))
	assert.False(t, HasUsableLocalVideo(task))
}

func TestCleanupExpiredVideoFilesRemovesOldFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIDEO_CACHE_DIR", dir)
	path := filepath.Join(dir, "task_old.mp4")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))
	past := time.Now().Add(-taskcommon.VideoContentTTL - time.Hour)
	require.NoError(t, os.Chtimes(path, past, past))

	CleanupExpiredVideoFiles()
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestCacheTaskVideoUsesPlainGETForSignedURL(t *testing.T) {
	t.Setenv("VIDEO_CACHE_DIR", t.TempDir())
	InitHttpClient()

	var sawAuth bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("x-goog-api-key") != "" {
			sawAuth = true
		}
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = io.WriteString(w, "signed-cdn-bytes")
	}))
	t.Cleanup(upstream.Close)

	task := &model.Task{
		TaskID: "task_signed_cdn",
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: upstream.URL + "/v1/videos/task_upstream/content?expires=4102444800&sign=abc",
		},
	}
	require.NoError(t, CacheTaskVideo(context.Background(), task, nil))
	assert.False(t, sawAuth)
	require.True(t, HasUsableLocalVideo(task))
	path, ok := SafeLocalVideoPath(task)
	require.True(t, ok)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("signed-cdn-bytes"), got)
}

func TestEnsureAPIKeySkipsSignedCDN(t *testing.T) {
	t.Parallel()

	signed := "https://cdn.example/v1/videos/task_upstream/content?expires=4102444800&sign=abc"
	assert.Equal(t, signed, ensureAPIKey(signed, "sk-test"))
	assert.NotContains(t, ensureAPIKey(signed, "sk-test"), "key=")
}

func TestShouldAttachProviderAuthSkipsSignedURL(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: 1}
	signed := "https://cdn.example/v.mp4?expires=4102444800&sign=abc"
	assert.False(t, shouldAttachProviderAuth(channel, signed))
}
