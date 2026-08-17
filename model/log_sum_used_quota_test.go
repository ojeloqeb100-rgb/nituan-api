package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSumUsedQuota_NetsRefundAgainstConsume(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	logs := []*Log{
		{Type: LogTypeConsume, Quota: 1000, Username: "alice", CreatedAt: now, ModelName: "test-model"},
		{Type: LogTypeConsume, Quota: 240, Username: "alice", CreatedAt: now, ModelName: "test-model"},
		{Type: LogTypeRefund, Quota: 208, Username: "alice", CreatedAt: now, ModelName: "test-model"},
		{Type: LogTypeTopup, Quota: 9999, Username: "alice", CreatedAt: now},
	}
	for _, log := range logs {
		require.NoError(t, LOG_DB.Create(log).Error)
	}

	stat, err := SumUsedQuota(0, 0, 0, "", "alice", "", 0, "")
	require.NoError(t, err)
	assert.Equal(t, 1032, stat.Quota)

	var refund Log
	require.NoError(t, LOG_DB.Where("type = ?", LogTypeRefund).First(&refund).Error)
	assert.Equal(t, LogTypeRefund, refund.Type)
	assert.Equal(t, 208, refund.Quota)
}
