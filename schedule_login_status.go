package main

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// LoginStatusScheduler 登录状态检查定时任务接口
type LoginStatusScheduler interface {
	Start() error
	Stop() error
	CheckLoginStatus() error
}

// DefaultLoginStatusScheduler 默认的登录状态检查实现
type DefaultLoginStatusScheduler struct {
	logger     *zap.Logger
	wxRobotSvc WxRobotService
	cron       *cron.Cron
}

// NewLoginStatusScheduler 创建新的登录状态检查定时任务
func NewLoginStatusScheduler(
	logger *zap.Logger,
	wxRobotSvc WxRobotService,
) LoginStatusScheduler {
	c := cron.New(cron.WithSeconds())
	return &DefaultLoginStatusScheduler{
		logger:     logger,
		wxRobotSvc: wxRobotSvc,
		cron:       c,
	}
}

// Start 启动登录状态检查定时任务 - 每1分钟执行一次
func (s *DefaultLoginStatusScheduler) Start() error {
	s.logger.Info("启动登录状态检查定时任务", zap.String("schedule", "每1分钟执行一次"))

	// 每1分钟执行一次
	cronExpr := "*/30 * * * * *"

	// 添加定时任务
	_, err := s.cron.AddFunc(cronExpr, func() {
		s.logger.Debug("开始执行登录状态检查任务")
		if err := s.CheckLoginStatus(); err != nil {
			s.logger.Error("登录状态检查任务执行失败", zap.Error(err))
		}
	})

	if err != nil {
		s.logger.Error("添加登录状态检查定时任务失败", zap.Error(err))
		return err
	}

	s.cron.Start()
	s.logger.Info("登录状态检查定时任务启动完成")
	return nil
}

// Stop 停止登录状态检查定时任务
func (s *DefaultLoginStatusScheduler) Stop() error {
	s.logger.Info("停止登录状态检查定时任务")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("登录状态检查定时任务停止完成")
	return nil
}

// CheckLoginStatus 检查登录状态的核心逻辑
func (s *DefaultLoginStatusScheduler) CheckLoginStatus() error {
	s.logger.Debug("开始检查用户登录状态")

	// 1. 查询状态为1的活跃用户
	users, err := s.wxRobotSvc.GetActiveUsers()
	if err != nil {
		s.logger.Error("获取活跃用户列表失败", zap.Error(err))
		return err
	}

	if len(users) == 0 {
		s.logger.Debug("没有找到活跃用户")
		return nil
	}

	s.logger.Info("找到活跃用户", zap.Int("count", len(users)))

	// 2. 逐个检查用户的登录状态
	successCount := 0
	errorCount := 0
	reloginCount := 0

	for _, user := range users {
		// 检查用户是否需要重新登录
		robot, err := s.wxRobotSvc.GetRobotByID(user.RobotID)
		if err != nil {
			s.logger.Error("获取机器人配置失败", zap.Uint("robot_id", user.RobotID), zap.Error(err))
			errorCount++
			continue
		}

		resp, err := s.wxRobotSvc.CheckCanSetAlias(robot.Address, user.Token)
		if err != nil {
			s.logger.Error("调用CheckCanSetAlias失败",
				zap.String("address", robot.Address),
				zap.String("token", user.Token),
				zap.Error(err))
			errorCount++
			continue
		}

		// 如果返回代码是300，表示需要重新登录
		if resp.Code == 300 {
			if err := s.wxRobotSvc.UpdateUserStatus(user.ID, 3); err != nil {
				s.logger.Error("更新用户状态为需要重新登录失败",
					zap.Uint("user_id", user.ID),
					zap.Error(err))
				errorCount++
				continue
			}
			s.logger.Info("用户需要重新登录，状态已更新",
				zap.Uint("user_id", user.ID),
				zap.String("wx_id", user.WxID),
				zap.Int("new_status", 3),
				zap.String("status_desc", "需要重新登录"))
			reloginCount++
		} else {
			successCount++
		}
	}

	s.logger.Info("登录状态检查任务完成",
		zap.Int("total", len(users)),
		zap.Int("success", successCount),
		zap.Int("need_relogin", reloginCount),
		zap.Int("error", errorCount))

	return nil
}

// processUserLoginStatus 处理单个用户的登录状态检查
func (s *DefaultLoginStatusScheduler) processUserLoginStatus(user WxUserLogin) error {
	s.logger.Debug("开始检查用户登录状态",
		zap.Uint("user_id", user.ID),
		zap.String("wx_id", user.WxID))

	// 这里可以添加额外的登录状态检查逻辑
	// 目前主要通过CheckCanSetAlias来判断是否需要重新登录

	return nil
}
