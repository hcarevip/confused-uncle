/*
 * File: 08-json-body.go
 * Date: 2025/9/9 23:44
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
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

/*
 1. JSON Body 流只能读一次。如果你需要对同一个请求体做多次绑定（例如既要绑定到 A，又要绑定到 B），请用 ShouldBindBodyWith（Gin 会把 Body 缓存到 context），
    否则第二次会拿到空数据
 2. Bind* = 自动 400 + Abort，快但“粗暴” 没机会自定义错误返回结构
 3. ShouldBind* = 只返回 error，更适合生产，只返回 error，是否返回 400/422，返回体格式，由你自己决定
 4. Bind / ShouldBind → 看请求的 Content-Type，可能是 JSON、form-data、query、XML 等
 5. BindJSON / ShouldBindJSON → 只解析 JSON，不管你传什么 Content-Type
*/
var validate = validator.New()

type CreateUserReq struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
	Email    string `json:"email"    validate:"required,email"`
	Age      int    `json:"age"      validate:"required,gte=18,lte=60"`
}

type ErrorResp struct {
	Code    string            `json:"code"`              // 业务错误码（如BadRequest/ ValidationError）
	Message string            `json:"message"`           // 总结性消息
	Details map[string]string `json:"details,omitempty"` // 字段级错误（key=字段名，value=错误原因）
}

func BadRequest(msg string, details map[string]string) (int, ErrorResp) {
	return http.StatusBadRequest, ErrorResp{Code: "BadRequest", Message: msg, Details: details}
}

func Unprocessable(details map[string]string) (int, ErrorResp) {
	return http.StatusUnprocessableEntity, ErrorResp{Code: "ValidationError", Message: "参数校验未通过", Details: details}
}
func BadRequestInner(msg string, details map[string]string) (int, map[string]any) {
	return http.StatusBadRequest, map[string]any{
		"code": "BadRequest", "message": msg, "details": details,
	}
}

func UnprocessableInner(details map[string]string) (int, map[string]any) {
	return http.StatusUnprocessableEntity, map[string]any{
		"code": "ValidationError", "message": "参数校验未通过", "details": details,
	}
}

// BindJSONStrict 读取 JSON Body -> 严格解码 -> 校验 -> 返回 DTO
// T 必须是指针类型传入（如 *dto.CreateUserReq）

func BindJSONStrict[T any](ctx *gin.Context, dst *T) (ok bool) {
	// Content-Type 检查（兼容 charset=utf-8 等参数）
	ct := ctx.GetHeader("Content-Type")
	// application/json; charset=utf-8
	// media type=application/json  params=map[charset:utf-8]
	mediaType, params, _ := mime.ParseMediaType(ct)
	fmt.Printf("主类型: %v  - 参数: %+v \n", mediaType, params)
	// 忽略大小写字符串比较
	if !strings.EqualFold(mediaType, "application/json") {
		status, body := BadRequest("Content-Type 必须为 application/json", nil)
		ctx.JSON(status, body)
		return false
	}
	// 读取原始 Body（只读一次）
	raw, err := ctx.GetRawData()
	if err != nil {
		status, body := BadRequest("读取请求体失败", nil)
		ctx.JSON(status, body)
		return false
	}
	// 严格 JSON 解码（禁止未知字段）
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		status, body := translateJSONDecodeError(err, raw)
		ctx.JSON(status, body)
		return false
	}
	// 防止多余垃圾数据：确保读到 EOF
	if dec.More() {
		status, body := BadRequest("JSON 解析失败：多余数据", nil)
		ctx.JSON(status, body)
		return false
	}
	// 校验规则（validator）
	if err := validate.Struct(dst); err != nil {
		status, body := translateValidationError(err)
		ctx.JSON(status, body)
		return false
	}
	return true
}

// translateJSONDecodeError: 把标准库 JSON 错误翻译成友好信息
func translateJSONDecodeError(err error, raw []byte) (int, map[string]any) {
	var se *json.SyntaxError
	var ute *json.UnmarshalTypeError

	switch {
	case errors.As(err, &se):
		// 语法错误：定位 offset
		snippet := locateSnippet(raw, se.Offset)
		return BadRequestInner(fmt.Sprintf("JSON 语法错误（位置 %d）", se.Offset), map[string]string{
			"_": snippet,
		})
	case errors.As(err, &ute):
		// 类型错误：字段/期待类型/实际类型
		field := ute.Field
		if field == "" && ute.Struct != "" {
			field = ute.Struct // 某些版本可能只给 Struct 路径
		}
		return BadRequestInner("字段类型不匹配", map[string]string{
			field: fmt.Sprintf("期望 %s，实际是 %s", ute.Type.String(), friendlyToken(ute.Value)),
		})
	case errors.Is(err, io.ErrUnexpectedEOF):
		return BadRequestInner("JSON 非完整：可能缺少右括号或引号", nil)
	case errors.Is(err, io.EOF):
		return BadRequestInner("请求体为空", nil)
	default:
		// 含有未知字段时，标准库报 "json: unknown field \"xxx\""
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			unk := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), "\"")
			return BadRequestInner("存在未声明的字段", map[string]string{
				unk: "该字段不被允许，请从请求体移除",
			})
		}
		return BadRequestInner("JSON 解析失败", map[string]string{
			"_": err.Error(),
		})
	}
}

func translateValidationError(err error) (int, map[string]any) {
	var inv *validator.InvalidValidationError
	if errors.As(err, &inv) {
		return BadRequestInner("参数校验配置错误", map[string]string{
			"_": inv.Error(),
		})
	}

	// 2) 正常的字段级校验错误（可能被包装，要用 errors.As）
	var verse validator.ValidationErrors
	if !errors.As(err, &verse) {
		return BadRequestInner("参数校验失败", map[string]string{
			"_": err.Error(),
		})
	}
	details := map[string]string{}
	for _, fe := range verse {
		field := fe.Field() // 默认是结构体字段名；可注册 TagNameFunc 把它改成 json 名
		switch fe.Tag() {
		case "required":
			details[field] = "必填"
		case "min":
			details[field] = "长度/数值小于最小值"
		case "max":
			details[field] = "长度/数值超过最大值"
		case "gte":
			details[field] = "必须大于等于最小值"
		case "lte":
			details[field] = "必须小于等于最大值"
		case "email":
			details[field] = "邮箱格式不正确"
		default:
			details[field] = "不满足约束：" + fe.Tag()
		}
	}
	return UnprocessableInner(details)
}

// locateSnippet: 给出错误附近 40 字符片段，便于排查
func locateSnippet(raw []byte, off int64) string {
	if off < 0 {
		off = 0
	}
	if off > int64(len(raw)) {
		off = int64(len(raw))
	}
	start := off - 40
	if start < 0 {
		start = 0
	}
	end := off + 40
	if end > int64(len(raw)) {
		end = int64(len(raw))
	}
	return string(raw[start:end])
}

func friendlyToken(v string) string {
	switch v {
	case "string":
		return "字符串"
	case "number":
		return "数字"
	case "object":
		return "对象"
	case "array":
		return "数组"
	case "boolean":
		return "布尔"
	case "null":
		return "null"
	default:
		return v
	}
}

func CreateAccount(ctx *gin.Context) {
	var req CreateUserReq
	if !BindJSONStrict(ctx, &req) {
		return
	}
	ctx.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": req,
	})
}

func RegisterJsonDataRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		api.POST("/create", CreateAccount)
	}
}
