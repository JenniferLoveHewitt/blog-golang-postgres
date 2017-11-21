package models

import "time"

type UserInfo struct {
	Id       int
	Login 	 string
	Email    string
	Password string
	Role     string
	Created  time.Time
}

func NewUserInfo(login string, email string, password string, role string) *UserInfo {
	return &UserInfo {
		0,
		login,
		email,
		password,
		role,
		time.Now(),
	}
}
