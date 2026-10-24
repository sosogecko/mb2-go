package db

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mb2-go/internal/config"
)

func Open(cfg config.MySQLConfig, log *zap.Logger) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now()
		},
	}
	conn, err := gorm.Open(mysql.Open(cfg.DSN()), gcfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	if log != nil {
		log.Info("mysql connected", zap.String("host", cfg.Host), zap.String("db", cfg.Database))
	}
	return conn, nil
}
