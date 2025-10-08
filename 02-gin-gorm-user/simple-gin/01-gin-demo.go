/*
 * File: 01-gin-demo.go
 * Date: 2025/9/9 20:35
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
	Gin是Go的 HTTP Web框架: 核心提供 路由、上下文（Context）、中间件、参数绑定与校验、渲染 等能力
	真正监听端口接收连接的底层仍然是 Go 标准库 net/http 的 http.Server
	Go的 net/http 每个请求一个 goroutine，调度器原生并发，性能与内存占用非常友好

	Context核心：是每个请求唯一的上下文对象
		包含请求 (*http.Request)、响应写入器 (ResponseWriter)、路由参数、处理器链、临时存储 Keys、错误收集 Errors、
			内容协商 Accepted 等
		接到请求 → 从 sync.Pool 取一个 Context → 绑定到这次请求，处理完毕 → 归还到池中复用
			因此：不要在请求结束后把 *gin.Context 存起来（比如丢进全局变量或异步队列）
	不要跨请求复用 Context：Gin 用 sync.Pool 复用 Context；请求结束后对象会回收
	Body 只能读一次：若既要校验又要记录原始体ShouldBindBodyWith（会缓存）或手动把 c.Request.Body 读到 []byte 后再
		io.NopCloser(bytes.NewReader(buf)) 还回去
	写响应的幂等性: 一旦写了头/体（c.Writer.Written() 为真），再改状态码/重复写会出错
	超时与取消: Gin 的 Context 不是 标准库 context.Context；取消/超时要用 c.Request.Context()
*/

func FirstGinDemo() {
	server := gin.Default() // 创建框架引擎
	server.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})
	err := server.Run(":8081")
	if err != nil {
		panic(err)
	}
}
