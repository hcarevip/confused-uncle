/*
 * File: 04-scopes-pagination.go
 * Date: 2025/10/7 19:28
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_orm

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

/*
	Scope本质把你经常写的查询条件、排序、分页逻辑，封装成可复用的函数，再用 db.Scopes() 组合调用
	Scope 的函数签名与执行机制
		固定格式：func(db *gorm.DB) *gorm.DB
		这个函数接收当前的 gorm.DB，返回修改后的 gorm.DB
	Scope 的使用注意事项
		不要在 Scope 里直接 Find()，Scope 应该只拼条件，执行在外层
		通过指针区分“未传入” vs “传了零值”
	SQL 的分页语句通常是
		SELECT * FROM user LIMIT <数量> OFFSET <起始行>;
		LIMIT 表示 取多少行  OFFSET 表示 跳过多少行（从第几条开始取）
		Offset = (page - 1) * pageSize
		那么第 1 页要跳过 0 行；第 2 页要跳过 1 页的 10 行； 第 3 页要跳过 2 页的 20 行；
*/

// 定义多个 Scope

func ActiveUsers(db *gorm.DB) *gorm.DB {
	return db.Where("status = ?", 1)
}

func OlderThan(age int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("age  > ?", age)
	}
}

func SortByBalance(desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		order := "balance asc"
		if desc {
			order = "balance desc"
		}
		return db.Order(order)
	}
}

type UserFilter struct {
	// GORM在处理结构体条件时，只会用“非零值字段”生成 WHERE 条件，而 int 类型的默认值是 0，它会被认为是「没赋值，不拼进 SQL」
	// 如当想查 status=0，被当成零值忽略 就需要使用指针就能区分 未传 与 确实传 0
	// 改成指针 status=0，会生成 WHERE status = 0
	Status   *int
	NameLike string
	MinAge   *int
	MaxAge   *int
	OrderBy  string
}

func WithFilter(f UserFilter) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if f.Status != nil {
			db = db.Where("status = ?", *f.Status)
		}
		if f.NameLike != "" {
			db = db.Where("name LIKE ?", "%"+f.NameLike+"%")
		}
		if f.MinAge != nil {
			db = db.Where("age >= ?", *f.MinAge)
		}
		if f.MaxAge != nil {
			db = db.Where("age <= ?", *f.MaxAge)
		}
		switch f.OrderBy {
		case "balance_desc":
			db = db.Order("balance desc")
		case "balance_asc":
			db = db.Order("balance asc")
		default:
			db = db.Order("created_at desc")
		}
		return db
	}
}

func ptr[T any](v T) *T {
	return &v
}

func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func QueryUserList(db *gorm.DB, f UserFilter, page, pageSize int) ([]User, int64, error) {
	var total int64
	var users []User
	query := db.Model(&User{}).Scopes(WithFilter(f))
	query.Count(&total)
	if total == 0 {
		return nil, 0, nil
	}
	err := query.Scopes(Paginate(page, pageSize)).Find(&users).Error
	return users, total, err
}
func TestScopesPagination() {
	db := New()
	var dbName string
	db.Raw("SELECT DATABASE()").Scan(&dbName)
	fmt.Println("当前 DATABASE() =", dbName)

	var users []User
	// SELECT * FROM user WHERE status=1 AND age>20 ORDER BY balance DESC;
	db.Scopes(ActiveUsers, OlderThan(10), SortByBalance(false)).Find(&users)
	fmt.Printf("Scopes组合查询结果-users: %+v \n", users)

	// Scopes 动态过滤,实际业务常见场景 用户列表页往往有这些过滤条件,如name模糊匹配, 年龄区间, 注册时间区间
	// 这些条件不是固定的，要根据前端传的参数决定是否拼入
	// ptr(1) 把整数 1 转换成 *int 类型的指针
	var users1 []User
	var filter = UserFilter{Status: ptr(1), NameLike: "Ali"}
	db.Scopes(WithFilter(filter)).Find(&users1)
	fmt.Printf("Scopes动态过滤查询结果-users1: %+v \n", users1)

	// 分页（Pagination） 通过 Limit 和 Offset 实现
	var users2 []User
	result := db.Scopes(WithFilter(filter), Paginate(1, 10)).Find(&users2)
	if result.Error != nil {
		log.Println("query failed:", result.Error)
	} else {
		fmt.Printf("分页查询结果-users2: %+v \n", users2)
	}

	// 推荐的标准分页 + 动态过滤模板
	userList, total, err := QueryUserList(db, filter, 1, 10)
	if err != nil {
		log.Println("query failed:", result.Error)
		return
	}
	fmt.Printf("查询结果-userList: %+v  总条数: %d \n", userList, total)
}
