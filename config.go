package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 配置结构体
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
	Database DatabaseConfig `mapstructure:"database"`
	Swagger  SwaggerConfig  `mapstructure:"swagger"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
	Port    string `mapstructure:"port"`
	Debug   bool   `mapstructure:"debug"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

type DatabaseConfig struct {
	Host                     string        `mapstructure:"host"`
	Port                     int           `mapstructure:"port"`
	Username                 string        `mapstructure:"username"`
	Password                 string        `mapstructure:"password"`
	Database                 string        `mapstructure:"database"`
	Charset                  string        `mapstructure:"charset"`
	ParseTime                bool          `mapstructure:"parse_time"`
	Loc                      string        `mapstructure:"loc"`
	MaxIdleConns             int           `mapstructure:"max_idle_conns"`
	MaxOpenConns             int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime          time.Duration `mapstructure:"conn_max_lifetime"`
	LogLevel                 string        `mapstructure:"log_level"`
	AllowMultiQueries        bool          `mapstructure:"allow_multi_queries"`
	UseCursorFetch           bool          `mapstructure:"use_cursor_fetch"`
	RewriteBatchedStatements bool          `mapstructure:"rewrite_batched_statements"`
}

type SwaggerConfig struct {
	Enable bool   `mapstructure:"enable"`
	Host   string `mapstructure:"host"`
	Port   int    `mapstructure:"port"`
}

// InitConfig 初始化配置
func InitConfig() (*Config, error) {
	// 获取环境变量，默认为开发环境
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	// 根据环境选择配置文件
	configName := "config-" + env
	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 环境变量支持
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败 (%s): %w", configName, err)
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 打印加载的配置文件信息
	fmt.Printf("===========================================\n")
	fmt.Printf("应用启动信息:\n")
	fmt.Printf("配置文件: %s.toml\n", configName)
	fmt.Printf("运行环境: %s\n", cfg.App.Env)
	fmt.Printf("应用名称: %s\n", cfg.App.Name)
	fmt.Printf("应用版本: %s\n", cfg.App.Version)
	fmt.Printf("调试模式: %t\n", cfg.App.Debug)
	fmt.Printf("服务地址: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("===========================================\n")

	return cfg, nil
}

// InitLogger 初始化日志
func InitLogger(cfg *Config) (*zap.Logger, error) {
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 创建日志目录
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建编码器
	var encoder zapcore.Encoder
	if cfg.Log.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var cores []zapcore.Core

	// 控制台输出
	if cfg.Log.Output == "stdout" || cfg.Log.Output == "both" {
		consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, consoleCore)
	}

	// 文件输出（带轮转）
	if cfg.Log.Output == "file" || cfg.Log.Output == "both" {
		// 配置日志轮转
		logRotate := &lumberjack.Logger{
			Filename:   cfg.Log.FilePath,
			MaxSize:    cfg.Log.MaxSize,    // MB
			MaxAge:     cfg.Log.MaxAge,     // 天
			MaxBackups: cfg.Log.MaxBackups, // 保留文件数
			Compress:   cfg.Log.Compress,   // 压缩
			LocalTime:  true,               // 使用本地时间
		}

		fileCore := zapcore.NewCore(encoder, zapcore.AddSync(logRotate), level)
		cores = append(cores, fileCore)
	}

	// 创建核心
	core := zapcore.NewTee(cores...)

	// 创建日志器
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Info("日志系统初始化完成",
		zap.String("level", cfg.Log.Level),
		zap.String("format", cfg.Log.Format),
		zap.String("output", cfg.Log.Output),
		zap.String("file_path", cfg.Log.FilePath),
		zap.Int("max_age_days", cfg.Log.MaxAge),
		zap.Int("max_size_mb", cfg.Log.MaxSize),
		zap.Int("max_backups", cfg.Log.MaxBackups),
		zap.Bool("compress", cfg.Log.Compress))

	return logger, nil
}
