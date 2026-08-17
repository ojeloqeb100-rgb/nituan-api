package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	videoLinkExpiredMessage      = "This video link has expired"
	videoContentSignedContextKey = "video_content_signed"
)

func videoProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func VideoProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}

	signed := c.GetBool(videoContentSignedContextKey)
	if signed {
		expires := strings.TrimSpace(c.Query("expires"))
		exp, err := strconv.ParseInt(expires, 10, 64)
		if err != nil || exp <= 0 || time.Now().Unix() >= exp {
			videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)
			return
		}
	}

	task, exists, err := loadVideoContentTask(c, taskID, signed)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to query task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}
	if task.Status != model.TaskStatusSuccess {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}
	if service.TaskLocalVideoExpired(task) {
		service.RemoveTaskLocalVideo(task)
		videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)
		return
	}

	if !service.HasUsableLocalVideo(task) {
		if err := service.CacheTaskVideo(c.Request.Context(), task, nil); err != nil {
			if errors.Is(err, service.ErrVideoLinkExpired) || service.TaskLocalVideoExpired(task) {
				videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)
				return
			}
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to materialize local video for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)
			return
		}
	}

	path, ok := service.SafeLocalVideoPath(task)
	if !ok {
		videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)
		return
	}
	if _, err := os.Stat(path); err != nil {
		videoProxyError(c, http.StatusGone, "invalid_request_error", videoLinkExpiredMessage)
		return
	}

	mime := strings.TrimSpace(task.PrivateData.LocalMimeType)
	if mime == "" {
		mime = "video/mp4"
	}
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", `inline; filename="video.mp4"`)
	c.Header("Cache-Control", "private, max-age=3600")
	c.File(path)
}

func loadVideoContentTask(c *gin.Context, taskID string, signed bool) (*model.Task, bool, error) {
	if signed {
		return model.GetByPublicTaskId(taskID)
	}
	userID := c.GetInt("id")
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		return nil, false, err
	}
	if (!exists || task == nil) && c.GetInt("role") >= common.RoleAdminUser {
		return model.GetByPublicTaskId(taskID)
	}
	return task, exists, err
}

func isTaskProxyContentURL(raw string, taskID string) bool {
	return model.IsLocalVideoContentURL(raw, taskID)
}
