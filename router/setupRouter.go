package router

import (
	"com.pulseIM/app/controller"
	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	api := r.Group("api")
	{
		api.POST("/migration_user_app_ab", controller.MigrationUserAppAB)
	}
}
