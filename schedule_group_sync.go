package main

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// GroupSyncScheduler 群组同步定时任务接口
type GroupSyncScheduler interface {
	Start() error
	Stop() error
	SyncGroupsForAllUsers() error
}

// DefaultGroupSyncScheduler 默认的群组同步定时任务实现
type DefaultGroupSyncScheduler struct {
	logger     *zap.Logger
	wxRobotSvc WxRobotService
	cron       *cron.Cron
}

// NewGroupSyncScheduler 创建新的群组同步定时任务
func NewGroupSyncScheduler(
	logger *zap.Logger,
	wxRobotSvc WxRobotService,
) GroupSyncScheduler {
	c := cron.New(cron.WithSeconds())
	return &DefaultGroupSyncScheduler{
		logger:     logger,
		wxRobotSvc: wxRobotSvc,
		cron:       c,
	}
}

// Start 启动群组同步定时任务 - 每3分钟执行一次
func (s *DefaultGroupSyncScheduler) Start() error {
	s.logger.Info("启动群组同步定时任务", zap.String("schedule", "每3分钟执行一次"))

	// 添加定时任务：每3分钟执行一次
	_, err := s.cron.AddFunc("0 */3 * * * *", func() {
		s.logger.Debug("开始执行群组同步任务")
		if err := s.SyncGroupsForAllUsers(); err != nil {
			s.logger.Error("群组同步任务执行失败", zap.Error(err))
		}
	})

	if err != nil {
		s.logger.Error("添加群组同步定时任务失败", zap.Error(err))
		return err
	}

	s.cron.Start()
	s.logger.Info("群组同步定时任务启动完成")
	return nil
}

// Stop 停止群组同步定时任务
func (s *DefaultGroupSyncScheduler) Stop() error {
	s.logger.Info("停止群组同步定时任务")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("群组同步定时任务停止完成")
	return nil
}

// SyncGroupsForAllUsers 为所有已初始化用户同步群组数据
func (s *DefaultGroupSyncScheduler) SyncGroupsForAllUsers() error {
	s.logger.Debug("开始为所有已初始化用户同步群组数据")

	// 1. 获取所有已初始化的用户
	users, err := s.wxRobotSvc.GetInitializedUsers()
	if err != nil {
		s.logger.Error("获取已初始化用户列表失败", zap.Error(err))
		return err
	}

	if len(users) == 0 {
		s.logger.Debug("没有找到已初始化的用户")
		return nil
	}

	s.logger.Info("找到已初始化用户", zap.Int("count", len(users)))

	// 2. 逐个用户同步群组数据
	successCount := 0
	errorCount := 0

	for _, user := range users {
		if err := s.syncGroupsForUser(user); err != nil {
			s.logger.Error("同步用户群组数据失败",
				zap.Uint("user_id", user.ID),
				zap.String("wx_id", user.WxID),
				zap.Error(err))
			errorCount++
			continue
		}
		successCount++
	}

	s.logger.Info("群组同步任务完成",
		zap.Int("total", len(users)),
		zap.Int("success", successCount),
		zap.Int("error", errorCount))

	return nil
}

// syncGroupsForUser 同步单个用户的群组数据
func (s *DefaultGroupSyncScheduler) syncGroupsForUser(user WxUserLogin) error {
	s.logger.Debug("开始同步用户群组数据",
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

	// 调用微信接口获取群列表
	groupResp, err := s.wxRobotSvc.GetGroupList(robot.Address, user.Token)
	if err != nil {
		s.logger.Error("获取群列表失败",
			zap.String("address", robot.Address),
			zap.String("token", user.Token),
			zap.Error(err))
		return err
	}

	if groupResp.Code != 200 {
		s.logger.Warn("获取群列表返回错误",
			zap.String("wx_id", user.WxID),
			zap.Int("code", groupResp.Code),
			zap.String("text", groupResp.Text))
		return nil // 不算作错误，可能是临时问题
	}

	// 处理群组数据同步
	return s.processGroupSync(user.WxID, groupResp)
}

// processGroupSync 处理群组数据同步逻辑
func (s *DefaultGroupSyncScheduler) processGroupSync(wxID string, groupResp *GroupListResponse) error {
	// 提取当前API返回的群ID列表
	currentGroupIDs := make([]string, 0, len(groupResp.Data.GroupList))

	// 保存或更新群组信息
	for _, group := range groupResp.Data.GroupList {
		groupID := group.UserName.Str
		groupNickName := group.NickName.Str

		currentGroupIDs = append(currentGroupIDs, groupID)

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

	// 删除数据库中存在但当前群列表中不存在的群组
	if err := s.wxRobotSvc.DeleteGroupsByWxIDNotInList(wxID, currentGroupIDs); err != nil {
		s.logger.Error("删除过期群组失败",
			zap.String("wx_id", wxID),
			zap.Error(err))
		return err
	}

	s.logger.Debug("用户群组数据同步完成",
		zap.String("wx_id", wxID),
		zap.Int("group_count", len(currentGroupIDs)))

	return nil
}