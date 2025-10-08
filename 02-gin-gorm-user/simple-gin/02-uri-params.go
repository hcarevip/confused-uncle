/*
 * File: 02-uri-params.go
 * Date: 2025/9/9 23:42
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"fmt"
	"net/http"
	"strconv"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

/*
	路由参数
		获取单个资源：GET /users/:id（路径参数标识唯一资源）
		列表 + 筛选：GET /users?role=admin&page=2&size=20（查询参数做过滤与分页）
		子资源：GET /users/:id/orders
*/

// -------- DTOs（请求绑定结构）--------

type UserURI struct {
	// 把路由路径参数（/users/:id）绑定到结构体字段
	// binding 告诉Gin使用 validator.v10 对该字段做校验
	ID string `uri:"id" binding:"required,uuid4"`
}
type OrderURI struct {
	ID string `uri:"id" binding:"required,numeric,len=19"` // 如：雪花ID 19位数字
}
type AssetPathURI struct {
	Filepath string `uri:"filepath" binding:"required"`
}

// -------- 通用错误响应 --------

type ErrResp struct {
	Error string `json:"error"`
	// omitempty: 序列化为JSON时，如果该字段是零值（空串、0、false、nil），就不输出这个字段
	// json:",omitempty"：控制序列化输出（只对结构体字段有效，对 gin.H/map 无效）
	Details string `json:"details,omitempty"`
}

func badRequest(ctx *gin.Context, err error) {
	ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrResp{
		Error:   "invalid uri params",
		Details: err.Error(),
	})
}

// -------- Handlers（演示路径参数用法）--------

func GetUserByID(ctx *gin.Context) {
	var u UserURI
	// ctx.Params [{id 3732b247-f000-4683-b20f-8e0be4f052ba}]
	if err := ctx.ShouldBindUri(&u); err != nil {
		badRequest(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"id":       u.ID,
		"fullPath": ctx.FullPath(), // /api/v1/users/:id
	})
}

func GetOrderByID(ctx *gin.Context) {
	var o OrderURI
	if err := ctx.ShouldBindUri(&o); err != nil {
		badRequest(ctx, err)
		return
	}
	// 如果需要数值型
	orderID, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		badRequest(ctx, err)
		return
	}
	// TODO: 查询订单...
	ctx.JSON(http.StatusOK, gin.H{
		"id":       orderID,
		"fullPath": ctx.FullPath(),
	})
}

// 子资源 + 路径参数 + Query 混用

type ListQuery struct {
	// form:"page,default=1" 表示：当该键缺失或空值时，先填充默认值，再进入校验（binding）
	// Query/表单：用 form:"..." + c.ShouldBindQuery/ShouldBind(&x) 作用是指定 Query/表单 参数名
	Page int    `form:"page,default=1"  binding:"gte=1,lte=1000"`
	Size int    `form:"size,default=20" binding:"gte=1,lte=200"`
	Sort string `form:"sort"` // e.g. -createdAt
}

func ListUserOrders(ctx *gin.Context) {
	var u UserURI
	// 用 URI 绑定器读取 /users/:id
	if err := ctx.ShouldBindUri(&u); err != nil {
		badRequest(ctx, err)
		return
	}
	var q ListQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		badRequest(ctx, err)
		return
	}
	// TODO: 查询 u.ID 的订单列表，分页/排序按 q
	ctx.JSON(http.StatusOK, gin.H{
		"userId": u.ID, "page": q.Page, "size": q.Size, "sort": q.Sort,
	})
}

// 通配符：捕获剩余路径（含斜杠）

func GetAsset(ctx *gin.Context) {
	var p AssetPathURI
	if err := ctx.ShouldBindUri(&p); err != nil {
		badRequest(ctx, err)
		return
	}
	// p.Filepath 如 /api/v1/assets/images/logo.png
	// TODO: 安全拼接，防止目录穿越（不要直接拼接相对路径）
	ctx.JSON(http.StatusOK, gin.H{"path": p.Filepath})
}

// 多参数绑定（层级资源）
// 自定义校验

func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("alphanumerics", func(fl validator.FieldLevel) bool {
			s := fl.Field().String()
			if s == "" {
				return false
			}
			for _, r := range s {
				if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
					continue
				}
				return false
			}
			return true
		})
	}
}

type RepoURI struct {
	Org  string `uri:"org"  binding:"required,alphanumerics"` // 自定义规则
	Repo string `uri:"repo" binding:"required,alphanumerics"`
}

func GetRepo(ctx *gin.Context) {
	var u RepoURI
	// gin.Params{gin.Param{Key:"org", Value:"1"}, gin.Param{Key:"repo", Value:"2"}} fullPath=/api/v1/repos/:org/:repo
	fmt.Printf("params=%#v fullPath=%s\n", ctx.Params, ctx.FullPath())
	if err := ctx.ShouldBindUri(&u); err != nil {
		badRequest(ctx, err)
		return
	}
	ctx.JSON(200, gin.H{"org": u.Org, "repo": u.Repo})
}

// -------- 路由注册 --------

func RegisterURIRoutesUriParams(r gin.IRouter) {
	api := r.Group("/api/v1")

	users := api.Group("/users")
	{
		users.GET("/:id", GetUserByID)
		users.GET("/:id/orders", ListUserOrders)
	}

	orders := api.Group("/orders")
	{
		orders.GET("/:id", GetOrderByID)
	}

	repo := api.Group("/repos")
	{
		repo.GET("/:org/:repo", GetRepo)
	}
	assets := api.Group("/assets")
	{
		// IMPORTANT: 通配符必须写在组里最后，避免抢占别的路由
		// /*filepath 会把余下所有内容（含 /）捕获到一个参数
		assets.GET("/*filepath", GetAsset)
	}
}
