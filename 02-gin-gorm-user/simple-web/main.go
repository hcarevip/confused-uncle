/*
 * File: main.go
 * Date: 2025/10/3 16:53
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"hcarevip/uucoming/02-gin-gorm-user/simple-web/internal/web"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

/*
目前大多数web框架都提供CORS的解决方案, 在gin中提供了一个middleware实现来解决
go get -u "github.com/gin-contrib/cors"
*/
func main() {
	// userHdl := web.NewUserHandle(nil)
	userHdl := web.NewUserHandle(&web.PasswordPolicy{})
	server := gin.Default()

	// 解决跨域
	server.Use(cors.New(cors.Config{
		// 一键允许任意来源（等价 Access-Control-Allow-Origin: *）与 AllowCredentials=true 互斥
		// AllowAllOrigins: true,

		// 白名单来源列表（精确匹配域名+端口+协议）
		// AllowOrigins: []string{"https://*.example.com", "https://admin.example.com"},

		// 自定义函数判断 Origin 是否允许（无上下文），需要动态允许某些来源，但只依赖 origin 本身
		// AllowOriginFunc: func(origin string) bool {
		//	 return strings.HasSuffix(origin, ".tenant.example.com")
		// },

		// 判断是否允许来源时，需要查看请求路径/头部等额外信息（带请求上下文）
		AllowOriginWithContextFunc: func(c *gin.Context, origin string) bool {
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
				return true
			}
			return origin == "http://127.0.0.1"
		},

		// 允许的 HTTP 方法
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},

		// 前端发送带 Token 的请求、或自定义头时必须声明
		AllowHeaders: []string{"Authorization", "Content-Type", "X-CSRF-Token"},

		// 是否允许携带用户凭证, 开启后不能和 AllowAllOrigins=true 并用；响应中的 Allow-Origin 不能是 *，必须是具体域名
		AllowCredentials: true,

		// 允许 WebSocket 协议跨域，如前端需要直接连你的 WS 服务
		AllowWebSockets: true,

		// 允许前端 JS 通过 fetch/xhr 读取的响应头白名单（默认JS只能读少数安全头）
		ExposeHeaders: []string{"X-Total-Count", "X-Request-Id"},

		// 浏览器缓存预检请求（OPTIONS）的时间，减少频繁预检，提升性能。常配 10m～24h
		MaxAge: 12 * time.Hour,

		// 允许在 AllowOrigins 里使用通配（如 https://*.example.com），比 AllowOriginFunc 简单
		AllowWildcard: true,

		// 自定义预检（OPTIONS）响应的状态码
		OptionsResponseStatusCode: http.StatusNoContent,

		// 允许来自浏览器扩展的跨域请求，你的后端 API 会被浏览器扩展直接调用
		AllowBrowserExtensions: false,

		// 允许自定义协议（scheme）的来源参与 CORS，你的前端不是浏览器里的 http/https 页面，而是桌面/移动壳
		// 里跑的 WebView，它用自定义 scheme 加载页面，并直接请求你的后端
		// CustomSchemas 只是告诉 CORS “这些协议是可信来源类型”，仍需要在 AllowOrigins（或自定义判断）里明确放行具体来源
		// AllowOrigins:  []string{"tauri://localhost", "capacitor://localhost"},
		CustomSchemas: []string{"tauri://", "capacitor://"},
	}))
	userHdl.RegisterRoutes(server)

	err := server.Run(":8081")
	if err != nil {
		panic(err)
	}
}
