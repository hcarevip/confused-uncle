/*
 * File: 02-crud-basic.go
 * Date: 2025/10/7 19:28
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_orm

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

/*
	基础 CRUD 与零值更新陷阱
	关键字详细解释
		1️⃣ primaryKey  标记字段为主键（PRIMARY KEY）GORM 通过此字段唯一标识记录；如果不指定，默认使用 ID 字段作为主键
		2️⃣ autoIncrement 自动递增主键（AUTO_INCREMENT）
		3️⃣ uniqueIndex / index  创建索引（唯一或普通）
		4️⃣ type 指定数据库中的列类型（MySQL 字段类型） type:varchar(32)
		5️⃣ comment 生成列注释，方便在数据库中查看表结构
		6️⃣ autoCreateTime / autoUpdateTime 自动写入时间戳
			CreatedAt time.Time `gorm:"autoCreateTime"` // 创建时写入
			UpdatedAt time.Time `gorm:"autoUpdateTime"` // 更新时写入
		7️⃣ decimal（精确小数） Go 没有原生小数类型（只有 float64，会有误差）数据库中可以手动指定 DECIMAL 类型
			Balance float64 `gorm:"type:decimal(10,2);comment:账户余额"`
		9️⃣ default 设置默认值： Status int `gorm:"default:1"`
		🔟 not null 强制非空： Name string `gorm:"type:varchar(64);not null"`
		unsigned = 无符号，只能表示 ≥ 0 的数，上限翻倍
	Find和 Scan 的根本区别
		Find 就是告诉 ORM：加载这个模型对象（实体）按模型定义
			自动处理时间、DeletedAt（软删除）
			调用生命周期钩子（BeforeFind / AfterFind）
			启动 ORM 特性（外键、关联、表名策略）
		Scan 就是告诉 ORM：只要把查询结果扫描进这个结构体里
			它只是执行 SQL,拿结果集,按列名对应字段名填进去
	Model vs Table
		Table("uu_user") 告诉 GORM：直接用这个表名执行 SQL，不要套任何模型逻辑
			GORM 只会生成最纯粹的 SQL，不带任何 ORM 特性
			通常用于统计报表、复杂多表 JOIN、临时表 / 视图、子查询场景
		Model(&User{}) 告诉 GORM：这次操作的目标是某个模型（结构体），请用它的表名、命名策略、软删除、钩子等所有 ORM 规则
			自动识别表名（可能是 users 或带前缀）
			自动加上 WHERE deleted_at IS NULL（如果模型有软删除）
			自动根据结构体字段匹配列名
			执行生命周期钩子（BeforeUpdate / AfterDelete 等）

*/

// User 定义模型
type User struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement;comment:主键ID"`
	UID   string `gorm:"uniqueIndex;type:varchar(32);not null;comment:业务唯一ID"`
	Name  string `gorm:"type:varchar(64);not null;comment:用户名"`
	Email string `gorm:"type:varchar(128);uniqueIndex;comment:邮箱"`
	Age   int    `gorm:"default:0;comment:年龄"`
	// decimal(18,2) 总共最多 18 位有效数字（包含小数部分）
	Balance   decimal.Decimal `gorm:"type:decimal(18,2) unsigned;default:0.00;comment:账户余额"`
	Status    int             `gorm:"default:1;comment:1=正常,0=禁用"`
	CreatedAt time.Time       `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt gorm.DeletedAt  `gorm:"index;comment:软删除时间"` // gorm.DeletedAt 启用逻辑删除
}

func Create(db *gorm.DB) {
	// 单条插入
	c, _ := decimal.NewFromString("1000")
	u := User{
		UID:     "U1001",
		Name:    "Alice",
		Status:  1,
		Age:     25,
		Balance: c,
		Email:   "Alice-U1001@gmail.com",
	}
	// 如果用 db.Create(users)（不加 &），GORM不会自动写回自增ID
	// GORM 默认忽略字段的 零值（0, "", false, nil），插入时只写非零值
	db.Create(&u)
	fmt.Println("插入后主键ID:", u.ID)
}

func CreateInBatches(db *gorm.DB) {
	c1, _ := decimal.NewFromString("1080")
	c2, _ := decimal.NewFromString("2800")
	users := []User{
		{
			UID:     "U1002",
			Name:    "Tom",
			Status:  2,
			Age:     20,
			Balance: c1,
			Email:   "Tom-U1002@gmail.com",
		},
		{
			UID:     "U1003",
			Name:    "Sam",
			Status:  3,
			Age:     15,
			Balance: c2,
			Email:   "Sam-U1001@gmail.com",
		},
	}
	db.CreateInBatches(&users, 10)
}

func Update(db *gorm.DB) {
	// 单字段更新
	db.Model(&User{}).Where("uid = ?", "U1001").Update("balance", 2000.8809)
}
func Updates(db *gorm.DB) {
	// 多字段更新
	db.Model(&User{}).Where("uid = ?", "U1002").Updates(map[string]interface{}{
		"name":    "Alice Zhang",
		"age":     18, // 会更新
		"balance": 3000.08,
	})
}

func StructUpdates(db *gorm.DB) {
	// 结构体更新（零值陷阱）GORM 会忽略 Age=0 和 Status=0，因为 0 被认为是零值
	/*
		u := User{
			UID:    "Alice L",
			Age:    0,
			Status: 0,
		}
		db.Model(&User{}).Where("uid = ?", "U1001").Updates(&u)
	*/

	// 解决
	// 1 推荐：显式传递键值，GORM 不会跳过零值  map[string]interface{}
	// 2 Select 强制选择字段
	u := User{
		UID:    "Alice L",
		Age:    0,
		Status: 0,
	}
	db.Model(&User{}).Select("uid", "age", "status").Where("id = ?", "1").Updates(&u)

	// 3 使用指针字段 *int / *bool

}

func SoftDelete(db *gorm.DB) {
	db.Delete(&User{}, "uid = ?", "U1003")
}

func HardDelete(db *gorm.DB) {
	// 调用 db.Unscoped() 后，本次链路不再自动附带 deleted_at IS NULL 条件
	// 没有 DeletedAt 字段时：GORM没有软删除，这时 Delete 本身就是物理删；Unscoped() 不影响结果
	db.Unscoped().Where("uid = ?", "U1003").Delete(&User{})
}

func TestCurd() {
	db := New()
	var dbName string
	db.Raw("SELECT DATABASE()").Scan(&dbName)
	fmt.Println("当前 DATABASE() =", dbName)

	// err := db.AutoMigrate(&User{}) // 映射成数据库表结构
	// if err != nil {
	// 	fmt.Println("AutoMigrate err:", err)
	// }

	// Create(db)
	// CreateInBatches(db)

	// 查询数据-按主键查找
	var u1 User
	db.First(&u1, 1)
	fmt.Printf("按主键查找结果-u1: %+v \n", u1)

	// 查询数据-按条件查找
	var u2 User
	db.Where("uid = ?", "U1002").First(&u2)
	fmt.Printf("按条件查找-u2: %+v \n", u2)

	var u3 []User
	db.Where("age > ?", "15").Find(&u3)
	fmt.Printf("按条件查找-u3: %+v \n", u3)

	// 查询列选择
	type UserLite struct {
		UID     string
		Name    string
		Balance decimal.Decimal `gorm:"type:decimal(18,2)"`
	}
	var rows []UserLite
	// 目标 []UserLite 不是模型，GORM无法从它推断表；必须用 Model(&User{}) 或 Table(...)
	// db.Model(&User{}).
	//	 Select("uid", "name", "balance").
	//	 Where("status = ?", 1).
	//	 Scan(&rows)
	db.Table("uu_user").
		Select("uid", "name", "balance").
		Where("status = ?", 1).
		Scan(&rows)
	fmt.Printf("按条件查找-列选择-rows: %+v \n", rows)

	// 查询统计
	var count int64
	db.Model(&User{}).Where("age > ?", 20).Count(&count)
	fmt.Println("满足条件的用户数:", count)

	// Update(db)
	// Updates(db)
	// StructUpdates(db)

	// 软删除（推荐）gorm.DeletedAt 只是把 deleted_at 写上时间
	// SoftDelete(db)

	// 物理删除
	HardDelete(db)
}
