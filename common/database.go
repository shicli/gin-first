package common

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/logging/logrus"
	"gorm.io/plugin/opentelemetry/tracing"
	"time"
)

var DB *gorm.DB

func InitDB() (db *gorm.DB) {
	username := viper.GetString("datasource.username")
	password := viper.GetString("datasource.password")
	host := viper.GetString("datasource.host")
	port := viper.GetString("datasource.port")
	database := viper.GetString("datasource.database")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port, database)

	// Set logger
	logger := logger.New(
		logrus.NewWriter(),
		logger.Config{
			SlowThreshold: time.Millisecond,
			LogLevel:      logger.Warn,
			Colorful:      false,
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger})
	if err != nil {
		panic(fmt.Errorf("连接数据库失败：%s", err))
	}

	// Set tracing and metrics
	// db.Use(tracing.NewPlugin())
	// Set only tracing
	if err := db.Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
		panic(err)
	}

	// 自动创建数据表
	//err = db.AutoMigrate(&model.User{})
	//if err != nil {
	//	panic(fmt.Errorf("创建表失败：%s", err))
	//}
	DB = db
	return DB

	//// 监控配置文件变化
	//viper.WatchConfig()
	//// 注意！！！配置文件发生变化后要同步到全局变量Conf
	//viper.OnConfigChange(func(in fsnotify.Event) {
	//	fmt.Println("配置文件被修改...")
	//	if err := viper.Unmarshal(&config.yml); err != nil {
	//		panic(fmt.Errorf("unmarshal conf failed, err:%s \n", err))
	//	}
	//})
}

func GetDB() *gorm.DB {
	return DB
}
