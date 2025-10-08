/*
 * File: main.go
 * Date: 2025/6/25 21:22
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"bytes"
	"fmt"
	simplegin "hcarevip/uucoming/02-gin-gorm-user/simple-gin"
	simpleorm "hcarevip/uucoming/02-gin-gorm-user/simple-orm"
	order "hcarevip/uucoming/02-gin-gorm-user/uploads"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

func UriParams(r gin.IRouter) {
	simplegin.RegisterURIRoutesUriParams(r)
	// 注册自定义校验
	simplegin.RegisterCustomValidators()
}

func QueryParams(r gin.IRouter) {
	simplegin.RegisterQueryRoutesQueryParams(r)
}

func HeaderParams(r gin.IRouter) {
	simplegin.RegisterQueryRoutesHeaderParams(r)
}

func CookieParams(r gin.IRouter) {
	simplegin.RegisterCookieRoutes(r)
}
func FormUrlencoded(r gin.IRouter) {
	_ = simplegin.UsernameCustomValidators()
	simplegin.RegisterUrlencodedRoutes(r)
}

func FileUpload(r gin.IRouter) {
	simplegin.RegisterUploadRoutes(r)
}

func JsonParams(r gin.IRouter) {
	simplegin.RegisterJsonDataRoutes(r)
}

func testProtobuf() {
	// 构造请求体
	req := &order.CreateOrderRequest{
		OrderId:    "20258945512316542123132",
		Amount:     100,
		Currency:   "PHP",
		CustomerId: "uu-coming",
		Email:      "uu-coming@gmail.com",
		Items:      []*order.CreateOrderRequest_Item{{Sku: "G-01", Qty: 2, Price: 88.8}},
	}
	b, err := proto.Marshal(req)
	if err != nil {
		panic(fmt.Errorf("proto marshal failed: %w", err))
	}

	// 2) 构造 HTTP 请求
	httpReq, err := http.NewRequest("POST", "http://localhost:8081/api/v1/test/protobuf", bytes.NewReader(b))
	if err != nil {
		panic(fmt.Errorf("new request failed: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")

	// 设置超时，避免卡死
	client := &http.Client{Timeout: 5 * time.Second}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		panic(fmt.Errorf("http do failed: %w", err))
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("read body failed: %w", err))
	}

	fmt.Println("HTTP:", resp.Status)
	fmt.Println("Body:", string(body))
}

func ProtobufBody(r gin.IRouter) {
	simplegin.RegisterProtobufRoutes(r)
}

func testMsgpack() {
	// 客户端
	payload := simplegin.CreateOrderReq{
		OrderID:    "564113213213212399",
		Amount:     199.99,
		Currency:   "PHP",
		CustomerID: "U1001",
		Email:      "juan@example.com",
		Items:      []simplegin.Item{{SKU: "G-01", Qty: 2, Price: 88.8}},
	}
	b, _ := msgpack.Marshal(&payload)
	req, _ := http.NewRequest("POST", "http://localhost:8081/api/v1/test/msgpack", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/msgpack")
	// 设置超时，避免卡死
	client := &http.Client{Timeout: 5 * time.Second}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("http do failed: %w", err))
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("read body failed: %w", err))
	}

	fmt.Println("HTTP:", resp.Status)
	fmt.Println("Body:", string(body))
}

func MsgpackBody(r gin.IRouter) {
	simplegin.RegisterMsgpackRoutes(r)
}

func MixedBinding(r gin.IRouter) {
	simplegin.RegisterMixedBindingRoutes(r)
}

func ValidationRules(r gin.IRouter) {
	simplegin.RegisterValidationRoutes(r)
}

func CustomValidation(r gin.IRouter) {
	simplegin.InitValidator()
	simplegin.RegisterCustomValidationRoutes(r)
}

func PaginationExample(r gin.IRouter) {
	simplegin.RegisterPaginationRoutes(r)
}

func SimpleGinTest() {
	// simplegin.FirstGinDemo()
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 路径参数
	// UriParams(r)

	// 查询参数
	// QueryParams(r)

	// header 参数
	//token, _ := simplegin.MakeToken()
	//fmt.Println("token:", token)
	//HeaderParams(r)

	// cookie 参数
	// CookieParams(r)

	// urlencoded 表单
	// FormUrlencoded(r)

	// 上传
	// FileUpload(r)

	// json 请求
	// JsonParams(r)

	// protobuf 请求
	// ProtobufBody(r)

	// msgpack 请求
	//MsgpackBody(r)

	// mixed-binding
	// MixedBinding(r)

	// 校验规则
	// ValidationRules(r)

	// 自定义校验
	// CustomValidation(r)

	PaginationExample(r)
	if err := r.Run(":8081"); err != nil {
		panic(err)
	}
}

func SimpleOrmTest() {
	// 01：数据库连接、日志、命名策略、连接池
	// simpleorm.New()
	// simpleorm.TestCurd()
	// simpleorm.TestQueryAdvanced()
	// simpleorm.TestScopesPagination()
	simpleorm.TestTransactionsLock()
}
func main() {
	// SimpleGinTest()
	SimpleOrmTest()
}
