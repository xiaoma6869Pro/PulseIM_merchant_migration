package models

type AppConf struct {
	DbAppA             string `yaml:"dbAppA" json:"dbAppA" binding:"required"`                         // 名字查 App-A
	OrganizationAppAId int    `yaml:"organizationAppAId" json:"organizationAppAId" binding:"required"` // 企业ID App-A
	DbAppB             string `yaml:"dbAppB" json:"dbAppB" binding:"required"`                         // 名字查 App-B
	OrganizationAppBId int    `yaml:"organizationAppBId" json:"organizationAppBId" binding:"required"` // 企业ID App-B
}
