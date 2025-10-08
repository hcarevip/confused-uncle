/*
 * File: 09-protobuf-body.go
 * Date: 2025/9/9 23:45
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"compress/gzip"
	"fmt"
	order "hcarevip/uucoming/02-gin-gorm-user/uploads"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

/*
	Protobuf是 Google开发的一种序列化协议
		相比 JSON/XML 更小（体积减少 50%~80%）、更快（解析速度提升数倍）
		新增字段不会导致旧客户端崩溃，旧字段保留可被新客户端识别
		Protobuf 是二进制，体积小，速度快（适合高并发/移动端/微服务内部通信）可跨语言
	安装两个插件
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
		把 GOPATH\bin 加到当前PowerShell会话的PATH(临时)
		$env:Path += ";$(go env GOPATH)\bin"
	执行：
		protoc -I . uploads\order.proto `
		--go_out=. --go_opt=paths=source_relative `
		--go-grpc_out=. --go-grpc_opt=paths=source_relative
*/

var DefaultPBMaxBody int64 = 1 << 20 // n << k = n * 2^k  1 x 2的20次方

func IsProtoContentType(ct string) bool {
	if ct == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	mt = strings.ToLower(mt)
	// 常见 proto 类型 + *+proto 后缀
	if mt == "application/x-protobuf" ||
		mt == "application/protobuf" ||
		mt == "application/grpc" ||
		mt == "application/vnd.google.protobuf" ||
		strings.HasSuffix(mt, "+proto") {
		return true
	}
	return false
}

func readBodyWithLimitAndMaybeGzip(r *http.Request, max int64) ([]byte, error) {
	// 体积限制 + gzip 解压
	if max <= 0 {
		max = DefaultPBMaxBody
	}
	// 超限就立刻报错（request body too large）目的就是兜住过大的请求体
	var reader io.Reader = http.MaxBytesReader(nil, r.Body, max)
	// defer 语句的参数是直接传入的
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			return
		}
	}(r.Body)
	// Content-Encoding
	switch strings.ToLower(r.Header.Get("Content-Encoding")) {
	case "", "identity": // 不处理
	case "gzip":
		gzr, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("invalid gzip: %w", err)
		}
		defer func(gzr *gzip.Reader) {
			err := gzr.Close()
			if err != nil {
				return
			}
		}(gzr)
		reader = gzr
	default:
		return nil, fmt.Errorf("unsupported Content-Encoding")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func BindProtoBody[T proto.Message](r *http.Request, msg T, maxBody int64) (int, error) {
	// 将请求体解码到proto.Message
	if !IsProtoContentType(r.Header.Get("Content-Type")) {
		return http.StatusUnsupportedMediaType, fmt.Errorf("unsupported Content-Type, want application/x-protobuf (or *+proto)")
	}
	raw, err := readBodyWithLimitAndMaybeGzip(r, maxBody)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("read body error: %w", err)
	}
	// 选择是否丢弃未知字段：推荐开启（前向兼容）
	unmarshal := proto.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshal.Unmarshal(raw, msg); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid protobuf: %w", err)
	}
	// 基于 validator/自定义规则的业务校验
	return http.StatusOK, nil
}

func CreateOrderPB(ctx *gin.Context) {
	req := &order.CreateOrderRequest{}
	status, err := BindProtoBody(ctx.Request, req, 1<<20)
	if err != nil {
		ctx.JSON(status, gin.H{"code": status, "message": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "message": "items must not be empty"})
		return
	}
	if req.Amount <= 0 {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "message": "amount must be > 0"})
		return
	}
	// TODO: 业务处理
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"orderId":  req.OrderId,
			"amount":   req.Amount,
			"currency": req.Currency,
			"items":    len(req.Items),
		},
	})
}

func RegisterProtobufRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		api.POST("/protobuf", CreateOrderPB)
	}
}
