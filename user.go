package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

//user 是当前的用户信息，包含token等敏感信息
type User struct {
	Id              uint64 `json:"id"`
	Nickname        string `json:"nickname"`
	Avatar          string `json:"avatar"`
	Token           uint64 `json:"token"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ActivationToken string `json:"activation_token"`
}

//联系人信息
type Contact struct {
	Id       uint64 `json:"-"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Alias    string `json:"alias,omitempty"`
	StrId    string `json:"strid"`
}

func contactinfo(w http.ResponseWriter, r *http.Request) {
	if cors(&w, r) {
		return
	}

	id := r.URL.Query().Get("strid")

	var clientResponse ClientResponse
	clientResponse.ResponseType = 8 //请求资源
	clientResponse.Status = 1       //默认是失败的请求

	db, err := getdb()
	if err != nil {
		println(err)
		clientResponse.Reason = "请求联系人信息时发生数据库错误"
		return
	}

	defer db.Close()

	var user User

	db.Find(&user, id)
	//如果没有id应用的user,则user为：
	//{"id":0,"nickname":"","avatar":""}

	info, err := json.Marshal(user)
	if err != nil {
		println(err.Error())
		clientResponse.Reason = "解析联系人信息时发生错误"
		return
	}

	clientResponse.Body = string(info)

	clientResponse.Status = 0
	clientResponseJson, err := json.Marshal(clientResponse)

	w.Write(clientResponseJson)

}

func getcontact(w http.ResponseWriter, r *http.Request) {
	if isPreflight := cors(&w, r); isPreflight == true {
		return
	}

	var clientResponse ClientResponse
	clientResponse.ResponseType = 7
	clientResponse.Status = 0

	defer func() {
		clientResponseJson, err := json.Marshal(clientResponse)
		if err != nil {
			return
		}
		println(string(clientResponseJson))
		w.Write(clientResponseJson)
	}()

	strId := r.URL.Query().Get("id")
	strToken := r.URL.Query().Get("token")

	db, err := getdb()
	if err != nil {
		println(err.Error())
		clientResponse.Status = 1
		clientResponse.Reason = "发生数据库错误"
		return
	}
	defer db.Close()

	id, err := strconv.ParseUint(strId, 10, 64)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = "用户id不合法，解析失败"
		return
	}

	token, err := strconv.ParseUint(strToken, 10, 64)
	if err != nil {
		clientResponse.Status = 1
		clientResponse.Reason = "用户token不合法，解析失败"
		return
	}

	var user User
	db.Find(&user, id)

	if user.Id == 0 {
		clientResponse.Status = 1
		clientResponse.Reason = "用户不存在"
		return
	}

	if user.Token != token {
		clientResponse.Status = 1
		clientResponse.Reason = "用户token不正确"
		return
	}

	//用户提交了合法凭据,查找数据库
	if clientResponse.Status == 0 {
		rows, err := db.Table("relationships").Select("users.id, users.nickname, users.avatar, relationships.alias").
			Where("id1 = (?)", id).Joins("left join users on relationships.id2 = users.id").Rows()
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
