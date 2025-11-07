package models

import "gorm.io/gorm"

type OrganizationUser struct {
	gorm.Model
	UniqueValue      string
	OrganizationId   uint   `gorm:"index"`     // 机构ID
	UserId           uint   `gorm:"index"`     // 用户ID
	ImUserId         string ``                 // 在 IM 的 user_id
	InvitationCode   string ``                 // 我的邀请码
	InvitationUserId uint   `gorm:"index"`     // 我的邀请人（0为系统或后台管理员生成）
	Status           int8   `gorm:"default:1"` // 状态
}

func OrganizationUserTbl() string {
	return "organization_user"
}
