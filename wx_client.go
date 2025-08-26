package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// 外部微信机器人API响应结构
type ExternalAPIResponse struct {
	Code int         `json:"Code"`
	Data interface{} `json:"Data"`
	Text string      `json:"Text"`
}

type GenAuthKeyRequest struct {
	Count int `json:"Count"`
	Days  int `json:"Days"`
}

type GenAuthKeyResponse struct {
	Code int      `json:"Code"`
	Data []string `json:"Data"`
	Text string   `json:"Text"`
}

type GetLoginQrCodeRequest struct {
	Check bool   `json:"Check"`
	Proxy string `json:"Proxy"`
}

type GetLoginQrCodeResponse struct {
	Code int `json:"Code"`
	Data struct {
		QrCodeUrl     string `json:"QrCodeUrl"`
		Txt           string `json:"Txt"`
		BaseResp      struct {
			Ret    int         `json:"ret"`
			ErrMsg interface{} `json:"errMsg"`
		} `json:"baseResp"`
		DeviceInfo    struct {
			DeviceBrand string `json:"deviceBrand"`
			DeviceName  string `json:"deviceName"`
			Imei        string `json:"imei"`
		} `json:"deviceInfo"`
		ExpiredTime   int    `json:"expiredTime"`
		QrCodeBase64  string `json:"qrCodeBase64"`
		UUID          string `json:"uuid"`
	} `json:"Data"`
	Text string `json:"Text"`
}

type CheckCanSetAliasResponse struct {
	Code int `json:"Code"`
	Data struct {
		BaseResponse struct {
			Ret    int `json:"ret"`
			ErrMsg struct {
				Str string `json:"str"`
			} `json:"errMsg"`
		} `json:"BaseResponse"`
		Results []struct {
			Title  string `json:"title"`
			Desc   string `json:"desc,omitempty"`
			Result string `json:"result"`
			IsPass bool   `json:"isPass"`
		} `json:"results"`
		Ticket     string `json:"ticket"`
		VerifyType int    `json:"verifyType"`
	} `json:"Data"`
	Text string `json:"Text"`
}

type CheckLoginStatusResponse struct {
	Code int `json:"Code"`
	Data struct {
		UUID                    string `json:"uuid"`
		State                   int    `json:"state"`
		WxID                    string `json:"wxid"`
		WxNewPass               string `json:"wxnewpass"`
		HeadImgUrl              string `json:"head_img_url"`
		PushLoginUrlExpiredTime int    `json:"push_login_url_expired_time"`
		NickName                string `json:"nick_name"`
		EffectiveTime           int    `json:"effective_time"`
		Unknow                  int    `json:"unknow"`
		Device                  string `json:"device"`
		Ret                     int    `json:"ret"`
		OthersInServerLogin     bool   `json:"othersInServerLogin"`
		TarGetServerIp          string `json:"tarGetServerIp"`
		UuId                    string `json:"uuId"`
		Msg                     string `json:"msg"`
	} `json:"Data"`
	Text string `json:"Text"`
}

type GetLoginStatusResponse struct {
	Code int `json:"Code"`
	Data struct {
		ExpiryTime   string `json:"expiryTime"`
		LoginErrMsg  string `json:"loginErrMsg"`
		LoginJournal struct {
			Count int           `json:"count"`
			Logs  []interface{} `json:"logs"`
		} `json:"loginJournal"`
		LoginState  int    `json:"loginState"`
		LoginTime   string `json:"loginTime"`
		OnlineDays  int    `json:"onlineDays"`
		OnlineTime  string `json:"onlineTime"`
		ProxyUrl    string `json:"proxyUrl"`
		TargetIp    string `json:"targetIp"`
		TotalOnline string `json:"totalOnline"`
	} `json:"Data"`
	Text string `json:"Text"`
}

type GetInitStatusResponse struct {
	Code int    `json:"Code"`
	Data bool   `json:"Data"`
	Text string `json:"Text"`
}

type DelayAuthKeyRequest struct {
	Days       int    `json:"Days"`
	ExpiryDate string `json:"ExpiryDate"`
	Key        string `json:"Key"`
}

type DelayAuthKeyResponse struct {
	Code int `json:"Code"`
	Data struct {
		ExpiryDate string `json:"expiryDate"`
	} `json:"Data"`
	Text string `json:"Text"`
}

type GetChatRoomInfoRequest struct {
	ChatRoomWxIdList []string `json:"ChatRoomWxIdList"`
}

type GetChatRoomInfoResponse struct {
	Code int `json:"Code"`
	Data struct {
		BaseResponse struct {
			Ret    int         `json:"ret"`
			ErrMsg interface{} `json:"errMsg"`
		} `json:"baseResponse"`
		ContactCount int `json:"contactCount"`
		ContactList  []struct {
			UserName struct {
				Str string `json:"str"`
			} `json:"userName"`
			NickName struct {
				Str string `json:"str"`
			} `json:"nickName"`
			ChatRoomOwner   string `json:"chatRoomOwner"`
			SmallHeadImgUrl string `json:"smallHeadImgUrl"`
			NewChatroomData struct {
				MemberCount        int `json:"member_count"`
				ChatroomMemberList []struct {
					UserName           string `json:"user_name"`
					NickName           string `json:"nick_name"`
					ChatroomMemberFlag int    `json:"chatroom_member_flag"`
					Unknow             string `json:"unknow,omitempty"`
				} `json:"chatroom_member_list"`
			} `json:"newChatroomData"`
		} `json:"contactList"`
	} `json:"Data"`
	Text string `json:"Text"`
}

type GroupListResponse struct {
	Code int `json:"Code"`
	Data struct {
		GroupList []struct {
			UserName struct {
				Str string `json:"str"`
			} `json:"userName"`
			NickName struct {
				Str string `json:"str"`
			} `json:"nickName"`
			ChatRoomOwner   string `json:"chatRoomOwner"`
			NewChatroomData struct {
				MemberCount        int `json:"member_count"`
				ChatroomMemberList []struct {
					UserName           string `json:"user_name"`
					NickName           string `json:"nick_name,omitempty"`
					ChatroomMemberFlag int    `json:"chatroom_member_flag"`
				} `json:"chatroom_member_list"`
			} `json:"newChatroomData"`
		} `json:"GroupList"`
		IsInitFinished bool `json:"IsInitFinished"`
		Count          int  `json:"count"`
	} `json:"Data"`
	Text string `json:"Text"`
}

// 微信API客户端
type WxAPIClient struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewWxAPIClient 创建新的微信API客户端
func NewWxAPIClient(logger *zap.Logger) *WxAPIClient {
	return &WxAPIClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// HTTP请求通用方法
func (c *WxAPIClient) makeRequest(method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.Debug("发送HTTP请求", zap.String("method", method), zap.String("url", url))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	c.logger.Debug("收到HTTP响应", zap.Int("status", resp.StatusCode), zap.Int("body_length", len(respBody)))

	return respBody, nil
}

// 判断响应是否成功
func (c *WxAPIClient) isSuccess(code int) bool {
	return code == 200
}

// 生成授权码
func (c *WxAPIClient) GenAuthKey(robotAddress, adminKey string, count, days int) (*GenAuthKeyResponse, error) {
	url := fmt.Sprintf("%s/admin/GenAuthKey1?key=%s", robotAddress, adminKey)
	reqBody := GenAuthKeyRequest{
		Count: count,
		Days:  days,
	}

	respBody, err := c.makeRequest("POST", url, reqBody)
	if err != nil {
		c.logger.Error("调用GenAuthKey失败", zap.Error(err))
		return nil, err
	}

	var resp GenAuthKeyResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析GenAuthKey响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("GenAuthKey调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("GenAuthKey调用成功", zap.Int("count", len(resp.Data)))
	return &resp, nil
}

// 获取登录二维码
func (c *WxAPIClient) GetLoginQrCode(robotAddress, authKey string, check bool, proxy string) (*GetLoginQrCodeResponse, error) {
	url := fmt.Sprintf("%s/login/GetLoginQrCodeNewX?key=%s", robotAddress, authKey)
	reqBody := GetLoginQrCodeRequest{
		Check: check,
		Proxy: proxy,
	}

	respBody, err := c.makeRequest("POST", url, reqBody)
	if err != nil {
		c.logger.Error("调用GetLoginQrCode失败", zap.Error(err))
		return nil, err
	}

	var resp GetLoginQrCodeResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析GetLoginQrCode响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("GetLoginQrCode调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("GetLoginQrCode调用成功")
	return &resp, nil
}

// 检查是否有安全风险
func (c *WxAPIClient) CheckCanSetAlias(robotAddress, authKey string) (*CheckCanSetAliasResponse, error) {
	url := fmt.Sprintf("%s/login/CheckCanSetAlias?key=%s", robotAddress, authKey)

	respBody, err := c.makeRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("调用CheckCanSetAlias失败", zap.Error(err))
		return nil, err
	}

	var resp CheckCanSetAliasResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析CheckCanSetAlias响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 对于CheckCanSetAlias，Code 200表示正常，Code 300表示需要重新登录，都是正常响应
	if resp.Code != 200 && resp.Code != 300 {
		c.logger.Warn("CheckCanSetAlias调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("CheckCanSetAlias调用成功")
	return &resp, nil
}

// 检查登录状态
func (c *WxAPIClient) CheckLoginStatus(robotAddress, authKey string) (*CheckLoginStatusResponse, error) {
	url := fmt.Sprintf("%s/login/CheckLoginStatus?key=%s", robotAddress, authKey)

	respBody, err := c.makeRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("调用CheckLoginStatus失败", zap.Error(err))
		return nil, err
	}

	var resp CheckLoginStatusResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析CheckLoginStatus响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 对于CheckLoginStatus，Code 200表示登录成功，Code 300表示不存在状态，都是正常响应
	if resp.Code != 200 && resp.Code != 300 {
		c.logger.Warn("CheckLoginStatus调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("CheckLoginStatus调用成功", zap.String("wxid", resp.Data.WxID), zap.Int("state", resp.Data.State))

	return &resp, nil
}

// 获取登录状态
func (c *WxAPIClient) GetLoginStatus(robotAddress, authKey string) (*GetLoginStatusResponse, error) {
	url := fmt.Sprintf("%s/login/GetLoginStatus?key=%s", robotAddress, authKey)

	respBody, err := c.makeRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("调用GetLoginStatus失败", zap.Error(err))
		return nil, err
	}

	var resp GetLoginStatusResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析GetLoginStatus响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("GetLoginStatus调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("GetLoginStatus调用成功", zap.Int("loginState", resp.Data.LoginState))
	return &resp, nil
}

// 检查初始化状态
func (c *WxAPIClient) GetInitStatus(robotAddress, authKey string) (*GetInitStatusResponse, error) {
	url := fmt.Sprintf("%s/login/GetInItStatus?key=%s", robotAddress, authKey)

	respBody, err := c.makeRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("调用GetInitStatus失败", zap.Error(err))
		return nil, err
	}

	var resp GetInitStatusResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析GetInitStatus响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("GetInitStatus调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("GetInitStatus调用成功", zap.Bool("isInitFinished", resp.Data))
	return &resp, nil
}

// 授权码延期
func (c *WxAPIClient) DelayAuthKey(robotAddress, adminKey, authKey string, days int) (*DelayAuthKeyResponse, error) {
	url := fmt.Sprintf("%s/admin/DelayAuthKey?key=%s", robotAddress, adminKey)
	reqBody := DelayAuthKeyRequest{
		Days:       days,
		ExpiryDate: "",
		Key:        authKey,
	}

	respBody, err := c.makeRequest("POST", url, reqBody)
	if err != nil {
		c.logger.Error("调用DelayAuthKey失败", zap.Error(err))
		return nil, err
	}

	var resp DelayAuthKeyResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析DelayAuthKey响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("DelayAuthKey调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("DelayAuthKey调用成功", zap.String("expiryDate", resp.Data.ExpiryDate))
	return &resp, nil
}

// 获取群详情
func (c *WxAPIClient) GetChatRoomInfo(robotAddress, authKey string, chatRoomIds []string) (*GetChatRoomInfoResponse, error) {
	url := fmt.Sprintf("%s/group/GetChatRoomInfo?key=%s", robotAddress, authKey)
	reqBody := GetChatRoomInfoRequest{
		ChatRoomWxIdList: chatRoomIds,
	}

	respBody, err := c.makeRequest("POST", url, reqBody)
	if err != nil {
		c.logger.Error("调用GetChatRoomInfo失败", zap.Error(err))
		return nil, err
	}

	var resp GetChatRoomInfoResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析GetChatRoomInfo响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("GetChatRoomInfo调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("GetChatRoomInfo调用成功", zap.Int("contactCount", resp.Data.ContactCount))
	return &resp, nil
}

// 获取群列表
func (c *WxAPIClient) GetGroupList(robotAddress, authKey string) (*GroupListResponse, error) {
	url := fmt.Sprintf("%s/group/GroupList?key=%s", robotAddress, authKey)

	respBody, err := c.makeRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("调用GetGroupList失败", zap.Error(err))
		return nil, err
	}

	var resp GroupListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		c.logger.Error("解析GetGroupList响应失败", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !c.isSuccess(resp.Code) {
		c.logger.Warn("GetGroupList调用失败", zap.Int("code", resp.Code), zap.String("text", resp.Text))
		return &resp, fmt.Errorf("API调用失败: %s", resp.Text)
	}

	c.logger.Info("GetGroupList调用成功", zap.Int("groupCount", len(resp.Data.GroupList)), zap.Bool("isInitFinished", resp.Data.IsInitFinished))
	return &resp, nil
}

// 消息发送相关结构体和接口

// SendTextRequest 发送文本消息请求（简化版）
type SendTextRequest struct {
	TextContent string `json:"TextContent"` // 文本内容
	ToUserName  string `json:"ToUserName"`  // 接收者用户名
}

// SendTextResponse 发送文本消息响应（简化版）
type SendTextResponse struct {
	ToUserName  string `json:"ToUserName"`
	ClientMsgId int64  `json:"ClientMsgId"`
	CreateTime  int64  `json:"CreateTime"`
	NewMsgId    int64  `json:"NewMsgId"`
}

// SendImageRequest 发送图片消息请求（简化版）
type SendImageRequest struct {
	ImageContent string `json:"ImageContent"` // 图片内容(base64)
	ToUserName   string `json:"ToUserName"`   // 接收者用户名
}

// SendImageResponse 发送图片消息响应（简化版）
type SendImageResponse struct {
	MsgId        int64  `json:"MsgId"`
	FromUserName string `json:"FromUserName"`
	ToUserName   string `json:"ToUserName"`
	CreateTime   int64  `json:"CreateTime"`
	NewMsgId     int64  `json:"NewMsgId"`
}

// SendTextAndImageRequest 同时发送文字和图片请求
type SendTextAndImageRequest struct {
	TextContent  string `json:"TextContent"`  // 文本内容
	ImageContent string `json:"ImageContent"` // 图片内容(base64)
	ToUserName   string `json:"ToUserName"`   // 接收者用户名
}

// SendTextAndImageResponse 同时发送文字和图片响应
type SendTextAndImageResponse struct {
	Success bool   `json:"Success"` // 发送是否成功
	Message string `json:"Message"` // 失败原因或成功消息
}

// 内部使用的复杂结构体（从原代码提取）

// SendTextMsgItem 文本消息项
type SendTextMsgItem struct {
	AtWxIDList   []string `json:"AtWxIDList"`   // @用户列表
	ImageContent string   `json:"ImageContent"` // 图片内容(通常为空)
	MsgType      int      `json:"MsgType"`      // 消息类型
	TextContent  string   `json:"TextContent"`  // 文本内容
	ToUserName   string   `json:"ToUserName"`   // 接收者用户名
}

// SendTextMessageRequest 发送文本消息请求
type SendTextMessageRequest struct {
	MsgItem []SendTextMsgItem `json:"MsgItem"`
}

// SendTextMessageRawResponse 原始发送文本消息响应
type SendTextMessageRawResponse struct {
	Code int    `json:"Code"`
	Text string `json:"Text"`
	Data []struct {
		IsSendSuccess bool   `json:"isSendSuccess"`
		TextContent   string `json:"textContent"`
		ToUserName    string `json:"toUSerName"`
		Resp          *struct {
			BaseResponse struct {
				Ret    int `json:"ret"`
				ErrMsg struct {
					Str string `json:"str,omitempty"`
				} `json:"errMsg"`
			} `json:"base_response"`
			Count           int `json:"count"`
			ChatSendRetList []struct {
				Ret        int `json:"ret"`
				ToUserName struct {
					Str string `json:"str"`
				} `json:"toUserName"`
				MsgId       int64 `json:"msgId"`
				ClientMsgId int64 `json:"clientMsgId"`
				CreateTime  int64 `json:"createTime"`
				ServerTime  int64 `json:"serverTime"`
				Type        int   `json:"type"`
				NewMsgId    int64 `json:"newMsgId"`
			} `json:"chat_send_ret_list"`
		} `json:"resp,omitempty"`
	} `json:"Data"`
}

// SendImageMsgItem 图片消息项
type SendImageMsgItem struct {
	AtWxIDList   []string `json:"AtWxIDList"`   // @用户列表
	ImageContent string   `json:"ImageContent"` // 图片内容(base64)
	MsgType      int      `json:"MsgType"`      // 消息类型
	TextContent  string   `json:"TextContent"`  // 文本内容
	ToUserName   string   `json:"ToUserName"`   // 接收者用户名
}

// SendImageNewMessageRequest 发送图片消息请求
type SendImageNewMessageRequest struct {
	MsgItem []SendImageMsgItem `json:"MsgItem"`
}

// SendImageNewMessageRawResponse 原始发送图片消息响应
type SendImageNewMessageRawResponse struct {
	Code int    `json:"Code"`
	Text string `json:"Text"`
	Data []struct {
		ErrMsg        string `json:"errMsg,omitempty"`
		ImageId       string `json:"imageId"`
		IsSendSuccess bool   `json:"isSendSuccess,omitempty"`
		ToUserName    string `json:"toUSerName"`
		Resp          *struct {
			BaseResponse struct {
				Ret    int `json:"ret"`
				ErrMsg struct {
					Str string `json:"str,omitempty"`
				} `json:"errMsg"`
			} `json:"baseResponse"`
			MsgId       int64 `json:"msgId"`
			ClientImgId struct {
				Str string `json:"str"`
			} `json:"clientImgId"`
			FromUserName struct {
				Str string `json:"str"`
			} `json:"fromUserName"`
			ToUserName struct {
				Str string `json:"str"`
			} `json:"toUserName"`
			TotalLen   int    `json:"totalLen"`
			StartPos   int    `json:"startPos"`
			DataLen    int    `json:"dataLen"`
			CreateTime int64  `json:"createTime"`
			NewMsgId   int64  `json:"newMsgId"`
			MsgSource  string `json:"msgSource"`
		} `json:"resp,omitempty"`
	} `json:"Data"`
}

// SendText 发送文本消息（简化版）
func (c *WxAPIClient) SendText(robotAddress, authKey string, req *SendTextRequest) (*SendTextResponse, error) {
	url := fmt.Sprintf("%s/message/SendTextMessage?key=%s", robotAddress, authKey)

	// 构建原始请求
	originalReq := &SendTextMessageRequest{
		MsgItem: []SendTextMsgItem{
			{
				AtWxIDList:   []string{},
				ImageContent: "",
				MsgType:      1, // 文本消息类型
				TextContent:  req.TextContent,
				ToUserName:   req.ToUserName,
			},
		},
	}

	jsonData, err := json.Marshal(originalReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %w", err)
	}

	c.logger.Info("发送文本消息请求",
		zap.String("url", url),
		zap.String("to_user", req.ToUserName),
		zap.Int("text_length", len(req.TextContent)))

	reqBody, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	reqBody.Header.Set("Content-Type", "application/json")
	reqBody.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(reqBody)
	if err != nil {
		return nil, fmt.Errorf("SendText 发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("SendText 读取响应数据失败: %w", err)
	}

	var rawResponse SendTextMessageRawResponse
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("SendText 解析响应数据失败: %w", err)
	}

	c.logger.Info("SendText 发送文本消息响应",
		zap.Int("code", rawResponse.Code),
		zap.Int("data_count", len(rawResponse.Data)))

	// 检查是否有数据返回
	if len(rawResponse.Data) == 0 {
		return nil, fmt.Errorf("SendText 发送文本消息失败: 无响应数据")
	}

	// 检查第一个结果的发送状态
	firstResult := rawResponse.Data[0]

	// 检查是否发送成功
	if !firstResult.IsSendSuccess {
		return nil, fmt.Errorf("SendText 发送文本消息失败: 发送状态为失败")
	}

	// 检查是否有Resp字段
	if firstResult.Resp == nil {
		return nil, fmt.Errorf("SendText 发送文本消息失败: 响应数据不完整")
	}

	// 检查响应状态
	if firstResult.Resp.BaseResponse.Ret != 0 {
		errMsg := firstResult.Resp.BaseResponse.ErrMsg.Str
		if errMsg == "" {
			errMsg = "未知错误"
		}
		return nil, fmt.Errorf("SendText 发送文本消息失败: %s", errMsg)
	}

	// 检查chat_send_ret_list是否有数据
	if len(firstResult.Resp.ChatSendRetList) == 0 {
		return nil, fmt.Errorf("SendText 发送文本消息失败: 无发送结果数据")
	}

	// 获取第一个发送结果
	sendRet := firstResult.Resp.ChatSendRetList[0]

	// 检查发送结果状态
	if sendRet.Ret != 0 {
		return nil, fmt.Errorf("SendText 发送文本消息失败: 发送结果状态码 %d", sendRet.Ret)
	}

	// 构建成功响应
	response := &SendTextResponse{
		ToUserName:  sendRet.ToUserName.Str,
		ClientMsgId: sendRet.ClientMsgId,
		CreateTime:  sendRet.CreateTime,
		NewMsgId:    sendRet.NewMsgId,
	}

	c.logger.Info("文本消息发送成功",
		zap.String("to_user", response.ToUserName),
		zap.Int64("client_msg_id", response.ClientMsgId),
		zap.Int64("create_time", response.CreateTime),
		zap.Int64("new_msg_id", response.NewMsgId))

	return response, nil
}

// SendImage 发送图片消息（简化版）
func (c *WxAPIClient) SendImage(robotAddress, authKey string, req *SendImageRequest) (*SendImageResponse, error) {
	url := fmt.Sprintf("%s/message/SendImageNewMessage?key=%s", robotAddress, authKey)

	// 构建原始请求
	originalReq := &SendImageNewMessageRequest{
		MsgItem: []SendImageMsgItem{
			{
				AtWxIDList:   []string{},
				ImageContent: req.ImageContent,
				MsgType:      3, // 图片消息类型
				TextContent:  "",
				ToUserName:   req.ToUserName,
			},
		},
	}

	jsonData, err := json.Marshal(originalReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %w", err)
	}

	c.logger.Info("发送图片消息请求",
		zap.String("url", url),
		zap.String("to_user", req.ToUserName),
		zap.Int("image_size", len(req.ImageContent)))

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应数据失败: %w", err)
	}

	var rawResponse SendImageNewMessageRawResponse
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("解析响应数据失败: %w", err)
	}

	c.logger.Info("发送图片消息响应",
		zap.Int("code", rawResponse.Code),
		zap.Int("data_count", len(rawResponse.Data)))

	// 检查是否有数据返回
	if len(rawResponse.Data) == 0 {
		return nil, fmt.Errorf("发送图片消息失败: 无响应数据")
	}

	// 检查第一个结果的发送状态
	firstResult := rawResponse.Data[0]

	// 如果有错误消息或发送失败标志
	if firstResult.ErrMsg != "" {
		return nil, fmt.Errorf("发送图片消息失败: %s", firstResult.ErrMsg)
	}

	// 检查是否有Resp字段且发送成功
	if firstResult.Resp == nil {
		return nil, fmt.Errorf("发送图片消息失败: 响应数据不完整")
	}

	// 检查响应状态
	if firstResult.Resp.BaseResponse.Ret != 0 {
		errMsg := firstResult.Resp.BaseResponse.ErrMsg.Str
		if errMsg == "" {
			errMsg = "未知错误"
		}
		return nil, fmt.Errorf("发送图片消息失败: %s", errMsg)
	}

	// 构建成功响应
	response := &SendImageResponse{
		MsgId:        firstResult.Resp.MsgId,
		FromUserName: firstResult.Resp.FromUserName.Str,
		ToUserName:   firstResult.Resp.ToUserName.Str,
		CreateTime:   firstResult.Resp.CreateTime,
		NewMsgId:     firstResult.Resp.NewMsgId,
	}

	c.logger.Info("图片消息发送成功",
		zap.Int64("msg_id", response.MsgId),
		zap.String("from_user", response.FromUserName),
		zap.String("to_user", response.ToUserName),
		zap.Int64("create_time", response.CreateTime),
		zap.Int64("new_msg_id", response.NewMsgId))

	return response, nil
}

// SendTextAndImage 同时发送文字和图片
func (c *WxAPIClient) SendTextAndImage(robotAddress, authKey string, req *SendTextAndImageRequest) (*SendTextAndImageResponse, error) {
	// 检查输入参数
	hasText := req.TextContent != ""
	hasImage := req.ImageContent != ""

	// 如果两者都为空，返回错误
	if !hasText && !hasImage {
		return &SendTextAndImageResponse{
			Success: false,
			Message: "文本内容和图片内容不能都为空",
		}, fmt.Errorf("SendTextAndImage 文本内容和图片内容不能都为空")
	}

	var textResp *SendTextResponse
	var imageResp *SendImageResponse
	var textErr, imageErr error

	// 根据条件发送文本消息
	if hasText {
		textReq := &SendTextRequest{
			TextContent: req.TextContent,
			ToUserName:  req.ToUserName,
		}

		textResp, textErr = c.SendText(robotAddress, authKey, textReq)
		if textErr != nil {
			c.logger.Error("SendTextAndImage 发送文本消息失败",
				zap.String("to_user", req.ToUserName),
				zap.Error(textErr))
		}
	}

	// 根据条件发送图片消息
	if hasImage {
		imageReq := &SendImageRequest{
			ImageContent: req.ImageContent,
			ToUserName:   req.ToUserName,
		}

		imageResp, imageErr = c.SendImage(robotAddress, authKey, imageReq)
		if imageErr != nil {
			c.logger.Error("SendTextAndImage 发送图片消息失败",
				zap.String("to_user", req.ToUserName),
				zap.Error(imageErr))
		}
	}

	// 判断整体是否成功并构建失败原因
	success := true
	var failureReasons []string

	if hasText && textErr != nil {
		success = false
		failureReasons = append(failureReasons, fmt.Sprintf("文本消息发送失败: %v", textErr))
	}
	if hasImage && imageErr != nil {
		success = false
		failureReasons = append(failureReasons, fmt.Sprintf("图片消息发送失败: %v", imageErr))
	}

	// 构建响应消息
	var message string
	if success {
		message = "消息发送成功"
	} else {
		message = strings.Join(failureReasons, "; ")
	}

	// 构建组合响应
	response := &SendTextAndImageResponse{
		Success: success,
		Message: message,
	}

	if success {
		logFields := []zap.Field{
			zap.String("to_user", req.ToUserName),
			zap.Bool("sent_text", hasText),
			zap.Bool("sent_image", hasImage),
		}
		if textResp != nil {
			logFields = append(logFields, zap.Int64("text_msg_id", textResp.NewMsgId))
		}
		if imageResp != nil {
			logFields = append(logFields, zap.Int64("image_msg_id", imageResp.NewMsgId))
		}

		c.logger.Info("SendTextAndImage 消息发送成功", logFields...)
	} else {
		c.logger.Warn("SendTextAndImage 部分消息发送失败",
			zap.String("to_user", req.ToUserName),
			zap.Bool("text_failed", hasText && textErr != nil),
			zap.Bool("image_failed", hasImage && imageErr != nil))
	}

	return response, nil
}

// CheckRobotHealth 检查机器人健康状态
func (c *WxAPIClient) CheckRobotHealth(robotAddress string) (bool, error) {
	// 确保地址以http://或https://开头
	if !strings.HasPrefix(robotAddress, "http://") && !strings.HasPrefix(robotAddress, "https://") {
		robotAddress = "http://" + robotAddress
	}

	// 发送简单的GET请求检查机器人状态
	req, err := http.NewRequest("GET", robotAddress, nil)
	if err != nil {
		c.logger.Error("创建健康检查请求失败",
			zap.String("robot_address", robotAddress),
			zap.Error(err))
		return false, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置超时时间
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error("健康检查请求失败",
			zap.String("robot_address", robotAddress),
			zap.Error(err))
		return false, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，200表示健康
	isHealthy := resp.StatusCode == http.StatusOK

	c.logger.Info("机器人健康检查完成",
		zap.String("robot_address", robotAddress),
		zap.Int("status_code", resp.StatusCode),
		zap.Bool("is_healthy", isHealthy))

	return isHealthy, nil
}
