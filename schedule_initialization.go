package main

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// InitializationScheduler 初始化状态检查定时任务接口
type InitializationScheduler interface {
	Start() error
	Stop() error
	CheckInitializationStatus() error
}

// DefaultInitializationScheduler 默认的初始化状态检查实现
type DefaultInitializationScheduler struct {
	logger     *zap.Logger
	wxRobotSvc WxRobotService
	cron       *cron.Cron
}

// NewInitializationScheduler 创建新的初始化状态检查定时任务
func NewInitializationScheduler(
	logger *zap.Logger,
	wxRobotSvc WxRobotService,
) InitializationScheduler {
	c := cron.New(cron.WithSeconds())
	return &DefaultInitializationScheduler{
		logger:     logger,
		wxRobotSvc: wxRobotSvc,
		cron:       c,
	}
}

// Start 启动定时任务 - 每30秒执行一次
func (s *DefaultInitializationScheduler) Start() error {
	s.logger.Info("启动初始化状态检查定时任务", zap.String("schedule", "每30秒执行一次"))

	// 每30秒执行一次
	cronExpr := "*/30 * * * * *"

	// 添加定时任务
	_, err := s.cron.AddFunc(cronExpr, func() {
		s.logger.Debug("开始执行初始化状态检查任务")
		if err := s.CheckInitializationStatus(); err != nil {
			s.logger.Error("初始化状态检查任务执行失败", zap.Error(err))
		}
	})

	if err != nil {
		s.logger.Error("添加初始化状态检查定时任务失败", zap.Error(err))
		return err
	}

	s.cron.Start()
	s.logger.Info("初始化状态检查定时任务启动完成")
	return nil
}

// Stop 停止定时任务
func (s *DefaultInitializationScheduler) Stop() error {
	s.logger.Info("停止初始化状态检查定时任务")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("初始化状态检查定时任务停止完成")
	return nil
}

// CheckInitializationStatus 检查初始化状态的核心逻辑
func (s *DefaultInitializationScheduler) CheckInitializationStatus() error {
	s.logger.Debug("开始检查初始化状态")

	// 1. 查询未初始化的用户
	users, err := s.getUninitializedUsers()
	if err != nil {
		return err
	}

	if len(users) == 0 {
		s.logger.Debug("没有找到未初始化的用户")
		return nil
	}

	s.logger.Info("找到未初始化用户", zap.Int("count", len(users)))

	// 2. 逐个检查用户的初始化状态
	for _, user := range users {
		if err := s.processUser(user); err != nil {
			s.logger.Error("处理用户失败",
				zap.Uint("user_id", user.ID),
				zap.String("wx_id", user.WxID),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// getUninitializedUsers 获取未初始化的用户列表
func (s *DefaultInitializationScheduler) getUninitializedUsers() ([]WxUserLogin, error) {
	return s.wxRobotSvc.GetUninitializedUsers()
}

// processUser 处理单个用户的初始化检查
func (s *DefaultInitializationScheduler) processUser(user WxUserLogin) error {
	s.logger.Debug("开始处理用户",
		zap.Uint("user_id", user.ID),
		zap.String("wx_id", user.WxID))

	// 获取机器人配置
	robot, err := s.wxRobotSvc.GetRobotByID(user.RobotID)
	if err != nil {
		s.logger.Error("获取机器人配置失败",
			zap.Uint("robot_id", user.RobotID),
			zap.Error(err))
		return err
	}

	// 1. 检查初始化状态
	initResp, err := s.wxRobotSvc.GetInitStatus(robot.Address, user.Token)
	if err != nil {
		s.logger.Error("调用GetInitStatus失败",
			zap.String("address", robot.Address),
			zap.String("token", user.Token),
			zap.Error(err))
		return err
	}

	// 判断是否初始化完成
	isInitialized := initResp.Code == 200 && initResp.Data

	if !isInitialized {
		s.logger.Debug("用户尚未初始化完成",
			zap.Uint("user_id", user.ID),
			zap.String("wx_id", user.WxID))

		// 检查是否存在该用户的群组数据
		groups, err := s.wxRobotSvc.GetGroupsByWxID(user.WxID)
		if err != nil {
			s.logger.Error("检查群组数据失败",
				zap.String("wx_id", user.WxID),
				zap.Error(err))
			return err
		}

		// 如果存在群组数据，说明之前已经获取过群列表，无需再次处理
		if len(groups) > 0 {
			s.logger.Debug("用户已有群组数据，跳过处理",
				zap.Uint("user_id", user.ID),
				zap.String("wx_id", user.WxID),
				zap.Int("group_count", len(groups)))
			return nil
		}

		// 没有群组数据且未初始化完成，继续等待
		return nil
	}

	s.logger.Info("用户初始化完成，开始获取群列表",
		zap.Uint("user_id", user.ID),
		zap.String("wx_id", user.WxID))

	// 2. 获取群列表
	groupResp, err := s.wxRobotSvc.GetGroupList(robot.Address, user.Token)
	if err != nil {
		s.logger.Error("获取群列表失败",
			zap.String("address", robot.Address),
			zap.String("token", user.Token),
			zap.Error(err))
		return err
	}

	// 3. 保存群信息
	if err := s.saveGroupInfo(user.WxID, groupResp); err != nil {
		s.logger.Error("保存群信息失败", zap.Error(err))
		return err
	}

	// 4. 更新用户初始化状态
	if err := s.wxRobotSvc.UpdateUserInitializationStatus(user.ID); err != nil {
		s.logger.Error("更新用户初始化状态失败", zap.Error(err))
		return err
	}

	s.logger.Info("用户初始化处理完成",
		zap.Uint("user_id", user.ID),
		zap.String("wx_id", user.WxID))

	return nil
}

// saveGroupInfo 保存群信息到数据库
func (s *DefaultInitializationScheduler) saveGroupInfo(wxID string, groupResp *GroupListResponse) error {
	if groupResp.Code != 200 || len(groupResp.Data.GroupList) == 0 {
		s.logger.Debug("没有群组数据需要保存", zap.String("wx_id", wxID))
		return nil
	}

	for _, group := range groupResp.Data.GroupList {
		groupID := group.UserName.Str
		groupNickName := group.NickName.Str

		// 保存或更新群组
		wxGroup := &WxGroup{
			WxID:          wxID,
			GroupID:       groupID,
			GroupNickName: groupNickName,
		}

		if err := s.wxRobotSvc.SaveOrUpdateGroup(wxGroup); err != nil {
			s.logger.Error("保存群组信息失败",
				zap.String("wx_id", wxID),
				zap.String("group_id", groupID),
				zap.Error(err))
			return err
		}
	}

	return nil
}
