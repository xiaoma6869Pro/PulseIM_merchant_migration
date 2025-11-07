package service

import (
	"fmt"
	"strconv"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
)

// 删除重复手机客户A和客户B
func removeDuplicates(info *models.UserMigrationModel) {
	monitorDuplicatePhone := make(map[string]struct{})
	for _, u := range info.DuplicateUserList {
		monitorDuplicatePhone[u.PhoneNumber] = struct{}{}
	}
	filtered := info.UserList[:0]
	for _, u := range info.UserList {
		if _, exists := monitorDuplicatePhone[u.PhoneNumber]; !exists {
			filtered = append(filtered, u)
		}
	}
	info.UserList = filtered
}

/*
		dbAppA: 数据库链接-app-A
	 	dbAppB: 数据链接-app-b
		organizationID：企业 app-A
*/
func GetVerifyUserAppAB(dbAppA, dbAppB string, organizationID int) (*models.UserMigrationModel, error) {
	// 查看数据库链接-app-A
	connectToDbAppA, err := db.GetConnectionDB(dbAppA)
	if err != nil {
		return nil, err
	}

	// 获取用户企业-app-A
	var userMigrationModel models.UserMigrationModel
	if err := connectToDbAppA.Table(models.OrganizationUserTbl()).Where("organization_id = ? AND deleted_at IS NULL", organizationID).Scan(&userMigrationModel.OrganizationUserList).Error; err != nil {
		return nil, err
	}

	// 获取用户Ids
	var userIDs []uint
	for _, u := range userMigrationModel.OrganizationUserList {
		userIDs = append(userIDs, u.UserId)
	}

	if err := connectToDbAppA.Table(models.UserTbl()).Where("deleted_at IS NULL and id IN ?", userIDs).Scan(&userMigrationModel.UserList).Error; err != nil {
		return nil, err
	}
	// 查看用户A-App userMigrationModel.UserList是否空白
	if len(userMigrationModel.UserList) > 0 {
		var phoneNumberList []string
		for _, user := range userMigrationModel.UserList {
			phoneNumberList = append(phoneNumberList, user.PhoneNumber)
		}
		//链接数据库 AppB
		connectToDbAppB, err := db.GetConnectionDB(dbAppB)
		if err != nil {
			return nil, err
		}
		// 找重复用户跟用户-App-B相关
		connectToDbAppB.Table(models.UserTbl()).Where("phone_number IN ?", phoneNumberList).Scan(&userMigrationModel.DuplicateUserList)
		if userMigrationModel.DuplicateUserList != nil {
			// 防止重复转移
			removeDuplicates(&userMigrationModel)
		}
	}
	return &userMigrationModel, nil
}

/*
   organizationAppBID: 数据库链接-app-B
   dbAppB: 数据链接-app-b
*/

func ImportUserAppAToAppB(dbAppB string, userMigrationModel models.UserMigrationModel, organizationAppBID int) error {
	if dbAppB == "" {
		return fmt.Errorf("数据库名字不能为空")
	}
	conn, err := db.GetConnectionDB(dbAppB)
	if err != nil {
		return err
	}

	var organization models.Organization
	// 查看企业 AppB
	if err := conn.Table(models.OrganizationTbl()).Where("id = ?", organizationAppBID).Find(&organization).Error; err != nil {
		return err
	}
	if organization.ID == 0 {
		return fmt.Errorf("组织不存在")
	}

	for _, u := range userMigrationModel.UserList {
		user := u
		// 查看用户企业A
		organizationUserA := models.OrganizationUser{}
		for _, org := range userMigrationModel.OrganizationUserList {
			if user.ID == org.UserId {
				organizationUserA = org
				break
			}
		}

		tx := conn.Begin()
		// 转移客户A企业到客户B企业
		user.ID = 0
		if err := tx.Table(models.UserTbl()).Create(&user).Error; err != nil {
			tx.Rollback()
			utils.Logger.Printf("创建用户失败: %v\n", err)
			continue
		}

		organizationUserA.OrganizationId = organization.ID
		organizationUserA.UniqueValue = strconv.Itoa(int(organization.ID)) + "-" + strconv.Itoa(int(user.ID))
		organizationUserA.UserId = user.ID
		organizationUserA.ImUserId = ""
		organizationUserA.ID = 0

		// 转移客户A企业到客户B企业
		if err := tx.Table(models.OrganizationUserTbl()).Create(&organizationUserA).Error; err != nil {
			tx.Rollback()
			utils.Logger.Printf("插入失败 organization_user %+v: %v\n", organizationUserA, err)
			continue
		}

		if err := tx.Commit().Error; err != nil {
			utils.Logger.Printf("提交失败:  客户：(%+v)\n企业用户(%+v): %v\n", user, organizationUserA, err)
			continue
		}
	}
	return nil
}
