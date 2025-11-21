package service

import (
	"fmt"
	"strconv"
	"sync"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
	"github.com/panjf2000/ants/v2"
)

func GetUniqueDuplicateUser(records []models.User) []models.User {
	seen := make(map[string]bool)
	var unique []models.User
	for _, record := range records {
		if seen[record.PhoneNumber] {
			continue
		}
		seen[record.PhoneNumber] = true
		unique = append(unique, record)
	}

	return unique
}

// 删除重复手机客户A和客户B
func removeDuplicates(info *models.UserMigrationModel) {
	monitorDuplicatePhone := make(map[string]struct{})
	for _, u := range info.DuplicateUserList {
		monitorDuplicatePhone[u.PhoneNumber] = struct{}{}
	}
	filtered := info.OriginalUserList[:0]
	for _, u := range info.OriginalUserList {
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

	if err := connectToDbAppA.Table(models.UserTbl()).Where("deleted_at IS NULL AND id IN ?", userIDs).Scan(&userMigrationModel.OriginalUserList).Error; err != nil {
		return nil, err
	}

	// 查看用户A-App userMigrationModel.UserList是否空白
	if len(userMigrationModel.OriginalUserList) > 0 {
		var phoneNumberList []string
		for _, user := range userMigrationModel.OriginalUserList {
			phoneNumberList = append(phoneNumberList, user.PhoneNumber)
		}
		//链接数据库 AppB
		connectToDbAppB, err := db.GetConnectionDB(dbAppB)
		if err != nil {
			return nil, err
		}
		// 找重复用户跟用户-App-B相关
		connectToDbAppB.Table(models.UserTbl()).Where("phone_number IN ? AND deleted_at IS NULL", phoneNumberList).Scan(&userMigrationModel.DuplicateUserList)
		if userMigrationModel.DuplicateUserList != nil {
			var duplicatePhoneStr string
			userMigrationModel.DuplicateUserList = GetUniqueDuplicateUser(userMigrationModel.DuplicateUserList)
			for _, u := range userMigrationModel.DuplicateUserList {
				duplicatePhoneStr += fmt.Sprintf("%s,", u.PhoneNumber)
			}
			utils.Logger.Printf("重复手机号记录: %v\n", duplicatePhoneStr)
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

	// 创建ants协程池
	pool, err := ants.NewPool(10, ants.WithPanicHandler(func(i interface{}) {
		utils.Logger.Printf("协程panic: %v", i)
	}))
	if err != nil {
		return fmt.Errorf("创建协程池失败: %v", err)
	}

	defer pool.Release()
	var wg sync.WaitGroup

	for _, u := range userMigrationModel.UserList {
		wg.Add(1)
		user := u

		pool.Submit(func() {
			defer wg.Done()

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
				return
			}

			organizationUserA.OrganizationId = organization.ID
			organizationUserA.UniqueValue = strconv.Itoa(int(organization.ID)) + "-" + strconv.Itoa(int(user.ID))
			organizationUserA.UserId = user.ID
			organizationUserA.ID = 0

			if organizationUserA.InvitationUserId > 0 {

				var _user models.User
				if err := conn.Table(models.UserTbl()).Where("id = ?", organizationUserA.InvitationUserId).Find(&_user).Error; err != nil {
					tx.Rollback()
					utils.Logger.Printf("获取失败:  客户_user：(%+v)\n错误(%+v)", _user, err)
					return
				}

				if _user.ID == 0 {
					organizationUserA.InvitationUserId = 0
				}
			}

			// 转移客户A企业到客户B企业
			if err := tx.Table(models.OrganizationUserTbl()).Create(&organizationUserA).Error; err != nil {
				tx.Rollback()
				utils.Logger.Printf("插入失败 organization_user %+v: %v\n", organizationUserA, err)
				return
			}

			if err := tx.Commit().Error; err != nil {
				utils.Logger.Printf("提交失败:  客户：(%+v)\n企业用户(%+v): %v\n", user, organizationUserA, err)
				return
			}
		})
	}
	wg.Wait()

	return nil
}

func AssignOrganizationToExitingClient(dbAppB string, organizationID int, info models.UserMigrationModel) error {

	if dbAppB == "" {
		return fmt.Errorf("数据库昵称不能为空")
	}

	if organizationID == 0 {
		return fmt.Errorf("没有找到手机号列表")
	}

	if info.DuplicateUserList == nil {
		return fmt.Errorf("暂时没有重复记录")
	}

	conn, _ := db.GetConnectionDB(dbAppB)

	// 创建ants协程池
	pool, err := ants.NewPool(10, ants.WithPanicHandler(func(i interface{}) {
		utils.Logger.Printf("协程panic: %v", i)
	}))
	if err != nil {
		return fmt.Errorf("创建协程池失败: %v", err)
	}

	defer pool.Release()
	var wg sync.WaitGroup

	for _, duplicateUser := range info.DuplicateUserList {
		wg.Add(1)
		var user models.User

		pool.Submit(func() {
			defer wg.Done()

			if err := conn.Table(models.UserTbl()).Where("phone_number = ?", duplicateUser.PhoneNumber).Find(&user).Error; err != nil {
				utils.Logger.Printf("获取重复客户记录失败: %v\n", duplicateUser.PhoneNumber)
				return
			}

			if user.ID == 0 {
				return
			}

			orgUser := models.OrganizationUser{
				OrganizationId:   uint(organizationID),
				UserId:           user.ID,
				UniqueValue:      strconv.Itoa(int(organizationID)) + "-" + strconv.Itoa(int(user.ID)),
				InvitationUserId: getPreviousOrgUser(info, duplicateUser.PhoneNumber).InvitationUserId,
			}

			if orgUser.InvitationUserId > 0 {
				var _user models.User
				if err := conn.Table(models.UserTbl()).Where("id = ?", orgUser.InvitationUserId).Find(&_user).Error; err != nil {
					utils.Logger.Printf("获取失败:  客户_user：(%+v)\n错误(%+v)", _user, err)
					return
				}
				if _user.ID == 0 {
					orgUser.InvitationUserId = 0
				}
			}
			if err := conn.Table(models.OrganizationUserTbl()).Create(&orgUser).Error; err != nil {
				utils.Logger.Printf("付客户企业出错误: %v\n", orgUser)
				return
			}
		})
	}
	wg.Wait()
	return nil
}

func getPreviousOrgUser(info models.UserMigrationModel, phoneNumber string) (org models.OrganizationUser) {
	var userID uint

	for _, user := range info.OriginalUserList {
		if user.PhoneNumber == phoneNumber {
			userID = user.ID
			break
		}
	}

	if userID == 0 {
		return models.OrganizationUser{}
	}

	for _, orgUser := range info.OrganizationUserList {
		if orgUser.UserId == userID {
			return orgUser
		}
	}

	return models.OrganizationUser{}
}
