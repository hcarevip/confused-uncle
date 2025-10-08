/*
 * File: 13-custom-validator.go
 * Date: 2025/9/9 23:46
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"errors"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

/*
	自定义 validator 注册与使用
	细节解释
		Gin 自动校验（ShouldBindJSON 等）
			Gin内部的 binding.Validator.Engine() 初始化时会执行
		自己创建的 validator 实例： v := validator.New()  v.Struct(req)
			这时候v的默认TagName是 validate（库默认值）
*/
// 定义全局 validator 变量
var (
	once        sync.Once
	ginValidate *validator.Validate

	// 统一正则（RE2 兼容，无先行断言）
	reAlphaNum = regexp.MustCompile(`^[A-Za-z0-9]+$`) // 仅字母数字
	reDigit    = regexp.MustCompile(`\d`)             // 至少 1 个数字
	reLetter   = regexp.MustCompile(`[A-Za-z]`)       // 至少 1 个字母
	reMobileCN = regexp.MustCompile(`^1[3-9]\d{9}$`)  // 中国大陆 11 位
)

func mobileValidator(fl validator.FieldLevel) bool {
	mobile := fl.Field().String()
	return reMobileCN.MatchString(mobile)
}

// password: ≥8 且仅字母数字，且至少 1 数字 + 1 字母
func passwordValidator(fl validator.FieldLevel) bool {
	pwd := fl.Field().String()

	// 长度（若允许非 ASCII，请用 utf8.RuneCountInString）
	if utf8.RuneCountInString(pwd) < 8 {
		return false
	}
	// 字符集限制
	if !reAlphaNum.MatchString(pwd) {
		return false
	}
	// 至少一个数字
	if !reDigit.MatchString(pwd) {
		return false
	}
	// 至少一个字母
	if !reLetter.MatchString(pwd) {
		return false
	}
	return true
}

// 让错误里的字段名显示为 json tag
func jsonTagName(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	if name == "-" || name == "" {
		return fld.Name
	}
	return name
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,password"` // 自定义 password
	Mobile   string `json:"mobile"   binding:"required,mobile"`   // 自定义 mobile
}

type APIError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Msg   string `json:"msg"`
}

// InitValidator 在项目初始化阶段（如 init() 或 main()）注册自定义 validator
func InitValidator() *validator.Validate {
	once.Do(func() {
		// 优先获取 Gin 的内置实例
		// 确保你把自定义规则注册到 Gin 实际在用的同一个 *validator.Validate 实例
		if eng, ok := binding.Validator.Engine().(*validator.Validate); ok && eng != nil {
			ginValidate = eng
		} else {
			// 极少数场景兜底
			ginValidate = validator.New()
		}

		// 告诉 Gin/validator 使用 `binding`作为tag 名（Gin 默认就是 binding，这里显式设置更直观）
		ginValidate.SetTagName("binding")

		// 自定义校验、字段名映射
		_ = ginValidate.RegisterValidation("mobile", mobileValidator)
		_ = ginValidate.RegisterValidation("password", passwordValidator)
		ginValidate.RegisterTagNameFunc(jsonTagName)
	})
	return ginValidate
}
func humanizeMsg(ve validator.FieldError) string {
	switch ve.Tag() {
	case "required":
		return "必填"
	case "min":
		return "长度过短"
	case "max":
		return "长度过长"
	case "mobile":
		return "手机号格式不正确"
	case "password":
		return "密码需≥8位，且包含字母与数字，仅限字母数字"
	default:
		return "格式不正确"
	}
}
func RegisterHandler(ctx *gin.Context) {
	var req RegisterRequest
	// 绑定 + 自动校验（触发 binding 标签 + 注册的自定义规则）
	// ShouldBind 会同时完成绑定 + 校验
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 精准处理校验错误
		var verse validator.ValidationErrors
		if errors.As(err, &verse) {
			out := make([]APIError, 0, len(verse))
			for _, ve := range verse {
				out = append(out, APIError{
					Field: ve.Field(), // 映射到json字段名
					Tag:   ve.Tag(),
					Msg:   humanizeMsg(ve),
				})
			}
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":  "VALIDATION_FAILED",
				"error": out,
			})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
	})
}

func RegisterCustomValidationRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		api.POST("/validation/register", RegisterHandler)
	}
}
