/*
 * File: 11-mixed-binding.go
 * Date: 2025/9/9 23:45
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validateMixed = validator.New()

type MinItem struct {
	Sku   string  `json:"sku" validate:"required"`
	Qty   int     `json:"qty" validate:"required,gt=0"`
	Price float64 `json:"price" validate:"required,gt=0"`
}
type MixedRequest struct {
	// 路径参数
	UserID string `uri:"user_id" validate:"required"` // /v1/users/:user_id
	// 查询参数
	Page    int    `form:"page" validate:"omitempty,min=1"`
	PerPage int    `form:"per_page" validate:"omitempty,min=1,max=200"`
	Search  string `form:"search" validate:"omitempty,max=64"`
	// 请求体
	OrderID    string  `json:"order_id" msgpack:"order_id" validate:"required,len=18"`
	Amount     float64 `json:"amount"   msgpack:"amount"   validate:"required,gt=0"`
	Currency   string  `json:"currency" msgpack:"currency" validate:"required,oneof=CNY USD PHP"`
	CustomerID string  `json:"customer_id" msgpack:"customer_id" validate:"required"`
	Email      string  `json:"email" msgpack:"email" validate:"omitempty,email"`
	Items      []Item  `json:"items" msgpack:"items" validate:"min=1,dive"`
}

func MixedBindURIQueryBody[T any](c *gin.Context, out *T) (int, error) {
	// 绑定 URI（必须最先，且失败→400）
	if err := c.ShouldBindUri(out); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid URI: %w", err)
	}
	// 绑定 Query（失败→400；只填充 form/query 标签对应字段）
	if err := c.ShouldBindQuery(out); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid query: %w", err)
	}
	//  绑定 Body
	ct := c.GetHeader("Content-Type")
	if ct != "" && !strings.Contains(strings.ToLower(ct), "json") {
		return http.StatusUnsupportedMediaType, fmt.Errorf("unsupported Content-Type %q", ct)
	}
	// if err := c.ShouldBindJSON(out); err != nil {
	//	 return http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err)
	// }
	raw, err := c.GetRawData()
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("read body error: %w", err)
	}
	// GetRawData() 会把 body 全部读掉 如果后面代码还要读body，就会发现已经 EOF
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw)) // 复原请求体
	// 反序列化 JSON
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err)
	}
	// 传 &out 会变成 指向指针的指针（**T）
	fmt.Printf("out: %+v\n", out)
	if err := validateMixed.Struct(out); err != nil {
		var verse validator.ValidationErrors
		if errors.As(err, &verse) {
			return http.StatusUnprocessableEntity, fmt.Errorf("validation failed: %s", verse.Error())
		}
		return http.StatusUnprocessableEntity, fmt.Errorf("validation error: %w", err)
	}
	return http.StatusOK, nil
}

func CreateOrderMixedBinding(ctx *gin.Context) {
	var req MixedRequest
	status, err := MixedBindURIQueryBody(ctx, &req)
	if err != nil {
		ctx.JSON(status, gin.H{"code": status, "message": err.Error()})
		return
	}
	// TODO: 业务逻辑……
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "ok",
		"data": gin.H{
			"user_id": req.UserID, "page": req.Page, "per_page": req.PerPage,
			"order_id": req.OrderID, "amount": req.Amount, "items": len(req.Items),
		},
	})
}

func RegisterMixedBindingRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		api.POST("/mixed/binging/:user_id/orders", CreateOrderMixedBinding)
	}
}
