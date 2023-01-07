package User

import (
	"Github.com/mhthrh/EventDriven/Model/Request"
	"Github.com/mhthrh/EventDriven/Utilitys/CryptoUtil"
	"Github.com/mhthrh/EventDriven/Utilitys/DbPool"
	"Github.com/mhthrh/EventDriven/Utilitys/Redis"
	"context"
	"fmt"
	"time"
)

type (
	User struct {
		Header   Request.Request `json:"header"`
		Name     string          `json:"name"  validate:"required,alphanum"`
		LastName string          `json:"lastName" validate:"required,alphanum"`
		UserName string          `json:"userName" validate:"required,alphanum"`
		PassWord string          `json:"passWord" validate:"required,printascii"`
		Email    string          `json:"email" validate:"required,email"`
		Avatar   string          `json:"avatar" validate:"required,base64"`
		status   int
		Token    string
	}
)

var (
	_db    *DbPool.Connection
	_redis *Redis.Client
	_ctx   *context.Context
)

func New(db *DbPool.Connection, redis *Redis.Client, ctx *context.Context) {
	_db = db
	_redis = redis
	_ctx = ctx
}

func (u *User) SignUp() error { //All
	defer func() {
		err := recover()
		if err != nil {
			return
		}
	}()
	exist := u.ExistUser()

	if exist {
		return fmt.Errorf("UserName dublicate")
	}
	c := CryptoUtil.NewKey()
	c.Text = u.PassWord
	encrypted := c.Sha256()

	result, err := _db.Db.ExecContext(*_ctx, fmt.Sprintf("INSERT INTO `MyDb`.`users`(`firstName`,`lastName`,`userName`,`password`,`email`,`createTime`,`stat`,`avatar`)VALUES('%s','%s','%s','%s','%s','%s','%d','%s')", u.Name, u.LastName, u.UserName, encrypted, u.Email, time.StampNano, 1, u.Avatar))
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return fmt.Errorf("user not inserted")
	}

	return nil
}

func (u *User) SignIn() (User, error) { //username and password
	user := User{
		Header:   u.Header,
		Name:     "",
		LastName: "",
		UserName: u.UserName,
		PassWord: "********",
		Email:    "",
		Avatar:   "",
	}

	defer func() {
		err := recover()
		if err != nil {
			return
		}
	}()
	crypto := CryptoUtil.NewKey()
	crypto.Text = u.PassWord
	encrypted := crypto.Sha256()

	fmt.Println(encrypted)
	rows, err := _db.Db.QueryContext(*_ctx, fmt.Sprintf("select firstName , lastName,userName,Email,stat,avatar from MyDb.users where userName='%s' and password='%s'", u.UserName, encrypted))
	if err != nil {
		return user, err
	}
	for rows.Next() {
		err := rows.Scan(&user.Name, &user.LastName, &user.UserName, &user.Email, &user.status, &user.Avatar)
		if err != nil {
			return user, err
		}
	}

	if user.status != 1 {
		return user, fmt.Errorf("user not longer active")
	}

	token, err := crypto.GenerateToken(user.UserName, user.Email)
	if err != nil {
		return user, err
	}
	c, err := _db.Db.ExecContext(*_ctx, fmt.Sprintf("UPDATE `MyDb`.`users` SET `Token` = '%s' WHERE `userName` = '%s'", token, user.UserName))
	if err != nil {
		return user, err
	}
	count, err := c.RowsAffected()
	if count != 1 {
		return user, err
	}

	user.Token = token
	return user, nil
}

func (u *User) ExistUser() bool { //username
	defer func() {
		err := recover()
		if err != nil {
			return
		}
	}()
	var count int
	rows, err := _db.Db.QueryContext(*_ctx, fmt.Sprintf("SELECT count(*) FROM MyDb.users where userName='%s'", u.UserName))
	if err != nil {
		return false
	}
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return false
		}
	}
	if count > 0 {
		return true
	}
	return false
}
func (u *User) TokenIsValid() (bool, error) {
	crypto := CryptoUtil.NewKey()
	var email string

	rows, err := _db.Db.QueryContext(*_ctx, fmt.Sprintf("select Email from MyDb.users where userName='%s' ", u.UserName))
	if err != nil {
		return false, err
	}
	for rows.Next() {
		err := rows.Scan(&email)
		if err != nil {
			return false, err
		}
	}

	valid, err := crypto.CheckSignKey(u.UserName, email, u.Token)
	if err != nil {
		return false, err
	}
	return valid, nil
}
