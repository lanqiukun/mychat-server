package main

type Message struct {
	Id                  uint64 `json:"id,omitempty"`
	Type                uint8  `json:"type,omitempty"`
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
	Id        uint64 `json:"id,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	WsToken   string `json:"ws_token,omitempty"`
}

type ClientResponse struct {
	ResponseType uint `json:"response_type"`
	ClientNotification
	ClientMessage
	Status uint8  `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ServerMessage struct {
	Message
	Sender_id   uint64 `json:"sender_id"`
	Receiver_id uint64 `json:"receiver_id"`
}
