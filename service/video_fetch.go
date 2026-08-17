package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

func CacheTaskVideo(ctx context.Context, task *model.Task, channel *model.Channel) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if TaskLocalVideoExpired(task) && task.PrivateData.LocalExpiresAt > 0 {
		return ErrVideoLinkExpired
	}
	lock := lockTaskVideo(task.TaskID)
	lock.Lock()
	defer lock.Unlock()
	if HasUsableLocalVideo(task) {
		return nil
	}

	if channel == nil && task.ChannelId > 0 {
		cached, err := model.CacheGetChannel(task.ChannelId)
		if err == nil {
			channel = cached
		}
	}

	source, contentType, err := resolveTaskVideoSource(ctx, task, channel)
	if err != nil {
		return err
	}
	defer source.Close()
	return PersistTaskVideo(task, source, contentType)
}

func resolveTaskVideoSource(ctx context.Context, task *model.Task, channel *model.Channel) (io.ReadCloser, string, error) {
	if stored := strings.TrimSpace(task.GetResultURL()); strings.HasPrefix(stored, "data:") {
		return dataURLReader(stored)
	}
	if stored := extractStoredDataURL(task); stored != "" {
		return dataURLReader(stored)
	}

	videoURL, header, client, err := resolveTaskVideoFetch(ctx, task, channel)
	if err != nil {
		return nil, "", err
	}
	if strings.HasPrefix(videoURL, "data:") {
		return dataURLReader(videoURL)
	}
	if signedVideoURLExpired(videoURL) {
		return nil, "", ErrVideoLinkExpired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, "", err
	}
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if client == nil {
		client = GetHttpClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return nil, "", ErrVideoLinkExpired
		}
		return nil, "", fmt.Errorf("upstream video fetch status %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	return resp.Body, contentType, nil
}

func resolveTaskVideoFetch(ctx context.Context, task *model.Task, channel *model.Channel) (string, http.Header, *http.Client, error) {
	header := make(http.Header)
	client, err := videoContentChannelClient(channel)
	if err != nil {
		return "", nil, nil, err
	}

	if stored := strings.TrimSpace(task.GetResultURL()); isFetchableVideoURL(stored, task.TaskID) {
		if shouldAttachProviderAuth(channel, stored) {
			attachProviderAuth(header, channel, task)
		}
		return stored, header, client, nil
	}

	if channel == nil {
		return "", nil, nil, fmt.Errorf("video source not available")
	}

	switch channel.Type {
	case constant.ChannelTypeGemini:
		apiKey := strings.TrimSpace(task.PrivateData.Key)
		if apiKey == "" {
			apiKey = strings.TrimSpace(channel.Key)
		}
		videoURL, resolveErr := getGeminiVideoURL(channel, task, apiKey)
		if resolveErr != nil {
			return "", nil, nil, resolveErr
		}
		if shouldAttachProviderAuth(channel, videoURL) && apiKey != "" {
			header.Set("x-goog-api-key", apiKey)
		}
		return videoURL, header, client, nil
	case constant.ChannelTypeVertexAi:
		videoURL, resolveErr := getVertexVideoURL(channel, task)
		if resolveErr != nil {
			return "", nil, nil, resolveErr
		}
		return videoURL, header, client, nil
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		baseURL := channel.GetBaseURL()
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		videoURL := strings.TrimRight(baseURL, "/") + "/v1/videos/" + task.GetUpstreamTaskID() + "/content"
		header.Set("Authorization", "Bearer "+channel.Key)
		return videoURL, header, client, nil
	default:
		return "", nil, nil, fmt.Errorf("video source not available")
	}
}

func isFetchableVideoURL(raw string, taskID string) bool {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "data:") {
		return false
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return false
	}
	return !model.IsLocalVideoContentURL(value, taskID)
}

func shouldAttachProviderAuth(channel *model.Channel, videoURL string) bool {
	if channel == nil || model.IsSignedVideoContentURL(videoURL) {
		return false
	}
	path := videoURL
	if parsed, err := url.Parse(videoURL); err == nil && parsed != nil {
		path = parsed.Path
	}
	switch channel.Type {
	case constant.ChannelTypeGemini:
		return strings.Contains(videoURL, "generativelanguage.googleapis.com") ||
			strings.Contains(videoURL, "aiplatform.googleapis.com")
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		return strings.Contains(path, "/v1/videos/") && strings.HasSuffix(path, "/content")
	default:
		return false
	}
}

func attachProviderAuth(header http.Header, channel *model.Channel, task *model.Task) {
	if header == nil || channel == nil {
		return
	}
	switch channel.Type {
	case constant.ChannelTypeGemini:
		apiKey := strings.TrimSpace(task.PrivateData.Key)
		if apiKey == "" {
			apiKey = strings.TrimSpace(channel.Key)
		}
		if apiKey != "" {
			header.Set("x-goog-api-key", apiKey)
		}
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		if key := strings.TrimSpace(channel.Key); key != "" {
			header.Set("Authorization", "Bearer "+key)
		}
	}
}

func videoContentChannelClient(channel *model.Channel) (*http.Client, error) {
	if channel == nil {
		return GetHttpClient(), nil
	}
	proxy := channel.GetSetting().Proxy
	if strings.TrimSpace(proxy) == "" {
		return GetHttpClient(), nil
	}
	return GetHttpClientWithProxy(proxy)
}

func signedVideoURLExpired(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return false
	}
	expires := strings.TrimSpace(parsed.Query().Get("expires"))
	if expires == "" {
		return false
	}
	ts, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || ts <= 0 {
		return false
	}
	return time.Now().Unix() >= ts
}

func extractStoredDataURL(task *model.Task) string {
	if task == nil {
		return ""
	}
	if strings.HasPrefix(strings.TrimSpace(task.PrivateData.ResultURL), "data:") {
		return strings.TrimSpace(task.PrivateData.ResultURL)
	}
	if url := extractVertexVideoURLFromTaskData(task); strings.HasPrefix(url, "data:") {
		return url
	}
	return ""
}

func dataURLReader(dataURL string) (io.ReadCloser, string, error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data url")
	}
	header := parts[0]
	payload := parts[1]
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return nil, "", fmt.Errorf("unsupported data url")
	}
	mimeType := strings.TrimPrefix(header, "data:")
	mimeType = strings.TrimSuffix(mimeType, ";base64")
	if mimeType == "" {
		mimeType = "video/mp4"
	}
	videoBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		videoBytes, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", err
		}
	}
	return io.NopCloser(bytes.NewReader(videoBytes)), mimeType, nil
}

func getGeminiVideoURL(channel *model.Channel, task *model.Task, apiKey string) (string, error) {
	if channel == nil || task == nil {
		return "", fmt.Errorf("invalid channel or task")
	}
	if url := extractGeminiVideoURLFromTaskData(task); url != "" {
		return url, nil
	}
	baseURL := constant.ChannelBaseURLs[channel.Type]
	if channel.GetBaseURL() != "" {
		baseURL = channel.GetBaseURL()
	}
	if GetTaskAdaptorFunc == nil {
		return "", fmt.Errorf("gemini task adaptor not found")
	}
	adaptor := GetTaskAdaptorFunc(constant.TaskPlatform(strconv.Itoa(channel.Type)))
	if adaptor == nil {
		return "", fmt.Errorf("gemini task adaptor not found")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API key not stored for task")
	}
	resp, err := adaptor.FetchTask(baseURL, apiKey, map[string]any{
		"task_id": task.GetUpstreamTaskID(),
		"action":  task.Action,
	}, channel.GetSetting().Proxy)
	if err != nil {
		return "", fmt.Errorf("fetch task failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read task response failed: %w", err)
	}
	taskInfo, parseErr := adaptor.ParseTaskResult(body)
	if parseErr == nil && taskInfo != nil && taskInfo.RemoteUrl != "" {
		return taskInfo.RemoteUrl, nil
	}
	if url := extractGeminiVideoURLFromPayload(body); url != "" {
		return url, nil
	}
	if parseErr != nil {
		return "", fmt.Errorf("parse task result failed: %w", parseErr)
	}
	return "", fmt.Errorf("gemini video url not found")
}

func extractGeminiVideoURLFromTaskData(task *model.Task) string {
	if task == nil || len(task.Data) == 0 {
		return ""
	}
	var payload map[string]any
	if err := common.Unmarshal(task.Data, &payload); err != nil {
		return ""
	}
	return extractGeminiVideoURLFromMap(payload)
}

func extractGeminiVideoURLFromPayload(body []byte) string {
	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return extractGeminiVideoURLFromMap(payload)
}

func extractGeminiVideoURLFromMap(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if uri, ok := payload["uri"].(string); ok && uri != "" {
		return uri
	}
	if resp, ok := payload["response"].(map[string]any); ok {
		if uri := extractGeminiVideoURLFromResponse(resp); uri != "" {
			return uri
		}
	}
	return ""
}

func extractGeminiVideoURLFromResponse(resp map[string]any) string {
	if resp == nil {
		return ""
	}
	if gvr, ok := resp["generateVideoResponse"].(map[string]any); ok {
		if uri := extractGeminiVideoURLFromGeneratedSamples(gvr); uri != "" {
			return uri
		}
	}
	if videos, ok := resp["videos"].([]any); ok {
		for _, video := range videos {
			if vm, ok := video.(map[string]any); ok {
				if uri, ok := vm["uri"].(string); ok && uri != "" {
					return uri
				}
			}
		}
	}
	if uri, ok := resp["video"].(string); ok && uri != "" {
		return uri
	}
	if uri, ok := resp["uri"].(string); ok && uri != "" {
		return uri
	}
	return ""
}

func extractGeminiVideoURLFromGeneratedSamples(gvr map[string]any) string {
	if gvr == nil {
		return ""
	}
	if samples, ok := gvr["generatedSamples"].([]any); ok {
		for _, sample := range samples {
			if sm, ok := sample.(map[string]any); ok {
				if video, ok := sm["video"].(map[string]any); ok {
					if uri, ok := video["uri"].(string); ok && uri != "" {
						return uri
					}
				}
			}
		}
	}
	return ""
}

func getVertexVideoURL(channel *model.Channel, task *model.Task) (string, error) {
	if channel == nil || task == nil {
		return "", fmt.Errorf("invalid channel or task")
	}
	if url := strings.TrimSpace(task.GetResultURL()); isFetchableVideoURL(url, task.TaskID) {
		return url, nil
	}
	if url := extractVertexVideoURLFromTaskData(task); url != "" {
		return url, nil
	}
	baseURL := constant.ChannelBaseURLs[channel.Type]
	if channel.GetBaseURL() != "" {
		baseURL = channel.GetBaseURL()
	}
	if GetTaskAdaptorFunc == nil {
		return "", fmt.Errorf("vertex task adaptor not found")
	}
	adaptor := GetTaskAdaptorFunc(constant.TaskPlatform(strconv.Itoa(channel.Type)))
	if adaptor == nil {
		return "", fmt.Errorf("vertex task adaptor not found")
	}
	key := getVertexTaskKey(channel, task)
	if key == "" {
		return "", fmt.Errorf("vertex key not available for task")
	}
	resp, err := adaptor.FetchTask(baseURL, key, map[string]any{
		"task_id": task.GetUpstreamTaskID(),
		"action":  task.Action,
	}, channel.GetSetting().Proxy)
	if err != nil {
		return "", fmt.Errorf("fetch task failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read task response failed: %w", err)
	}
	taskInfo, parseErr := adaptor.ParseTaskResult(body)
	if parseErr == nil && taskInfo != nil && strings.TrimSpace(taskInfo.Url) != "" {
		return taskInfo.Url, nil
	}
	if url := extractVertexVideoURLFromPayload(body); url != "" {
		return url, nil
	}
	if parseErr != nil {
		return "", fmt.Errorf("parse task result failed: %w", parseErr)
	}
	return "", fmt.Errorf("vertex video url not found")
}

func getVertexTaskKey(channel *model.Channel, task *model.Task) string {
	if task != nil {
		if key := strings.TrimSpace(task.PrivateData.Key); key != "" {
			return key
		}
	}
	if channel == nil {
		return ""
	}
	keys := channel.GetKeys()
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			return key
		}
	}
	return strings.TrimSpace(channel.Key)
}

func extractVertexVideoURLFromTaskData(task *model.Task) string {
	if task == nil || len(task.Data) == 0 {
		return ""
	}
	return extractVertexVideoURLFromPayload(task.Data)
}

func extractVertexVideoURLFromPayload(body []byte) string {
	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	resp, ok := payload["response"].(map[string]any)
	if !ok || resp == nil {
		return ""
	}
	if videos, ok := resp["videos"].([]any); ok && len(videos) > 0 {
		if video, ok := videos[0].(map[string]any); ok && video != nil {
			if b64, _ := video["bytesBase64Encoded"].(string); strings.TrimSpace(b64) != "" {
				mime, _ := video["mimeType"].(string)
				enc, _ := video["encoding"].(string)
				return buildVideoDataURL(mime, enc, b64)
			}
		}
	}
	if b64, _ := resp["bytesBase64Encoded"].(string); strings.TrimSpace(b64) != "" {
		enc, _ := resp["encoding"].(string)
		return buildVideoDataURL("", enc, b64)
	}
	if video, _ := resp["video"].(string); strings.TrimSpace(video) != "" {
		if strings.HasPrefix(video, "data:") || strings.HasPrefix(video, "http://") || strings.HasPrefix(video, "https://") {
			return video
		}
		enc, _ := resp["encoding"].(string)
		return buildVideoDataURL("", enc, video)
	}
	return ""
}

func buildVideoDataURL(mimeType string, encoding string, base64Data string) string {
	mime := strings.TrimSpace(mimeType)
	if mime == "" {
		enc := strings.TrimSpace(encoding)
		if enc == "" {
			enc = "mp4"
		}
		if strings.Contains(enc, "/") {
			mime = enc
		} else {
			mime = "video/" + enc
		}
	}
	return "data:" + mime + ";base64," + base64Data
}

func ensureAPIKey(uri, key string) string {
	if key == "" || uri == "" || model.IsSignedVideoContentURL(uri) {
		return uri
	}
	if strings.Contains(uri, "key=") {
		return uri
	}
	if strings.Contains(uri, "?") {
		return fmt.Sprintf("%s&key=%s", uri, key)
	}
	return fmt.Sprintf("%s?key=%s", uri, key)
}
