package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// refundUsedQuotaReconcileOptionKey marks that the one-time historical
// refund used_quota / quota_data reconcile has completed. The algorithm
// itself is reentrant; the key only skips a repeated full-table scan.
const refundUsedQuotaReconcileOptionKey = "refund_used_quota_reconcile_v1"

const quotaDataHourExpr = "(created_at - (created_at % 3600))"

type userTypeQuotaRow struct {
	UserId int `gorm:"column:user_id"`
	Type   int `gorm:"column:type"`
	Quota  int `gorm:"column:quota"`
}

type channelTypeQuotaRow struct {
	ChannelId int   `gorm:"column:channel_id"`
	Type      int   `gorm:"column:type"`
	Quota     int64 `gorm:"column:quota"`
}

type modelHourQuotaRow struct {
	UserId     int    `gorm:"column:user_id"`
	ModelName  string `gorm:"column:model_name"`
	BucketHour int64  `gorm:"column:bucket_hour"`
	Quota      int    `gorm:"column:quota"`
}

// ReconcileRefundUsedQuotaIfNeeded runs the historical refund used_quota
// reconcile once per database. Failures are returned so the caller can log
// them; the option key is written only after a successful run.
func ReconcileRefundUsedQuotaIfNeeded() error {
	if DB == nil || LOG_DB == nil {
		return errors.New("database is not initialized")
	}
	var option Option
	err := DB.Where(&Option{Key: refundUsedQuotaReconcileOptionKey}).First(&option).Error
	if err == nil && option.Value == "1" {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("read reconcile marker: %w", err)
	}
	if err := ReconcileRefundUsedQuota(); err != nil {
		return err
	}
	option = Option{Key: refundUsedQuotaReconcileOptionKey, Value: "1"}
	if err := DB.Save(&option).Error; err != nil {
		return fmt.Errorf("write reconcile marker: %w", err)
	}
	return nil
}

// ReconcileRefundUsedQuota aligns historical used_quota / channel.used_quota /
// quota_data with consume−refund. It assigns the net from logs; it does not
// subtract a refund total from the current counters. Safe to run twice.
// Does not change user.quota, request_count, or token.used_quota.
func ReconcileRefundUsedQuota() error {
	if DB == nil || LOG_DB == nil {
		return errors.New("database is not initialized")
	}

	SaveQuotaDataCache()

	var refundUserIDs []int
	if err := LOG_DB.Model(&Log{}).
		Where("type = ?", LogTypeRefund).
		Distinct("user_id").
		Pluck("user_id", &refundUserIDs).Error; err != nil {
		return fmt.Errorf("list refund users: %w", err)
	}
	if len(refundUserIDs) == 0 {
		common.SysLog("refund used_quota reconcile: no type=6 logs, nothing to do")
		return nil
	}

	if err := reconcileUserUsedQuota(refundUserIDs); err != nil {
		return err
	}
	if err := reconcileChannelUsedQuota(); err != nil {
		return err
	}
	if err := reconcileQuotaDataNets(refundUserIDs); err != nil {
		return err
	}
	common.SysLog("refund used_quota reconcile finished")
	return nil
}

func reconcileUserUsedQuota(userIDs []int) error {
	var rows []userTypeQuotaRow
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, type, COALESCE(SUM(quota), 0) AS quota").
		Where("user_id IN ? AND type IN ?", userIDs, []int{LogTypeConsume, LogTypeRefund}).
		Group("user_id, type").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("sum user consume/refund: %w", err)
	}

	consumeByUser := make(map[int]int, len(userIDs))
	refundByUser := make(map[int]int, len(userIDs))
	for _, row := range rows {
		switch row.Type {
		case LogTypeConsume:
			consumeByUser[row.UserId] = row.Quota
		case LogTypeRefund:
			refundByUser[row.UserId] = row.Quota
		}
	}

	var users []User
	if err := DB.Select("id", "used_quota").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return fmt.Errorf("load users: %w", err)
	}

	updated := 0
	for _, user := range users {
		net := netUsedQuota(consumeByUser[user.Id], refundByUser[user.Id])
		if user.UsedQuota == net {
			continue
		}
		if err := DB.Model(&User{}).Where("id = ?", user.Id).Update("used_quota", net).Error; err != nil {
			return fmt.Errorf("assign user %d used_quota: %w", user.Id, err)
		}
		updated++
	}
	common.SysLog(fmt.Sprintf("refund used_quota reconcile: updated %d/%d users", updated, len(users)))
	return nil
}

func reconcileChannelUsedQuota() error {
	var refundChannelIDs []int
	if err := LOG_DB.Model(&Log{}).
		Where("type = ? AND channel_id <> ?", LogTypeRefund, 0).
		Distinct("channel_id").
		Pluck("channel_id", &refundChannelIDs).Error; err != nil {
		return fmt.Errorf("list refund channels: %w", err)
	}
	if len(refundChannelIDs) == 0 {
		return nil
	}

	var rows []channelTypeQuotaRow
	if err := LOG_DB.Model(&Log{}).
		Select("channel_id, type, COALESCE(SUM(quota), 0) AS quota").
		Where("channel_id IN ? AND type IN ?", refundChannelIDs, []int{LogTypeConsume, LogTypeRefund}).
		Group("channel_id, type").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("sum channel consume/refund: %w", err)
	}

	consumeByChannel := make(map[int]int64, len(refundChannelIDs))
	refundByChannel := make(map[int]int64, len(refundChannelIDs))
	for _, row := range rows {
		switch row.Type {
		case LogTypeConsume:
			consumeByChannel[row.ChannelId] = row.Quota
		case LogTypeRefund:
			refundByChannel[row.ChannelId] = row.Quota
		}
	}

	var channels []Channel
	if err := DB.Select("id", "used_quota").Where("id IN ?", refundChannelIDs).Find(&channels).Error; err != nil {
		return fmt.Errorf("load channels: %w", err)
	}

	updated := 0
	for _, channel := range channels {
		net := netChannelUsedQuota(consumeByChannel[channel.Id], refundByChannel[channel.Id])
		if channel.UsedQuota == net {
			continue
		}
		if err := DB.Model(&Channel{}).Where("id = ?", channel.Id).Update("used_quota", net).Error; err != nil {
			return fmt.Errorf("assign channel %d used_quota: %w", channel.Id, err)
		}
		updated++
	}
	common.SysLog(fmt.Sprintf("refund used_quota reconcile: updated %d/%d channels", updated, len(channels)))
	return nil
}

func reconcileQuotaDataNets(userIDs []int) error {
	var refundBuckets []modelHourQuotaRow
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, model_name, "+quotaDataHourExpr+" AS bucket_hour, COALESCE(SUM(quota), 0) AS quota").
		Where("user_id IN ? AND type = ?", userIDs, LogTypeRefund).
		Group("user_id, model_name, " + quotaDataHourExpr).
		Scan(&refundBuckets).Error; err != nil {
		return fmt.Errorf("sum refund quota_data buckets: %w", err)
	}
	if len(refundBuckets) == 0 {
		return nil
	}

	var consumeBuckets []modelHourQuotaRow
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, model_name, "+quotaDataHourExpr+" AS bucket_hour, COALESCE(SUM(quota), 0) AS quota").
		Where("user_id IN ? AND type = ?", userIDs, LogTypeConsume).
		Group("user_id, model_name, " + quotaDataHourExpr).
		Scan(&consumeBuckets).Error; err != nil {
		return fmt.Errorf("sum consume quota_data buckets: %w", err)
	}

	consumeByBucket := make(map[quotaDataBucket]int, len(consumeBuckets))
	for _, row := range consumeBuckets {
		consumeByBucket[quotaDataBucket{UserId: row.UserId, ModelName: row.ModelName, Hour: row.BucketHour}] = row.Quota
	}

	var currentBuckets []modelHourQuotaRow
	if err := DB.Model(&QuotaData{}).
		Select("user_id, model_name, created_at AS bucket_hour, COALESCE(SUM(quota), 0) AS quota").
		Where("user_id IN ?", userIDs).
		Group("user_id, model_name, created_at").
		Scan(&currentBuckets).Error; err != nil {
		return fmt.Errorf("sum current quota_data: %w", err)
	}
	currentByBucket := make(map[quotaDataBucket]int, len(currentBuckets))
	for _, row := range currentBuckets {
		currentByBucket[quotaDataBucket{UserId: row.UserId, ModelName: row.ModelName, Hour: row.BucketHour}] = row.Quota
	}

	usernameByUser := make(map[int]string)
	inserted := 0
	for _, refund := range refundBuckets {
		key := quotaDataBucket{UserId: refund.UserId, ModelName: refund.ModelName, Hour: refund.BucketHour}
		expected := consumeByBucket[key] - refund.Quota
		current := currentByBucket[key]
		correction := expected - current
		if correction >= 0 {
			continue
		}
		username := usernameByUser[refund.UserId]
		if username == "" {
			username, _ = GetUsernameById(refund.UserId, true)
			usernameByUser[refund.UserId] = username
		}
		row := &QuotaData{
			UserID:    refund.UserId,
			Username:  username,
			ModelName: refund.ModelName,
			CreatedAt: refund.BucketHour,
			Count:     0,
			Quota:     correction,
		}
		if err := DB.Create(row).Error; err != nil {
			return fmt.Errorf("insert quota_data correction user=%d hour=%d: %w", refund.UserId, refund.BucketHour, err)
		}
		currentByBucket[key] = current + correction
		inserted++
	}
	common.SysLog(fmt.Sprintf("refund used_quota reconcile: inserted %d quota_data corrections", inserted))
	return nil
}

type quotaDataBucket struct {
	UserId    int
	ModelName string
	Hour      int64
}

func netUsedQuota(consume, refund int) int {
	if consume < refund {
		return 0
	}
	net := consume - refund
	if net > common.MaxQuota {
		return common.MaxQuota
	}
	return net
}

func netChannelUsedQuota(consume, refund int64) int64 {
	if consume < refund {
		return 0
	}
	return consume - refund
}
