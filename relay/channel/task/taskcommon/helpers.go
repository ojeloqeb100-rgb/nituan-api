package taskcommon

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// UnmarshalMetadata converts a map[string]any metadata to a typed struct via JSON round-trip.
// This replaces the repeated pattern: json.Marshal(metadata) → json.Unmarshal(bytes, &target).
func UnmarshalMetadata(metadata map[string]any, target any) error {
	if metadata == nil {
		return nil
	}
	// Prevent metadata from overriding model fields to avoid billing bypass.
	delete(metadata, "model")
	metaBytes, err := common.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata failed: %w", err)
	}
	if err := common.Unmarshal(metaBytes, target); err != nil {
		return fmt.Errorf("unmarshal metadata failed: %w", err)
	}
	return nil
}

// DefaultString returns val if non-empty, otherwise fallback.
func DefaultString(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

// DefaultInt returns val if non-zero, otherwise fallback.
func DefaultInt(val, fallback int) int {
	if val == 0 {
		return fallback
	}
	return val
}

// EncodeLocalTaskID encodes an upstream operation name to a URL-safe base64 string.
// Used by Gemini/Vertex to store upstream names as task IDs.
func EncodeLocalTaskID(name string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(name))
}

// DecodeLocalTaskID decodes a base64-encoded upstream operation name.
func DecodeLocalTaskID(id string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// BuildProxyURL returns a host-relative content path for the public task ID.
// Callers that need an absolute URL should use BuildPublicProxyURL so local
// and production hosts are taken from the current request or ServerAddress.
func BuildProxyURL(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ""
	}
	return "/v1/videos/" + taskID + "/content"
}

// BuildPublicProxyURL builds a content URL that follows the current request
// Host, then the configured ServerAddress, then a root-relative path.
func BuildPublicProxyURL(c *gin.Context, taskID string) string {
	path := BuildProxyURL(taskID)
	if path == "" {
		return ""
	}
	if origin := RequestOrigin(c); origin != "" {
		return origin + path
	}
	if base := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/"); base != "" {
		return base + path
	}
	return path
}

const VideoContentTTL = 12 * time.Hour

func VideoContentExpiry(task *model.Task) int64 {
	if task != nil && task.PrivateData.LocalExpiresAt > 0 {
		return task.PrivateData.LocalExpiresAt
	}
	return time.Now().Add(VideoContentTTL).Unix()
}

func BuildSignedProxyURL(taskID string, expires int64) string {
	path := BuildProxyURL(taskID)
	if path == "" || expires <= 0 {
		return path
	}
	return path + "?" + videoContentQuery(taskID, expires)
}

func BuildPublicSignedProxyURL(c *gin.Context, taskID string, expires int64) string {
	base := BuildPublicProxyURL(c, taskID)
	if base == "" || expires <= 0 {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&" + videoContentQuery(taskID, expires)
	}
	return base + "?" + videoContentQuery(taskID, expires)
}

func videoContentQuery(taskID string, expires int64) string {
	exp := strconv.FormatInt(expires, 10)
	return "expires=" + url.QueryEscape(exp) + "&sign=" + url.QueryEscape(SignVideoContent(taskID, exp))
}

func SignVideoContent(taskID, expires string) string {
	return common.GenerateHMAC("video-content-v1\n" + strings.TrimSpace(taskID) + "\n" + strings.TrimSpace(expires))
}

func VerifyVideoContentSign(taskID, expires, sign string) bool {
	taskID = strings.TrimSpace(taskID)
	expires = strings.TrimSpace(expires)
	sign = strings.TrimSpace(sign)
	if taskID == "" || expires == "" || sign == "" {
		return false
	}
	expected, err := hex.DecodeString(SignVideoContent(taskID, expires))
	if err != nil || len(expected) == 0 {
		return false
	}
	provided, err := hex.DecodeString(sign)
	if err != nil || len(provided) == 0 {
		return false
	}
	return hmac.Equal(expected, provided)
}

// RequestOrigin returns scheme://host from the current request. It does not
// invent 127.0.0.1; missing Host falls through to the caller.
func RequestOrigin(c *gin.Context) string {
	host, scheme := requestHostAndScheme(c)
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func requestHostAndScheme(c *gin.Context) (host, scheme string) {
	if c == nil || c.Request == nil {
		return "", ""
	}
	host = strings.TrimSpace(c.Request.Host)
	if host == "" {
		if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwarded != "" {
			host = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
	}
	if host == "" {
		return "", ""
	}
	scheme = "http"
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = strings.ToLower(strings.TrimSpace(strings.Split(proto, ",")[0]))
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	return host, scheme
}

// Status-to-progress mapping constants for polling updates.
const (
	ProgressSubmitted  = "10%"
	ProgressQueued     = "20%"
	ProgressInProgress = "30%"
	ProgressComplete   = "100%"
)

// ---------------------------------------------------------------------------
// BaseBilling — embeddable no-op implementations for TaskAdaptor billing methods.
// Adaptors that do not need custom billing can embed this struct directly.
// ---------------------------------------------------------------------------

type BaseBilling struct{}

// EstimateBilling returns nil (no extra ratios; use base model price).
func (BaseBilling) EstimateBilling(_ *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

// AdjustBillingOnSubmit returns nil (no submit-time adjustment).
func (BaseBilling) AdjustBillingOnSubmit(_ *relaycommon.RelayInfo, _ []byte) map[string]float64 {
	return nil
}

// AdjustBillingOnComplete returns 0 (keep pre-charged amount).
func (BaseBilling) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}
