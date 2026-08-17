package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// refundRequestCountReconcileOptionKey marks that the one-time historical
// failure-refund request_count / quota_data.count reconcile has completed.
// v1 used_quota reconcile left request_count unchanged; this key is new so
// startup will scan again even if refund_used_quota_reconcile_v1 already ran.
const refundRequestCountReconcileOptionKey = "refund_request_count_reconcile_v1"

type userConsumeCountRow struct {
	UserId int `gorm:"column:user_id"`
	Count  int `gorm:"column:cnt"`
}

type modelHourCountRow struct {
	UserId     int    `gorm:"column:user_id"`
	ModelName  string `gorm:"column:model_name"`
	BucketHour int64  `gorm:"column:bucket_hour"`
	Count      int    `gorm:"column:cnt"`
}

// ReconcileRefundRequestCountIfNeeded runs the historical failure-refund
// request_count reconcile once per database. Failures are returned so the
// caller can log them; the option key is written only after a successful run.
func ReconcileRefundRequestCountIfNeeded() error {
	if DB == nil || LOG_DB == nil {
		return errors.New("database is not initialized")
	}
	var option Option
	err := DB.Where(&Option{Key: refundRequestCountReconcileOptionKey}).First(&option).Error
	if err == nil && option.Value == "1" {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("read request_count reconcile marker: %w", err)
	}
	if err := ReconcileRefundRequestCount(); err != nil {
		return err
	}
	option = Option{Key: refundRequestCountReconcileOptionKey, Value: "1"}
	if err := DB.Save(&option).Error; err != nil {
		return fmt.Errorf("write request_count reconcile marker: %w", err)
	}
	return nil
}

// ReconcileRefundRequestCount assigns user.request_count and quota_data.count
// from consume_count − failure_refund_count. Price-recalculate type=6 logs
// are excluded. Safe to run twice. Does not change user.quota or used_quota.
func ReconcileRefundRequestCount() error {
	if DB == nil || LOG_DB == nil {
		return errors.New("database is not initialized")
	}

	SaveQuotaDataCache()

	var refundLogs []Log
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, model_name, created_at, other").
		Where("type = ?", LogTypeRefund).
		Find(&refundLogs).Error; err != nil {
		return fmt.Errorf("list refund logs: %w", err)
	}

	failureByUser := make(map[int]int)
	failureByBucket := make(map[quotaDataBucket]int)
	for _, log := range refundLogs {
		if !isFailureTaskRefundOther(log.Other) {
			continue
		}
		failureByUser[log.UserId]++
		hour := log.CreatedAt - (log.CreatedAt % 3600)
		failureByBucket[quotaDataBucket{UserId: log.UserId, ModelName: log.ModelName, Hour: hour}]++
	}
	if len(failureByUser) == 0 {
		common.SysLog("refund request_count reconcile: no failure refund logs, nothing to do")
		return nil
	}

	userIDs := make([]int, 0, len(failureByUser))
	for userID := range failureByUser {
		userIDs = append(userIDs, userID)
	}

	if err := reconcileUserRequestCount(userIDs, failureByUser); err != nil {
		return err
	}
	if err := reconcileQuotaDataCounts(userIDs, failureByBucket); err != nil {
		return err
	}
	common.SysLog("refund request_count reconcile finished")
	return nil
}

func isFailureTaskRefundOther(otherJSON string) bool {
	if otherJSON == "" {
		return true
	}
	var other map[string]interface{}
	if err := common.UnmarshalJsonStr(otherJSON, &other); err != nil {
		return true
	}
	if kind, ok := other["refund_kind"].(string); ok && kind != "" {
		return kind == RefundKindTaskFailure
	}
	if _, ok := other["pre_consumed_quota"]; ok {
		return false
	}
	if _, ok := other["actual_quota"]; ok {
		return false
	}
	return true
}

func reconcileUserRequestCount(userIDs []int, failureByUser map[int]int) error {
	var rows []userConsumeCountRow
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, COUNT(*) AS cnt").
		Where("user_id IN ? AND type = ?", userIDs, LogTypeConsume).
		Group("user_id").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("count consume logs: %w", err)
	}

	consumeByUser := make(map[int]int, len(rows))
	for _, row := range rows {
		consumeByUser[row.UserId] = row.Count
	}

	var users []User
	if err := DB.Select("id", "request_count").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return fmt.Errorf("load users: %w", err)
	}

	updated := 0
	for _, user := range users {
		net := netRequestCount(consumeByUser[user.Id], failureByUser[user.Id])
		if user.RequestCount == net {
			continue
		}
		if err := DB.Model(&User{}).Where("id = ?", user.Id).Update("request_count", net).Error; err != nil {
			return fmt.Errorf("assign user %d request_count: %w", user.Id, err)
		}
		updated++
	}
	common.SysLog(fmt.Sprintf("refund request_count reconcile: updated %d/%d users", updated, len(users)))
	return nil
}

func reconcileQuotaDataCounts(userIDs []int, failureByBucket map[quotaDataBucket]int) error {
	if len(failureByBucket) == 0 {
		return nil
	}

	var consumeBuckets []modelHourCountRow
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, model_name, "+quotaDataHourExpr+" AS bucket_hour, COUNT(*) AS cnt").
		Where("user_id IN ? AND type = ?", userIDs, LogTypeConsume).
		Group("user_id, model_name, " + quotaDataHourExpr).
		Scan(&consumeBuckets).Error; err != nil {
		return fmt.Errorf("count consume quota_data buckets: %w", err)
	}
	consumeByBucket := make(map[quotaDataBucket]int, len(consumeBuckets))
	for _, row := range consumeBuckets {
		consumeByBucket[quotaDataBucket{UserId: row.UserId, ModelName: row.ModelName, Hour: row.BucketHour}] = row.Count
	}

	var currentBuckets []modelHourCountRow
	if err := DB.Model(&QuotaData{}).
		Select("user_id, model_name, created_at AS bucket_hour, COALESCE(SUM(count), 0) AS cnt").
		Where("user_id IN ?", userIDs).
		Group("user_id, model_name, created_at").
		Scan(&currentBuckets).Error; err != nil {
		return fmt.Errorf("sum current quota_data count: %w", err)
	}
	currentByBucket := make(map[quotaDataBucket]int, len(currentBuckets))
	for _, row := range currentBuckets {
		currentByBucket[quotaDataBucket{UserId: row.UserId, ModelName: row.ModelName, Hour: row.BucketHour}] = row.Count
	}

	usernameByUser := make(map[int]string)
	inserted := 0
	for key, failureCount := range failureByBucket {
		expected := consumeByBucket[key] - failureCount
		current := currentByBucket[key]
		correction := expected - current
		if correction >= 0 {
			continue
		}
		username := usernameByUser[key.UserId]
		if username == "" {
			username, _ = GetUsernameById(key.UserId, true)
			usernameByUser[key.UserId] = username
		}
		row := &QuotaData{
			UserID:    key.UserId,
			Username:  username,
			ModelName: key.ModelName,
			CreatedAt: key.Hour,
			Count:     correction,
			Quota:     0,
		}
		if err := DB.Create(row).Error; err != nil {
			return fmt.Errorf("insert quota_data count correction user=%d hour=%d: %w", key.UserId, key.Hour, err)
		}
		currentByBucket[key] = current + correction
		inserted++
	}
	common.SysLog(fmt.Sprintf("refund request_count reconcile: inserted %d quota_data count corrections", inserted))
	return nil
}

func netRequestCount(consume, failure int) int {
	if consume < failure {
		return 0
	}
	return consume - failure
}
