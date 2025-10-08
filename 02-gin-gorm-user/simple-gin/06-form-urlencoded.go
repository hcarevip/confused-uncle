/*
 * File: 06-form-urlencoded.go
 * Date: 2025/9/9 23:44
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

/*
表单默认的提交方式
	username=uu-coming&password=123456&age=18 Content-Type: application/x-www-form-urlencoded，主要用于传统表单
	x-www-form-urlencoded 的数据在 Body，而 ShouldBindQuery 只看 URL 部分
*/

var usernameAllowed = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*[A-Za-z0-9]$`)

// 连续特殊字符（不允许出现）
var usernameConsecutiveSpecial = regexp.MustCompile(`[._-]{2,}`)

// 常见黑名单
var usernameBlacklist = map[string]struct{}{
	"admin":  {},
	"root":   {},
	"system": {},
	"test":   {},
	"guest":  {},
}

// UsernameCustomValidators 自定义校验器
func UsernameCustomValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return errors.New("validator engine not found")
	}
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		fmt.Printf("field Tag: %v  field Name: %v \n", field.Tag, field.Name)
		if name := field.Tag.Get("json"); name != "" && name != "-" {
			return name
		}
		if name := field.Tag.Get("form"); name != "" && name != "-" {
			return name
		}
		return field.Name
	})
	// 无空白字符（包含空格、制表、换行等）
	_ = v.RegisterValidation("no_space", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		// 任意 Unicode 空白都不允许
		for _, r := range s {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' {
				return false
			}
		}
		return true
	})
	// 用户名格式：仅 A-Z a-z 0-9 . _ -；首尾必须是字母/数字；禁止连续特殊字符
	_ = v.RegisterValidation("username_fmt", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if utf8.RuneCountInString(s) < 1 {
			return false
		}
		if !usernameAllowed.MatchString(s) {
			return false
		}
		if usernameConsecutiveSpecial.MatchString(s) {
			return false
		}
		// 额外限制：不允许 “.-”、“-.”、“._”、“_.” 直接相邻
		if strings.Contains(s, "._") || strings.Contains(s, "_.") ||
			strings.Contains(s, ".-") || strings.Contains(s, "-.") {
			return false
		}
		return true
	})
	// 用户名黑名单（大小写不敏感）
	_ = v.RegisterValidation("username_blacklist", func(fl validator.FieldLevel) bool {
		s := strings.ToLower(fl.Field().String())
		_, banned := usernameBlacklist[s]
		return !banned
	})
	return nil
}

type LoginForm struct {
	// Username string `form:"username" binding:"required"`
	Username string `form:"username" binding:"required,no_space,username_fmt,username_blacklist"`
	Password string `form:"password" binding:"required"`
}

func LoginHandler(ctx *gin.Context) {
	var form LoginForm
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"username": form.Username})
}

func RegisterUrlencodedRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/users")
	{
		api.POST("/test/login", LoginHandler)
	}
}
