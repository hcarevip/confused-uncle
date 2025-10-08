/*
 * File: 01-connect.go
 * Date: 2025/10/7 19:28
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_orm

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

/*
	数据库连接、日志、命名策略、连接池
	关键机制
		gorm.Open(): 创建 ORM 引擎,不会立刻连数据库，只有执行 SQL 时才真正取连接
		NamingStrategy:
			SingularTable: true  表名不加复数
			TablePrefix: "app_"  表名加统一前缀
			NameReplacer:        自定义驼峰映射
		sql.DB 连接池
			MaxOpenConns  最大连接数
			MaxIdleConns  空闲连接
			ConnMaxLifetime  连接最大存活时间  建议值 20~30 分钟
			ConnMaxIdleTime  空闲最大存活时间  建议值 5~10 分钟
			当连接池满时，新的请求会等待连接释放，而不是打爆数据库，这正是背压保护机制
*/

// New 数据库初始化函数
func New() *gorm.DB {
	dsn := getDSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true, // 缓存预编译语句，加快热点路径执行
		Logger: logger.New(log.New(os.Stdout, "[GORM]", log.LstdFlags), logger.Config{
			SlowThreshold: 200 * time.Millisecond, // 慢查询阈值
			LogLevel:      logger.Warn,            // Warn：仅记录慢查询
			Colorful:      true,
		}),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,  // 表名不加 s
			TablePrefix:   "uu_", // 可选：统一前缀
		},
	})
	if err != nil {
		panic("GORM init failed: " + err.Error())
	}
	// 获取底层连接池对象
	sqlDB, _ := db.DB()
	configurePool(sqlDB)
	ping(sqlDB)
	log.Println("✅ DB connected successfully.")
	return db
}
func ping(sqlDB *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		panic("DB ping failed: " + err.Error())
	}
}
func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(100)                 // 最大连接数（并发上限）
	db.SetMaxIdleConns(20)                  // 最大空闲连接
	db.SetConnMaxLifetime(30 * time.Minute) // 连接最长存活时间
	db.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接存活时间
}
func getDSN() string {
	if v := os.Getenv("DSN"); v != "" {
		return v
	}
	return "root:a12345678@tcp(192.168.126.129:30822)/uu_coming?charset=utf8mb4&parseTime=True&loc=Local"
}
