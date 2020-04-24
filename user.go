package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type UserBasic struct {
	Id       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type Contact struct {
	Id       uint64 `json:"-"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Alias    string `json:"alias"`
	StrId    string `json:"strid"`
}

type User struct {
	UserBasic
	WsToken uint64 `json:"ws_token"`
}

func userinfo(w http.ResponseWriter, r *http.Request) {
	cors(&w, r)

	id := r.URL.Query().Get("id")

	db, err := getdb()
	defer db.Close()

	if err != nil {
		println(err.Error())
		return
	}
	var user User

	db.Find(&user, id)
	//如果没有id应用的user,则user为：
	//{"id":0,"nickname":"","avatar":""}

	info, err := json.Marshal(user)
	if err != nil {
		return
	}

	w.Write([]byte(info))
}

func getcontact(w http.ResponseWriter, r *http.Request) {
	cors(&w, r)

	var clientResponse ClientResponse
	clientResponse.ResponseType = 6
	clientResponse.Status = 0

	defer func() {
		clientResponseJson, err := json.Marshal(clientResponse)
		if err != nil {
			return
		}
		w.Write([]byte(clientResponseJson))
	}()

	id := r.URL.Query().Get("id")
	token := r.URL.Query().Get("token")

	db, err := getdb()
	if err != nil {
		println(err.Error())
		clientResponse.Status = 1
		clientResponse.Reason = "发生数据库错误"
		return
	}
	defer db.Close()

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = "用户id不合法，解析失败"
		return
	}

	wsToken, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = "用户token不合法，解析失败"
		return
	}

	var user User
	db.Find(&user, userId)

	if user.Id == 0 {
		clientResponse.Status = 1
		clientResponse.Reason = "用户不存在"
		return
	}

	if user.WsToken != wsToken {
		clientResponse.Status = 1
		clientResponse.Reason = "用户token不正确"
		return
	}

	//用户提交了合法凭据,查找数据库
	if clientResponse.Status == 0 {
		rows, err := db.Table("relationships").Select("users.id, users.nickname, users.avatar, relationships.alias").
			Where("id1 = (?)", userId).Joins("left join users on relationships.id2 = users.id").Rows()
		if err != nil {
			println(err.Error())
			clientResponse.Status = 1
			clientResponse.Reason = "发生数据库错误"
			return
		}
		var contacts []Contact
		var contact Contact
		for rows.Next() {

			err := db.ScanRows(rows, &contact)
			if err != nil {
				println(err.Error())
			}

			//id 转 str id
			contact.StrId = strconv.FormatUint(contact.Id, 10)

			contact.Id = 0

			contacts = append(contacts, contact)
		}

		if contactsJson, err := json.Marshal(contacts); err != nil {
			clientResponse.Status = 1
			clientResponse.Reason = "解析数据库信息时发生错误"
			return
		} else {
			clientResponse.Body = string(contactsJson)
		}
	}

}
