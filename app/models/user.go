package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	UniqueValue  string
	Account      string `gorm:"type:varchar(100);unique_index"`
	AreaCode     string // 区域（手机号）
	PhoneNumber  string `gorm:"type:varchar(100);unique_index"` // 手机号
	Email        string // 邮箱
	Nickname     string // 昵称
	FaceUrl      string // 头像
	Password     string // 密码
	IsInternal   uint8  // 是否内部账号（0否、1是）
	Port         string // 端口：1.user客户端、 2.organization商户管理端、 3.manage管理后台端
	Platform     uint   // 平台：1.IOS、 2.Android、 3.Windows、 4.OSX、 5.Web、 6.MiniWeb、 7.Linux、 8.Android Pad、 9.IPad、 10.admin
	DeviceId     string // 设备字符串
	Ip           string // IP
	Region       string // IP区域
	OtpSecret    string // OTP密钥
	Status       int    `gorm:"default:1"` // 状态
	RealName     string
	IdCardNumber string
	VerifyStatus int `gorm:"default:0;index"`
}

func UserTbl() string {
	return "user"
}

type UserMigrationModel struct {
	DuplicateUserList    []User // 验证客户A表根据客户B
	OriginalUserList     []User // 源数据
	UserList             []User // 客户A表
	OrganizationUserList []OrganizationUser
}
