package models

import "gorm.io/gorm"

type UserApp struct {
	gorm.Model
	UserID       uint
	AppPackageID uint
}

func UserAppTbl() string {
	return "user_app"
}
