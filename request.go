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

func wstokenuserid(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "DNT,X-Mx-ReqToken,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, TRACE, CONNECT, OPTIONS")
	w.Header().Set("Content-Type", "application/json")

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

		tx := db.Begin()
		if err := db.Create(&user).Error; err != nil {
			tx.Rollback()
		}

		if err := db.Create(&authorRelation).Error; err != nil {
			tx.Rollback()
		}
		tx.Commit()
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

	var tmpResponse ClientResponse
	tmpResponse.ResponseType = 5

	tempconn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		println("create temp connection faild")
	}

	defer func() {
		tempconn.Close()
	}()

	id := r.URL.Query().Get("id")
	token := r.URL.Query().Get("token")

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		println("invalid token parsed")
		tmpResponse.Reason = "用户id不合法，解析失败"
	}

	println("new ws request2")

	wsToken, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		println("invalid token parsed")
		tmpResponse.Reason = "用户token不合法，解析失败"
	}

	println("new ws request3")

	db, err := getdb()
	defer db.Close()
	if err != nil {

		println(err.Error())
		tmpResponse.Reason = "发生数据库错误"
	}

	println("new ws request4")

	var user User

	db.Find(&user, id)

	if user.Id == 0 {
		println("no such user")
		tmpResponse.Reason = "用户不存在"
	}

	if user.WsToken != wsToken {
		println("invalid token")
		tmpResponse.Reason = "用户token不正确"
	}

	if tmpResponse.Reason != "" {
		tmpResponse.Status = 1
	}

	tempResponseJson, err := json.Marshal(tempconn)

	if err := tempconn.WriteMessage(websocket.TextMessage, tempResponseJson); err != nil {
		return
	}

	println("new ws request5")

	serveWs(w, r, userId)
}
