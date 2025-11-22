package service

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"com.pulseIM/app/models"
	"com.pulseIM/app/utils"
	"com.pulseIM/db"
)

func DecryptMD5(hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	hashType := "md5"
	apiURL := fmt.Sprintf("https://md5decrypt.net/en/Api/api.php?hash=%s&hash_type=%s&email=%s&code=%s",
		url.QueryEscape(hash),
		hashType,
		url.QueryEscape("deanna_abshire@gmail.com"),
		"a29ca0d4fd7dd8ed",
	)

	fmt.Printf("正在解密哈希值: %s\n", hash)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("无法读取响应: %v", err)
	}

	result := strings.TrimSpace(string(body))

	if result == "" || result == "NOT_FOUND" || strings.Contains(result, "ERROR") {
		return "", fmt.Errorf("哈希值在数据库中未找到")
	}

	return result, nil
}

func CheckSecurePassword(dbName string) error {
	mysql, err := db.GetConnectionDB(dbName)
	if err != nil {
		return fmt.Errorf("无法连接到数据库: %w", err)
	}

	var userList []models.User
	if err := mysql.Table(models.UserTbl()).Limit(10).Where("deleted_at IS NULL").Find(&userList).Error; err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	for _, user := range userList {
		decodedPassword, _ := DecryptMD5(user.Password)
		if decodedPassword != "" {
			utils.Logger.Printf("用户 (%d-%s) 使用了弱密码", user.ID, decodedPassword)
		}
	}

	return nil
}
