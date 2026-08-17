package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResultURLPrefersStoredUpstreamThenData(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID: "task_public",
		PrivateData: TaskPrivateData{
			ResultURL: "https://api.migeapi.com/signed/v.mp4",
		},
		Data: json.RawMessage(`{"data":{"result":{"videos":[{"url":["https://cdn.example/other.mp4"]}]}}}`),
	}
	assert.Equal(t, "https://api.migeapi.com/signed/v.mp4", task.GetResultURL())
}

func TestGetResultURLReadsWrappedTaskData(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID: "task_public",
		Data:   json.RawMessage(`{"data":{"status":"completed","result":{"videos":[{"url":["https://cdn.example/v.mp4"]}]}}}`),
	}
	assert.Equal(t, "https://cdn.example/v.mp4", task.GetResultURL())
}

func TestGetResultURLReadsMetadataURL(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID: "task_public",
		Data:   json.RawMessage(`{"metadata":{"url":"https://cdn.example/meta.mp4"}}`),
	}
	assert.Equal(t, "https://cdn.example/meta.mp4", task.GetResultURL())
}

func TestIsSignedVideoContentURL(t *testing.T) {
	t.Parallel()

	assert.True(t, IsSignedVideoContentURL("https://cdn.example/v1/videos/task_upstream/content?expires=1787508561&sign=abc"))
	assert.False(t, IsSignedVideoContentURL("http://localhost:3000/v1/videos/task_public/content"))
	assert.False(t, IsSignedVideoContentURL("https://cdn.example/v.mp4"))
}

func TestIsLocalVideoContentURLIgnoresSignedQuery(t *testing.T) {
	t.Parallel()

	assert.True(t, IsLocalVideoContentURL("https://video.example/v1/videos/task_public/content?expires=1&sign=abc", "task_public"))
	assert.True(t, IsLocalVideoContentURL("/v1/videos/task_public/content?expires=1&sign=abc", "task_public"))
	assert.False(t, IsLocalVideoContentURL("https://cdn.example/v1/videos/task_upstream/content?expires=1&sign=abc", "task_public"))
}

func TestGetResultURLReadsSignedUpstreamContentFromMetadata(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID: "task_public",
		PrivateData: TaskPrivateData{
			ResultURL: "http://localhost:3000/v1/videos/task_public/content",
		},
		Data: json.RawMessage(`{"id":"task_upstream","metadata":{"url":"https://cdn.example/v1/videos/task_upstream/content?expires=1787508561&sign=abc"},"object":"video","status":"completed"}`),
	}
	assert.Equal(t, "https://cdn.example/v1/videos/task_upstream/content?expires=1787508561&sign=abc", task.GetResultURL())
}

func TestGetResultURLSkipsLocalContentPlaceholder(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID: "task_public",
		PrivateData: TaskPrivateData{
			ResultURL: "https://relay.example/v1/videos/task_public/content",
		},
		Data: json.RawMessage(`{"data":{"result":{"videos":[{"url":"https://cdn.example/from-data.mp4"}]}}}`),
	}
	assert.Equal(t, "https://cdn.example/from-data.mp4", task.GetResultURL())
}

func TestGetResultURLFallsBackToLegacyFailReason(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID:     "task_public",
		FailReason: "https://cdn.example/legacy.mp4",
	}
	assert.Equal(t, "https://cdn.example/legacy.mp4", task.GetResultURL())
}

func TestToOpenAIVideoUsesLocalContentURL(t *testing.T) {
	t.Parallel()

	task := &Task{
		TaskID: "task_public",
		Status: TaskStatusSuccess,
		PrivateData: TaskPrivateData{
			ResultURL: "https://api.migeapi.com/signed/v.mp4",
		},
	}
	got := task.ToOpenAIVideo()
	require.NotNil(t, got)
	url, _ := got.Metadata["url"].(string)
	assert.Equal(t, "/v1/videos/task_public/content", url)
	assert.NotContains(t, url, "migeapi.com")
}
