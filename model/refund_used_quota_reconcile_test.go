package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const reconcileTestHour int64 = 1_700_000_000 - (1_700_000_000 % 3600)

func seedReconcileUser(t *testing.T, id int, quota int, usedQuota int, requestCount int) {
	t.Helper()
	user := &User{
		Id:           id,
		Username:     "reconcile_user_" + strconv.Itoa(id),
		Password:     "password",
		Quota:        quota,
		UsedQuota:    usedQuota,
		RequestCount: requestCount,
		Status:       common.UserStatusEnabled,
		AffCode:      "aff_" + strconv.Itoa(id),
	}
	require.NoError(t, DB.Create(user).Error)
}

func seedReconcileChannel(t *testing.T, id int, usedQuota int64) {
	t.Helper()
	ch := &Channel{Id: id, Name: "reconcile_channel", Key: "sk-reconcile", UsedQuota: usedQuota, Status: common.ChannelStatusEnabled}
	require.NoError(t, DB.Create(ch).Error)
}

func seedReconcileToken(t *testing.T, id int, userId int, usedQuota int) {
	t.Helper()
	token := &Token{Id: id, UserId: userId, Key: "sk-reconcile-token", Name: "reconcile", UsedQuota: usedQuota, Status: common.TokenStatusEnabled}
	require.NoError(t, DB.Create(token).Error)
}

func seedReconcileLog(t *testing.T, userId int, channelId int, logType int, quota int, modelName string, createdAt int64) {
	t.Helper()
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    userId,
		Username:  "reconcile_user",
		Type:      logType,
		Quota:     quota,
		ModelName: modelName,
		ChannelId: channelId,
		CreatedAt: createdAt,
	}).Error)
}

func seedReconcileQuotaData(t *testing.T, userId int, modelName string, quota int, count int, createdAt int64) {
	t.Helper()
	require.NoError(t, DB.Create(&QuotaData{
		UserID:    userId,
		Username:  "reconcile_user",
		ModelName: modelName,
		CreatedAt: createdAt,
		Quota:     quota,
		Count:     count,
	}).Error)
}

func loadUserAccounting(t *testing.T, id int) User {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota", "used_quota", "request_count").Where("id = ?", id).First(&user).Error)
	return user
}

func loadChannelUsedQuota(t *testing.T, id int) int64 {
	t.Helper()
	var ch Channel
	require.NoError(t, DB.Select("used_quota").Where("id = ?", id).First(&ch).Error)
	return ch.UsedQuota
}

func loadTokenUsedQuota(t *testing.T, id int) int {
	t.Helper()
	var token Token
	require.NoError(t, DB.Select("used_quota").Where("id = ?", id).First(&token).Error)
	return token.UsedQuota
}

func sumQuotaDataQuotaForUser(t *testing.T, userId int) int {
	t.Helper()
	var total int
	require.NoError(t, DB.Model(&QuotaData{}).Where("user_id = ?", userId).Select("COALESCE(SUM(quota), 0)").Scan(&total).Error)
	return total
}

func sumQuotaDataCountForUser(t *testing.T, userId int) int {
	t.Helper()
	var total int
	require.NoError(t, DB.Model(&QuotaData{}).Where("user_id = ?", userId).Select("COALESCE(SUM(count), 0)").Scan(&total).Error)
	return total
}

func TestReconcileRefundUsedQuota_RewritesDirtyUsedQuotaAndIsIdempotent(t *testing.T) {
	truncateTables(t)

	const userID, channelID, tokenID = 101, 201, 301
	const walletQuota, requestCount = 19876, 4
	const successConsume, failedConsume = 124, 208
	const dirtyUsed = successConsume + failedConsume
	const tokenUsed = 50

	seedReconcileUser(t, userID, walletQuota, dirtyUsed, requestCount)
	seedReconcileChannel(t, channelID, int64(dirtyUsed))
	seedReconcileToken(t, tokenID, userID, tokenUsed)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, successConsume, "ok-model", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, failedConsume, "fail-model", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeRefund, failedConsume, "fail-model", reconcileTestHour+60)
	seedReconcileQuotaData(t, userID, "ok-model", successConsume, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "fail-model", failedConsume, 1, reconcileTestHour)

	require.NoError(t, ReconcileRefundUsedQuota())

	user := loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, successConsume, user.UsedQuota)
	assert.Equal(t, requestCount, user.RequestCount)
	assert.Equal(t, int64(successConsume), loadChannelUsedQuota(t, channelID))
	assert.Equal(t, tokenUsed, loadTokenUsedQuota(t, tokenID))
	assert.Equal(t, successConsume, sumQuotaDataQuotaForUser(t, userID))
	assert.Equal(t, 2, sumQuotaDataCountForUser(t, userID))

	require.NoError(t, ReconcileRefundUsedQuota())

	user = loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, successConsume, user.UsedQuota)
	assert.Equal(t, requestCount, user.RequestCount)
	assert.Equal(t, int64(successConsume), loadChannelUsedQuota(t, channelID))
	assert.Equal(t, tokenUsed, loadTokenUsedQuota(t, tokenID))
	assert.Equal(t, successConsume, sumQuotaDataQuotaForUser(t, userID))
	assert.Equal(t, 2, sumQuotaDataCountForUser(t, userID))
}

func TestReconcileRefundUsedQuota_DoesNotDoubleSubtractCleanNewPath(t *testing.T) {
	truncateTables(t)

	const userID, channelID = 102, 202
	const walletQuota, requestCount = 5000, 2
	const consumeQuota = 2080
	const netUsed = 0

	seedReconcileUser(t, userID, walletQuota, netUsed, requestCount)
	seedReconcileChannel(t, channelID, int64(netUsed))
	seedReconcileLog(t, userID, channelID, LogTypeConsume, consumeQuota, "fail-model", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeRefund, consumeQuota, "fail-model", reconcileTestHour+30)
	seedReconcileQuotaData(t, userID, "fail-model", consumeQuota, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "fail-model", -consumeQuota, 0, reconcileTestHour)

	require.NoError(t, ReconcileRefundUsedQuota())

	user := loadUserAccounting(t, userID)
	assert.Equal(t, walletQuota, user.Quota)
	assert.Equal(t, netUsed, user.UsedQuota)
	assert.Equal(t, requestCount, user.RequestCount)
	assert.Equal(t, int64(netUsed), loadChannelUsedQuota(t, channelID))
	assert.Equal(t, 0, sumQuotaDataQuotaForUser(t, userID))
	assert.Equal(t, 1, sumQuotaDataCountForUser(t, userID))
}

func TestReconcileRefundUsedQuota_QuotaDataAlreadyNegativeIsNotDeductedAgain(t *testing.T) {
	truncateTables(t)

	const userID, channelID = 103, 203
	const consumeA, consumeB, refundB = 1000, 240, 208

	seedReconcileUser(t, userID, 8000, consumeA+consumeB-refundB, 3)
	seedReconcileChannel(t, channelID, int64(consumeA+consumeB-refundB))
	seedReconcileLog(t, userID, channelID, LogTypeConsume, consumeA, "a", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeConsume, consumeB, "b", reconcileTestHour)
	seedReconcileLog(t, userID, channelID, LogTypeRefund, refundB, "b", reconcileTestHour+10)
	seedReconcileQuotaData(t, userID, "a", consumeA, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "b", consumeB, 1, reconcileTestHour)
	seedReconcileQuotaData(t, userID, "b", -refundB, 0, reconcileTestHour)

	require.NoError(t, ReconcileRefundUsedQuota())
	require.NoError(t, ReconcileRefundUsedQuota())

	assert.Equal(t, consumeA+consumeB-refundB, loadUserAccounting(t, userID).UsedQuota)
	assert.Equal(t, consumeA+consumeB-refundB, sumQuotaDataQuotaForUser(t, userID))
	assert.GreaterOrEqual(t, sumQuotaDataCountForUser(t, userID), 0)
}

func TestReconcileRefundUsedQuota_LeavesUsersWithoutRefundsUntouched(t *testing.T) {
	truncateTables(t)

	const dirtyID, cleanID = 104, 105
	seedReconcileUser(t, dirtyID, 1000, 9999, 7)
	seedReconcileUser(t, cleanID, 2000, 333, 2)
	seedReconcileLog(t, dirtyID, 0, LogTypeConsume, 100, "m", reconcileTestHour)
	seedReconcileLog(t, dirtyID, 0, LogTypeRefund, 40, "m", reconcileTestHour)
	seedReconcileLog(t, cleanID, 0, LogTypeConsume, 50, "m", reconcileTestHour)

	require.NoError(t, ReconcileRefundUsedQuota())

	assert.Equal(t, 60, loadUserAccounting(t, dirtyID).UsedQuota)
	assert.Equal(t, 7, loadUserAccounting(t, dirtyID).RequestCount)
	assert.Equal(t, 333, loadUserAccounting(t, cleanID).UsedQuota)
	assert.Equal(t, 2, loadUserAccounting(t, cleanID).RequestCount)
}

func TestReconcileRefundUsedQuotaIfNeeded_SkipsSecondScanButAlgorithmStaysReentrant(t *testing.T) {
	truncateTables(t)

	const userID = 106
	seedReconcileUser(t, userID, 1000, 300, 1)
	seedReconcileLog(t, userID, 0, LogTypeConsume, 300, "m", reconcileTestHour)
	seedReconcileLog(t, userID, 0, LogTypeRefund, 100, "m", reconcileTestHour)

	require.NoError(t, ReconcileRefundUsedQuotaIfNeeded())
	assert.Equal(t, 200, loadUserAccounting(t, userID).UsedQuota)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", userID).Update("used_quota", 300).Error)
	require.NoError(t, ReconcileRefundUsedQuotaIfNeeded())
	assert.Equal(t, 300, loadUserAccounting(t, userID).UsedQuota, "option key skips a repeat scan")

	require.NoError(t, ReconcileRefundUsedQuota())
	assert.Equal(t, 200, loadUserAccounting(t, userID).UsedQuota, "direct reconcile stays reentrant")
}
