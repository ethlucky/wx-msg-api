package main

// API响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type QRCodeResponse struct {
	QRCode        string `json:"qr_code"`
	Token         string `json:"token"`
	ExpireTime    int64  `json:"expire_time"`
	QrCodeBase64  string `json:"qrCodeBase64"`
}

type LoginStatusResponse struct {
	Status   int    `json:"status"` // 0未扫码 1已扫码 2登录成功 3登录失败
	WxID     string `json:"wx_id"`
	NickName string `json:"nick_name"`
	Message  string `json:"message"`
}

// DTO 对象用于保存操作
type SaveUserRequest struct {
	RobotID         uint   `json:"robot_id" binding:"required"`
	Token           string `json:"token" binding:"required"`
	WxID            string `json:"wx_id" binding:"required"`
	NickName        string `json:"nick_name"`
	HasSecurityRisk int    `json:"has_security_risk"`
	IsMessageBot    int    `json:"is_message_bot"`
}

// 创建机器人配置请求
type CreateRobotRequest struct {
	Address     string   `json:"address" binding:"required"`
	AdminKey    string   `json:"admin_key" binding:"required"`
	OwnerID     uint     `json:"owner_id" binding:"required"`
	Description string   `json:"description"`
	AdminUsers  []string `json:"admin_users"`
}

// 更新机器人配置请求
type UpdateRobotRequest struct {
	Address     string   `json:"address" binding:"required"`
	AdminKey    string   `json:"admin_key" binding:"required"`
	OwnerID     uint     `json:"owner_id" binding:"required"`
	Description string   `json:"description"`
	AdminUsers  []string `json:"admin_users"`
}

// 账单统计请求
type BillStatsRequest struct {
	GroupID   string `form:"group_id"`
	GroupNick string `form:"group_nick"`
	PageNo    int    `form:"page_no,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=10" binding:"min=1,max=100"`
	OwnerID   uint   `form:"owner_id" binding:"required"`
}

// 账单统计响应
type BillStatsResponse struct {
	GroupID     string `json:"group_id"`
	GroupNick   string `json:"group_nick"`
	TotalAmount string `json:"total_amount"`
	Count       int64  `json:"count"`
}

// 分页信息
type PaginationInfo struct {
	PageNo     int   `json:"page_no"`
	PageSize   int   `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// 账单统计分页响应
type BillStatsPaginatedResponse struct {
	List       []BillStatsResponse `json:"list"`
	Pagination PaginationInfo      `json:"pagination"`
}

// 账单查询请求
type BillQueryRequest struct {
	CreateTimeStart string `form:"create_time_start"` // 创建时间开始，格式：yyyy-mm-dd hh:mi:ss
	CreateTimeEnd   string `form:"create_time_end"`   // 创建时间结束，格式：yyyy-mm-dd hh:mi:ss
	GroupName       string `form:"group_name"`        // 群名称
	GroupID         string `form:"group_id"`          // 群ID
	Status          string `form:"status"`            // 账单状态
	PageNum         int    `form:"page_num,default=1" binding:"min=1"`
	PageSize        int    `form:"page_size,default=10" binding:"min=1,max=100"`
	OwnerID         uint   `form:"owner_id" binding:"required"`
}

// 账单信息响应
type BillInfoResponse struct {
	ID          uint   `json:"id"`
	GroupName   string `json:"group_name"`
	GroupID     string `json:"group_id"`
	Dollar      string `json:"dollar"`
	Rate        string `json:"rate"`
	Amount      string `json:"amount"`
	Remark      string `json:"remark"`
	Operator    string `json:"operator"`
	MsgTime     int64  `json:"msg_time"`
	Status      string `json:"status"`
	OwnerID     uint   `json:"owner_id"`
	CreateTime  string `json:"create_time"`
	UpdateTime  string `json:"update_time"`
}

// 账单查询分页响应
type BillQueryPaginatedResponse struct {
	List       []BillInfoResponse `json:"list"`
	Pagination PaginationInfo     `json:"pagination"`
}