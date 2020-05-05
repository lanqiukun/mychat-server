package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"gopkg.in/gomail.v2"
)

type RegisterEmailPassword struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

//接受nickname、email、password来注册一个用户
func emailRegister(w http.ResponseWriter, r *http.Request) {
	println("来了")
	if cors(&w, r) {
		w.WriteHeader(200)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(403)
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

	//获取email
	decoder := json.NewDecoder(r.Body)

	var registerEmailInfo RegisterEmailPassword
	err := decoder.Decode(&registerEmailInfo)
	if err != nil {
		println(err.Error())
		clientResponse.Reason = "解析请求失败"
		return
	}

	println(registerEmailInfo.Email)

	if !isValidEmailAddress(registerEmailInfo.Email) {
		w.WriteHeader(413)
		clientResponse.Reason = "邮箱地址格式不正确"
		return
	}

	if len(registerEmailInfo.Password) < 6 {
		w.WriteHeader(413)
		clientResponse.Reason = "密码长度不足~"
		return
	}

	passwordHash, err := HashPassword(registerEmailInfo.Password)
	if err != nil {
		clientResponse.Reason = "序列化密码失败了"
		println(err.Error())
		return
	}

	//邮箱地址格式正确，密码长度足够，查询数据库是否已存在该邮箱的记录

	db, err := getdb()
	defer db.Close()
	if err != nil {
		println(err.Error())
		return
	}

	var user User
	db.Where("email = ?", registerEmailInfo.Email).First(&user)
	if user.Id != 0 {
		clientResponse.Reason = "该邮箱已被注册"
		return
	}

	//开始发送注册邮件
	token, err := generateActivationToken(45)

	if err != nil {
		clientResponse.Reason = "向用户发送注册邮件时发生错误"
		println(err.Error())
		return
	}

	go sendRegisterEmailAndCreateUser(registerEmailInfo.Email, token, registerEmailInfo.Nickname, passwordHash)

	clientResponse.Status = 0

}

func sendRegisterEmailAndCreateUser(target, token, nickname, passwordHash string) {
	m := gomail.NewMessage()

	m.SetHeader("From", "1543323033@qq.com")
	m.SetHeader("To", target)

	m.SetHeader("Subject", "Welcome to MyChat! Click the link below to activate you email account!")
	EmailActivationUrl := EmailActivatingHostUrl + "?token=" + token
	m.SetBody("text/html", `<a href=`+EmailActivationUrl+`>`+EmailActivationUrl+`</a>`)

	d := gomail.NewDialer("smtp.qq.com", 465, "1543323033@qq.com", "qrjnzcuvustfgdag")

	if err := d.DialAndSend(m); err != nil {
		println(err.Error())
	}

	//创建用户, 前面已经确定数据库中没有该用户的记录了
	db, err := getdb()
	defer db.Close()
	if err != nil {
		println(err.Error())
		return
	}

	var user User
	user.Email = target
	user.ActivationToken = token
	user.Password = passwordHash
	user.Token = rand.Uint64()
	user.Nickname = nickname

	db.Create(&user)

}

func activateEmailAccount(w http.ResponseWriter, r *http.Request) {
	//get请求不用考虑请求预检
	activation_token := r.URL.Query().Get("token")
	println(activation_token)

	var clientResponse ClientResponse

	clientResponse.Status = 1
	clientResponse.Reason = "unknown"

	defer func() {
		clientResponseJson, err := json.Marshal(clientResponse)
		if err != nil {
			println(err.Error())
		}
		w.Write(clientResponseJson)
	}()

	db, err := getdb()
	defer db.Close()
	if err != nil {
		clientResponse.Reason = "发生数据库错误"
		println(err.Error())
		return
	}

	var user User
	db.Where("activation_token = ?", activation_token).First(&user)

	if user.Id == 0 {
		clientResponse.Reason = "你是不是已经激活过该邮箱了呢？链接已经失效了哈哈哈~"
		return
	}

	if user.ActivationToken != activation_token {
		clientResponse.Reason = "该链接无效偶~"
		return
	}

	//链接、activation_token有效，删除activation_code

	//
	err = db.Table("users").Where("id = (?)", user.Id).Update("activation_token", "").Error

	if err != nil {
		clientResponse.Reason = "发生数据库错误"
		return
	}

	clientResponse.Status = 0

	//注册成功，添加好友,并重定向用户至注册成功界面

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

	db.Create(&authorRelation1)
	db.Create(&authorRelation2)

	http.Redirect(w, r, EmailRegisterSucceedUrl+"/register-succeed"+"?newcomer="+user.Email, 302)

	return
}
