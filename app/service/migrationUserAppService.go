package service

import (
	"fmt"
	"sync"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
	"github.com/panjf2000/ants/v2"
)

func splitCountIntoBatches(totalCount int64, batchSize int) []int {
	var batches []int
	remaining := int(totalCount)

	for remaining > 0 {
		if remaining >= batchSize {
			batches = append(batches, batchSize)
			remaining -= batchSize
		} else {
			batches = append(batches, remaining)
			remaining = 0
		}
	}

	return batches
}

// MigrationUserAppService 按排除组织实现用户应用迁移
func MigrationUserAppService(excludeOrgIDs []int, dbName string) error {
	if excludeOrgIDs == nil && dbName == "" {
		return fmt.Errorf("excludeOrgIDs或dbName不能为空")
	}

	mysql, err := db.GetConnectionDB(dbName)
	if err != nil {
		return err
	}

	var countOrgUser int64
	whereOrgUser := "deleted_at IS NULL AND organization_id NOT IN ?"
	if err := mysql.Table(models.OrganizationUserTbl()).Where(whereOrgUser, excludeOrgIDs).Count(&countOrgUser).Error; err != nil {
		return err
	}

	if countOrgUser == 0 {
		return fmt.Errorf("展示没有客户企业: %d", countOrgUser)
	}

	const batchSize = 50000
	var offset int
	batches := splitCountIntoBatches(countOrgUser, batchSize)

	pool, err := ants.NewPool(50, ants.WithPanicHandler(func(i interface{}) {
		utils.Logger.Printf("协程panic: %v", i)
	}))
	if err != nil {
		return fmt.Errorf("创建协程池失败: %v", err)
	}
	defer pool.Release()
	var wg sync.WaitGroup

	for _, currentBatchSize := range batches {
		var orgUserList []models.OrganizationUser
		err := mysql.Table(models.OrganizationUserTbl()).Where(whereOrgUser, excludeOrgIDs).Offset(offset).Limit(currentBatchSize).Find(&orgUserList).Error
		if err != nil {
			return err
		}

		if orgUserList == nil {
			break
		}

		for _, orgUser := range orgUserList {
			wg.Add(1)
			u := orgUser
			pool.Submit(func() {
				defer wg.Done()
				userApp := models.UserApp{
					AppPackageID: 1,
					UserID:       u.UserId,
				}
				if err := mysql.Table(models.UserAppTbl()).Create(&userApp).Error; err != nil {
					utils.Logger.Printf("创建客户失败：%d", userApp.UserID)
				}
			})
		}
		offset += currentBatchSize
	}
	wg.Wait()
	return nil
}
