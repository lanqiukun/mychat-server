package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const authorId = 63495249

type ClientRequest struct {
	Code string `json:"code"`
}

func cors(w *http.ResponseWriter, r *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Headers", "DNT,X-Mx-ReqToken,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, TRACE, CONNECT, OPTIONS")
	(*w).Header().Set("Content-Type", "application/json")
}

func getidtoken(w http.ResponseWriter, r *http.Request) {

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
		user.Token = token
		user.Avatar = ghui.AvatarUrl

		var authorRelation1 Relationship
		var authorRelation2 Relationship

		authorRelation1.Id1 = user.Id
		authorRelation1.Id2 = authorId
		authorRelation1.Alias = "Mychat Author"
		authorRelation1.CreatedAt = time.Now().Unix()

		authorRelation2.Id1 = authorId
		authorRelation2.Id2 = user.Id
		authorRelation2.Alias = ""
		authorRelation2.CreatedAt = time.Now().Unix()

		db.Create(&user)
		db.Create(&authorRelation1)
		db.Create(&authorRelation2)
	} else {
		//用户已经在数据库中
		db.Table("users").Where("id = (?)", user.Id).Update("token", token)
	}

	//生成client response
	clientResponse.Status = 0
	clientResponse.Id = user.Id
	clientResponse.AvatarUrl = user.Avatar
	clientResponse.NickName = user.Nickname
	clientResponse.Token = strconv.FormatUint(token, 10)

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

//如果请求合法,则相应成功信息并返回websocket连接给调用者
//如果失败则返回响应失败信息和error并关闭websocket连接
func establishWsConn(w http.ResponseWriter, r *http.Request) {
	authenticate(w, r)
}

func authenticate(w http.ResponseWriter, r *http.Request) error {

	var fatalError error = nil

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		println(err.Error())
		return err
	}

	var tempResponse ClientResponse
	tempResponse.ResponseType = 5
	tempResponse.Status = 0

	defer func() {
		if tempResponseJson, err := json.Marshal(tempResponse); err == nil {
			conn.WriteMessage(websocket.TextMessage, tempResponseJson)
		}

		if fatalError != nil {
			conn.Close()
		}
	}()

	id := r.URL.Query().Get("id")
	token := r.URL.Query().Get("token")

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		println("invalid id parsed")
		tempResponse.Status = 1
		tempResponse.Reason = "用户id不合法，解析失败"

		fatalError = fmt.Errorf(tempResponse.Reason)
		return fatalError
	}

	wsToken, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		println("invalid token parsed")
		tempResponse.Status = 1
		tempResponse.Reason = "用户token不合法，解析失败"

		fatalError = fmt.Errorf(tempResponse.Reason)
		return fatalError
	}

	db, err := getdb()
	if err != nil {
		println(err.Error())
		fatalError = err
		return fatalError
	}
	defer db.Close()

	var user User

	db.Find(&user, userId)

	if user.Id == 0 {
		println("no such user")
		tempResponse.Status = 1
		tempResponse.Reason = "用户不存在"

		fatalError = fmt.Errorf(tempResponse.Reason)
		return fatalError
	}

	if user.Token != wsToken {
		println("invalid token")
		tempResponse.Status = 1
		tempResponse.Reason = "用户token不正确"

		fatalError = fmt.Errorf(tempResponse.Reason)
		return fatalError
	}

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

		var clientMessage ClientMessage
		for _, sm := range unread {
			clientMessage.Message = sm.Message
			clientMessage.Sender_str_id = strconv.FormatUint(sm.Sender_id, 10)
			clientMessage.Receiver_str_id = strconv.FormatUint(sm.Receiver_id, 10)

			cmpl.Lock()
			cmp <- clientMessage
			cmpl.Unlock()
		}
		go readPump(userId, conn)

	} else {
		println("user was already online")
		tempResponse.Status = 1
		tempResponse.Reason = "用户已经在线"

		fatalError = fmt.Errorf(tempResponse.Reason)
		return fatalError
	}

	return nil
}
