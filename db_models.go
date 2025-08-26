package main

import (
	"time"
)

// 数据库模型
type WxRobotConfig struct {
	ID          uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	Address     string        `json:"address" gorm:"type:varchar(255);not null;comment:机器人地址"`
	AdminKey    string        `json:"admin_key" gorm:"type:varchar(255);not null;comment:管理密钥"`
	OwnerID     uint          `json:"owner_id" gorm:"not null;comment:所属公司ID"`
	Description string        `json:"description" gorm:"type:varchar(500);comment:文本描述"`
	AdminUsers  string        `json:"admin_users" gorm:"type:text;comment:管理员用户列表，用逗号分隔"`
	CreateTime  time.Time     `json:"create_time" gorm:"autoCreateTime;comment:创建时间"`
	UpdateTime  time.Time     `json:"update_time" gorm:"autoUpdateTime;comment:修改时间"`
	UserLogins  []WxUserLogin `json:"user_logins" gorm:"foreignKey:RobotID"`
}

func (WxRobotConfig) TableName() string {
	return "wx_robot_configs"
}

type WxUserLogin struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	RobotID         uint      `json:"robot_id" gorm:"not null;comment:关联的机器人ID"`
	Token           string    `json:"token" gorm:"type:varchar(500);comment:登录令牌"`
	WxID            string    `json:"wx_id" gorm:"type:varchar(100);comment:微信ID"`
	NickName        string    `json:"nick_name" gorm:"type:varchar(100);comment:微信昵称"`
	ExtensionTime   time.Time `json:"extension_time" gorm:"comment:延期时间"`
	HasSecurityRisk int       `json:"has_security_risk" gorm:"default:0;comment:是否有安全风险 0否 1是"`
	ExpirationTime  time.Time `json:"expiration_time" gorm:"comment:过期时间"`
	Status          int       `json:"status" gorm:"default:1;comment:状态 1正常 2风控 3需要重新登录"`
	IsInitialized   int       `json:"is_initialized" gorm:"default:0;comment:是否初始化完成 0未初始化 1初始化完成"`
	IsMessageBot    int       `json:"is_message_bot" gorm:"default:0;comment:是否是消息机器人 0不是 1是"`
	CreateTime      time.Time `json:"create_time" gorm:"autoCreateTime;comment:创建时间"`
	UpdateTime      time.Time `json:"update_time" gorm:"autoUpdateTime;comment:修改时间"`
}

func (WxUserLogin) TableName() string {
	return "wx_user_logins"
}

type WxGroup struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	WxID          string    `json:"wx_id" gorm:"type:varchar(100);not null;comment:微信ID"`
	GroupID       string    `json:"group_id" gorm:"type:varchar(100);not null;comment:群组ID"`
	GroupNickName string    `json:"group_nick_name" gorm:"type:varchar(200);comment:群组昵称"`
	CreateTime    time.Time `json:"create_time" gorm:"autoCreateTime;comment:创建时间"`
	UpdateTime    time.Time `json:"update_time" gorm:"autoUpdateTime;comment:修改时间"`
}

func (WxGroup) TableName() string {
	return "wx_groups"
}

type WxBillInfo struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupName   string    `json:"group_name" gorm:"type:varchar(50);not null;comment:群组名称"`
	GroupID     string    `json:"group_id" gorm:"type:varchar(50);not null;comment:群组Id"`
	Dollar      string    `json:"dollar" gorm:"type:varchar(20);comment:金额(外币)"`
	Rate        string    `json:"rate" gorm:"type:varchar(20);comment:汇率"`
	Amount      string    `json:"amount" gorm:"type:decimal(15,2);comment:金额(RMB)"`
	Remark      string    `json:"remark" gorm:"type:text;comment:备注"`
	Operator    string    `json:"operator" gorm:"type:varchar(20);comment:操作人名称"`
	MsgTime     int64     `json:"msg_time" gorm:"comment:账单时间"`
	Status      string    `json:"status" gorm:"type:char(2);comment:清账状态(0 为未清账, 1 为已清账)"`
	OwnerID     uint      `json:"owner_id" gorm:"not null;comment:所属公司ID"`
	CreateTime  time.Time `json:"create_time" gorm:"autoCreateTime;comment:创建时间"`
	UpdateTime  time.Time `json:"update_time" gorm:"autoUpdateTime;comment:修改时间"`
}

func (WxBillInfo) TableName() string {
	return "wx_bill_info"
}


type WxGroupMessage struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID    string    `json:"group_id" gorm:"type:varchar(100);not null;comment:群组ID"`
	WxNickName string    `json:"wx_nick_name" gorm:"type:varchar(100);not null;comment:微信昵称"`
	Content    string    `json:"content" gorm:"type:text;not null;comment:消息内容"`
	MsgType    int       `json:"msg_type" gorm:"not null;comment:消息类型"`
	MsgTime    int64     `json:"msg_time" gorm:"not null;comment:消息时间戳"`
	OwnerID    uint      `json:"owner_id" gorm:"not null;comment:所属公司ID"`
	CreateTime time.Time `json:"create_time" gorm:"autoCreateTime;comment:创建时间"`
	UpdateTime time.Time `json:"update_time" gorm:"autoUpdateTime;comment:修改时间"`
}

func (WxGroupMessage) TableName() string {
	return "wx_group_messages"
}