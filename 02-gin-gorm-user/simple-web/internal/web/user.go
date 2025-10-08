/*
 * File: user.go
 * Date: 2025/10/3 16:40
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package web

import (
	"errors"
	"regexp"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	emailRexExprPattern = `^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`
	// go 官方的正则不支持先行(?= ...)/后行断言
	// pwdRexExprPattern = `^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d]{8,}$`
	// 组合判定用的子模式（RE2 兼容）
	reHasLetterPattern  = `[A-Za-z]`
	reHasDigitPattern   = `[0-9]`
	reHasSpecialPattern = `[^A-Za-z0-9]`
	reOnlyAlbumPattern  = `^[A-Za-z0-9]+$`
	reNoSpacePattern    = `^\S+$`
)

// PasswordPolicy 可配置的密码策略
type PasswordPolicy struct {
	MinLen         int  // 默认 8
	MaxLen         int  // 0 表示不限制
	AllowSpecial   bool // 是否允许特殊字符
	RequireLetter  bool // 是否必须包含字母
	RequireDigit   bool // 是否必须包含数字
	RequireSpecial bool // 是否必须包含特殊字符（只有 AllowSpecial=true 时才有意义）
	NoSpace        bool // 是否禁止空白字符
}
type UserHandle struct {
	// 预编译提高性能 会解析并编译正则；如果语法非法会立刻 panic。校验的是正则表达式本身是否有效，不是去校验你的输入字符串
	// 避免每次匹配都重新解析&编译（regexp.MatchString(pattern, s) 每次都会编译一次，开销大
	emailRexExpr   *regexp.Regexp
	hasLetterExpr  *regexp.Regexp
	hasDigitExpr   *regexp.Regexp
	hasSpecialExpr *regexp.Regexp
	onlyAlbumExpr  *regexp.Regexp
	noSpaceExpr    *regexp.Regexp

	// policy
	pwdPolicy PasswordPolicy
}

func NewUserHandle(policy *PasswordPolicy) *UserHandle {
	p := PasswordPolicy{
		MinLen:         8,
		MaxLen:         16,
		AllowSpecial:   false,
		RequireLetter:  true,
		RequireDigit:   true,
		RequireSpecial: false,
		NoSpace:        true,
	}
	if policy != nil {
		if policy.MinLen > 0 {
			p.MinLen = policy.MinLen
		}
		if policy.MaxLen > 0 {
			p.MaxLen = policy.MaxLen
		}
		p.AllowSpecial = policy.AllowSpecial
		p.RequireLetter = policy.RequireLetter
		p.RequireDigit = policy.RequireDigit
		p.RequireSpecial = policy.RequireSpecial
		p.NoSpace = policy.NoSpace
	}
	return &UserHandle{
		emailRexExpr:   regexp.MustCompile(emailRexExprPattern),
		hasLetterExpr:  regexp.MustCompile(reHasLetterPattern),
		hasDigitExpr:   regexp.MustCompile(reHasDigitPattern),
		hasSpecialExpr: regexp.MustCompile(reHasSpecialPattern),
		onlyAlbumExpr:  regexp.MustCompile(reOnlyAlbumPattern),
		noSpaceExpr:    regexp.MustCompile(reNoSpacePattern),
		pwdPolicy:      p,
	}
}

// RegisterRoutes 与用户相关的路由定义在这个UserHandle上
func (h *UserHandle) RegisterRoutes(server *gin.Engine) {
	api := server.Group("/api/v1/")
	api.POST("user/signup", h.Signup)
	api.POST("user/login", h.Login)
	api.GET("user/profile", h.Profile)
	api.GET("user/edit", h.Edit)
}

// ValidateEmail 邮箱校验
func (h *UserHandle) ValidateEmail(s string) bool {
	return h.emailRexExpr.MatchString(s)
}

// ValidatePassword 密码校验：返回是否通过 + 具体错误（便于前端提示）
func (h *UserHandle) ValidatePassword(pwd string) (bool, []string) {
	var errs []string
	n := utf8.RuneCountInString(pwd)

	// 长度
	if n < h.pwdPolicy.MinLen {
		errs = append(errs, "密码长度不足")
	}
	if h.pwdPolicy.MaxLen > 0 && n > h.pwdPolicy.MaxLen {
		errs = append(errs, "密码长度超限")
	}

	// 空白
	if h.pwdPolicy.NoSpace && !h.noSpaceExpr.MatchString(pwd) {
		errs = append(errs, "密码包含空白字符")
	}

	// 字符集限制  MatchString 是用已经编译好的程序去匹配输入 s；返回是否匹配到
	if !h.pwdPolicy.AllowSpecial && !h.onlyAlbumExpr.MatchString(pwd) {
		errs = append(errs, "密码仅允许字母和数字")
	}

	// 组成要求
	if h.pwdPolicy.RequireLetter && !h.hasLetterExpr.MatchString(pwd) {
		errs = append(errs, "密码至少包含一个字母")
	}
	if h.pwdPolicy.RequireDigit && !h.hasDigitExpr.MatchString(pwd) {
		errs = append(errs, "密码至少包含一个数字")
	}
	if h.pwdPolicy.AllowSpecial && h.pwdPolicy.RequireSpecial && !h.hasSpecialExpr.MatchString(pwd) {
		errs = append(errs, "密码至少包含一个特殊字符")
	}

	return len(errs) == 0, errs
}
func (h *UserHandle) ValidatePasswordErr(pwd string) error {
	ok, errs := h.ValidatePassword(pwd)
	if ok {
		return nil
	}
	return errors.New(joinErrs(errs))
}

func joinErrs(es []string) string {
	switch len(es) {
	case 0:
		return ""
	case 1:
		return es[0]
	default:
		out := es[0]
		for i := 1; i < len(es); i++ {
			out += "；" + es[i]
		}
		return out
	}
}

func (h *UserHandle) Signup(ctx *gin.Context) {
	// 定义内部结构体接收请求（优先使用内部结构体） 与前端传过来的字段一致
	type SignupReq struct {
		Email           string `form:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	var req SignupReq
	// ctx.Bind 解析/校验失败：Gin 自动写 400 响应并 Abort，代码不会再往下走
	// ShouldBind 解析/校验失败：只返回 error，不写响应也不中止，是否返回 400/422 由你自己决定
	if err := ctx.ShouldBind(&req); err != nil {
		// 失败时自己处理
		var vErr validator.ValidationErrors
		// 判断捕获到的错误 err 是不是由字段校验（Validator）产生的，如果是就把具体的字段错误集合（ValidationErrors）取出来返回给前端
		if errors.As(err, &vErr) {
			ctx.JSON(422, gin.H{"code": "VALIDATION_FAILED", "errs": vErr})
		} else {
			ctx.JSON(400, gin.H{"code": "BIND_FAILED", "err": err.Error()})
		}
		return
	}
	// 校验字段
	if req.Password != req.ConfirmPassword {
		ctx.JSON(422, gin.H{"code": "VALIDATION_FAILED", "errs": "密码输入不一致"})
		return
	}
	if !h.ValidateEmail(req.Email) {
		ctx.JSON(422, gin.H{
			"code": "VALIDATION_FAILED",
			"errs": "邮箱格式不正确，请输入有效的邮箱地址",
		})
		return
	}
	if ok, errs := h.ValidatePassword(req.Password); !ok {
		ctx.JSON(422, gin.H{
			"code": "VALIDATION_FAILED",
			"errs": errs, // errs 是 []string，例如 ["密码长度不足", "密码至少包含一个数字"]
		})
		return
	}

	ctx.JSON(200, gin.H{"code": "SUCCESS", "data": req})
}

func (h *UserHandle) Login(ctx *gin.Context) {}

func (h *UserHandle) Profile(ctx *gin.Context) {}

func (h *UserHandle) Edit(ctx *gin.Context) {}
