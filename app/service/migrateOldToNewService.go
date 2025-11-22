package service

import (
	"fmt"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
)

// MigrationNewUserAppInOldDbToNewDb 在旧数据迁移新注册客户转移到新数据库
func MigrationNewUserAppInOldDbToNewDb(oldDbName, newDbName string) error {
	if oldDbName == "" || newDbName == "" {
		return fmt.Errorf("数据库名不能为空")
	}

	mysqlOldDb, err := db.GetConnectionDB(oldDbName)
	if err != nil {
		return err
	}

	mysqlNewDb, err := db.GetConnectionDB(newDbName)
	if err != nil {
		return err
	}

	var countOldUser int64
	if err := mysqlOldDb.Table(models.UserTbl()).Where("deleted_at IS NULL").Count(&countOldUser).Error; err != nil {
		return err
	}

	const batchSize = 50000
	var offset int
	batches := splitCountIntoBatches(countOldUser, batchSize)

	// 创建新ants协程池运行并发
	//pool, err := ants.NewPool(30, ants.WithPanicHandler(func(i interface{}) {
	//	utils.Logger.Printf("协程panic: %v", i)
	//}))
	//if err != nil {
	//	return fmt.Errorf("创建协程池失败: %v", err)
	//}
	//defer pool.Release()
	//
	//var wg sync.WaitGroup

	for _, currentBatchSize := range batches {
		var oldUserList []models.User
		if err := mysqlOldDb.Table(models.UserTbl()).Where("deleted_at IS NULL").Offset(offset).Limit(currentBatchSize).Find(&oldUserList).Error; err != nil {
			return err
		}

		if len(oldUserList) == 0 {
			break
		}
		//_, _, line, _ := runtime.Caller(0)
		//	utils.Logger.Printf("Error Line: %d, OrganizationUserTbl: %v, 错误：%v", line, user, err)
		//	_, _, line, _ := runtime.Caller(0)
		//	utils.Logger.Printf("Error Line: %d, OrganizationTbl: %v, 错误：%v", line, oldUser, err)

		for _, oldUser := range oldUserList {
			user := oldUser
			//wg.Add(1)
			//pool.Submit(func() {
			//	defer wg.Done()

			// 检查旧库用户是否已存在新库中
			var findUser models.User
			if err := mysqlNewDb.Table(models.UserTbl()).Where("phone_number = ? AND deleted_at IS NULL", user.PhoneNumber).Find(&findUser).Error; err == nil && findUser.ID > 0 {
				continue
			}

			// 验证旧组织用户
			var oldOrgUser models.OrganizationUser
			if err := mysqlOldDb.Table(models.OrganizationUserTbl()).Where("deleted_at IS NULL AND user_id = ?", user.ID).Find(&oldOrgUser).Error; err != nil {
				continue
			}
			if oldOrgUser.ID == 0 {
				continue
			}

			// 验证旧组织用户与新旧组织是否匹配
			var organization models.Organization
			if err := mysqlNewDb.Table(models.OrganizationTbl()).Where("id = ? AND deleted_at IS NULL", oldOrgUser.OrganizationId).Find(&organization).Error; err != nil {
				continue
			}

			if organization.ID == 0 {
				continue
			}

			tx := mysqlNewDb.Begin()
			defer func() {
				// 防止panic出现问题
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			// 提交记录到新数据库
			newUser := user
			newUser.ID = 0
			if err := tx.Create(&newUser).Error; err != nil {
				tx.Rollback()
				utils.Logger.Printf("迁移用户失败：%v, 用户手机号 %s ", err, newUser.PhoneNumber)
				continue
			}

			newOrgUser := oldOrgUser
			newOrgUser.ID = 0
			newOrgUser.UserId = newUser.ID
			if err := tx.Create(&newOrgUser).Error; err != nil {
				tx.Rollback()
				utils.Logger.Printf("迁移企业客户失败：%v, OrgID: %d, 客户：%d", err, newOrgUser.OrganizationId, oldUser.ID)
				continue
			}

			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				utils.Logger.Printf("提交失败：%v, OrgID: %d, 客户：%d", err, newOrgUser.OrganizationId, oldUser.ID)
			}
			//})
		}
		offset += currentBatchSize
	}

	//wg.Wait()
	return nil
}
