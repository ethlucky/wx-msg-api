package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// 响应工具函数
func (rm *RouterManager) successResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

func (rm *RouterManager) errorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Code:    -1,
		Message: message,
		Data:    nil,
	})
}

func (rm *RouterManager) badRequestResponse(c *gin.Context, message string) {
	rm.errorResponse(c, http.StatusBadRequest, message)
}

func (rm *RouterManager) notFoundResponse(c *gin.Context, message string) {
	rm.errorResponse(c, http.StatusNotFound, message)
}

func (rm *RouterManager) internalErrorResponse(c *gin.Context, message string) {
	rm.errorResponse(c, http.StatusInternalServerError, message)
}

// RouterManager 路由管理器
type RouterManager struct {
	logger              *zap.Logger
	service             WxRobotService
	messageSendStrategy MessageSendStrategy
}

// NewRouterManager 创建路由管理器
func NewRouterManager(logger *zap.Logger, service WxRobotService) *RouterManager {
	return &RouterManager{
		logger:              logger,
		service:             service,
		messageSendStrategy: NewRandomMessageSendStrategy(), // 默认使用随机策略
	}
}

// InitRoutes 初始化路由
func (rm *RouterManager) InitRoutes(cfg *Config) *gin.Engine {
	// 设置运行模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", rm.healthCheck)

	// Swagger文档路由 - 根据配置决定是否启用
	if cfg.Swagger.Enable {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		rm.logger.Info("Swagger文档已启用",
			zap.String("url", fmt.Sprintf("http://%s:%d/swagger/index.html", cfg.Swagger.Host, cfg.Swagger.Port)))
	} else {
		rm.logger.Info("Swagger文档已禁用")
	}

	// API路由组
	apiV1 := router.Group("/api/wx/v1")
	{
		// 微信机器人配置相关接口
		robots := apiV1.Group("/robots")
		{
			robots.GET("/", rm.getRobotList)               // 获取机器人列表
			robots.POST("/", rm.createRobot)               // 创建机器人配置
			robots.GET("/:id", rm.getRobotById)            // 获取单个机器人信息
			robots.PUT("/:id", rm.updateRobot)             // 修改机器人配置
			robots.GET("/:id/health", rm.checkRobotHealth) // 检查机器人健康状态
		}

		// 微信用户登录相关接口
		users := apiV1.Group("/users")
		{
			users.GET("/robot/:robotId", rm.getUsersByRobot)                 // 获取指定机器人的用户列表
			users.POST("/authorize", rm.authorizeUser)                       // 获取授权信息
			users.POST("/qrcode", rm.getQRCode)                              // 获取二维码
			users.GET("/status/:robotId/:token", rm.checkLoginStatus)        // 检查登录状态
			users.POST("/save", rm.saveUser)                                 // 保存用户数据
			users.DELETE("/:id", rm.deleteUser)                              // 删除用户
			users.GET("/login-status/:id", rm.getLoginStatus)                // 获取在线状态
			users.POST("/message-bot-status/:id", rm.updateMessageBotStatus) // 更新消息机器人状态
		}

		// 授权管理相关接口
		auth := apiV1.Group("/auth")
		{
			auth.POST("/extend/:robotId", rm.extendAuth) // 延期授权
		}

		// 消息发送相关接口
		messages := apiV1.Group("/messages/group")
		{
			messages.POST("/send-text", rm.sendText)               // 发送文本消息
			messages.POST("/send-image", rm.sendImage)             // 发送图片消息
			messages.POST("/send-text-image", rm.sendTextAndImage) // 发送文字和图片
			messages.POST("/set-strategy", rm.setMessageStrategy)  // 设置消息发送策略
		}

		// 群组管理相关接口
		groups := apiV1.Group("/groups")
		{
			groups.GET("/user/:wxId", rm.getGroupsByWxID) // 获取指定用户的群组列表
			groups.GET("/search", rm.searchGroupsByName)  // 按群名称模糊搜索群组
		}

		// 账单统计相关接口
		bills := apiV1.Group("/bills")
		{
			bills.GET("/stats", rm.getBillStatistics) // 获取账单统计信息
			bills.GET("/list", rm.getBillList)        // 查询账单列表
		}
	}

	return router
}

// healthCheck 健康检查
func (rm *RouterManager) healthCheck(c *gin.Context) {
	// 检查各个组件的健康状态
	health := gin.H{
		"status":     "ok",
		"message":    "服务正常运行",
		"timestamp":  time.Now().Format(time.RFC3339),
		"components": gin.H{},
	}

	components := health["components"].(gin.H)
	overallStatus := "ok"

	// 检查数据库连接
	if err := rm.service.CheckDatabaseHealth(); err != nil {
		components["database"] = gin.H{"status": "error", "message": "数据库连接失败", "error": err.Error()}
		overallStatus = "error"
	} else {
		components["database"] = gin.H{"status": "ok", "message": "数据库连接正常"}
	}

	// 设置整体状态
	health["status"] = overallStatus
	if overallStatus == "error" {
		health["message"] = "部分组件异常"
	}

	// 根据整体状态返回适当的HTTP状态码
	if overallStatus == "ok" {
		c.JSON(http.StatusOK, health)
	} else {
		c.JSON(http.StatusServiceUnavailable, health)
	}
}

// API处理函数

// getRobotList 获取机器人列表
// @Summary 获取机器人列表
// @Description 获取所有机器人配置及其关联的用户信息
// @Tags robots
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=[]WxRobotConfig} "查询成功"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /robots/ [get]
func (rm *RouterManager) getRobotList(c *gin.Context) {
	robots, err := rm.service.GetRobotList()
	if err != nil {
		rm.internalErrorResponse(c, "查询机器人列表失败")
		return
	}

	rm.successResponse(c, "查询成功", robots)
}

// createRobot 创建机器人配置
// @Summary 创建机器人配置
// @Description 创建新的微信机器人配置
// @Tags robots
// @Accept json
// @Produce json
// @Param robot body CreateRobotRequest true "机器人配置信息"
// @Success 200 {object} APIResponse{data=WxRobotConfig} "创建成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /robots/ [post]
func (rm *RouterManager) createRobot(c *gin.Context) {
	var req CreateRobotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 验证必填字段
	if req.Address == "" || req.AdminKey == "" || req.OwnerID == 0 {
		rm.badRequestResponse(c, "机器人地址、管理密钥和所属公司ID为必填项")
		return
	}

	// 构建 WxRobotConfig 对象
	robot := WxRobotConfig{
		Address:     req.Address,
		AdminKey:    req.AdminKey,
		OwnerID:     req.OwnerID,
		Description: req.Description,
		AdminUsers:  strings.Join(req.AdminUsers, ","), // 将数组转为逗号分隔字符串
	}

	if err := rm.service.CreateRobot(&robot); err != nil {
		rm.internalErrorResponse(c, "创建机器人配置失败")
		return
	}

	rm.successResponse(c, "创建成功", robot)
}

// getRobotById 获取单个机器人信息
// @Summary 获取单个机器人信息
// @Description 根据ID获取机器人详细信息
// @Tags robots
// @Accept json
// @Produce json
// @Param id path uint true "机器人ID"
// @Success 200 {object} APIResponse{data=WxRobotConfig} "查询成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Router /robots/{id} [get]
func (rm *RouterManager) getRobotById(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		rm.badRequestResponse(c, "机器人ID不能为空")
		return
	}

	// 解析ID
	robotId, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		rm.badRequestResponse(c, "机器人ID格式错误")
		return
	}

	robot, err := rm.service.GetRobotByID(uint(robotId))
	if err != nil {
		rm.notFoundResponse(c, "机器人不存在")
		return
	}

	rm.successResponse(c, "查询成功", robot)
}

// updateRobot 修改机器人配置
// @Summary 修改机器人配置
// @Description 更新机器人配置信息
// @Tags robots
// @Accept json
// @Produce json
// @Param id path uint true "机器人ID"
// @Param robot body UpdateRobotRequest true "机器人配置信息"
// @Success 200 {object} APIResponse{data=WxRobotConfig} "修改成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /robots/{id} [put]
func (rm *RouterManager) updateRobot(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		rm.badRequestResponse(c, "机器人ID不能为空")
		return
	}

	// 解析ID
	robotId, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		rm.badRequestResponse(c, "机器人ID格式错误")
		return
	}

	var req UpdateRobotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 验证必填字段
	if req.Address == "" || req.AdminKey == "" || req.OwnerID == 0 {
		rm.badRequestResponse(c, "机器人地址、管理密钥和所属公司ID为必填项")
		return
	}

	// 检查机器人是否存在
	existingRobot, err := rm.service.GetRobotByID(uint(robotId))
	if err != nil {
		rm.notFoundResponse(c, "机器人不存在")
		return
	}

	// 构建更新的机器人配置对象
	robot := WxRobotConfig{
		ID:          uint(robotId),
		Address:     req.Address,
		AdminKey:    req.AdminKey,
		OwnerID:     req.OwnerID,
		Description: req.Description,
		AdminUsers:  strings.Join(req.AdminUsers, ","), // 将数组转为逗号分隔字符串
		CreateTime:  existingRobot.CreateTime,          // 保留创建时间
	}

	if err := rm.service.UpdateRobot(&robot); err != nil {
		rm.internalErrorResponse(c, "修改机器人配置失败")
		return
	}

	rm.successResponse(c, "修改成功", robot)
}

// getUsersByRobot 获取指定机器人的用户列表
// @Summary 获取机器人用户列表
// @Description 获取指定机器人的所有用户登录信息
// @Tags users
// @Accept json
// @Produce json
// @Param robotId path string true "机器人ID"
// @Success 200 {object} APIResponse{data=[]WxUserLogin} "查询成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/robot/{robotId} [get]
func (rm *RouterManager) getUsersByRobot(c *gin.Context) {
	robotId := c.Param("robotId")
	if robotId == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "机器人ID不能为空",
			Data:    nil,
		})
		return
	}

	users, err := rm.service.GetUsersByRobot(robotId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code:    -1,
			Message: "查询用户列表失败",
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "查询成功",
		Data:    users,
	})
}

// authorizeUser 获取授权信息
// @Summary 获取授权信息
// @Description 为指定机器人生成授权token
// @Tags users
// @Accept json
// @Produce json
// @Param request body object{robot_id=uint} true "请求参数"
// @Success 200 {object} APIResponse{data=object{token=string,robot_id=uint}} "获取成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/authorize [post]
func (rm *RouterManager) authorizeUser(c *gin.Context) {
	var req struct {
		RobotID uint `json:"robot_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "参数错误: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查机器人是否存在
	robot, err := rm.service.GetRobotByID(req.RobotID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "机器人不存在",
			Data:    nil,
		})
		return
	}

	// 调用微信机器人API获取授权token
	authResp, err := rm.service.GenAuthKey(robot.Address, robot.AdminKey, 1, 365)
	if err != nil {
		rm.logger.Error("调用GenAuthKey失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code:    -1,
			Message: "获取授权信息失败: " + err.Error(),
			Data:    nil,
		})
		return
	}

	if len(authResp.Data) == 0 {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code:    -1,
			Message: "获取授权信息失败: 返回数据为空",
			Data:    nil,
		})
		return
	}

	authKey := authResp.Data[0]

	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "获取授权信息成功",
		Data: map[string]interface{}{
			"token":    authKey,
			"robot_id": req.RobotID,
		},
	})
}

// getQRCode 获取二维码
// @Summary 获取登录二维码
// @Description 生成微信登录二维码
// @Tags users
// @Accept json
// @Produce json
// @Param request body object{token=string,robot_id=uint} true "请求参数"
// @Success 200 {object} APIResponse{data=QRCodeResponse} "获取成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/qrcode [post]
func (rm *RouterManager) getQRCode(c *gin.Context) {
	var req struct {
		Token   string `json:"token" binding:"required"`
		RobotID uint   `json:"robot_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "参数错误: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// 获取机器人信息
	robot, err := rm.service.GetRobotByID(req.RobotID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "机器人不存在",
			Data:    nil,
		})
		return
	}

	// 调用微信机器人API获取二维码
	qrResp, err := rm.service.GetLoginQrCode(robot.Address, req.Token, false, "")
	if err != nil {
		rm.logger.Error("调用GetLoginQrCode失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code:    -1,
			Message: "获取二维码失败: " + err.Error(),
			Data:    nil,
		})
		return
	}

	qrResponse := QRCodeResponse{
		QRCode:       qrResp.Data.QrCodeUrl,
		Token:        req.Token,
		ExpireTime:   time.Now().Add(5 * time.Minute).Unix(),
		QrCodeBase64: qrResp.Data.QrCodeBase64,
	}

	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "获取二维码成功",
		Data:    qrResponse,
	})
}

// checkLoginStatus 检查登录状态（仅检查，不保存）
// @Summary 检查登录状态
// @Description 检查用户扫码登录状态
// @Tags users
// @Accept json
// @Produce json
// @Param robotId path string true "机器人ID"
// @Param token path string true "授权token"
// @Success 200 {object} APIResponse{data=LoginStatusResponse} "检查成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/status/{robotId}/{token} [get]
func (rm *RouterManager) checkLoginStatus(c *gin.Context) {
	robotIdStr := c.Param("robotId")
	token := c.Param("token")

	if robotIdStr == "" || token == "" {
		rm.badRequestResponse(c, "robotId和token不能为空")
		return
	}

	robotId, err := strconv.ParseUint(robotIdStr, 10, 32)
	if err != nil {
		rm.badRequestResponse(c, "robotId参数错误")
		return
	}

	// 获取机器人信息
	robot, err := rm.service.GetRobotByID(uint(robotId))
	if err != nil {
		rm.notFoundResponse(c, "机器人不存在")
		return
	}

	// 调用微信机器人API检查登录状态
	loginResp, err := rm.service.CheckLoginStatus(robot.Address, token)
	if err != nil {
		rm.logger.Error("调用CheckLoginStatus失败", zap.Error(err))
		rm.internalErrorResponse(c, "检查登录状态失败: "+err.Error())
		return
	}

	// 根据外部API响应的Code判断状态
	var status LoginStatusResponse
	switch loginResp.Code {
	case 200:
		// Code 200时还需要检查state字段，只有state为2才是真正的登录成功
		if loginResp.Data.State == 2 {
			// 登录成功，包含完整用户信息
			status = LoginStatusResponse{
				Status:   2,
				WxID:     loginResp.Data.WxID,
				NickName: loginResp.Data.NickName,
				Message:  "登录成功",
			}
		} else {
			// Code 200但state不为2，视为二维码已过期或不存在
			status = LoginStatusResponse{
				Status:  0,
				Message: "二维码已过期或不存在",
			}
		}
	case 300:
		// 不存在状态（二维码过期或其他原因）
		status = LoginStatusResponse{
			Status:  0,
			Message: "二维码已过期或不存在",
		}
	default:
		// 其他错误状态
		status = LoginStatusResponse{
			Status:  3,
			Message: "检查登录状态失败",
		}
	}

	rm.successResponse(c, "检查成功", status)
}

// saveUser 保存用户数据
// @Summary 保存用户数据
// @Description 保存用户登录信息到数据库
// @Tags users
// @Accept json
// @Produce json
// @Param request body SaveUserRequest true "用户数据"
// @Success 200 {object} APIResponse{data=WxUserLogin} "保存成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/save [post]
func (rm *RouterManager) saveUser(c *gin.Context) {
	var req SaveUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 检查机器人是否存在
	robot, err := rm.service.GetRobotByID(req.RobotID)
	if err != nil {
		rm.notFoundResponse(c, "关联的机器人不存在")
		return
	}

	// 检查是否有安全风险
	hasRisk := req.HasSecurityRisk
	if hasRisk == 0 {
		riskResp, err := rm.service.CheckCanSetAlias(robot.Address, req.Token)
		if err == nil {
			for _, result := range riskResp.Data.Results {
				if !result.IsPass {
					hasRisk = 1
					break
				}
			}
		}
	}

	// 构建用户数据
	user := WxUserLogin{
		RobotID:         req.RobotID,
		Token:           req.Token,
		WxID:            req.WxID,
		NickName:        req.NickName,
		ExtensionTime:   time.Now().Add(24 * time.Hour * 365),
		ExpirationTime:  time.Now().Add(24 * time.Hour * 365),
		HasSecurityRisk: hasRisk,
		Status:          1,
		IsMessageBot:    req.IsMessageBot,
	}

	if err := rm.service.SaveUser(&user); err != nil {
		rm.internalErrorResponse(c, "保存用户数据失败")
		return
	}

	rm.successResponse(c, "保存成功", user)
}

// deleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户（不删除关联的群组数据）
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/{id} [delete]
func (rm *RouterManager) deleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		rm.badRequestResponse(c, "用户ID不能为空")
		return
	}

	if err := rm.service.DeleteUser(id); err != nil {
		rm.internalErrorResponse(c, "删除用户失败")
		return
	}

	rm.successResponse(c, "删除成功", nil)
}

// getLoginStatus 获取在线状态
// @Summary 获取用户在线状态
// @Description 获取用户当前的在线状态
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} APIResponse "获取成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "用户不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/login-status/{id} [get]
func (rm *RouterManager) getLoginStatus(c *gin.Context) {
	idStr := c.Param("id")

	if idStr == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "id不能为空",
			Data:    nil,
		})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "id参数错误",
			Data:    nil,
		})
		return
	}

	// 通过用户ID获取用户信息
	user, err := rm.service.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "用户不存在",
			Data:    nil,
		})
		return
	}

	// 获取机器人信息
	robot, err := rm.service.GetRobotByID(user.RobotID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "关联的机器人不存在",
			Data:    nil,
		})
		return
	}

	// 调用微信机器人API获取登录状态
	statusResp, err := rm.service.GetLoginStatus(robot.Address, user.Token)
	if err != nil {
		rm.logger.Error("调用GetLoginStatus失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code:    -1,
			Message: "获取登录状态失败: " + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "获取成功",
		Data:    statusResp,
	})
}

// extendAuth 延期授权
// @Summary 延期授权
// @Description 延长机器人授权有效期
// @Tags auth
// @Accept json
// @Produce json
// @Param robotId path string true "机器人ID"
// @Param request body object{days=int} true "延期天数"
// @Success 200 {object} APIResponse "延期成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人或用户不存在"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /auth/extend/{robotId} [post]
func (rm *RouterManager) extendAuth(c *gin.Context) {
	robotIdStr := c.Param("robotId")

	var req struct {
		Days int `json:"days" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "参数错误: " + err.Error(),
			Data:    nil,
		})
		return
	}

	if robotIdStr == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "robotId不能为空",
			Data:    nil,
		})
		return
	}

	robotId, err := strconv.ParseUint(robotIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Code:    -1,
			Message: "robotId参数错误",
			Data:    nil,
		})
		return
	}

	// 获取机器人信息
	robot, err := rm.service.GetRobotByID(uint(robotId))
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "机器人不存在",
			Data:    nil,
		})
		return
	}

	// 从robot关联的用户中获取token（假设取第一个有效用户的token）
	users, err := rm.service.GetUsersByRobot(robotIdStr)
	if err != nil || len(users) == 0 {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "未找到该机器人关联的用户",
			Data:    nil,
		})
		return
	}

	// 取第一个有效用户的token
	var token string
	for _, user := range users {
		if user.Status == 1 && user.Token != "" { // 状态正常且有token
			token = user.Token
			break
		}
	}

	if token == "" {
		c.JSON(http.StatusNotFound, APIResponse{
			Code:    -1,
			Message: "未找到有效的用户token",
			Data:    nil,
		})
		return
	}

	// 调用微信机器人API延期授权
	extendResp, err := rm.service.DelayAuthKey(robot.Address, robot.AdminKey, token, req.Days)
	if err != nil {
		rm.logger.Error("调用DelayAuthKey失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code:    -1,
			Message: "延期授权失败: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// 更新数据库中的用户延期时间
	newExpiry, _ := time.Parse("2006-01-02", extendResp.Data.ExpiryDate)
	rm.service.UpdateUserExtension(uint(robotId), token, newExpiry)

	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "延期成功",
		Data:    extendResp,
	})
}

// sendText 发送文本消息
// @Summary 发送文本消息
// @Description 向指定群组发送文本消息
// @Tags messages
// @Accept json
// @Produce json
// @Param request body object{text_content=string,to_user_name=string} true "文本消息参数"
// @Success 200 {object} APIResponse "发送成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "未找到消息机器人"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /messages/group/send-text [post]
func (rm *RouterManager) sendText(c *gin.Context) {
	var req struct {
		TextContent string `json:"text_content" binding:"required"`
		ToUserName  string `json:"to_user_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 通过策略获取消息机器人信息
	botInfo, err := rm.service.GetMessageBotByStrategy(req.ToUserName, rm.messageSendStrategy)
	if err != nil {
		rm.notFoundResponse(c, "未找到对应的消息机器人")
		return
	}

	// 构建发送请求
	sendReq := &SendTextRequest{
		TextContent: req.TextContent,
		ToUserName:  req.ToUserName,
	}

	// 调用服务发送文本消息
	resp, err := rm.service.SendText(botInfo.Robot.Address, botInfo.User.Token, sendReq)
	if err != nil {
		rm.logger.Error("发送文本消息失败", zap.Error(err))
		rm.internalErrorResponse(c, "发送文本消息失败: "+err.Error())
		return
	}

	rm.successResponse(c, "文本消息发送成功", resp)
}

// sendImage 发送图片消息
// @Summary 发送图片消息
// @Description 向指定群组发送图片消息
// @Tags messages
// @Accept json
// @Produce json
// @Param request body object{image_content=string,to_user_name=string} true "图片消息参数"
// @Success 200 {object} APIResponse "发送成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "未找到消息机器人"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /messages/group/send-image [post]
func (rm *RouterManager) sendImage(c *gin.Context) {
	var req struct {
		ImageContent string `json:"image_content" binding:"required"`
		ToUserName   string `json:"to_user_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 通过策略获取消息机器人信息
	botInfo, err := rm.service.GetMessageBotByStrategy(req.ToUserName, rm.messageSendStrategy)
	if err != nil {
		rm.notFoundResponse(c, "未找到对应的消息机器人")
		return
	}

	// 构建发送请求
	sendReq := &SendImageRequest{
		ImageContent: req.ImageContent,
		ToUserName:   req.ToUserName,
	}

	// 调用服务发送图片消息
	resp, err := rm.service.SendImage(botInfo.Robot.Address, botInfo.User.Token, sendReq)
	if err != nil {
		rm.logger.Error("发送图片消息失败", zap.Error(err))
		rm.internalErrorResponse(c, "发送图片消息失败: "+err.Error())
		return
	}

	rm.successResponse(c, "图片消息发送成功", resp)
}

// sendTextAndImage 同时发送文字和图片
// @Summary 发送文本和图片消息
// @Description 向指定群组同时发送文本和图片消息
// @Tags messages
// @Accept json
// @Produce json
// @Param request body object{text_content=string,image_content=string,to_user_name=string} true "混合消息参数"
// @Success 200 {object} APIResponse "发送成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "未找到消息机器人"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /messages/group/send-text-image [post]
func (rm *RouterManager) sendTextAndImage(c *gin.Context) {
	var req struct {
		TextContent  string `json:"text_content"`
		ImageContent string `json:"image_content"`
		ToUserName   string `json:"to_user_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 检查至少有一个内容不为空
	if req.TextContent == "" && req.ImageContent == "" {
		rm.badRequestResponse(c, "文本内容和图片内容不能都为空")
		return
	}

	// 通过策略获取消息机器人信息
	botInfo, err := rm.service.GetMessageBotByStrategy(req.ToUserName, rm.messageSendStrategy)
	if err != nil {
		rm.notFoundResponse(c, "未找到对应的消息机器人")
		return
	}

	// 构建发送请求
	sendReq := &SendTextAndImageRequest{
		TextContent:  req.TextContent,
		ImageContent: req.ImageContent,
		ToUserName:   req.ToUserName,
	}

	// 调用服务发送文字和图片
	resp, err := rm.service.SendTextAndImage(botInfo.Robot.Address, botInfo.User.Token, sendReq)
	if err != nil {
		rm.logger.Error("发送文字和图片失败", zap.Error(err))
		rm.internalErrorResponse(c, "发送文字和图片失败: "+err.Error())
		return
	}

	rm.successResponse(c, "消息发送完成", resp)
}

// setMessageStrategy 设置消息发送策略
// @Summary 设置消息发送策略
// @Description 设置系统的消息发送策略（随机或轮询）
// @Tags messages
// @Accept json
// @Produce json
// @Param request body object{strategy=string} true "策略参数 (random/round_robin)"
// @Success 200 {object} APIResponse "设置成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Router /messages/group/set-strategy [post]
func (rm *RouterManager) setMessageStrategy(c *gin.Context) {
	var req struct {
		Strategy string `json:"strategy" binding:"required"` // round_robin, random
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	switch req.Strategy {
	case "round_robin":
		rm.messageSendStrategy = NewRoundRobinMessageSendStrategy()
		rm.logger.Info("消息发送策略已切换为: 轮询")
	case "random":
		rm.messageSendStrategy = NewRandomMessageSendStrategy()
		rm.logger.Info("消息发送策略已切换为: 随机")
	default:
		rm.badRequestResponse(c, "无效的策略类型，支持: round_robin, random")
		return
	}

	rm.successResponse(c, "策略设置成功", map[string]string{
		"strategy": req.Strategy,
	})
}

// getGroupsByWxID 获取指定用户的群组列表
// @Summary 获取用户群组列表
// @Description 获取指定微信用户的所有群组信息
// @Tags groups
// @Accept json
// @Produce json
// @Param wxId path string true "微信ID"
// @Success 200 {object} APIResponse{data=[]WxGroup} "查询成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /groups/user/{wxId} [get]
func (rm *RouterManager) getGroupsByWxID(c *gin.Context) {
	wxId := c.Param("wxId")
	if wxId == "" {
		rm.badRequestResponse(c, "微信ID不能为空")
		return
	}

	groups, err := rm.service.GetGroupsByWxID(wxId)
	if err != nil {
		rm.internalErrorResponse(c, "查询用户群组列表失败")
		return
	}

	rm.successResponse(c, "查询成功", groups)
}

// searchGroupsByName 按群名称模糊搜索群组
// @Summary 搜索群组
// @Description 根据群名称进行模糊搜索
// @Tags groups
// @Accept json
// @Produce json
// @Param groupNickName query string true "群名称"
// @Success 200 {object} APIResponse{data=[]WxGroup} "搜索成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /groups/search [get]
func (rm *RouterManager) searchGroupsByName(c *gin.Context) {
	groupNickName := c.Query("groupNickName")
	if groupNickName == "" {
		rm.badRequestResponse(c, "群名称参数不能为空")
		return
	}

	groups, err := rm.service.SearchGroupsByName(groupNickName)
	if err != nil {
		rm.internalErrorResponse(c, "搜索群组失败")
		return
	}

	rm.successResponse(c, "搜索成功", groups)
}

// updateMessageBotStatus 更新消息机器人状态
// @Summary 更新消息机器人状态
// @Description 设置用户是否为消息机器人
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{is_message_bot=int} true "消息机器人状态"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /users/message-bot-status/{id} [post]
func (rm *RouterManager) updateMessageBotStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		rm.badRequestResponse(c, "ID不能为空")
		return
	}

	// 解析ID
	parsedId, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		rm.badRequestResponse(c, "ID格式错误")
		return
	}

	var req struct {
		IsMessageBot int `json:"is_message_bot"` // 0不是 1是
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 验证参数值
	if req.IsMessageBot != 0 && req.IsMessageBot != 1 {
		rm.badRequestResponse(c, "is_message_bot参数必须为0或1")
		return
	}

	// 调用服务更新消息机器人状态
	if err := rm.service.UpdateMessageBotStatus(uint(parsedId), req.IsMessageBot); err != nil {
		rm.internalErrorResponse(c, "更新消息机器人状态失败")
		return
	}

	rm.successResponse(c, "更新成功", map[string]interface{}{
		"id":             uint(parsedId),
		"is_message_bot": req.IsMessageBot,
	})
}

// getBillStatistics 获取账单统计信息（分页）
// @Summary 获取账单统计信息（分页）
// @Description 根据群组ID和群组昵称获取账单统计信息，按group_id和group_name分组统计金额总数，支持分页
// @Tags bills
// @Accept json
// @Produce json
// @Param group_id query string false "群组ID"
// @Param group_nick query string false "群组昵称"
// @Param page_no query int false "页码，默认1" default(1) minimum(1)
// @Param page_size query int false "每页大小，默认10" default(10) minimum(1) maximum(100)
// @Param owner_id query uint true "所属公司ID"
// @Success 200 {object} APIResponse{data=BillStatsPaginatedResponse} "获取成功"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 500 {object} APIResponse "内部服务器错误"
// @Router /bills/stats [get]
func (rm *RouterManager) getBillStatistics(c *gin.Context) {
	var req BillStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.PageNo <= 0 {
		req.PageNo = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	stats, err := rm.service.GetBillStatistics(req)
	if err != nil {
		rm.internalErrorResponse(c, "获取账单统计失败")
		return
	}

	rm.successResponse(c, "获取成功", stats)
}

// @Summary 查询账单列表
// @Description 根据条件查询账单信息，支持分页
// @Tags bills
// @Accept json
// @Produce json
// @Param create_time_start query string false "创建时间开始，格式：yyyy-mm-dd hh:mi:ss"
// @Param create_time_end query string false "创建时间结束，格式：yyyy-mm-dd hh:mi:ss"
// @Param group_name query string false "群名称"
// @Param group_id query string false "群ID"
// @Param status query string false "账单状态"
// @Param page_num query int false "页码，默认1"
// @Param page_size query int false "每页大小，默认10，最大100"
// @Param owner_id query uint true "所属公司ID"
// @Success 200 {object} APIResponse{data=BillQueryPaginatedResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /bills/list [get]
func (rm *RouterManager) getBillList(c *gin.Context) {
	var req BillQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		rm.badRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	billList, err := rm.service.GetBillList(req)
	if err != nil {
		rm.internalErrorResponse(c, "查询账单列表失败")
		return
	}

	rm.successResponse(c, "查询成功", billList)
}

// checkRobotHealth 检查机器人健康状态
// @Summary 检查机器人健康状态
// @Description 通过HTTP请求检查指定机器人的健康状态
// @Tags robots
// @Accept json
// @Produce json
// @Param id path string true "机器人ID"
// @Success 200 {object} APIResponse{data=object{status=string,address=string,response_time=string}} "机器人健康"
// @Failure 400 {object} APIResponse "参数错误"
// @Failure 404 {object} APIResponse "机器人不存在"
// @Failure 503 {object} APIResponse "机器人不健康"
// @Router /robots/{id}/health [get]
func (rm *RouterManager) checkRobotHealth(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		rm.badRequestResponse(c, "机器人ID不能为空")
		return
	}

	// 解析ID
	robotId, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		rm.badRequestResponse(c, "机器人ID格式错误")
		return
	}

	// 获取机器人信息
	robot, err := rm.service.GetRobotByID(uint(robotId))
	if err != nil {
		rm.notFoundResponse(c, "机器人不存在")
		return
	}

	// 检查机器人健康状态
	startTime := time.Now()
	isHealthy, err := rm.service.CheckRobotHealth(robot.Address)
	responseTime := time.Since(startTime)

	if err != nil {
		rm.logger.Error("检查机器人健康状态失败",
			zap.String("address", robot.Address),
			zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, APIResponse{
			Code:    -1,
			Message: "机器人不健康: " + err.Error(),
			Data: map[string]interface{}{
				"status":        "unhealthy",
				"address":       robot.Address,
				"response_time": responseTime.String(),
				"error":         err.Error(),
			},
		})
		return
	}

	if !isHealthy {
		c.JSON(http.StatusServiceUnavailable, APIResponse{
			Code:    -1,
			Message: "机器人不健康",
			Data: map[string]interface{}{
				"status":        "unhealthy",
				"address":       robot.Address,
				"response_time": responseTime.String(),
			},
		})
		return
	}

	rm.successResponse(c, "机器人健康", map[string]interface{}{
		"status":        "healthy",
		"address":       robot.Address,
		"response_time": responseTime.String(),
	})
}
