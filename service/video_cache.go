package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
)

const (
	defaultVideoCacheMaxBytes int64 = 512 << 20
	videoCacheMaintainEvery         = 15 * time.Minute
)

var (
	ErrVideoLinkExpired = errors.New("video link expired")
	videoTaskLocks      sync.Map
)

func VideoCacheDir() string {
	if dir := strings.TrimSpace(os.Getenv("VIDEO_CACHE_DIR")); dir != "" {
		return filepath.Clean(dir)
	}
	if info, err := os.Stat("/data"); err == nil && info.IsDir() {
		return filepath.Join("/data", "video-cache")
	}
	return filepath.Join("data", "video-cache")
}

func videoCacheMaxBytes() int64 {
	if mb := common.GetEnvOrDefault("VIDEO_CACHE_MAX_MB", 0); mb > 0 {
		return int64(mb) << 20
	}
	return defaultVideoCacheMaxBytes
}

func StartVideoCacheMaintenance() {
	if !common.IsMasterNode {
		return
	}
	go func() {
		maintainVideoCache(context.Background())
		ticker := time.NewTicker(videoCacheMaintainEvery)
		defer ticker.Stop()
		for range ticker.C {
			maintainVideoCache(context.Background())
		}
	}()
}

func maintainVideoCache(ctx context.Context) {
	CleanupExpiredVideoFiles()
	BackfillRecentTaskVideos(ctx)
}

func TaskLocalVideoExpired(task *model.Task) bool {
	if task == nil || task.PrivateData.LocalExpiresAt <= 0 {
		return false
	}
	return time.Now().Unix() >= task.PrivateData.LocalExpiresAt
}

func HasUsableLocalVideo(task *model.Task) bool {
	if task == nil || TaskLocalVideoExpired(task) {
		return false
	}
	path, ok := SafeLocalVideoPath(task)
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

func SafeLocalVideoPath(task *model.Task) (string, bool) {
	if task == nil {
		return "", false
	}
	raw := strings.TrimSpace(task.PrivateData.LocalPath)
	if raw == "" {
		var err error
		raw, err = defaultVideoCachePath(task.TaskID)
		if err != nil {
			return "", false
		}
	}
	clean := filepath.Clean(raw)
	root := filepath.Clean(VideoCacheDir()) + string(os.PathSeparator)
	if !strings.HasPrefix(clean+string(os.PathSeparator), root) && clean != filepath.Clean(VideoCacheDir()) {
		return "", false
	}
	return clean, true
}

func defaultVideoCachePath(taskID string) (string, error) {
	id := sanitizeVideoTaskID(taskID)
	if id == "" {
		return "", fmt.Errorf("invalid task id")
	}
	dir := VideoCacheDir()
	return filepath.Join(dir, id+".mp4"), nil
}

func sanitizeVideoTaskID(taskID string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(taskID) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func PersistTaskVideo(task *model.Task, body io.Reader, contentType string) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	dest, err := defaultVideoCachePath(task.TaskID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(body, videoCacheMaxBytes()+1))
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	if written <= 0 {
		os.Remove(tmp)
		return fmt.Errorf("empty video body")
	}
	if written > videoCacheMaxBytes() {
		os.Remove(tmp)
		return fmt.Errorf("video exceeds maximum cache size")
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	task.PrivateData.LocalPath = dest
	task.PrivateData.LocalExpiresAt = time.Now().Add(taskcommon.VideoContentTTL).Unix()
	mime := strings.TrimSpace(contentType)
	if mime == "" || mime == "application/octet-stream" {
		mime = "video/mp4"
	}
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = strings.TrimSpace(mime[:idx])
	}
	task.PrivateData.LocalMimeType = mime
	if task.ID > 0 {
		if err := task.SavePrivateData(); err != nil {
			return err
		}
	}
	return nil
}

func RemoveTaskLocalVideo(task *model.Task) {
	if task == nil {
		return
	}
	if path, ok := SafeLocalVideoPath(task); ok {
		_ = os.Remove(path)
	}
	if task.PrivateData.LocalPath == "" && task.PrivateData.LocalExpiresAt == 0 {
		return
	}
	task.PrivateData.LocalPath = ""
	task.PrivateData.LocalExpiresAt = 0
	task.PrivateData.LocalMimeType = ""
	if task.ID > 0 {
		_ = task.SavePrivateData()
	}
}

func CleanupExpiredVideoFiles() {
	dir := VideoCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			common.SysError("video cache cleanup failed: " + err.Error())
		}
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") {
			info, infoErr := entry.Info()
			if infoErr == nil && now.Sub(info.ModTime()) > time.Hour {
				_ = os.Remove(filepath.Join(dir, name))
			}
			continue
		}
		path := filepath.Join(dir, name)
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		expiredByAge := now.Sub(info.ModTime()) > taskcommon.VideoContentTTL
		taskID := strings.TrimSuffix(name, filepath.Ext(name))
		if model.DB != nil {
			if task, exists, err := model.GetByPublicTaskId(taskID); err == nil && exists && task != nil {
				if TaskLocalVideoExpired(task) || expiredByAge {
					RemoveTaskLocalVideo(task)
				}
				continue
			}
		}
		if expiredByAge {
			_ = os.Remove(path)
		}
	}
}

func BackfillRecentTaskVideos(ctx context.Context) {
	since := time.Now().Add(-taskcommon.VideoContentTTL).Unix()
	for _, task := range model.ListRecentSuccessfulTasks(since, 50) {
		if ctx.Err() != nil {
			return
		}
		if task == nil || task.Platform == constant.TaskPlatformSuno {
			continue
		}
		if HasUsableLocalVideo(task) {
			continue
		}
		if TaskLocalVideoExpired(task) {
			continue
		}
		if upstream := strings.TrimSpace(task.GetResultURL()); model.IsSignedVideoContentURL(upstream) {
			if signedVideoURLExpired(upstream) {
				continue
			}
		}
		if err := CacheTaskVideo(ctx, task, nil); err != nil && !errors.Is(err, ErrVideoLinkExpired) {
			logger.LogWarn(ctx, fmt.Sprintf("video backfill skipped for task %s", task.TaskID))
		}
	}
}

func lockTaskVideo(taskID string) *sync.Mutex {
	actual, _ := videoTaskLocks.LoadOrStore(taskID, &sync.Mutex{})
	return actual.(*sync.Mutex)
}
