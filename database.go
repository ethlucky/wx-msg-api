package main

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DatabaseManager 数据库管理器
type DatabaseManager struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDatabaseManager 创建数据库管理器
func NewDatabaseManager(cfg *Config, logger *zap.Logger) (*DatabaseManager, error) {
	db, err := connectDatabase(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &DatabaseManager{
		db:     db,
		logger: logger,
	}, nil
}

// GetDB 获取数据库连接
func (dm *DatabaseManager) GetDB() *gorm.DB {
	return dm.db
}

// Close 关闭数据库连接
func (dm *DatabaseManager) Close() error {
	if dm.db != nil {
		if sqlDB, err := dm.db.DB(); err == nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// connectDatabase 初始化数据库连接
func connectDatabase(cfg *Config, logger *zap.Logger) (*gorm.DB, error) {
	// 构建简化的DSN - 先用最基本的参数测试
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	logger.Info("正在连接数据库", zap.String("dsn", dsn))
	
	// GORM日志级别
	var logLevel gormlogger.LogLevel
	switch cfg.Database.LogLevel {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "warn":
		logLevel = gormlogger.Warn
	case "info":
		logLevel = gormlogger.Info
	default:
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		logger.Error("数据库连接测试失败", zap.Error(err))
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	logger.Info("数据库连接成功",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Database),
		zap.String("charset", cfg.Database.Charset))

	// 数据库连接成功
	logger.Info("数据库连接成功")
	return db, nil
}


// CheckDatabaseHealth 检查数据库健康状态
func (dm *DatabaseManager) CheckDatabaseHealth() error {
	if dm.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	
	if sqlDB, err := dm.db.DB(); err == nil {
		return sqlDB.Ping()
	} else {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}
}

