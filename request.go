package main

import (
	"encoding/json"
	"fmt"
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

func cors(w *http.ResponseWriter, r *http.Request) bool {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Headers", "DNT,X-Mx-ReqToken,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, TRACE, CONNECT, OPTIONS")

	//如果将要返回的是json
	//(*w).Header().Set("Content-Type", "application/json")

	//如果是跨域预检请求，那就别再继续执行。
	if r.Method == "OPTIONS" {
		(*w).WriteHeader(200)
		return true
	}

	return false
}

type EmailPassword struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

//如果用户提交的邮箱或密码正确，则返回id和令牌
func verifyemailpassword(w http.ResponseWriter, r *http.Request) {
	if isPreflight := cors(&w, r); isPreflight == true {
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(405)
		println("no post request")
		return
	}

	var clientResponse ClientResponse
	clientResponse.Status = 1
	clientResponse.Reason = "unknown"

	defer func() {
		clientResponseJson, err := json.Marshal(clientResponse)
		if err != nil {
			println(err.Error())
			return
		}
		w.Write(clientResponseJson)
	}()

	decoder := json.NewDecoder(r.Body)

	var user User

	err := decoder.Decode(&user)
	if err != nil {
		println(err.Error())
		clientResponse.Reason = "请求解析失败"
		w.WriteHeader(500)
		return
	}

	email := user.Email
	password := user.Password

	db, err := getdb()
	if err != nil {
		println(err.Error())
		clientResponse.Reason = "连接数据库时发生错误"
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	db.Where("email = ?", email).First(&user)

	if user.Id == 0 {
		clientResponse.Reason = "邮箱账号或密码不正确"
		w.WriteHeader(200)
		return
	}

	validEmailPassword := CheckPasswordHash(password, user.Password)
	if validEmailPassword != true {
		clientResponse.Reason = "邮箱账号或密码不正确"
		w.WriteHeader(200)
		return
	}

	//至此证明用户提交的邮箱和密码正确

	clientResponse.StrId = strconv.FormatUint(user.Id, 10)
	clientResponse.Token = strconv.FormatUint(user.Token, 10)
	clientResponse.NickName = user.Nickname
	clientResponse.AvatarUrl = user.Avatar

	clientResponse.Status = 0
	clientResponse.Reason = ""
}

func authenticationcredentials(w http.ResponseWriter, r *http.Request) {

	if isPreflight := cors(&w, r); isPreflight == true {
		return
	}

	strId := r.URL.Query().Get("id")
	strToken := r.URL.Query().Get("token")

	validCredentials := false

	defer func() {
		if validCredentials == true {
			w.Write([]byte(`{"valid_credentials": true}`))
		} else {
			w.Write([]byte(`{"valid_credentials": false}`))
		}
	}()

	db, err := getdb()
	if err != nil {
		println(err.Error())
		return
	}
	defer db.Close()

	id, err := strconv.ParseUint(strId, 10, 64)
	if err != nil {
		return
	}

	token, _ := strconv.ParseUint(strToken, 10, 64)
	if err != nil {
		return
	}

	var user User
	db.Find(&user, id)

	if user.Id == 0 {
		return
	}

	if user.Token != token {
		return
	}

	validCredentials = true

}

func getidtoken(w http.ResponseWriter, r *http.Request) {

	if isPreflight := cors(&w, r); isPreflight == true {
		return
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
	clientResponse.StrId = strconv.FormatUint(user.Id, 10)
	clientResponse.AvatarUrl = user.Avatar
	clientResponse.NickName = user.Nickname
	clientResponse.Token = strconv.FormatUint(token, 10)

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

	var clientResponse ClientResponse
	clientResponse.ResponseType = 7
	clientResponse.Status = 0

	defer func() {
		if clientResponseJson, err := json.Marshal(clientResponse); err == nil {
			writeLock.Lock()
			conn.WriteMessage(websocket.TextMessage, clientResponseJson)
			writeLock.Unlock()
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
		clientResponse.Status = 1
		clientResponse.Reason = "用户id不合法，解析失败"

		fatalError = fmt.Errorf(clientResponse.Reason)
		return fatalError
	}

	wsToken, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		println("invalid token parsed")
		clientResponse.Status = 1
		clientResponse.Reason = "用户token不合法，解析失败"

		fatalError = fmt.Errorf(clientResponse.Reason)
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
		clientResponse.Status = 1
		clientResponse.Reason = "用户不存在"

		fatalError = fmt.Errorf(clientResponse.Reason)
		return fatalError
	}

	if user.Token != wsToken {
		println("invalid token")
		clientResponse.Status = 1
		clientResponse.Reason = "用户token不正确"

		fatalError = fmt.Errorf(clientResponse.Reason)
		return fatalError
	}

	connectionPoolLock.Lock()
	_, ok := connectionPool[userId]
	connectionPoolLock.Unlock()

	//用户刚上线，查找属于它的未读消息并将查找结果放入消息池
	if !ok {

		connectionPoolLock.Lock()
		connectionPool[userId] = conn
		connectionPoolLock.Unlock()

		var unread []ServerMessage

		db.Where("receiver_id = ? AND received_at = 0", userId).Limit(100).Find(&unread)

		var clientMessage ClientMessage
		for _, sm := range unread {
			clientMessage.Message = sm.Message
			clientMessage.Sender_str_id = strconv.FormatUint(sm.Sender_id, 10)
			clientMessage.Receiver_str_id = strconv.FormatUint(sm.Receiver_id, 10)

			clientMessagePoolLock.Lock()
			clientMessagePool <- clientMessage
			clientMessagePoolLock.Unlock()
		}
		go readPump(userId, conn)

	} else {
		println("user was already online")
		clientResponse.Status = 1
		clientResponse.Reason = "用户已经在线"
		clientResponse.Code = 1

		fatalError = fmt.Errorf(clientResponse.Reason)
		return fatalError
	}

	return nil
}
