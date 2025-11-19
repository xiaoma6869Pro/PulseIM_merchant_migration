package controller

import (
	"fmt"

	"com.pulseIM/app/controller/base"
	"com.pulseIM/app/models"
	"com.pulseIM/app/service"
	"com.pulseIM/app/utils"
	"github.com/gin-gonic/gin"
)

func MigrationUserAppAB(c *gin.Context) {
	var params models.AppConf
	if err := c.ShouldBindJSON(&params); err != nil {
		base.ResponseJson(c, utils.CodeWrongParams, err.Error(), nil)
		return
	}

	result, err := service.GetVerifyUserAppAB(params.DbAppA, params.DbAppB, params.OrganizationAppAId)
	if err != nil {
		base.ResponseJson(c, utils.CodeInternalServerError, err.Error(), nil)
		return
	}

	if len(result.UserList) == 0 {
		base.ResponseJson(c, utils.CodeInternalServerError, "AppUserA 暂时没数据", nil)
		return
	}

	if err := service.ImportUserAppAToAppB(params.DbAppB, *result, params.OrganizationAppBId); err != nil {
		base.ResponseJson(c, utils.CodeInternalServerError, err.Error(), nil)
		return
	}

	base.ResponseJson(c, utils.CodeSuccess, "用户导入成功", map[string]interface{}{
		"total_count": len(result.UserList),
	})
}

// 获取账户交易限制
func getLimitAccountTransaction(length int, userInfo models.UserMigrationModel) (*models.UserMigrationModel, error) {
	if length <= 0 {
		return nil, fmt.Errorf("长度必须大于0")
	}
	result := userInfo
	if len(userInfo.UserList) > length {
		result.UserList = userInfo.UserList[:length]
	}
	return &result, nil
}

// RunMigration 运行数据迁移
func RunMigration(conf models.AppConf) {
	result, err := service.GetVerifyUserAppAB(conf.DbAppA, conf.DbAppB, conf.OrganizationAppAId)
	if err != nil {
		fmt.Printf("验证用户AppAB出错误:%s\n", err.Error())
		return
	}
	if len(result.UserList) == 0 {
		fmt.Println("========================AppUserA 暂时没数据=======================")
		return
	}

	//ret, err := getLimitAccountTransaction(10, *result)
	//if err != nil {
	//	fmt.Printf("获取账户交易限制失败%+v\n", err)
	//	return
	//}

	//if err := service.ImportUserAppAToAppB(conf.DbAppB, *result, conf.OrganizationAppBId); err != nil {
	//	fmt.Printf("转移客户A到客户B出现错误:%s\n", err.Error())
	//	return
	//}

	//if err := service.AssignOrganizationToExitingClient(conf.DbAppB, conf.OrganizationAppBId, *result); err != nil {
	//	fmt.Printf("内部服务有问题%v", err)
	//
	//}

	fmt.Println("=========================== 所有交易迁移成功 ===========================")
}
