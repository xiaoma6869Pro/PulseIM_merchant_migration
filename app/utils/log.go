package utils

import (
	"fmt"
	"log"
	"os"
)

var (
	Logger *log.Logger
)

func InitLog() error {
	logFile, err := os.OpenFile("migrationFile.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("启动日子文件失败: %w", err)
	}
	Logger = log.New(logFile, "", log.LstdFlags)
	return nil
}
