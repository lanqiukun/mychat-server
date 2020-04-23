package main

import (
	"encoding/json"
	"net/http"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type User struct {
	Id       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	WsToken  uint64 `json:"ws_token`
}

func userinfo(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	db, err := gorm.Open("mysql", "root:Edison3306#@(lowb.top:3306)/mychat?charset=utf8&parseTime=True&loc=Local")
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

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(info))
}
