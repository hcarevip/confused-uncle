/*
 * File: 05-cookie-params.go
 * Date: 2025/9/9 23:43
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
)

/*
Cookie 基础 & 在 Gin 中的读写要点
浏览器绝不会把 A 域的 Cookie 发送到 B 域
SameSite只决定B 域的 Cookie 在跨站时能不能随请求回B 域
	Strict → 不发
	Lax → 不发（子请求不带）
	None; Secure → 会发（需 HTTPS）

【作用】
- 在浏览器与服务器之间携带少量状态（会话 ID等）。服务端通过响应头Set-Cookie写入，浏览器之后在同站/符合条件的请求中通过Cookie头回发

【Gin 读取】
- Gin **没有** ShouldBindCookie（和 Header 不同），常用：
    1) c.Cookie("name") -> (string, error)         // 取单个；不存在返回 ErrNoCookie
    2) c.Request.Cookies() -> []*http.Cookie       // 取全部
- 不建议直接解析 c.Request.Header.Get("Cookie") 的原始串（自己拆分容易踩格式边界）

【Gin 写入】
- 两种写法：
    1) http.SetCookie(c.Writer, &http.Cookie{...}) // 原生写法，**可直接设置 SameSite**
    2) c.SetSameSite(mode); c.SetCookie(...)       // Gin 包装；SameSite 需先调用 SetSameSite
- 删除 Cookie：同名 + 同 Path/Domain，MaxAge=-1（并通常加一个过去的 Expires）

【关键属性】
- Name/Value：值不要含分号/控制字符；复杂内容先做 URL/Base64 编码
- Path：不指定时按 RFC 计算默认路径（当前 URL 的“目录”）；**建议显式设置 "/"** 以便全站可用
- Domain：不设=host-only；设成 ".example.com" 可覆盖子域（注意安全边界）
- MaxAge / Expires：
    - MaxAge=0 或不设 → 会话 Cookie（关浏览器后清除）
    - MaxAge>0 → 持久化（配合 Expires）
    - MaxAge<0 → 删除
- Secure：仅在 HTTPS 发送（生产环境强烈建议开启）
- HttpOnly：JS 无法读取（防止 XSS 窃取会话）
- SameSite：
    - Lax（推荐默认）：跨站子请求不带；顶级导航 GET 会带
    - Strict：最严格，仅同站点才带（第三方登录回跳可能受影响）
    - None：任何场景都带；**必须 Secure=true 且在 HTTPS 下**，常用于跨子域/iframe
    - DefaultMode：未显式设置的等价；现代浏览器多按 Lax 处理，但以浏览器实现为准

【多值与大小限制】
- 一次响应可发送多个 Set-Cookie（**一条只设置一个**）
- 单个 Cookie ~4KB 上限（含属性）；总数也有限制，别塞大对象

【安全建议】
- 服务端会话：Cookie 仅存 session id（HttpOnly+Secure+SameSite=Lax），真正状态存后端（Redis 等）
- 防篡改：需要把少量字段放 Cookie 可用签名（如 gorilla/secure cookie）；签名≠加密，敏感数据不要明文放 Cookie
- CSRF：同站策略 + 非幂等请求校验 CSRF Token（Cookie + Header 双通道比对）

【常见坑】
- SameSite=None 未配 Secure 或非 HTTPS → 浏览器会直接丢弃，看起来“写了但读不回”
- Path/Domain 不一致 → 删除/覆盖失败（必须与原属性一致）。
- 本地开发 http://localhost 测 SameSite=None; Secure → 不会回带；请用 Lax 或本地 HTTPS
*/

func newID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// ======= 读 Cookie =======
// 返回值：string（若不存在返回空串）与布尔（是否存在）
func getCookie(ctx *gin.Context, name string) (string, bool) {
	v, err := ctx.Cookie(name)
	if err != nil {
		return "", false
	}
	return v, true
}

// ======= 写 Cookie（使用 http.SetCookie 显式控制 SameSite） =======
// 注意：SameSite=None 必须搭配 Secure=true，允许任何场景，但前提是 HTTPS
func setCookie(c *gin.Context, name, value string, maxAgeSec int,
	path, domain string, secure, httpOnly bool, sameSite http.SameSite) {
	ck := &http.Cookie{
		Name:   name,   // Cookie 的键 （不要空格、逗号、分号等分隔符）
		Value:  value,  // Cookie 的值 如含特殊字符，先 URL/Base64 编码，避免被浏览器拆错
		Path:   path,   // 作用路径。浏览器只有在访问该path（及其子路径）时才会携带这个 Cookie。常用 "/"
		Domain: domain, // 作用域域名 空=仅此主机（host-only） 设为 "example.com" 可覆盖其子域（a.example.com）
		// MaxAge生存期（秒） >0=持久；=0=会话 Cookie（浏览器会话结束清除）；<0=让浏览器删除这个 Cookie
		SameSite: sameSite, // 跨站携带策略：Lax / Strict / None（None 必须配合 Secure 且需 HTTPS）
		Secure:   secure,   // 仅在 HTTPS 连接上发送。生产环境强烈建议 true（
		HttpOnly: httpOnly, // JS 无法读取 document.cookie 中的这个键，防止 XSS 窃取会话
		// XSS 攻击者让受害站点的页面执行恶意 JavaScript，脚本在受害站点的同源上下文运行，可直接读页面数据、调用站点 API、冒充用户
		// Expires:  calcExpires(maxAgeSec), 现代浏览器以 Max-Age 为准，没写 Max-Age 时看 Expires
		// 浏览器中Expires如何显示为Session，浏览器把它当作浏览器会话结束就清除的 Cookie
	}
	switch {
	case maxAgeSec > 0:
		// 持久 Cookie：同时设置 MaxAge 和 Expires（向后兼容）
		ck.MaxAge = maxAgeSec
		ck.Expires = time.Now().Add(time.Duration(maxAgeSec) * time.Second)
	case maxAgeSec == 0:
		// 会话 Cookie：只设置 MaxAge=0，**不要设置 Expires**
		// 在 Go 的实现中，MaxAge=0 会省略 Max-Age 字段，从而成为“会话 Cookie”
		ck.MaxAge = 0
		// ck.Expires = time.Time{} // 零值，序列化时不写 Expires
	default:
		// 删除 Cookie：MaxAge<0，且 Expires 设到过去
		ck.MaxAge = -1
		ck.Expires = time.Unix(1, 0)
	}
	http.SetCookie(c.Writer, ck)
}

// ======= 签名 Cookie（防篡改）=======
var hashKey = []byte("W5lLWFwaSIsInN1YiI6ImNhcmwiLCJle")
var blockKey []byte = nil
var sCookie = securecookie.New(hashKey, blockKey)

// SetSignedCookie 把 map 数据签名序列化到 Cookie（防篡改）
func SetSignedCookie(ctx *gin.Context, name string, data map[string]string, maxAgeSec int, sameSite http.SameSite) error {
	encoded, err := sCookie.Encode(name, data)
	if err != nil {
		return err
	}
	setCookie(ctx, name, encoded, maxAgeSec, "/", "", true, true, sameSite)
	return nil
}
func GetSignedCookie(ctx *gin.Context, name string) (map[string]string, bool) {
	raw, ok := getCookie(ctx, name)
	if !ok {
		return nil, false
	}
	var out map[string]string
	if err := sCookie.Decode(name, raw, &out); err != nil {
		return nil, false
	}
	return out, true
}

func RegisterCookieRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/cookies")
	{
		api.GET("/set/lax", setLax)       // SameSite=Lax（最常用，会话/登录）
		api.GET("/set/strict", setStrict) // SameSite=Strict（最安全，几乎不跨站）
		api.GET("/set/none", setNone)     // SameSite=None; Secure（跨站使用，如子域/iframe）
		api.GET("/read", readAll)         // 读取 demo
		// api.GET("/clear", clearAll)         // 清理 demo
		api.GET("/signed/set", setSigned)   // 签名 Cookie 写
		api.GET("/signed/read", readSigned) // 签名 Cookie 读
	}
}

// --- Handlers ---
func setLax(ctx *gin.Context) {
	id := newID(12)
	setCookie(ctx, "sid", id, 10, "/", "", false, true, http.SameSiteLaxMode)
	ctx.JSON(http.StatusOK, gin.H{"ok": true, "sid": id, "sameSite": "Lax"})
}
func setStrict(ctx *gin.Context) {
	// 更严格：仅同站点请求才会带上（防 CSRF），但可能影响第三方登录跳转回流
	id := newID(12)
	setCookie(ctx, "sid_strict", id, 3600*24, "/", "", true, true, http.SameSiteStrictMode)
	ctx.JSON(200, gin.H{"ok": true, "sid_strict": id, "sameSite": "Strict"})
}
func setNone(ctx *gin.Context) {
	// 跨站场景（如 SPA 跨子域、嵌入 iframe）必须 SameSite=None；且必须 Secure=true，并且仅在 HTTPS 下有效
	if !strings.HasPrefix(ctx.Request.URL.Scheme, "https") && ctx.Request.TLS == nil {
		// 在本地 http 环境说明一下
		ctx.JSON(200, gin.H{
			"warn": "SameSite=None; Secure cookie 需要 HTTPS 才会被浏览器接受。请用 https://localhost 或在生产环境测试。",
		})
		return
	}
	id := newID(12)
	setCookie(ctx, "sid_none", id, 3600, "/", "", true, true, http.SameSiteNoneMode)
	ctx.JSON(200, gin.H{"ok": true, "sid_none": id, "sameSite": "None; Secure"})
}
func setSigned(ctx *gin.Context) {
	payload := map[string]string{
		"user": "alice",
		"rid":  newID(8),
	}
	if err := SetSignedCookie(ctx, "ss", payload, 3600, http.SameSiteLaxMode); err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}
func readAll(ctx *gin.Context) {
	sid, _ := getCookie(ctx, "sid")
	strict, _ := getCookie(ctx, "sid_strict")
	none, _ := getCookie(ctx, "sid_none")
	ctx.JSON(200, gin.H{
		"sid":        sid,
		"sid_strict": strict,
		"sid_none":   none,
	})
}

// CSRFProtect ：CSRF Token 校验（演示版：从 Cookie + Header 双通道比对）
// 跨站请求伪造
func CSRFProtect(cookieName, headerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		cv, ok := getCookie(c, cookieName)
		hv := c.GetHeader(headerName)
		if !ok || cv == "" || hv == "" || !secureCompare(cv, hv) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf_invalid"})
			return
		}
		c.Next()
	}
}
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func readSigned(ctx *gin.Context) {
	data, ok := GetSignedCookie(ctx, "ss")
	if !ok {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "no/invalid signed cookie"})
		return
	}
	ctx.JSON(200, gin.H{"data": data})
}
