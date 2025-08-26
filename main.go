// Package main wx-msg-api
// @title WeChat Robot API
// @version 1.0
// @description WeChat robot management API service
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8886
// @BasePath /api/wx/v1
// @schemes http https

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "wx-msg-api/docs"

	"go.uber.org/zap"
)

func main() {
	// 初始化配置
	cfg, err := InitConfig()
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	// 初始化日志
	logger, err := InitLogger(cfg)
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	logger.Info("应用启动", zap.String("name", cfg.App.Name), zap.String("version", cfg.App.Version))

	// 初始化数据库
	dbManager, err := NewDatabaseManager(cfg, logger)
	if err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}


	// 初始化微信机器人服务
	wxRobotSvc := NewWxRobotService(dbManager.GetDB(), logger)

	// 初始化路由管理器
	routerMgr := NewRouterManager(logger, wxRobotSvc)

	// 初始化路由
	router := routerMgr.InitRoutes(cfg)

	// 初始化定时任务
	scheduler := NewInitializationScheduler(logger, wxRobotSvc)

	// 初始化群组同步定时任务
	groupSyncScheduler := NewGroupSyncScheduler(logger, wxRobotSvc)

	// 初始化登录状态检查定时任务
	loginStatusScheduler := NewLoginStatusScheduler(logger, wxRobotSvc)


	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 启动定时任务
	if err := scheduler.Start(); err != nil {
		logger.Error("启动定时任务失败", zap.Error(err))
	}

	// 启动群组同步定时任务
	if err := groupSyncScheduler.Start(); err != nil {
		logger.Error("启动群组同步定时任务失败", zap.Error(err))
	}

	// 启动登录状态检查定时任务
	if err := loginStatusScheduler.Start(); err != nil {
		logger.Error("启动登录状态检查定时任务失败", zap.Error(err))
	}


	// 启动服务器
	go func() {
		logger.Info("HTTP服务器启动", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	gracefulShutdown(server, logger, scheduler, groupSyncScheduler, loginStatusScheduler, dbManager)
}

// gracefulShutdown 优雅关闭
func gracefulShutdown(server *http.Server, logger *zap.Logger, scheduler InitializationScheduler, groupSyncScheduler GroupSyncScheduler, loginStatusScheduler LoginStatusScheduler, dbManager *DatabaseManager) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("服务器正在关闭...")

	// 停止定时任务
	if scheduler != nil {
		if err := scheduler.Stop(); err != nil {
			logger.Error("停止定时任务失败", zap.Error(err))
		}
	}

	// 停止群组同步定时任务
	if groupSyncScheduler != nil {
		if err := groupSyncScheduler.Stop(); err != nil {
			logger.Error("停止群组同步定时任务失败", zap.Error(err))
		}
	}

	// 停止登录状态检查定时任务
	if loginStatusScheduler != nil {
		if err := loginStatusScheduler.Stop(); err != nil {
			logger.Error("停止登录状态检查定时任务失败", zap.Error(err))
		}
	}


	ctx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器强制关闭", zap.Error(err))
	}

	// 关闭数据库连接
	if dbManager != nil {
		if err := dbManager.Close(); err != nil {
			logger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}

	logger.Info("服务器已关闭")
}
