package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func closeConnection(userId uint64) {
	connectionPoolLock.Lock()
	conn, ok := connectionPool[userId]
	if ok {
		conn.Close()
		delete(connectionPool, userId)
	}
	connectionPoolLock.Unlock()
}

func readPump(userId uint64, conn *websocket.Conn) {

	defer closeConnection(userId)

	conn.SetReadLimit(200)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var clientMessage ClientMessage //client message
		println(string(message))
		println("我拿到一条消息")

		err = json.Unmarshal(message, &clientMessage)
		if err != nil {
			println("json unmarshal failed")
			return
		}

		// println(cm.Id)
		// println(cm.Sender_str_id)
		// println(cm.Receiver_str_id)
		// println(cm.Type)
		// println(cm.Body)
		// println(cm.Created_at)
		// println(cm.Modified_at)
		// println(cm.Sender_deleted_at)
		// println(cm.Receiver_deleted_at)
		// println(cm.Withdrawn_at)

		clientMessage.Created_at = time.Now().Unix()
		clientMessage.Id = 0

		clientMessagePoolLock.Lock()
		clientMessagePool <- clientMessage
		clientMessagePoolLock.Unlock()
	}
}

func writePump() {
	for clientMessage := range clientMessagePool {
		var err error
		var serverMessage ServerMessage

		//将clientMessage转为serverMessage
		serverMessage.Sender_id, err = strconv.ParseUint(clientMessage.Sender_str_id, 10, 64)
		if err != nil {
			return
		}
		serverMessage.Receiver_id, err = strconv.ParseUint(clientMessage.Receiver_str_id, 10, 64)
		if err != nil {
			return
		}

		serverMessage.Message = clientMessage.Message

		connectionPoolLock.RLock()
		targetConnection, ok := connectionPool[serverMessage.Receiver_id]
		connectionPoolLock.RUnlock()

		db, err := getdb()
		defer db.Close()
		if err != nil {
			println(err.Error())
			return
		}

		var clientResponse ClientResponse

		clientResponse.ClientMessage = clientMessage
		print("message type is: ")
		println(clientResponse.ClientMessage.Type)
		clientResponse.ResponseType = 0
		clientResponse.Status = 0

		if ok {
			msgJson, err := json.Marshal(clientResponse)
			if err != nil {
				continue
			}

			//尝试向目标用户发送消息
			writeLock.Lock()
			err = targetConnection.WriteMessage(websocket.TextMessage, msgJson)
			writeLock.Unlock()
			if err == nil {
				serverMessage.Received_at = time.Now().Unix()
				println("我推了一条消息出去")
			}

		}

		//如果存在，则更新数据库，否则新增
		// var temp ServerMessage
		if serverMessage.Id != 0 {

			db.Table("server_messages").Where("id = (?)", serverMessage.Id).Update("received_at", time.Now().Unix())
			// db.Table("server_messages").Where("id = (?)", sm.Id).Updates(map[string]interface{}{"received_at": 18})
			// db.Table("server_messages").Where("id IN (?)", []int{10, 11}).Updates(map[string]interface{}{"received_at": 0, "age": 18})
		} else {

			db.Create(&serverMessage)
		}

	}
}
