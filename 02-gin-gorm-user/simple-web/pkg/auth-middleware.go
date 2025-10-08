/*
 * File: auth-middleware.go
 * Date: 2025/10/7 14:24
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package pkg

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TokenSource 从请求中抽取 Token 的策略

// Claims 一些常用字段的强类型读取（可选，调用方按需使用）
type Claims struct {
	Subject   string
	Issuer    string
	Audience  any
	ExpiresAt *time.Time
	IssuedAt  *time.Time
	NotBefore *time.Time
	Raw       map[string]any
}

type TokenSource int

const (
	FromAuthHeader TokenSource = iota // Authorization: Bearer <token>
	FromQuery                         // ?access_token=<token>
)

// 用户在 context 中的 key

const CtxKeyClaims = "auth.claims"
const CtxKeySubject = "auth.sub"

// Options 为中间件配置

type Options struct {
	// 支持 HS256 或 RS256
	SigningMethod string // HS256 | RS256

	// HS256：对称秘钥；RS256：公钥验证（只读）
	HMACSecret   []byte
	RSAPublicKey *rsa.PublicKey

	// 允许从哪些位置拿 token，按顺序尝试
	TokenSources []TokenSource

	// Authorization 头前缀（默认 "Bearer "）
	AuthHeaderPrefix string

	// Query 参数名（当启用 FromQuery 时）
	QueryName string

	// 允许的时钟偏移（默认 60s）
	ClockSkew time.Duration

	// 必须包含的 Claim（键存在即可）
	RequiredClaims []string

	// 可选：对 claim 做更复杂校验；返回非 nil 则拒绝
	ValidateClaims func(map[string]any) error

	// 如果为 true：失败时不拦截，继续向后（资源层可判断是否已鉴别）
	Optional bool

	// 统一未授权响应
	OnUnauthorized func(c *gin.Context, code int, msg string)
}

func (o *Options) withDefaults() *Options {
	cp := *o
	if cp.TokenSources == nil {
		cp.TokenSources = []TokenSource{FromAuthHeader, FromQuery}
	}
	if cp.AuthHeaderPrefix == "" {
		cp.AuthHeaderPrefix = "Bearer "
	}
	if cp.QueryName == "" {
		cp.QueryName = "access_token"
	}
	if cp.ClockSkew == 0 {
		cp.ClockSkew = 60 * time.Second
	}
	if cp.OnUnauthorized == nil {
		cp.OnUnauthorized = func(c *gin.Context, code int, msg string) {
			// c.Abort() / c.AbortWithStatusJSON(...)：标记中止，后续中间件/handler 不会再执行
			c.AbortWithStatusJSON(code, gin.H{
				"code":  "UNAUTHORIZED",
				"error": msg,
			})
		}
	}
	return &cp
}

// AuthMiddleware 返回 Gin 中间件
func AuthMiddleware(opt Options) gin.HandlerFunc {
	o := opt.withDefaults()
	// 准备 jwt 解析器（允许时钟偏移）
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{o.SigningMethod}),
		jwt.WithLeeway(o.ClockSkew),
	)
	keyFunc := func(t *jwt.Token) (any, error) {
		switch o.SigningMethod {
		case "HS256":
			return o.HMACSecret, nil
		case "RS256":
			return o.RSAPublicKey, nil
		default:
			return nil, errors.New("unsupported signing method")
		}
	}
	return func(c *gin.Context) {
		raw := extractToken(c, o)
		if raw == "" {
			if o.Optional {
				c.Next()
				return
			}
			o.OnUnauthorized(c, http.StatusUnauthorized, "missing token")
			return
		}
		claims := jwt.MapClaims{}
		if _, err := parser.ParseWithClaims(raw, claims, keyFunc); err != nil {
			if o.Optional {
				c.Next()
				return
			}
			o.OnUnauthorized(c, http.StatusUnauthorized, "invalid token: "+err.Error())
			return
		}
		// 检查必要的标准时间声明（nbf/exp iat 由 jwt 库+leeway处理）
		for _, k := range o.RequiredClaims {
			if _, ok := claims[k]; !ok {
				if o.Optional {
					c.Next()
					return
				}
				o.OnUnauthorized(c, http.StatusForbidden, "missing claim: "+k)
				return
			}
		}
		// 业务自定义校验
		if o.ValidateClaims != nil {
			if err := o.ValidateClaims(claims); err != nil {
				if o.Optional {
					c.Next() // 先放行（让后面的中间件/处理器全跑完）执行完后会回到当前位置
					return
				}
				o.OnUnauthorized(c, http.StatusForbidden, "claim validation failed: "+err.Error())
				return
			}
		}
		// 放入 Context，供后续 handler 使用
		c.Set(CtxKeyClaims, map[string]any(claims))
		if sub, _ := claims["sub"].(string); sub != "" {
			c.Set(CtxKeySubject, sub)
		}
		c.Next()
	}
}

func extractToken(c *gin.Context, o *Options) string {
	for _, src := range o.TokenSources {
		switch src {
		case FromAuthHeader:
			h := c.GetHeader("Authorization")
			if h != "" && strings.HasPrefix(h, o.AuthHeaderPrefix) {
				return strings.TrimSpace(h[len(o.AuthHeaderPrefix):])
			}
		case FromQuery:
			if v := c.Query(o.QueryName); v != "" {
				return v
			}
		}
	}
	return ""
}
