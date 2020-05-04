package main

import (
	"errors"
)

//如果不合法否则生成error
func validIdToken(id, token uint64) error {

	db, err := getdb()
	defer db.Close()
	if err != nil {
		println(err.Error())
		return errors.New("发生数据库错误")
	}

	var user User

	db.Find(&user, id)

	if user.Id == 0 {
		return errors.New("用户不存在")
	}

	if user.Token != token {
		return errors.New(("用户凭据无效"))
	}

	return nil
}
