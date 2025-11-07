package db

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 线程安全的Map，用于存储多个数据库连接
var (
	DatabaseConnections = make(map[string]*gorm.DB)
	dbMutex             sync.RWMutex
)

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type DatabaseConfig struct {
	Databases map[string]DBConfig `yaml:"databases"`
}

func LoadConfig(filepath string) (*DatabaseConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config DatabaseConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &config, nil
}

func InitDatabases(configPath string) error {
	log.Println("正在加载数据库配置...")

	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("正在加载数据库配置失败: %w", err)
	}

	if len(config.Databases) == 0 {
		return fmt.Errorf("YAML文件中未配置数据库 ")
	}

	log.Printf("找到 %d 数据库(s)链接", len(config.Databases))

	successCount := 0
	failedDBs := []string{}

	for dbName, dbConfig := range config.Databases {
		log.Printf("🔌 正在链接 '%s' 数据库...", dbName)

		db, err := connectDB(dbConfig)
		if err != nil {
			log.Printf("数据库链接失败 '%s': %v", dbName, err)
			failedDBs = append(failedDBs, dbName)
			continue
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Printf("获取数据库实例失败 '%s': %v", dbName, err)
			failedDBs = append(failedDBs, dbName)
			continue
		}

		if err := sqlDB.Ping(); err != nil {
			log.Printf("❌ 数据库Ping失败  '%s': %v", dbName, err)
			failedDBs = append(failedDBs, dbName)
			continue
		}

		// 线程安全地将连接存储到Map中
		dbMutex.Lock()
		DatabaseConnections[dbName] = db
		dbMutex.Unlock()

		successCount++
		log.Printf("已连接 '%s' → %s@%s:%s/%s",
			dbName, dbConfig.User, dbConfig.Host, dbConfig.Port, dbConfig.Name)
	}

	log.Printf("已连接成功: %d/%d 数据库", successCount, len(config.Databases))

	if len(failedDBs) > 0 {
		log.Printf("连接失败: %v", failedDBs)
		return fmt.Errorf("连接到某些数据库失败 : %v", failedDBs)
	}

	log.Println("所有数据库连接成功！")

	return nil
}

func connectDB(config DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Name,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 获取底层SQL数据库以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

func GetConnectionDB(name string) (*gorm.DB, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	db, exists := DatabaseConnections[name]
	println()
	if !exists {
		return nil, fmt.Errorf("数据库链接 '%s' 不存在", name)
	}
	return db, nil
}
func GetAllDBNames() []string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	names := make([]string, 0, len(DatabaseConnections))
	for name := range DatabaseConnections {
		names = append(names, name)
	}
	return names
}

func HasDB(name string) bool {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	_, exists := DatabaseConnections[name]
	return exists
}

func GetConnectionCount() int {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	return len(DatabaseConnections)
}

func CloseDatabases() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	log.Println("数据库链接关闭...")

	for dbName, db := range DatabaseConnections {
		if db != nil {
			sqlDB, _ := db.DB()
			if err := sqlDB.Close(); err != nil {
				log.Printf("关闭'%s'数据库时出错 %v", dbName, err)
			} else {
				log.Printf("'%s' 数据库链接已关闭", dbName)
			}
		}
	}

	log.Printf("全部 %d 数据库链接(s)已关闭", len(DatabaseConnections))
}

func HealthCheck() map[string]string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	status := make(map[string]string)

	for name, db := range DatabaseConnections {
		sqlDB, err := db.DB()
		if err != nil {
			status[name] = "error: " + err.Error()
			continue
		}

		if err := sqlDB.Ping(); err != nil {
			status[name] = "disconnected: " + err.Error()
		} else {
			status[name] = "connected"
		}
	}
	return status
}
