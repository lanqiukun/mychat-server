package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
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
	cpl.Lock()
	conn, ok := cp[userId]
	if ok {
		conn.Close()
		delete(cp, userId)
	}
	cpl.Unlock()
}

func readPump(userId uint64, conn *websocket.Conn) {

	defer closeConnection(userId)

	conn.SetReadLimit(maxMessageSize)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var cm ClientMessage //client message
		println(string(message))
		println("我拿到一条消息")

		err = json.Unmarshal(message, &cm)
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

		cm.Created_at = time.Now().Unix()
		cm.Id = 0
		cmp <- cm

	}
}

func writePump() {
	for cm := range cmp {
		var err error
		var sm ServerMessage

		sm.Sender_id, err = strconv.ParseUint(cm.Sender_str_id, 10, 64)
		if err != nil {
			return
		}
		sm.Receiver_id, err = strconv.ParseUint(cm.Receiver_str_id, 10, 64)
		if err != nil {
			return
		}

		//将cm转为smgs
		sm.Message = cm.Message

		cpl.RLock()
		targetConnection, ok := cp[sm.Receiver_id]
		cpl.RUnlock()

		db, err := gorm.Open("mysql", "root:Edison939878#@(lowb.top:3306)/mychat?charset=utf8&parseTime=True&loc=Local")
		if err != nil {
			println(err.Error())
			return
		}

		if ok {
			msgJson, err := json.Marshal(cm)
			if err != nil {
				continue
			}

			//尝试向目标用户发送消息
			cpl.Lock()
			err = targetConnection.WriteMessage(websocket.TextMessage, []byte(msgJson))
			cpl.Unlock()
			if err == nil {
				sm.Received_at = time.Now().Unix()
			}

		}

		//如果存在，则更新数据库，否则新增
		// var temp ServerMessage
		if sm.Id != 0 {

			db.Table("server_messages").Where("id = (?)", sm.Id).Update("received_at", time.Now().Unix())
			// db.Table("server_messages").Where("id = (?)", sm.Id).Updates(map[string]interface{}{"received_at": 18})
			// db.Table("server_messages").Where("id IN (?)", []int{10, 11}).Updates(map[string]interface{}{"received_at": 0, "age": 18})
		} else {

			db.Create(&sm)
		}

		db.Close()

	}
}

func serveWs(w http.ResponseWriter, r *http.Request, userId uint64) {

	cpl.Lock()
	_, ok := cp[userId]
	cpl.Unlock()

	//用户刚上线，查找属于它的未读消息并将查找结果放入消息池
	if !ok {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		cpl.Lock()
		cp[userId] = conn
		cpl.Unlock()

		db, err := gorm.Open("mysql", "root:123@(192.168.31.73:3306)/mychat?charset=utf8&parseTime=True&loc=Local")
		if err != nil {
			println(err.Error())
			return
		}
		defer db.Close()

		var unread []ServerMessage

		db.Where("receiver_id = ? AND received_at = 0", userId).Limit(100).Find(&unread)

		var cm ClientMessage
		for _, sm := range unread {
			cm.Message = sm.Message
			cm.Sender_str_id = strconv.FormatUint(sm.Sender_id, 10)
			cm.Receiver_str_id = strconv.FormatUint(sm.Receiver_id, 10)

			cmpl.Lock()
			cmp <- cm
			cmpl.Unlock()
		}
		go readPump(userId, conn)

	} else {
		println("already")
	}

}
