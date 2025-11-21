package service

import (
	"github.com/panjf2000/ants/v2"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"

	"fmt"
	"sync"
	"time"

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
	if err := mysql.Unscoped().Table(models.OrganizationUserTbl()).Where("deleted_at IS NOT NULL AND organization_id = ?", orgID).Scan(&organizationUserList).Error; err != nil {
		return fmt.Errorf("获取组织用户失败: %v", err)
	}
	if organizationUserList == nil {
		return fmt.Errorf("暂时没组织用户")
	}

	// 创建ants协程池
	pool, err := ants.NewPool(30, ants.WithPanicHandler(func(i interface{}) {
		utils.Logger.Printf("协程panic: %v", i)
	}))
	if err != nil {
		return fmt.Errorf("创建协程池失败: %v", err)
	}
	defer pool.Release()
	var wg sync.WaitGroup

	userBindMultipleOrgList := ""

	// 使用协程池处理
	for _, organizationUser := range organizationUserList {
		wg.Add(1)
		orgUser := organizationUser
		pool.Submit(func() {
			defer wg.Done()

			var user models.User
			if err := mysql.Table(models.UserTbl()).Where("deleted_at IS NULL AND id = ?", orgUser.UserId).Find(&user).Error; err != nil {
				utils.Logger.Printf("获取客户失败: %v", orgUser)
				return
			}

			if user.ID == 0 {
				return
			}

			// 查看客户绑定跟另外企业
			var countOrgUser int64
			if err := mysql.Table(models.OrganizationUserTbl()).Where("user_id = ? AND deleted_at IS NULL", user.ID).Count(&countOrgUser).Error; err != nil {
				utils.Logger.Printf("统计企业客户失败: %v", orgUser)
				return
			}
			if countOrgUser > 0 {
				userBindMultipleOrgList += fmt.Sprintf("%d,", user.ID)
				return
			}
			user.UniqueValue = fmt.Sprintf("%s%s-%s", user.AreaCode, user.PhoneNumber, time.Now().Format("20060102"))
			user.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			if err := mysql.Table(models.UserTbl()).Where("id = ?", user.ID).Updates(user).Error; err != nil {
				utils.Logger.Printf("删除客户失败: %v", user)
			}
		})
	}
	wg.Wait()
	utils.Logger.Printf("客户绑定多企业列表：%v", userBindMultipleOrgList)
	return nil
}
