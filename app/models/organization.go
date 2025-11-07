package models

import (
	"time"

	"gorm.io/gorm"
)

type Organization struct {
	gorm.Model
	ImServerId        uint      `gorm:"index"` // IM服务器ID
	Name              string    // 名称
	FullName          string    // 全称（认证名称）
	IdType            string    // 证件类型
	IdNumber          string    // 证件号
	IdPic             string    // 证件图
	IdRegisterTime    time.Time // 证件注册时间
	LegalID1          string    // 法人证件图1
	LegalID2          string    // 法人证件图2
	Commitment        string    // 信息安全承诺书
	Ico               string    // ico图标
	Logo              string    // logo图片
	Slogan            string    // 带宣传标语的图片
	Intro             string    // 简介
	ContactPerson     string    // 联系人姓名
	AreaCode          string    // 区域（手机号）
	PhoneNumber       string    // 联系手机号
	Address           string    // 地址
	Scale             string    // 企业规模
	Profession        string    // 行业类型
	InTime            time.Time // 入驻时间
	AuthTime          time.Time // 认证时间
	ExpiryTime        time.Time // 授权的到期时间
	Prefix            string    // 邀请码前缀
	Code              string    // 企业码
	ShowAuthInfo      uint8     // 是否显示企业信息
	Level             uint      // 等级，如：1普遍、2高级
	TelegramJoinCheck int8      // 是否检查关注飞机
	Status            int8      `gorm:"default:1"` // 状态
	TrtcOption        int       // 腾讯音视频开关 0 关闭 1打开
}

func OrganizationTbl() string {
	return "organization"
}
