/*
 * File: 10-msgpack-body.go
 * Date: 2025/9/9 23:45
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/vmihailenco/msgpack/v5"
)

/*
	MsgPack 和 Protobuf 都是数据序列化格式
		作用就是把结构化数据（对象、结构体）转换成字节流（传输/存储），然后在另一端再解码还原
		都是二进制格式，体积比 JSON/XML 小
	Content-Type允许：application/msgpack、application/x-msgpack、以及 *+msgpack（如 application/vnd.myapp+msgpack）
	Content-Encoding允许空/identity（不压缩）可选支持 gzip
	适用场景：内部服务/移动端对接，追求更小体积与更快解码 | 需要数组/二进制/时间等高效表示
*/

var (
	DefaultMsgpackMaxCompressed   int64 = 1 << 20 // 1MiB，压缩前
	DefaultMsgpackMaxDecompressed int64 = 2 << 20 // 2MiB，解压后
	validateMsgpack                     = validator.New()
)

type Item struct {
	// msgpack:"sku"：告诉 MessagePack 库（如msgpack）用 sku 作为键名
	// validate面向 validator 包，手动在任意地方调用 validate.Struct(obj) 时生效
	SKU   string  `json:"sku" msgpack:"sku" validate:"required"`
	Qty   int     `json:"qty" msgpack:"qty" validate:"required,gt=0"`
	Price float64 `json:"price" msgpack:"price" validate:"required,gt=0"`
}
type CreateOrderReq struct {
	OrderID    string  `json:"order_id" msgpack:"order_id" validate:"required,len=18"`
	Amount     float64 `json:"amount"   msgpack:"amount"   validate:"required,gt=0"`
	Currency   string  `json:"currency" msgpack:"currency" validate:"required,oneof=CNY USD PHP"`
	CustomerID string  `json:"customer_id" msgpack:"customer_id" validate:"required"`
	Email      string  `json:"email" msgpack:"email" validate:"omitempty,email"`
	Items      []Item  `json:"items" msgpack:"items" validate:"min=1,dive"`
}

// BytesReader : 小工具，避免每次都 bytes.NewReader(raw)
type BytesReader struct {
	b   []byte // 源数据
	off int    // 已读取偏移（0..len(b)）
}

func NewBytesReader(b []byte) *BytesReader {
	return &BytesReader{b: b}
}

func (br *BytesReader) Read(p []byte) (int, error) {
	// 如果已经到末尾，直接 EOF
	if br.off >= len(br.b) {
		return 0, io.EOF
	}
	// 把接收者 br.b（一个 []byte）尽可能多地拷贝到调用方给的缓冲 p，返回拷贝的字节数 n
	n := copy(p, br.b[br.off:])
	br.off += n
	if br.off >= len(br.b) {
		// 本次读完了最后的字节——允许带 EOF 返回
		return n, io.EOF
	}
	return n, nil // 还有数据，err 必须为 nil
}

func IsMsgpackContentType(ct string) bool {
	if ct == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	mt = strings.ToLower(mt)
	return mt == "application/msgpack" ||
		mt == "application/x-msgpack" ||
		strings.HasSuffix(mt, "+msgpack")
}

// readBodyWithLimitAndMaybeGzipMsgpack：压缩前/后双限流 + gzip 解析
func readBodyWithLimitAndMaybeGzipMsgpack(w http.ResponseWriter, r *http.Request,
	maxCompressed, maxDecompressed int64) ([]byte, int, error) {
	if maxCompressed <= 0 {
		maxCompressed = DefaultMsgpackMaxCompressed
	}
	if maxDecompressed <= 0 {
		maxDecompressed = DefaultMsgpackMaxDecompressed
	}
	// 压缩前限流（必须传 w，自动 413 + close）
	r.Body = http.MaxBytesReader(w, r.Body, maxCompressed)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(r.Body)
	ce := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding")))
	var reader io.Reader = r.Body
	switch ce {
	case "", "identity":
	// 直接读
	case "gzip":
		gzr, err := gzip.NewReader(reader)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid gzip: %w", err)
		}
		defer func(gzr *gzip.Reader) {
			err := gzr.Close()
			if err != nil {
				return
			}
		}(gzr)
		reader = io.LimitReader(gzr, maxDecompressed) // 解压后再次限流
	default:
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("unsupported Content-Encoding %q", ce)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	return data, http.StatusOK, nil
}

// BindMsgpackBody 读取并解码为 v（指针），并进行可选校验；返回 (status, error)
func BindMsgpackBody[T any](w http.ResponseWriter, r *http.Request, v *T,
	maxCompressed, maxDecompressed int64) (int, error) {
	if !IsMsgpackContentType(r.Header.Get("Content-Type")) {
		return http.StatusUnsupportedMediaType, fmt.Errorf("unsupported Content-Type, need application/msgpack (or *+msgpack)")
	}
	// 读取（含 gzip / 双限流）
	raw, status, err := readBodyWithLimitAndMaybeGzipMsgpack(w, r, maxCompressed, maxDecompressed)
	if err != nil {
		return status, fmt.Errorf("read body error: %w", err)
	}
	// 解码（严格模式）
	dec := msgpack.NewDecoder(NewBytesReader(raw))
	dec.SetCustomStructTag("msgpack")
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		// 默认 map[string]any；你也可以按需切换到 map[any]any
		return d.DecodeMap()
	})
	if err := dec.Decode(v); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid msgpack: %w", err)
	}
	// 业务校验
	if err := validateMsgpack.Struct(v); err != nil {
		var verse validator.ValidationErrors
		if errors.As(err, &verse) {
			return http.StatusUnprocessableEntity, fmt.Errorf("validation failed: %s", verse.Error())
		}
		return http.StatusUnprocessableEntity, fmt.Errorf("validation error: %w", err)
	}
	return http.StatusOK, nil
}

func CreateOrderMsgpack(ctx *gin.Context) {
	var req CreateOrderReq
	status, err := BindMsgpackBody(ctx.Writer, ctx.Request, &req, 1<<20, 2<<20)
	if err != nil {
		ctx.JSON(status, gin.H{"code": status, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

func RegisterMsgpackRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		api.POST("/msgpack", CreateOrderMsgpack)
	}
}
