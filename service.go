package main

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 微信机器人服务接口
type WxRobotService interface {
	// 外部API调用
	GenAuthKey(robotAddress, adminKey string, count, days int) (*GenAuthKeyResponse, error)
	GetLoginQrCode(robotAddress, authKey string, check bool, proxy string) (*GetLoginQrCodeResponse, error)
	CheckCanSetAlias(robotAddress, authKey string) (*CheckCanSetAliasResponse, error)
	CheckLoginStatus(robotAddress, authKey string) (*CheckLoginStatusResponse, error)
	GetLoginStatus(robotAddress, authKey string) (*GetLoginStatusResponse, error)
	GetInitStatus(robotAddress, authKey string) (*GetInitStatusResponse, error)
	DelayAuthKey(robotAddress, adminKey, authKey string, days int) (*DelayAuthKeyResponse, error)
	GetChatRoomInfo(robotAddress, authKey string, chatRoomIds []string) (*GetChatRoomInfoResponse, error)
	GetGroupList(robotAddress, authKey string) (*GroupListResponse, error)

	// 消息发送接口
	SendText(robotAddress, authKey string, req *SendTextRequest) (*SendTextResponse, error)
	SendImage(robotAddress, authKey string, req *SendImageRequest) (*SendImageResponse, error)
	SendTextAndImage(robotAddress, authKey string, req *SendTextAndImageRequest) (*SendTextAndImageResponse, error)

	// 数据库操作
	GetRobotList() ([]WxRobotConfig, error)
	CreateRobot(robot *WxRobotConfig) error
	UpdateRobot(robot *WxRobotConfig) error
	GetUsersByRobot(robotId string) ([]WxUserLogin, error)
	GetRobotByID(id uint) (*WxRobotConfig, error)
	GetUserByID(id uint) (*WxUserLogin, error)
	SaveUser(user *WxUserLogin) error
	DeleteUser(id string) error
	UpdateUserExtension(robotId uint, token string, newExpiry time.Time) error
	GetInitializedUsers() ([]WxUserLogin, error)
	GetUninitializedUsers() ([]WxUserLogin, error)
	GetActiveUsers() ([]WxUserLogin, error)
	UpdateUserInitializationStatus(userID uint) error
	UpdateUserStatus(userID uint, status int) error
	UpdateMessageBotStatus(userID uint, isMessageBot int) error
	SaveOrUpdateGroup(group *WxGroup) error
	DeleteGroupsByWxIDNotInList(wxID string, groupIDs []string) error
	GetGroupsByWxID(wxID string) ([]WxGroup, error)
	SearchGroupsByName(groupNickName string) ([]WxGroup, error)
	GetMessageBotByStrategy(groupId string, strategy MessageSendStrategy) (*MessageBotInfo, error)
	CheckDatabaseHealth() error
	CheckRobotHealth(robotAddress string) (bool, error)

	// 账单处理相关
	GetMaxMsgTimeFromMessages() (int64, error)
	GetGroupByGroupID(groupID string) (*WxGroup, error)
	CreateBill(bill *WxBillInfo) error
	
	// 账单统计相关
	GetBillStatistics(req BillStatsRequest) (*BillStatsPaginatedResponse, error)
	GetBillList(req BillQueryRequest) (*BillQueryPaginatedResponse, error)
}

// 微信机器人服务实现
type wxRobotService struct {
	apiClient *WxAPIClient
	db        *gorm.DB
	logger    *zap.Logger
}

// NewWxRobotService 创建微信机器人服务
func NewWxRobotService(db *gorm.DB, logger *zap.Logger) WxRobotService {
	return &wxRobotService{
		apiClient: NewWxAPIClient(logger),
		db:        db,
		logger:    logger,
	}
}

// 生成授权码
func (s *wxRobotService) GenAuthKey(robotAddress, adminKey string, count, days int) (*GenAuthKeyResponse, error) {
	return s.apiClient.GenAuthKey(robotAddress, adminKey, count, days)
}

// 获取登录二维码
func (s *wxRobotService) GetLoginQrCode(robotAddress, authKey string, check bool, proxy string) (*GetLoginQrCodeResponse, error) {
	return s.apiClient.GetLoginQrCode(robotAddress, authKey, check, proxy)
}

// 检查是否有安全风险
func (s *wxRobotService) CheckCanSetAlias(robotAddress, authKey string) (*CheckCanSetAliasResponse, error) {
	return s.apiClient.CheckCanSetAlias(robotAddress, authKey)
}

// 检查登录状态
func (s *wxRobotService) CheckLoginStatus(robotAddress, authKey string) (*CheckLoginStatusResponse, error) {
	return s.apiClient.CheckLoginStatus(robotAddress, authKey)
}

// 获取登录状态
func (s *wxRobotService) GetLoginStatus(robotAddress, authKey string) (*GetLoginStatusResponse, error) {
	return s.apiClient.GetLoginStatus(robotAddress, authKey)
}

// 检查初始化状态
func (s *wxRobotService) GetInitStatus(robotAddress, authKey string) (*GetInitStatusResponse, error) {
	return s.apiClient.GetInitStatus(robotAddress, authKey)
}

// 授权码延期
func (s *wxRobotService) DelayAuthKey(robotAddress, adminKey, authKey string, days int) (*DelayAuthKeyResponse, error) {
	return s.apiClient.DelayAuthKey(robotAddress, adminKey, authKey, days)
}

// 获取群详情
func (s *wxRobotService) GetChatRoomInfo(robotAddress, authKey string, chatRoomIds []string) (*GetChatRoomInfoResponse, error) {
	return s.apiClient.GetChatRoomInfo(robotAddress, authKey, chatRoomIds)
}

// 获取群列表
func (s *wxRobotService) GetGroupList(robotAddress, authKey string) (*GroupListResponse, error) {
	return s.apiClient.GetGroupList(robotAddress, authKey)
}

// 发送文本消息（简化版）
func (s *wxRobotService) SendText(robotAddress, authKey string, req *SendTextRequest) (*SendTextResponse, error) {
	return s.apiClient.SendText(robotAddress, authKey, req)
}

// 发送图片消息（简化版）
func (s *wxRobotService) SendImage(robotAddress, authKey string, req *SendImageRequest) (*SendImageResponse, error) {
	return s.apiClient.SendImage(robotAddress, authKey, req)
}

// 同时发送文字和图片
func (s *wxRobotService) SendTextAndImage(robotAddress, authKey string, req *SendTextAndImageRequest) (*SendTextAndImageResponse, error) {
	return s.apiClient.SendTextAndImage(robotAddress, authKey, req)
}

// 数据库操作方法

// GetRobotList 获取机器人列表
func (s *wxRobotService) GetRobotList() ([]WxRobotConfig, error) {
	var robots []WxRobotConfig
	if err := s.db.Preload("UserLogins").Find(&robots).Error; err != nil {
		s.logger.Error("查询机器人列表失败", zap.Error(err))
		return nil, err
	}
	return robots, nil
}

// CreateRobot 创建机器人配置
func (s *wxRobotService) CreateRobot(robot *WxRobotConfig) error {
	if err := s.db.Create(robot).Error; err != nil {
		s.logger.Error("创建机器人配置失败", zap.Error(err))
		return err
	}
	return nil
}

// UpdateRobot 更新机器人配置
func (s *wxRobotService) UpdateRobot(robot *WxRobotConfig) error {
	if err := s.db.Save(robot).Error; err != nil {
		s.logger.Error("更新机器人配置失败", zap.Error(err))
		return err
	}
	return nil
}

// GetUsersByRobot 获取指定机器人的用户列表
func (s *wxRobotService) GetUsersByRobot(robotId string) ([]WxUserLogin, error) {
	var users []WxUserLogin
	if err := s.db.Where("robot_id = ?", robotId).Find(&users).Error; err != nil {
		s.logger.Error("查询用户列表失败", zap.Error(err))
		return nil, err
	}
	return users, nil
}

// GetRobotByID 根据ID获取机器人配置
func (s *wxRobotService) GetRobotByID(id uint) (*WxRobotConfig, error) {
	var robot WxRobotConfig
	if err := s.db.First(&robot, id).Error; err != nil {
		return nil, err
	}
	return &robot, nil
}

// GetUserByID 根据ID获取用户信息
func (s *wxRobotService) GetUserByID(id uint) (*WxUserLogin, error) {
	var user WxUserLogin
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// SaveUser 保存用户登录信息（saveOrUpdate逻辑：先更新，不存在则创建）
func (s *wxRobotService) SaveUser(user *WxUserLogin) error {
	// 先尝试查找现有记录，基于robot_id和wx_id的组合
	var existingUser WxUserLogin
	err := s.db.Where("robot_id = ? AND wx_id = ?", user.RobotID, user.WxID).First(&existingUser).Error

	if err == nil {
		// 记录存在，执行更新操作
		// 保留原有的ID和创建时间
		user.ID = existingUser.ID
		user.CreateTime = existingUser.CreateTime
		user.UpdateTime = time.Now()

		if err := s.db.Save(user).Error; err != nil {
			s.logger.Error("更新用户登录信息失败", zap.Error(err))
			return err
		}
		s.logger.Info("用户登录信息已更新", zap.String("wxid", user.WxID), zap.String("nickname", user.NickName))
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// 记录不存在，创建新记录
		if err := s.db.Create(user).Error; err != nil {
			s.logger.Error("创建用户登录信息失败", zap.Error(err))
			return err
		}
		s.logger.Info("用户登录成功", zap.String("wxid", user.WxID), zap.String("nickname", user.NickName))
	} else {
		// 其他数据库错误
		s.logger.Error("查询用户记录失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteUser 删除用户
func (s *wxRobotService) DeleteUser(id string) error {
	// 首先获取用户信息，以获取wx_id用于日志记录
	var user WxUserLogin
	if err := s.db.First(&user, id).Error; err != nil {
		s.logger.Error("查询用户信息失败", zap.Error(err))
		return err
	}

	// 删除用户记录（不删除群组信息，因为群组可能被其他用户使用）
	if err := s.db.Delete(&WxUserLogin{}, id).Error; err != nil {
		s.logger.Error("删除用户失败", zap.Error(err))
		return err
	}

	s.logger.Info("成功删除用户", zap.String("wx_id", user.WxID), zap.String("nickname", user.NickName))
	return nil
}

// UpdateUserExtension 更新用户延期时间
func (s *wxRobotService) UpdateUserExtension(robotId uint, token string, newExpiry time.Time) error {
	var user WxUserLogin
	if err := s.db.Where("robot_id = ? AND token = ?", robotId, token).First(&user).Error; err == nil {
		user.ExtensionTime = newExpiry
		user.ExpirationTime = newExpiry
		return s.db.Save(&user).Error
	}
	return nil
}

// GetInitializedUsers 获取已初始化的用户列表
func (s *wxRobotService) GetInitializedUsers() ([]WxUserLogin, error) {
	var users []WxUserLogin
	if err := s.db.Where("is_initialized = ? AND status = ?", 1, 1).Find(&users).Error; err != nil {
		s.logger.Error("查询已初始化用户失败", zap.Error(err))
		return nil, err
	}
	return users, nil
}

// GetUninitializedUsers 获取未初始化的用户列表
func (s *wxRobotService) GetUninitializedUsers() ([]WxUserLogin, error) {
	var users []WxUserLogin
	if err := s.db.Where("is_initialized = ? AND status = ?", 0, 1).Find(&users).Error; err != nil {
		s.logger.Error("查询未初始化用户失败", zap.Error(err))
		return nil, err
	}
	return users, nil
}

// UpdateUserInitializationStatus 更新用户初始化状态
func (s *wxRobotService) UpdateUserInitializationStatus(userID uint) error {
	result := s.db.Model(&WxUserLogin{}).Where("id = ?", userID).Update("is_initialized", 1)
	if result.Error != nil {
		s.logger.Error("更新用户初始化状态失败", zap.Uint("user_id", userID), zap.Error(result.Error))
		return result.Error
	}
	s.logger.Debug("用户初始化状态更新完成", zap.Uint("user_id", userID))
	return nil
}

// SaveOrUpdateGroup 保存或更新群组信息
func (s *wxRobotService) SaveOrUpdateGroup(group *WxGroup) error {
	var existing WxGroup
	result := s.db.Where("wx_id = ? AND group_id = ?", group.WxID, group.GroupID).First(&existing)

	if result.Error != nil {
		// 群不存在，创建新记录
		if err := s.db.Create(group).Error; err != nil {
			s.logger.Error("创建群记录失败", zap.Error(err))
			return err
		}
		s.logger.Debug("成功创建群记录",
			zap.String("wx_id", group.WxID),
			zap.String("group_id", group.GroupID),
			zap.String("group_nick_name", group.GroupNickName))
	} else {
		// 群已存在，更新昵称（如果有变化）
		if existing.GroupNickName != group.GroupNickName {
			existing.GroupNickName = group.GroupNickName
			if err := s.db.Save(&existing).Error; err != nil {
				s.logger.Error("更新群记录失败", zap.Error(err))
				return err
			}
			s.logger.Debug("成功更新群记录",
				zap.String("wx_id", group.WxID),
				zap.String("group_id", group.GroupID),
				zap.String("group_nick_name", group.GroupNickName))
		}
	}
	return nil
}

// DeleteGroupsByWxIDNotInList 删除数据库中存在但群列表中没有的群
func (s *wxRobotService) DeleteGroupsByWxIDNotInList(wxID string, groupIDs []string) error {
	if len(groupIDs) == 0 {
		// 如果群列表为空，删除该用户的所有群
		result := s.db.Where("wx_id = ?", wxID).Delete(&WxGroup{})
		if result.Error != nil {
			s.logger.Error("删除用户所有群记录失败", zap.String("wx_id", wxID), zap.Error(result.Error))
			return result.Error
		}
		if result.RowsAffected > 0 {
			s.logger.Info("删除用户所有群记录", zap.String("wx_id", wxID), zap.Int64("count", result.RowsAffected))
		}
		return nil
	}

	result := s.db.Where("wx_id = ? AND group_id NOT IN ?", wxID, groupIDs).Delete(&WxGroup{})
	if result.Error != nil {
		s.logger.Error("删除群记录失败", zap.String("wx_id", wxID), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected > 0 {
		s.logger.Info("删除过期群记录",
			zap.String("wx_id", wxID),
			zap.Int64("count", result.RowsAffected))
	}

	return nil
}

// GetGroupsByWxID 获取用户的群列表
func (s *wxRobotService) GetGroupsByWxID(wxID string) ([]WxGroup, error) {
	var groups []WxGroup
	if err := s.db.Where("wx_id = ?", wxID).Find(&groups).Error; err != nil {
		s.logger.Error("查询用户群列表失败", zap.String("wx_id", wxID), zap.Error(err))
		return nil, err
	}
	return groups, nil
}

// SearchGroupsByName 按群名称模糊搜索群组
func (s *wxRobotService) SearchGroupsByName(groupNickName string) ([]WxGroup, error) {
	var groups []WxGroup
	if err := s.db.Where("group_nick_name LIKE ?", "%"+groupNickName+"%").Find(&groups).Error; err != nil {
		s.logger.Error("按群名称搜索群组失败", zap.String("group_nick_name", groupNickName), zap.Error(err))
		return nil, err
	}
	return groups, nil
}

// GetActiveUsers 获取状态为1的用户列表
func (s *wxRobotService) GetActiveUsers() ([]WxUserLogin, error) {
	var users []WxUserLogin
	if err := s.db.Where("status = ?", 1).Find(&users).Error; err != nil {
		s.logger.Error("查询活跃用户失败", zap.Error(err))
		return nil, err
	}
	return users, nil
}

// UpdateUserStatus 更新用户状态
func (s *wxRobotService) UpdateUserStatus(userID uint, status int) error {
	result := s.db.Model(&WxUserLogin{}).Where("id = ?", userID).Update("status", status)
	if result.Error != nil {
		s.logger.Error("更新用户状态失败", zap.Uint("user_id", userID), zap.Int("status", status), zap.Error(result.Error))
		return result.Error
	}
	s.logger.Info("用户状态更新完成", zap.Uint("user_id", userID), zap.Int("status", status))
	return nil
}

// UpdateMessageBotStatus 更新消息机器人状态
func (s *wxRobotService) UpdateMessageBotStatus(userID uint, isMessageBot int) error {
	// 首先检查用户是否存在
	var user WxUserLogin
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		s.logger.Error("用户不存在", zap.Uint("user_id", userID), zap.Error(err))
		return err
	}

	// 更新消息机器人状态
	result := s.db.Model(&WxUserLogin{}).Where("id = ?", userID).Update("is_message_bot", isMessageBot)
	if result.Error != nil {
		s.logger.Error("更新消息机器人状态失败", zap.Uint("user_id", userID), zap.Int("is_message_bot", isMessageBot), zap.Error(result.Error))
		return result.Error
	}
	s.logger.Info("消息机器人状态更新完成", zap.Uint("user_id", userID), zap.Int("is_message_bot", isMessageBot))
	return nil
}

// GetMessageBotByStrategy 通过策略获取消息机器人信息
func (s *wxRobotService) GetMessageBotByStrategy(groupId string, strategy MessageSendStrategy) (*MessageBotInfo, error) {
	return strategy.GetMessageBot(s.db, groupId, s.logger)
}

// CheckDatabaseHealth 检查数据库健康状态
func (s *wxRobotService) CheckDatabaseHealth() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// CheckRobotHealth 检查机器人健康状态
func (s *wxRobotService) CheckRobotHealth(robotAddress string) (bool, error) {
	return s.apiClient.CheckRobotHealth(robotAddress)
}

// GetMaxMsgTimeFromMessages 获取wx_group_messages表中最大的msg_time
func (s *wxRobotService) GetMaxMsgTimeFromMessages() (int64, error) {
	var maxMsgTime int64
	err := s.db.Model(&WxGroupMessage{}).Select("COALESCE(MAX(msg_time), 0)").Scan(&maxMsgTime).Error
	if err != nil {
		s.logger.Error("获取最大消息时间失败", zap.Error(err))
		return 0, err
	}
	return maxMsgTime, nil
}


// GetGroupByGroupID 通过群组ID获取群组信息
func (s *wxRobotService) GetGroupByGroupID(groupID string) (*WxGroup, error) {
	var group WxGroup
	if err := s.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		s.logger.Error("获取群组信息失败", zap.String("group_id", groupID), zap.Error(err))
		return nil, err
	}
	return &group, nil
}

// CreateBill 创建账单
func (s *wxRobotService) CreateBill(bill *WxBillInfo) error {
	if err := s.db.Create(bill).Error; err != nil {
		s.logger.Error("创建账单失败", zap.Error(err))
		return err
	}
	s.logger.Info("账单创建成功", zap.Uint("bill_id", bill.ID))
	return nil
}


// GetBillStatistics 获取账单统计信息（分页）
func (s *wxRobotService) GetBillStatistics(req BillStatsRequest) (*BillStatsPaginatedResponse, error) {
	// 构建基础查询
	baseQuery := s.db.Model(&WxBillInfo{}).
		Select("group_id, group_name as group_nick, SUM(CAST(amount AS DECIMAL(15,2))) as total_amount, COUNT(*) as count").
		Where("owner_id = ?", req.OwnerID).
		Group("group_id, group_name")
	
	// 根据条件过滤
	if req.GroupID != "" {
		baseQuery = baseQuery.Where("group_id = ?", req.GroupID)
	}
	if req.GroupNick != "" {
		baseQuery = baseQuery.Where("group_name LIKE ?", "%"+req.GroupNick+"%")
	}
	
	// 获取总数量（从分组结果中计算）
	var totalCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) as grouped_results", 
		s.db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return baseQuery.Session(&gorm.Session{DryRun: true})
		}))
	
	if err := s.db.Raw(countQuery).Scan(&totalCount).Error; err != nil {
		s.logger.Error("获取统计总数量失败", zap.Error(err))
		return nil, err
	}
	
	// 计算分页信息
	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))
	offset := (req.PageNo - 1) * req.PageSize
	
	// 执行分页查询
	query := baseQuery.Offset(offset).Limit(req.PageSize)
	rows, err := query.Rows()
	if err != nil {
		s.logger.Error("查询账单统计失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	
	// 处理结果
	var results []BillStatsResponse
	for rows.Next() {
		var result BillStatsResponse
		var totalAmount float64
		
		err := rows.Scan(&result.GroupID, &result.GroupNick, &totalAmount, &result.Count)
		if err != nil {
			s.logger.Error("扫描统计结果失败", zap.Error(err))
			continue
		}
		
		// 格式化金额
		result.TotalAmount = fmt.Sprintf("%.2f", totalAmount)
		results = append(results, result)
	}
	
	// 构建分页信息
	pagination := PaginationInfo{
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    req.PageNo < totalPages,
		HasPrev:    req.PageNo > 1,
	}
	
	response := &BillStatsPaginatedResponse{
		List:       results,
		Pagination: pagination,
	}
	
	return response, nil
}

// GetBillList 查询账单列表（分页）
func (s *wxRobotService) GetBillList(req BillQueryRequest) (*BillQueryPaginatedResponse, error) {
	// 构建基础查询
	query := s.db.Model(&WxBillInfo{}).Where("owner_id = ?", req.OwnerID)
	
	// 根据条件过滤
	if req.CreateTimeStart != "" {
		if startTime, err := time.Parse("2006-01-02 15:04:05", req.CreateTimeStart); err == nil {
			// 将时间转换为时间戳进行比较
			startTimestamp := startTime.Unix()
			query = query.Where("msg_time >= ?", startTimestamp)
		}
	}
	
	if req.CreateTimeEnd != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", req.CreateTimeEnd); err == nil {
			// 将时间转换为时间戳进行比较
			endTimestamp := endTime.Unix()
			query = query.Where("msg_time <= ?", endTimestamp)
		}
	}
	
	if req.GroupName != "" {
		query = query.Where("group_name LIKE ?", "%"+req.GroupName+"%")
	}
	
	if req.GroupID != "" {
		query = query.Where("group_id = ?", req.GroupID)
	}
	
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	
	// 获取总数量
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		s.logger.Error("获取账单总数量失败", zap.Error(err))
		return nil, err
	}
	
	// 计算分页信息
	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))
	offset := (req.PageNum - 1) * req.PageSize
	
	// 执行分页查询
	var bills []WxBillInfo
	if err := query.Offset(offset).Limit(req.PageSize).Order("create_time DESC").Find(&bills).Error; err != nil {
		s.logger.Error("查询账单列表失败", zap.Error(err))
		return nil, err
	}
	
	// 转换为响应格式
	var results []BillInfoResponse
	for _, bill := range bills {
		result := BillInfoResponse{
			ID:         bill.ID,
			GroupName:  bill.GroupName,
			GroupID:    bill.GroupID,
			Dollar:     bill.Dollar,
			Rate:       bill.Rate,
			Amount:     bill.Amount,
			Remark:     bill.Remark,
			Operator:   bill.Operator,
			MsgTime:    bill.MsgTime,
			Status:     bill.Status,
			OwnerID:    bill.OwnerID,
			CreateTime: bill.CreateTime.Format("2006-01-02 15:04:05"),
			UpdateTime: bill.UpdateTime.Format("2006-01-02 15:04:05"),
		}
		results = append(results, result)
	}
	
	// 构建分页信息
	pagination := PaginationInfo{
		PageNo:     req.PageNum,
		PageSize:   req.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    req.PageNum < totalPages,
		HasPrev:    req.PageNum > 1,
	}
	
	response := &BillQueryPaginatedResponse{
		List:       results,
		Pagination: pagination,
	}
	
	return response, nil
}
