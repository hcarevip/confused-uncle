/*
 * File: 03-query-advanced.go
 * Date: 2025/10/7 19:28
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_orm

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func TestQueryAdvanced() {
	db := New()
	var dbName string
	db.Raw("SELECT DATABASE()").Scan(&dbName)
	fmt.Println("当前 DATABASE() =", dbName)

	// 选择字段（Select）没被SELECT填充过的使用原模型结构体接收 → 保留零值
	var users []User
	fields := []string{"id", "name", "age"}
	db.Select(fields).Find(&users)
	fmt.Printf("选择字段查询结果-users: %+v \n", users)

	// 选择字段（Select）使用 Map (Find(&[]map[string]any{}))
	var user1 []map[string]interface{}
	db.Model(&User{}).Select("uid", "name").Find(&user1)
	fmt.Printf("选择字段查询结果-user1: %+v \n", user1)

	// 过滤（Where、Not、Or、IN、LIKE）
	var user2 User
	db.Where("status = ?", 1).Or("age > ?", 15).Find(&user2)
	fmt.Printf("过滤查询结果-user2: %+v \n", user2)

	// 结构体与 Map 条件
	var user3 User
	// 结构体只会使用非零字段
	db.Where(&User{Status: 1, Age: 20}).Find(&user3)
	fmt.Printf("结构体过滤查询结果-user3: %+v \n", user3)

	var user4 User
	// Map：所有键值都会参与过滤
	db.Where(map[string]interface{}{"status": 1, "age": 20}).Find(&user4)
	fmt.Printf("Map过滤查询结果-user4: %+v \n", user4)

	var user5 []User
	db.Where("id in ?", []uint64{1, 2}).Find(&user5)
	fmt.Printf("IN 过滤查询结果-user5: %+v \n", user5)

	var user6 []User
	db.Where("name LIKE ?", "Ali%").Find(&user6)
	fmt.Printf("LIKE 过滤查询结果-user6: %+v \n", user6)

	// 排序（Order）
	var user7 []User
	db.Order("balance desc").Find(&user7)
	fmt.Printf("排序查询结果-user7: %+v \n", user7)
	// 多字段排序
	var user8 []User
	db.Order("status asc").Order("age desc").Find(&user8)
	fmt.Printf("多字段排序查询结果-user8: %+v \n", user8)
	// 排序风险：SQL 注入点,它只负责把字符串拼进去，不会做任何检查、转义、过滤
	// order := "name desc; DROP TABLE user;"
	// db.Order(order).Find(&users) 最终执行： SELECT * FROM user ORDER BY name desc; DROP TABLE user;

	// 正确的安全写法：白名单机制
	var user9 []User
	safeOrder(db, "age", "DESC").Find(&user9) // age DESC
	fmt.Printf("字段排序安全查询结果-user9: %+v \n", user9)

	// 分组（Group、Having） Having() → 分组后条件过滤
	type Result struct {
		Status int
		Total  int64
	}
	var results []Result
	db.Model(&User{}).
		Select("status, COUNT(*) as total").
		Group("status").
		Having("COUNT(*) > ?", 0).
		Scan(&results)
	//  [{Status:1 Total:1} {Status:2 Total:1}]
	fmt.Printf("分组查询结果-results: %+v \n", results)

	// 原生 SQL（Raw / Exec）
	type Stat struct {
		Status int
		Total  int64
	}
	var stats []Stat
	// 读取（Raw + Scan）
	db.Raw("SELECT status, COUNT(*) as total FROM uu_user GROUP BY status").Scan(&stats)
	//  [{Status:1 Total:1} {Status:2 Total:1}]
	fmt.Printf("Raw 查询结果-stats: %+v \n", stats)

	// 执行（Exec）
	tx := db.Exec("UPDATE uu_user SET status=? WHERE uid=?", 3, "U1002")
	fmt.Println(tx.RowsAffected)
}

func safeOrder(db *gorm.DB, field, dir string) *gorm.DB {
	safeCols := map[string]bool{"name": true, "age": true, "balance": true}
	dir = strings.ToUpper(dir)
	if !safeCols[field] {
		field = "created_at" // 默认字段
	}
	if dir != "DESC" && dir != "ASC" {
		dir = "ASC"
	}
	return db.Order(fmt.Sprintf("%s %s", field, dir))
}
