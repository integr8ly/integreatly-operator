package types

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PostgresStratKey    = "postgres"
	RedisStratKey       = "redis"
	BlobstorageStratKey = "blobstorage"
)

type StrategyProvider interface {
	ReconcileStrategyMap(context.Context, client.Client, *StrategyTimeConfig) error
}

type StartTime struct {
	Time time.Time
	Day  time.Weekday
}

type StrategyTimeConfig struct {
	BackupStartTime      StartTime
	MaintenanceStartTime StartTime
}

func (t *StrategyTimeConfig) SetBackupTime(hour int, minute int) {
	t.BackupStartTime = StartTime{
		Time: time.Date(0, time.January, 1, hour, minute, 0, 0, time.UTC),
	}
}

func (t *StrategyTimeConfig) SetMaintenanceTime(day time.Weekday, hour int, minute int) {
	t.MaintenanceStartTime = StartTime{
		Time: time.Date(0, time.January, 1, hour, minute, 0, 0, time.UTC),
		Day:  day,
	}
}

func NewStrategyTimeConfig(backupHour int, backupMinute int, maintenanceDay time.Weekday, maintenanceHour int, maintenanceMinute int) *StrategyTimeConfig {
	config := &StrategyTimeConfig{}
	config.SetBackupTime(backupHour, backupMinute)
	config.SetMaintenanceTime(maintenanceDay, maintenanceHour, maintenanceMinute)
	return config
}
