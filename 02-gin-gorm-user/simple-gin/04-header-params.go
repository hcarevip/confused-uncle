/*
 * File: 04-header-params.go
 * Date: 2025/9/9 23:43
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

/*
	作用：把请求头按 header:"Name" 标签绑定到结构体字段，并按 binding 做校验
	多值：ShouldBindHeader 取首个值；若需全部值，用 c.Request.Header.Values("X-...")
*/

// =================== 通用错误响应 ===================

type ErrRespForHeader struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func unauthorized(ctx *gin.Context, msg string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrRespForHeader{
		Error:   "unauthorized",
		Details: msg,
	})
}

// =================== Header 绑定 DTO ===================
// 用 ShouldBindHeader 把常见头部安全取出并校验

type CommonHeader struct {
	// 业务相关：请求链路ID（允许为空；若中间件未生成，这里可能为空）
	RequestID string `header:"X-Request-Id" binding:"omitempty,min=8,max=128"`

	// 内容协商/客户端信息
	UserAgent      string `header:"User-Agent"       binding:"required,min=3,max=512"`
	AcceptLanguage string `header:"Accept-Language"  binding:"omitempty,max=128"`
	ContentType    string `header:"Content-Type"     binding:"omitempty,oneof=application/json application/xml multipart/form-data application/x-www-form-urlencoded"`

	// 鉴权头（这里只做存在性/读取；格式校验由鉴权中间件完成）
	Authorization string `header:"Authorization" binding:"omitempty,min=7"` // "Bearer <token>" 至少 7 字符
}

// EchoHeaders 示例：回显已绑定的 Header
func EchoHeaders(ctx *gin.Context) {
	var h CommonHeader
	if err := ctx.ShouldBindHeader(&h); err != nil {
		badRequest(ctx, err)
		return
	}
	// 多值头读取：Accept-Language: en_US,...
	languages := ctx.Request.Header.Values("Accept-Language")
	fmt.Printf("AcceptLanguage: %+v\n", languages)

	ctx.JSON(http.StatusOK, gin.H{
		"headers":   h,
		"clientIP":  ctx.ClientIP(),
		"route":     ctx.FullPath(),
		"xDebugAll": ctx.Request.Header.Values("X-Debug"),
	})
}

// =================== Request-ID 中间件 ===================
// 规则：若来路已有 X-Request-Id 则复用，否则生成一个新的，注入到响应与上下文

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rid := strings.TrimSpace(ctx.Request.Header.Get("X-Request-Id"))
		if rid == "" {
			rid = newRequestID()
		}
		ctx.Header("X-Request-Id", rid) // 写响应头
		ctx.Set("requestID", rid)       // 把值放进Gin Context的键值对里，后续处理器都能拿到ctx.GetString("requestID")
		ctx.Next()                      // 继续执行剩余的处理器（中间件链）
	}
}

// =================== JWT 鉴权中间件（Bearer） ===================
// 依赖：github.com/golang-jwt/jwt/v5

type JWTClaims struct {
	Role string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

func extractBearer(h string) (string, error) {
	h = strings.TrimSpace(h)
	if h == "" {
		return "", errors.New("missing Authorization header")
	}
	// 宽松大小写处理
	if len(h) < 7 || !strings.EqualFold(h[:6], "Bearer") {
		return "", errors.New("authorization must be Bearer <token>")
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("empty bearer token")
	}
	return strings.TrimSpace(parts[1]), nil
}
func AuthnJWTHS256(secret string) gin.HandlerFunc {
	secretBytes := []byte(secret)
	return func(ctx *gin.Context) {
		rawAuth := ctx.GetHeader("Authorization")
		tokenStr, err := extractBearer(rawAuth)
		if err != nil {
			unauthorized(ctx, err.Error())
			return
		}
		token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
			// 只接受 HS256
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return secretBytes, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			unauthorized(ctx, "invalid token")
			return
		}
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			unauthorized(ctx, "invalid claims")
			return
		}
		// 过期校验（ParseWithClaims已经校验 exp，但这里演示如何读取）
		if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
			unauthorized(ctx, "token expired")
			return
		}
		// 注入上下文，便于后续 handler 使用
		ctx.Set("sub", claims.Subject)
		ctx.Set("role", claims.Role)
		ctx.Next()
	}
}

// 基于角色的简单授权（配合上面的 AuthnJWTHS256 使用）

func RequireRole(allow ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allow))
	for _, r := range allow {
		allowed[r] = struct{}{}
	}
	return func(ctx *gin.Context) {
		role, _ := ctx.Get("role")
		if roleStr, ok := role.(string); ok {
			if _, ok := allowed[roleStr]; ok {
				ctx.Next()
				return
			}
		}
		ctx.AbortWithStatusJSON(http.StatusForbidden, ErrRespForHeader{
			Error:   "role not allowed",
			Details: fmt.Sprintf("role not allowed: %v", role),
		})
	}
}

func Me(ctx *gin.Context) {
	var h CommonHeader
	_ = ctx.ShouldBindHeader(&h)
	sub, _ := ctx.Get("sub")
	role, _ := ctx.Get("role")
	reqID, _ := ctx.Get("requestID")
	ctx.JSON(http.StatusOK, gin.H{
		"subject":   sub,
		"role":      role,
		"requestID": reqID,
		"userAgent": h.UserAgent,
	})
}
func newRequestID() string {
	// 生成 16 字节随机ID（hex编码，32位） crypto/rand.Read
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func MakeToken() (string, error) {
	secret := []byte("3ca7c401e8812acre67ade85fe6316eb")
	now := time.Now()
	claims := JWTClaims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "carl",
			Issuer:    "skyline-api",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}
func RegisterQueryRoutesHeaderParams(r gin.IRouter) {
	r.Use(RequestID())
	api := r.Group("/api/v1/headers")
	{
		api.GET("/echo", EchoHeaders)
	}
	secure := r.Group("/api/v1/secure")
	{
		// 认证
		secure.Use(AuthnJWTHS256("3ca7c401e8812acre67ade85fe6316eb"))
		secure.GET("/me", Me)
		// 认证 + 授权
		secure.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})
	}
}
