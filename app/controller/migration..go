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
	}

	base.ResponseJson(c, utils.CodeSuccess, "用户导入成功", map[string]interface{}{
		"total_count": len(result.UserList),
	})
}

func RunMigration(conf models.AppConf) {
	result, err := service.GetVerifyUserAppAB(conf.DbAppA, conf.DbAppB, conf.OrganizationAppAId)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(result.UserList) == 0 {
		fmt.Println("========================AppUserA 暂时没数据=======================")
		return
	}

	if err := service.ImportUserAppAToAppB(conf.DbAppB, *result, conf.OrganizationAppBId); err != nil {
		fmt.Println(err)
	}
}
