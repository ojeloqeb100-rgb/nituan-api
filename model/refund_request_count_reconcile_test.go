package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedReconcileLogWithOther(t *testing.T, userId int, channelId int, logType int, quota int, modelName string, createdAt int64, other map[string]interface{}) {
	t.Helper()
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    userId,
		Username:  "reconcile_user",
		Type:      logType,
		Quota:     quota,
		ModelName: modelName,
		ChannelId: channelId,
		CreatedAt: createdAt,
		Other:     common.MapToJsonStr(other),
	}).Error)
}

func TestIsFailureTaskRefundOther(t *testing.T) {
	assert.True(t, isFailureTaskRefundOther(""))
	assert.True(t, isFailureTaskRefundOther(common.MapToJsonStr(map[string]interface{}{
		"reason":  "task failed: upstream error",
		"task_id": "task_1",
	})))
	assert.True(t, isFailureTaskRefundOther(common.MapToJsonStr(map[string]interface{}{
		"refund_kind": RefundKindTaskFailure,
		"reason":      "构图失败",
		"task_id":     "mj_1",
	})))
	assert.False(t, isFailureTaskRefundOther(common.MapToJsonStr(map[string]interface{}{
		"pre_consumed_quota": 5000,
		"actual_quota":       3000,
	})))
	assert.False(t, isFailureTaskRefundOther(common.MapToJsonStr(map[string]interface{}{
		"refund_kind":        RefundKindQuotaRecalculate,
		"pre_consumed_quota": 5000,
		"actual_quota":       3000,
	})))
}

func TestReconcileRefundRequestCount_DirtyTwoRequestsBecomesOneAndIsIdempotent(t *testing.T) {
	truncateTables(t)

	const userID, channelID = 201, 301
	const walletQuota, dirtyRequestCount = 19876, 2
	const successConsume, failedConsume = 124, 208

	seedReconcileUser(t, userID, walletQuota, successConsume, dirtyRequestCount)
	seedReconcileChannel(t, channelID, int64(successConsume))
	seedReconcileLog(t, userID, channelID, LogTypeConsume, successConsume, "ok-model", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, failedConsume, "fail-model", reconcileTestHour)
	seedReconcileLogWithOther(t, userID, channelID, LogTypeRefund, failedConsume, "fail-model", reconcileTestHour+60, map[string]interface{}{
		"reason":  "task failed: upstream error",
		"task_id": "task_fail",
	})
	seedReconcileQuotaData(t, userID, "ok-model", successConsume, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "fail-model", failedConsume, 1, reconcileTestHour)

	require.NoError(t, ReconcileRefundRequestCount())

	user := loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, successConsume, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)
	assert.Equal(t, 1, sumQuotaDataCountForUser(t, userID))

	require.NoError(t, ReconcileRefundRequestCount())

	user = loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, successConsume, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)
	assert.Equal(t, 1, sumQuotaDataCountForUser(t, userID))
}

func TestReconcileRefundRequestCount_AlreadyCorrectUserIsNotDecremented(t *testing.T) {
	truncateTables(t)

	const userID, channelID = 202, 302
	const walletQuota = 5000
	const successConsume, failedConsume = 124, 2080

	seedReconcileUser(t, userID, walletQuota, successConsume, 1)
	seedReconcileChannel(t, channelID, int64(successConsume))
	seedReconcileLog(t, userID, channelID, LogTypeConsume, successConsume, "ok-model", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, failedConsume, "fail-model", reconcileTestHour)
	seedReconcileLogWithOther(t, userID, channelID, LogTypeRefund, failedConsume, "fail-model", reconcileTestHour+30, map[string]interface{}{
		"refund_kind": RefundKindTaskFailure,
		"reason":      "task failed",
	})
	seedReconcileQuotaData(t, userID, "ok-model", successConsume, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "fail-model", failedConsume, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "fail-model", -failedConsume, -1, reconcileTestHour)

	require.NoError(t, ReconcileRefundRequestCount())

	user := loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, successConsume, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)
	assert.Equal(t, 1, sumQuotaDataCountForUser(t, userID))
}

func TestReconcileRefundRequestCount_IgnoresPriceRecalculateRefunds(t *testing.T) {
	truncateTables(t)

	const userID, channelID = 203, 303
	const walletQuota, requestCount = 8000, 1
	const consumeQuota, refundDelta = 5000, 2000

	seedReconcileUser(t, userID, walletQuota, consumeQuota-refundDelta, requestCount)
	seedReconcileChannel(t, channelID, int64(consumeQuota-refundDelta))
	seedReconcileLog(t, userID, channelID, LogTypeConsume, consumeQuota, "ok-model", reconcileTestHour)
	seedReconcileLogWithOther(t, userID, channelID, LogTypeRefund, refundDelta, "ok-model", reconcileTestHour+10, map[string]interface{}{
		"refund_kind":        RefundKindQuotaRecalculate,
		"pre_consumed_quota": consumeQuota,
		"actual_quota":       consumeQuota - refundDelta,
	})
	seedReconcileQuotaData(t, userID, "ok-model", consumeQuota, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "ok-model", -refundDelta, 0, reconcileTestHour)

	require.NoError(t, ReconcileRefundRequestCount())

	user := loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, requestCount, user.RequestCount)
	assert.Equal(t, 1, sumQuotaDataCountForUser(t, userID))
}

func TestReconcileRefundRequestCount_MixedFailureAndRecalculateOnlyCountsFailure(t *testing.T) {
	truncateTables(t)

	const userID, channelID = 204, 304
	const walletQuota, dirtyRequestCount = 9000, 3
	const successConsume, failedConsume, recalcRefund = 1000, 400, 200

	seedReconcileUser(t, userID, walletQuota, successConsume+failedConsume-recalcRefund, dirtyRequestCount)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, successConsume, "ok-model", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, failedConsume, "fail-model", reconcileTestHour)
	seedReconcileLogWithOther(t, userID, channelID, LogTypeRefund, failedConsume, "fail-model", reconcileTestHour+20, map[string]interface{}{
		"refund_kind": RefundKindTaskFailure,
		"reason":      "upstream error",
	})
	seedReconcileLogWithOther(t, userID, channelID, LogTypeRefund, recalcRefund, "ok-model", reconcileTestHour+40, map[string]interface{}{
		"pre_consumed_quota": successConsume + recalcRefund,
		"actual_quota":       successConsume,
	})
	seedReconcileQuotaData(t, userID, "ok-model", successConsume, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "fail-model", failedConsume, 1, reconcileTestHour)

	require.NoError(t, ReconcileRefundRequestCount())

	user := loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, 1, user.RequestCount)
	assert.Equal(t, 1, sumQuotaDataCountForUser(t, userID))
}

func TestReconcileRefundRequestCountIfNeeded_RunsDespiteUsedQuotaMarkerThenSkips(t *testing.T) {
	truncateTables(t)

	const userID = 205
	seedReconcileUser(t, userID, 1000, 300, 2)
	seedReconcileLog(t, userID, 0, LogTypeConsume, 200, "ok-model", reconcileTestHour)
	seedReconcileLog(t, userID, 0, LogTypeConsume, 100, "fail-model", reconcileTestHour)
	seedReconcileLog(t, userID, 0, LogTypeRefund, 100, "fail-model", reconcileTestHour)
	require.NoError(t, DB.Create(&Option{Key: refundUsedQuotaReconcileOptionKey, Value: "1"}).Error)

	require.NoError(t, ReconcileRefundRequestCountIfNeeded())
	assert.Equal(t, 1, loadUserAccounting(t, userID).RequestCount)
	assert.Equal(t, 1000, loadUserAccounting(t, userID).Quota)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", userID).Update("request_count", 2).Error)
	require.NoError(t, ReconcileRefundRequestCountIfNeeded())
	assert.Equal(t, 2, loadUserAccounting(t, userID).RequestCount, "option key skips a repeat scan")

	require.NoError(t, ReconcileRefundRequestCount())
	assert.Equal(t, 1, loadUserAccounting(t, userID).RequestCount, "direct reconcile stays reentrant")
}

func TestReverseUserUsedQuotaAndRequestCount_ClampsRequestCount(t *testing.T) {
	truncateTables(t)

	const userID = 206
	seedReconcileUser(t, userID, 5000, 100, 1)

	ReverseUserUsedQuotaAndRequestCount(userID, 100)
	user := loadUserAccounting(t, userID)
	assert.Equal(t, 5000, user.Quota)
	assert.Equal(t, 0, user.UsedQuota)
	assert.Equal(t, 0, user.RequestCount)

	ReverseUserUsedQuotaAndRequestCount(userID, 50)
	user = loadUserAccounting(t, userID)
	assert.Equal(t, 0, user.RequestCount)
}

func TestReverseUserUsedQuota_LeavesRequestCountUnchanged(t *testing.T) {
	truncateTables(t)

	const userID = 207
	seedReconcileUser(t, userID, 5000, 200, 3)

	ReverseUserUsedQuota(userID, 80)
	user := loadUserAccounting(t, userID)
	assert.Equal(t, 120, user.UsedQuota)
	assert.Equal(t, 3, user.RequestCount)
}
