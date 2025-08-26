package main

import (
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MessageBotInfo 消息机器人信息
type MessageBotInfo struct {
	User  *WxUserLogin
	Robot *WxRobotConfig
}

// messageBotQueryResult 数据库查询结果结构
type messageBotQueryResult struct {
	UserID        uint   `json:"user_id"`
	UserToken     string `json:"user_token"`
	UserWxID      string `json:"user_wx_id"`
	UserNickName  string `json:"user_nick_name"`
	RobotID       uint   `json:"robot_id"`
	RobotAddress  string `json:"robot_address"`
	RobotAdminKey string `json:"robot_admin_key"`
}

// MessageSendStrategy 消息发送策略接口
type MessageSendStrategy interface {
	GetMessageBot(db *gorm.DB, groupId string, logger *zap.Logger) (*MessageBotInfo, error)
}

// RoundRobinMessageSendStrategy 轮询消息机器人策略
type RoundRobinMessageSendStrategy struct {
	currentIndex int
}

// RandomMessageSendStrategy 随机消息机器人策略
type RandomMessageSendStrategy struct {
	rand *rand.Rand
}

// NewRoundRobinMessageSendStrategy 创建轮询策略
func NewRoundRobinMessageSendStrategy() MessageSendStrategy {
	return &RoundRobinMessageSendStrategy{
		currentIndex: 0,
	}
}

// NewRandomMessageSendStrategy 创建随机策略
func NewRandomMessageSendStrategy() MessageSendStrategy {
	return &RandomMessageSendStrategy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// queryMessageBots 查询所有可用的消息机器人
func queryMessageBots(db *gorm.DB, groupId string, logger *zap.Logger) ([]messageBotQueryResult, error) {
	var results []messageBotQueryResult

	err := db.Table("wx_groups g").
		Select(`u.id as user_id, u.token as user_token, u.wx_id as user_wx_id, u.nick_name as user_nick_name,
			r.id as robot_id, r.address as robot_address, r.admin_key as robot_admin_key`).
		Joins("JOIN wx_user_logins u ON g.wx_id = u.wx_id").
		Joins("JOIN wx_robot_configs r ON u.robot_id = r.id").
		Where("g.group_id = ? AND u.status = 1 AND u.is_message_bot = 1 AND u.has_security_risk = 0", groupId).
		Find(&results).Error

	if err != nil {
		logger.Error("查询消息机器人列表失败",
			zap.String("group_id", groupId),
			zap.Error(err))
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到可用的消息机器人")
	}

	return results, nil
}

// buildMessageBotInfo 根据查询结果构建MessageBotInfo
func buildMessageBotInfo(result messageBotQueryResult) *MessageBotInfo {
	user := &WxUserLogin{
		ID:       result.UserID,
		Token:    result.UserToken,
		WxID:     result.UserWxID,
		NickName: result.UserNickName,
		RobotID:  result.RobotID,
	}

	robot := &WxRobotConfig{
		ID:       result.RobotID,
		Address:  result.RobotAddress,
		AdminKey: result.RobotAdminKey,
	}

	return &MessageBotInfo{
		User:  user,
		Robot: robot,
	}
}

// GetMessageBot 轮询策略实现
func (s *RoundRobinMessageSendStrategy) GetMessageBot(db *gorm.DB, groupId string, logger *zap.Logger) (*MessageBotInfo, error) {
	results, err := queryMessageBots(db, groupId, logger)
	if err != nil {
		return nil, err
	}

	// 轮询选择
	selectedIndex := s.currentIndex % len(results)
	s.currentIndex = (s.currentIndex + 1) % len(results)

	selectedBot := buildMessageBotInfo(results[selectedIndex])

	logger.Info("使用轮询消息机器人策略",
		zap.String("group_id", groupId),
		zap.String("wx_id", selectedBot.User.WxID),
		zap.String("robot_address", selectedBot.Robot.Address),
		zap.Int("selected_index", selectedIndex),
		zap.Int("total_count", len(results)))

	return selectedBot, nil
}

// GetMessageBot 随机策略实现
func (s *RandomMessageSendStrategy) GetMessageBot(db *gorm.DB, groupId string, logger *zap.Logger) (*MessageBotInfo, error) {
	results, err := queryMessageBots(db, groupId, logger)
	if err != nil {
		return nil, err
	}

	// 随机选择
	selectedIndex := s.rand.Intn(len(results))
	selectedBot := buildMessageBotInfo(results[selectedIndex])

	logger.Info("使用随机消息机器人策略",
		zap.String("group_id", groupId),
		zap.String("wx_id", selectedBot.User.WxID),
		zap.String("robot_address", selectedBot.Robot.Address),
		zap.Int("selected_index", selectedIndex),
		zap.Int("total_count", len(results)))

	return selectedBot, nil
}
