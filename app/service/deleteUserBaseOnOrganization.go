package service

import (
	"fmt"
	"strconv"
	"time"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
	"gorm.io/gorm"
)

// DeleteUserBaseOrganizationID 软删除用户根据企业ID
func DeleteUserBaseOrganizationID(dbName string, orgID int) error {

	if dbName == "" {
		return fmt.Errorf("dbName为必填写")
	}
	if orgID == 0 {
		return fmt.Errorf("orgID为必填写")
	}

	mysql, err := db.GetConnectionDB(dbName)
	if err != nil {
		return fmt.Errorf("链接数据库失败: %v", err)
	}

	// 获取企业客户
	var organizationUserList []models.OrganizationUser
	if err := mysql.Table(models.OrganizationUserTbl()).Limit(100).Where("deleted_at IS NULL AND organization_id = ?", orgID).Scan(&organizationUserList).Error; err != nil {
		return fmt.Errorf("获取组织用户失败: %v", err)
	}
	if organizationUserList == nil {
		return fmt.Errorf("暂时没组织用户")
	}

	// 使用组织用户列表检查客户根据ID
	for _, organizationUser := range organizationUserList {
		var user models.User
		if err := mysql.Table(models.UserTbl()).Where("id = ? AND deleted_at IS NULL", organizationUser.UserId).First(&user).Error; err != nil {
			utils.Logger.Printf("获取客户失败: %v", organizationUser)
			continue
		}

		if user.ID == 0 {
			continue
		}

		tx := mysql.Begin()

		user.UniqueValue = strconv.Itoa(int(orgID)) + "-" + strconv.Itoa(int(user.ID))
		user.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
		if err := tx.Table(models.UserTbl()).Where("id = ?", user.ID).Updates(user).Error; err != nil {
			tx.Rollback()
			utils.Logger.Printf("删除客户失败: %v", user)
		}

		organizationUser.UniqueValue = strconv.Itoa(int(orgID)) + "-" + strconv.Itoa(int(user.ID))
		organizationUser.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
		if err := tx.Table(models.OrganizationUserTbl()).Where("user_id = ? and organization_id = ?", organizationUser.UserId, orgID).Updates(&organizationUser).Error; err != nil {
			tx.Rollback()
			utils.Logger.Printf("更新企业客户失败: %v", user)
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			utils.Logger.Printf("提交失败: %v", organizationUser)
		}
	}
	return nil
}
