/*
 * File: 03-query-params.go
 * Date: 2025/9/9 23:43
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"encoding"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

/*
筛选 / 分页 / 排序 / 搜索 / 可选开关 都走 QueryString（?page=2&status=paid）
在 Gin 中的绑定入口 ctx.ShouldBindQuery(&dst) → 只从 URL 查询串绑定

*/

// -------- 通用错误响应 --------

type ErrRespForQuery struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func badRequestForQuery(ctx *gin.Context, err error) {
	ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrRespForQuery{
		Error:   "参数异常",
		Details: err.Error(),
	})
}

// -------- 基础分页/排序（含默认值与校验） --------
// 规则：page/size 默认值 + 范围校验；sort 限定白名单

type ListQueryDemo struct {
	// binding Gin 专用
	Page int `form:"page,default=1"  binding:"gte=1,lte=1000"`
	Size int `form:"size,default=20" binding:"gte=1,lte=200"`
	// omitempty: 当字段是零值时跳过后续校验,如果客户端没传 sort（或传了空串），不会触发后面的 oneof 校验，直接通过
	// oneof：值必须在给定“候选集合”中；候选项用空格分隔，"name"、"-name"、"createdAt"、"-createdAt"
	// binding:"omitempty"：跳过校验（绑定/验证阶段），与输出无关
	Sort string `form:"sort"            binding:"omitempty,oneof=name -name createdAt -createdAt"`
	// 绑定后 Limit == nil
	Limit *int `form:"limit" binding:"omitempty,gte=0,lte=1000"`
}
type ListUsersResp struct {
	Page  int    `json:"page"`
	Size  int    `json:"size"`
	Sort  string `json:"sort,omitempty"` // ← 为空则不输出
	Limit *int   `json:"limit,omitempty"`
}

// Handler：GET /api/v1/users?page=2&size=50&sort=-createdAt

func ListUsers(ctx *gin.Context) {
	var query ListQueryDemo
	f := reflect.ValueOf(&query).Elem().FieldByName("Sort")
	fmt.Println("implements on addr?",
		f.Addr().Type().Implements(reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()))
	if err := ctx.ShouldBindQuery(&query); err != nil {
		badRequestForQuery(ctx, err)
		return
	}
	fmt.Println("1) after ShouldBindQuery, query =", query)
	// TODO: 业务
	// 模拟业务默认值：没传 limit 就用 20
	limit := 20
	if query.Limit != nil {
		limit = *query.Limit
	}
	/*
		json:",omitempty"：控制序列化输出（只对结构体字段有效，对 gin.H/map 无效）gin.H 是 map[string]any
		ctx.JSON(http.StatusOK, gin.H{
			"page": query.Page, "size": query.Size, "sort": query.Sort, "limit": limit,
		})
	*/
	ctx.JSON(200, ListUsersResp{
		Page:  query.Page,
		Size:  query.Size,
		Sort:  query.Sort,
		Limit: &limit,
	})
}

// 多值参数：数组/集合
// 方式 A：重复 key（推荐） ?tags=a&tags=b&tags=c  -> []string{"a","b","c"}
// 方式 B：逗号分隔        ?tags=a,b,c               -> 自定义 CSV 类型解析

type TagsRepeat struct {
	// dive: 告诉 validator：对切片/数组/映射的每个元素逐一应用后续规则
	// 没有 dive 时，min/max 会作用在切片长度，有了 dive 才会作用在元素本身
	Tags []string `form:"tags" binding:"omitempty,dive,min=1,max=32"` // 重复 key
}

type SomeTest []string

// UnmarshalText Go 的参数一律按值传递。即便是指针接收器，传进来的也是指针的一个副本
func (st *SomeTest) UnmarshalText(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" {
		// c = nil 只是把这个副本指针改成 nil，对调用方没影响
		// 要改调用方持有的值，必须对指针解引用赋值：*c
		*st = nil
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return errors.New("empty item in csv")
		}
		out = append(out, p)
	}
	*st = out // 相当于 *c = SomeTest(out)
	return nil
}

/*
var _ Interface = (*Type)(nil) = 编译期保证 *Type 实现了 Interface
把这个 *CSV 值 赋给左边的接口类型变量（encoding.TextUnmarshaler）
编译器在赋值时会检查：*CSV 的方法集是否满足 encoding.TextUnmarshaler（即是否声明了 UnmarshalText([]byte) error）
如果不满足，直接编译报错；满足则静默通过。变量名用 _（空白标识符）表示不使用，无运行时成本
func (c *CSV) UnmarshalText 指针接收者，则 只有 *CSV 实现接口，CSV（值）不实现，所以要用 (*CSV)(nil) 来触发检查
只有编译期检查，运行时没有额外开销（(*CSV)(nil) 不会被解引用）
*/
var _ encoding.TextUnmarshaler = (*SomeTest)(nil) // 把 nil 转成 *CSV 类型

type CSV []string
type TagsCSV struct {
	Tags CSV `form:"tags" binding:"omitempty,dive,min=1,max=32"`
}

func ListItems(c *gin.Context) {
	var r TagsRepeat // tags=a&tags=b&tags=c
	var v TagsCSV    // tags=a,b,c
	_ = c.ShouldBindQuery(&r)
	_ = c.ShouldBindQuery(&v) // 两种都支持，谁有值用谁
	var tags []string
	if len(r.Tags) > 0 {
		tags = r.Tags
	} else {
		tags = []string(v.Tags)
	}
	c.JSON(200, gin.H{"tags": tags})
}

// --------  时间范围绑定（时区/格式） 如GET /api/v1/logs?from=2025-09-10&to=2025-09-12T18:00:01 --------

type TimeRangeQuery struct {
	// from 用日期，to 用时间戳示例；展示多格式解析
	From time.Time `form:"from" time_format:"2006-01-02" time_location:"Asia/Shanghai" binding:"required"`
	To   time.Time `form:"to"   time_format:"2006-01-02T15:04:05" time_utc:"true"      binding:"required,gtfield=From"`
}

func GetLogs(ctx *gin.Context) {
	var r TimeRangeQuery
	if err := ctx.ShouldBindQuery(&r); err != nil {
		badRequestForQuery(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"from": r.From, "to": r.To, "durationHours": r.To.Sub(r.From).Hours(),
	})
}

// 直接用原生 API（GetQueryArray / QueryMap）

func RawQueryExample(ctx *gin.Context) {
	// 单值（有/无）
	if v, ok := ctx.GetQuery("q"); ok {
		_ = v
	}

	// 多值
	tags := ctx.QueryArray("tags")
	// Map：filters[name]=alice&filters[age]=20 → map[name:alice age:20]
	filters := ctx.QueryMap("filters")

	ctx.JSON(200, gin.H{
		"tags": tags, "filters": filters,
	})
}

// 严格模式：拒绝未知参数 有些审计要求不接受未声明的查询键，下面是简单实现

func EnforceKnownKeys(ctx *gin.Context, allow map[string]struct{}) error {
	q := ctx.Request.URL.Query()
	for key := range q {
		if _, ok := allow[key]; !ok {
			return fmt.Errorf("unknown query key: %s", key)
		}
	}
	return nil
}

func StrictListUsers(ctx *gin.Context) {
	allow := map[string]struct{}{"page": {}, "size": {}, "sort": {}}
	if err := EnforceKnownKeys(ctx, allow); err != nil {
		badRequestForQuery(ctx, err)
		return
	}
	var q ListQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		badRequest(ctx, err)
		return
	}
	ctx.JSON(200, gin.H{"page": q.Page, "size": q.Size, "sort": q.Sort})
}
func RegisterQueryRoutesQueryParams(r gin.IRouter) {
	api := r.Group("/api/v1")
	{
		api.GET("/users", ListUsers)
		api.GET("/items", ListItems)
		api.GET("/logs", GetLogs)
		api.GET("/raw", RawQueryExample)
		api.GET("/users/strict", StrictListUsers)
	}
}
