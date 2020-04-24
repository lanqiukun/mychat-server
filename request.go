package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type ClientRequest struct {
	Code string `json"code"`
}

func cors(w *http.ResponseWriter, r *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Headers", "DNT,X-Mx-ReqToken,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, TRACE, CONNECT, OPTIONS")
	(*w).Header().Set("Content-Type", "application/json")
}

func wstokenuserid(w http.ResponseWriter, r *http.Request) {

	cors(&w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		println("preflight request")
	}

	if r.Method != "POST" {
		w.WriteHeader(405)
		println("no post request")
		return
	}

	var clientRequest ClientRequest
	var clientResponse ClientResponse

	defer func() {
		clientResponseJson, err := json.Marshal(clientResponse)
		if err != nil {
			println(err.Error())
		}

		//发送client response
		w.Write([]byte(clientResponseJson))
	}()

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&clientRequest)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = err.Error()
		return
	}

	ght, err := requestGitHubToken(clientRequest.Code)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = err.Error()
		return
	}

	ghui, err := getGitHubUserInfo(ght)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = err.Error()
		return
	}

	db, err := getdb()
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = "发生数据库错误"
		return
	}
	defer db.Close()

	var user User
	db.Find(&user, ghui.Id)

	token := rand.Uint64()
	if user.Id == 0 {
		//用户不在数据库中
		user.Id = ghui.Id
		user.Nickname = ghui.Login
		user.WsToken = token
		user.Avatar = ghui.AvatarUrl

		var authorRelation Relationship

		authorRelation.Id1 = user.Id
		authorRelation.Id2 = 98688141287751680
		authorRelation.Alias = "Mychat Author"
		authorRelation.CreatedAt = time.Now().Unix()

		db.Create(&user)
		db.Create(&authorRelation)
	} else {
		//用户已经在数据库中
		db.Table("users").Where("id = (?)", user.Id).Update("ws_token", token)
	}

	//生成client response
	clientResponse.Status = 0
	clientResponse.Id = user.Id
	clientResponse.AvatarUrl = user.Avatar
	clientResponse.NickName = user.Nickname
	clientResponse.WsToken = strconv.FormatUint(token, 10)

}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func establishwsconn(w http.ResponseWriter, r *http.Request) {

	var tempResponse ClientResponse
	tempResponse.ResponseType = 5
	tempResponse.Status = 0

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		println("create temp connection faild")
	}

	id := r.URL.Query().Get("id")
	token := r.URL.Query().Get("token")

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		println("invalid token parsed")
		tempResponse.Status = 1
		tempResponse.Reason = "用户id不合法，解析失败"
	}

	println("new ws request2")

	wsToken, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		println("invalid token parsed")
		tempResponse.Status = 1
		tempResponse.Reason = "用户token不合法，解析失败"
	}

	println("new ws request3")

	db, err := getdb()
	if err != nil {
		println(err.Error())
		return
	}
	defer db.Close()

	println("new ws request4")

	var user User

	db.Find(&user, id)

	if user.Id == 0 {
		println("no such user")
		tempResponse.Status = 1
		tempResponse.Reason = "用户不存在"
	}

	if user.WsToken != wsToken {
		println("invalid token")
		tempResponse.Status = 1
		tempResponse.Reason = "用户token不正确"
	}

	tempResponseJson, err := json.Marshal(conn)

	//不管用户提供的身份信息是否合法,都响应一下用户
	if err := conn.WriteMessage(websocket.TextMessage, tempResponseJson); err != nil {
		return
	}

	//如果发现已知错误,则关闭websocket连接
	if tempResponse.Status == 1 {
		conn.Close()
		return
	}

	println("new ws request5")

	println("new ws request6")

	cpl.Lock()
	_, ok := cp[userId]
	cpl.Unlock()

	//用户刚上线，查找属于它的未读消息并将查找结果放入消息池
	if !ok {

		cpl.Lock()
		cp[userId] = conn
		cpl.Unlock()

		var unread []ServerMessage

		db.Where("receiver_id = ? AND received_at = 0", userId).Limit(100).Find(&unread)
		println("new ws request7")

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
		return
	}
	println("new ws request8")

}
