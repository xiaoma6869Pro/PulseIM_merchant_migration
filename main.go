package main

import (
	"fmt"
	"log"
	"os"

	"com.pulseIM/app/models"
	"com.pulseIM/app/service"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
	"com.pulseIM/router"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func allowCrossOrigin(r *gin.Engine) {
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
}

func automateRunMigrationAB() {
	conf := "config/appConfig.yaml"
	data, err := os.ReadFile(conf)
	if err != nil {
		fmt.Errorf("读取配置文件失败 %w", err)
		return
	}
	var appConf models.AppConf
	err = yaml.Unmarshal(data, &appConf)
	if err != nil {
		fmt.Errorf("YAML解析失败: %w", err)
		return
	}
	//controller.RunMigration(appConf)
	//if err := service.MigrationUserAppService([]int{92, 93, 94, 95}, appConf.DbAppA); err != nil {
	//	fmt.Println(err)
	//}
	if err := service.MigrationNewUserAppInOldDbToNewDb(appConf.DbAppA, appConf.DbAppB); err != nil {
		log.Fatalf(err.Error())
	}
}

func runApiService() {
	r := gin.Default()
	allowCrossOrigin(r)
	router.SetupRouter(r)
	r.Run(":8080")
}

func main() {
	configPath := "config/db.yaml"

	err := utils.InitLog()
	if err != nil {
		log.Fatalf("初始化日子失败: %v", err)
	}

	if err := db.InitDatabases(configPath); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.CloseDatabases()
	automateRunMigrationAB()
	//runApiService()

}
