package main

type Message struct {
	Id   uint64 `json:"id,omitempty"`
	Type uint8  `json:"type"` //这里不能加omitempty,否则type为0时将不产生type字段
	//文本类型0
	//文件类型1
	//图片类型2
	//视频类型3
	//音频类型4
	Body                string `json:"body,omitempty"`
	Created_at          int64  `json:"created_at,omitempty"`
	Modified_at         int64  `json:"modified_at,omitempty"`
	Sender_deleted_at   int64  `json:"sender_deleted_at,omitempty"`
	Receiver_deleted_at int64  `json:"receiver_deleted_at,omitempty"`
	Withdrawn_at        int64  `json:"withdrawn_at,omitempty"`
	Received_at         int64  `json:"Received_at,omitempty"`
}

type ClientMessage struct {
	Message
	Sender_str_id   string `json:"sender_str_id,omitempty"`
	Receiver_str_id string `json:"receiver_str_id,omitempty"`
}

type ClientNotification struct {
	NickName  string `json:"nickname,omitempty"`
	StrId     string `json:"strid,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	Token     string `json:"token,omitempty"`
}

type ClientResponse struct {
	//好友消息0
	//好友请求1
	//群消息2
	//群请求3
	//系统消息4
	//连接状态消息5
	//好友列表请求6
	//一般请求/响应7
	//请求资源8
	//更新资源9
	ResponseType uint `json:"response_type"`
	ClientNotification
	ClientMessage

	//正常0	失败1
	Status uint8  `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type ServerMessage struct {
	Message
	Sender_id   uint64 `json:"sender_id"`
	Receiver_id uint64 `json:"receiver_id"`
}
