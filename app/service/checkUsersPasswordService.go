package service

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

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

func CheckSecurePassword(dbName string) error {
	mysql, err := db.GetConnectionDB(dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	var userList []models.User
	if err := mysql.Table(models.UserTbl()).Limit(10).Where("deleted_at IS NULL").Find(&userList).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	for _, user := range userList {
		decodedPassword, _ := DecryptMD5(user.Password)

		isStrong := isStrongPassword(decodedPassword)

		if isStrong {
			utils.Logger.Printf("User ID %d has a strong password", user.ID)
		} else {
			utils.Logger.Printf("User ID %d has a weak password", user.ID)
		}
	}

	return nil
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	criteriaCount := 0
	if hasUpper {
		criteriaCount++
	}
	if hasLower {
		criteriaCount++
	}
	if hasNumber {
		criteriaCount++
	}
	if hasSpecial {
		criteriaCount++
	}

	return criteriaCount >= 3
}
