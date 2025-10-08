/*
 * File: 12-validation-rules.go
 * Date: 2025/9/9 23:46
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validateRules *validator.Validate

// 常见枚举

var (
	ISO4217 = map[string]struct{}{
		"USD": {}, "PHP": {}, "CNY": {}, "HKD": {}, "EUR": {}, "JPY": {}, "KRW": {}, "SGD": {},
	}
	OrderCurrencyEnum = ISO4217
	OrderSourceEnum   = map[string]struct{}{"PC": {}, "H5": {}, "APP": {}}
)

//  自定义正则

var (
	/*
		(?=.*[A-Za-z]) 正向前瞻，要求后续字符串中必须至少有一个英文字母（大小写均可）
		(?=.*\d) 正向前瞻，要求后续字符串中必须至少有一个数字
		(?=.*[^A-Za-z0-9]) 正向前瞻，要求后续字符串中必须至少有一个非字母非数字的字符（也就是特殊符号，如 @ # $ % 等）
		.{8,32} 任意字符（除换行符）长度必须在 8 到 32 位之间
		(?:0|[1-9]\d*) 非捕获分组，表示金额的整数部分  0：允许单独为 0 [1-9]\d*：非零开头的数字（1,2,...99999）
	*/
	// go 不支持 (?=...)、(?!...) 这类前瞻/后顾语法
	// rePasswordStrong = regexp.MustCompile(`^(?=.*[A-Za-z])(?=.*\d)(?=.*[^A-Za-z0-9]).{8,32}$`)
	reAmount2dp  = regexp.MustCompile(`^(?:0|[1-9]\d*)(?:\.\d{1,2})?$`) // 金额最多2位小数
	reUUIDv4     = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	reHasLetter  = regexp.MustCompile(`[A-Za-z]`)
	reHasDigit   = regexp.MustCompile(`[0-9]`)
	reHasSpecial = regexp.MustCompile(`[^A-Za-z0-9]`)
)

func IsStrongPassword(pw string) bool {
	n := utf8.RuneCountInString(pw)
	if n < 8 || n > 32 {
		return false
	}
	return reHasLetter.MatchString(pw) &&
		reHasDigit.MatchString(pw) &&
		reHasSpecial.MatchString(pw)
}

// Init 创建并返回全局 validator
func Init() *validator.Validate {
	if validateRules != nil {
		return validateRules
	}

	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})
	// 自定义规则注册
	// not blank: 去除首尾空格后不得为空
	_ = v.RegisterValidation("not-blank", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind().String() != "string" {
			return false
		}
		return strings.TrimSpace(fl.Field().String()) != ""
	})
	// trimmed: 不允许首尾空格
	_ = v.RegisterValidation("trimmed", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind().String() != "string" {
			return false
		}
		s := fl.Field().String()
		return s == strings.TrimSpace(s)
	})

	// password
	_ = v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind().String() != "string" {
			return false
		}
		return IsStrongPassword(fl.Field().String())
	})
	// amount
	_ = v.RegisterValidation("amount2dp", func(fl validator.FieldLevel) bool {
		switch fl.Field().Kind().String() {
		case "string":
			return reAmount2dp.MatchString(fl.Field().String())
		case "float64", "float32":
			// 统一转 string 做正则校验，避免浮点精度坑
			// .10f 把一个浮点数格式化为定点小数字符串，保留 10 位小数，四舍五入并补零
			// fl.Field().Float() 获取对应字段当前的数值 返回一个 float64
			s := fmt.Sprintf("%.10f", fl.Field().Float())
			s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
			return reAmount2dp.MatchString(s)
		case "int", "int64", "int32":
			return fl.Field().Int() > 0
		default:
			return false
		}
	})

	// in_currency: ISO4217 枚举
	_ = v.RegisterValidation("in_currency", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind().String() != "string" {
			return false
		}
		_, ok := OrderCurrencyEnum[strings.ToUpper(fl.Field().String())]
		return ok
	})

	// in_source: 业务来源枚举
	_ = v.RegisterValidation("in_source", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind().String() != "string" {
			return false
		}

		_, ok := OrderSourceEnum[strings.ToUpper(fl.Field().String())]
		return ok
	})

	//  uuid4: 严格 uuid v4
	_ = v.RegisterValidation("uuid4strict", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind().String() != "string" {
			return false
		}
		return reUUIDv4.MatchString(fl.Field().String())
	})

	//  time fmt=2006-01-02T15:04:05Z07:00（标签参数指定布局）
	_ = v.RegisterValidation("time-fmt", func(fl validator.FieldLevel) bool {
		param := fl.Param()
		if param == "" {
			// 默认 RFC3339
			param = time.RFC3339 // 2006-01-02T15:04:05Z07:00
		}
		val := fl.Field().String()
		_, err := time.Parse(param, val)
		return err == nil
	})

	// slice_unique: 切片去重校验（string / int）
	_ = v.RegisterValidation("slice_unique", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		switch field.Kind().String() {
		case "slice":
			seenS := map[string]struct{}{}
			seenI := map[int64]struct{}{}
			for i := 0; i < field.Len(); i++ {
				elem := field.Index(i)
				switch elem.Kind().String() {
				case "string":
					s := elem.String()
					if _, ok := seenS[s]; ok {
						return false
					}
					seenS[s] = struct{}{}
				case "int", "int64", "int32":
					n := elem.Int()
					if _, ok := seenI[n]; ok {
						return false
					}
					seenI[n] = struct{}{}
				default:
					return false
				}
			}
			return true
		default:
			return false

		}
	})

	//  max runes=n：以字符数限制长度（支持中文）
	_ = v.RegisterValidation("max-runes", func(fl validator.FieldLevel) bool {
		// 当你写 validate:"max-runes=20" 时，fl.Param() 会返回 "20"（类型是 string）
		param := fl.Param()
		var limit int
		// 解析出来的值写入你提供的变量，Sscanf 根据格式串 %d 解析出一个整数，然后把这个整数写到 limit 所在的内存中，需要指针
		if _, err := fmt.Sscanf(param, "%d", &limit); err != nil || limit < 0 {
			return false
		}
		return utf8.RuneCountInString(fl.Field().String()) <= limit
	})

	validateRules = v
	return validateRules
}

func ValidateEnter(i any) error {
	v := Init()
	// 会反射遍历你的结构体 i，错误通常可断言为 validator.ValidationErrors，里面是一组 FieldError，可用来生成你现在的中文提示
	return v.Struct(i)
}

// Translate 将 validator 错误转中文可读
func Translate(err error) map[string]string {
	out := map[string]string{}
	var verse validator.ValidationErrors
	if !errors.As(err, &verse) {
		out["_"] = err.Error()
		return out
	}
	for _, fe := range verse {
		field := fe.Field() // 直接用字段名；可配合 tag jsonName 做映射
		switch fe.Tag() {
		case "required":
			out[field] = "必填"
		case "not-blank":
			out[field] = "不能为空或全空白"
		case "trimmed":
			out[field] = "首尾不能有空格"
		case "email":
			out[field] = "邮箱格式不正确"
		case "uuid4strict":
			out[field] = "必须为 UUID v4"
		case "in_currency":
			out[field] = "不在支持的币种列表"
		case "in_source":
			out[field] = "来源非法（允许：PC/H5/APP）"
		case "oneof":
			out[field] = fmt.Sprintf("不在允许集合：%s", fe.Param())
		case "len":
			out[field] = fmt.Sprintf("长度必须为 %s", fe.Param())
		case "min":
			out[field] = fmt.Sprintf("最小值为 %s", fe.Param())
		case "max":
			out[field] = fmt.Sprintf("最大值为 %s", fe.Param())
		case "gte":
			out[field] = fmt.Sprintf("必须 ≥ %s", fe.Param())
		case "lte":
			out[field] = fmt.Sprintf("必须 ≤ %s", fe.Param())
		case "gt":
			out[field] = fmt.Sprintf("必须 > %s", fe.Param())
		case "lt":
			out[field] = fmt.Sprintf("必须 < %s", fe.Param())
		case "time-fmt":
			p := fe.Param()
			if p == "" {
				p = time.RFC3339
			}
			out[field] = fmt.Sprintf("时间格式不正确，要求：%s", p)
		case "amount2dp":
			out[field] = "金额格式不正确（非负，最多2位小数）"
		case "password":
			out[field] = "密码需含字母、数字、特殊字符，长度8-32"
		case "slice_unique":
			out[field] = "数组内存在重复值"
		case "max-runes":
			out[field] = fmt.Sprintf("字符数不能超过 %s", fe.Param())
		default:
			out[field] = "参数不合法"
		}
	}
	return out
}

type OrderItem struct {
	SKU   string  `json:"sku"    validate:"required,not-blank,trimmed,max-runes=64"`
	Qty   int     `json:"qty"    validate:"required,gt=0,lte=999999"`
	Price string  `json:"price"  validate:"required,amount2dp"` // 用 string 避免浮点误差
	Note  *string `json:"note"   validate:"omitempty,max-runes=200"`
}

type CreateOrderValidationReq struct {
	OrderID    string      `json:"order_id"     validate:"required,trimmed,uuid4strict"`
	Amount     string      `json:"amount"       validate:"required,amount2dp"`
	Currency   string      `json:"currency"     validate:"required,in_currency"`
	CustomerID string      `json:"customer_id"  validate:"required,not-blank,trimmed"`
	Email      string      `json:"email"        validate:"required,email"`
	Source     string      `json:"source"       validate:"required,in_source"`
	Items      []OrderItem `json:"items"        validate:"required,min=1,dive,required"`
	// 交叉校验：EndTime >= StartTime（字符串 RFC3339）
	StartTime string `json:"start_time" validate:"omitempty,time-fmt"` // 默认 RFC3339
	// gte field=StartTime 与同一结构体的其他字段比较（用于时间顺序、数值依赖）
	EndTime string `json:"end_time"   validate:"omitempty,time-fmt=2006-01-02 15:04:05"`
	// 切片唯一
	Tags     []string `json:"tags" validate:"omitempty,slice_unique,max=20,dive,max-runes=32"`
	Password string   `json:"password" validate:"required,password"`
}

// OrderItemResp 返回体定义
type OrderItemResp struct {
	SKU       string  `json:"sku"`
	Qty       int     `json:"qty"`
	Price     string  `json:"price"`      // 原样回显（字符串金额）
	LineTotal string  `json:"line_total"` // qty * price，保留2位小数字符串
	Note      *string `json:"note,omitempty"`
}

type CreateOrderResp struct {
	Code       int             `json:"code"`
	Msg        string          `json:"msg"`
	OrderID    string          `json:"order_id"`
	Amount     string          `json:"amount"` // 原样回显
	Currency   string          `json:"currency"`
	CustomerID string          `json:"customer_id"`  // 可按需脱敏
	Email      string          `json:"email_masked"` // 脱敏后的邮箱
	Source     string          `json:"source"`
	Tags       []string        `json:"tags,omitempty"`
	StartTime  string          `json:"start_time,omitempty"`
	EndTime    string          `json:"end_time,omitempty"`
	Items      []OrderItemResp `json:"items"`
	ItemsTotal string          `json:"items_total"` // Σ(line_total)，2位小数
	ServerTime string          `json:"server_time"` // RFC3339
	Checks     struct {
		AmountEqualsItemsTotal bool   `json:"amount_equals_items_total"`
		Note                   string `json:"note,omitempty"`
	} `json:"checks"`
}

func maskEmail(s string) string {
	at := strings.IndexByte(s, '@')
	if at <= 1 {
		return "***"
	}
	name := s[:at]
	domain := s[at:]
	// 保留首尾各1个字符，中间用*
	var mid = ""
	if len(name) > 2 {
		mid = strings.Repeat("*", len(name)-2)
	}
	return name[:1] + mid + name[len(name)-1:] + domain
}

// 工具函数（精确金额运算 + 脱敏）
func mulPrice(qty int, priceStr string) (string, error) {
	// 用 big.Rat 做十进制精确乘法：price * qty → 保留2位小数
	var r big.Rat
	if _, ok := r.SetString(priceStr); !ok {
		return "", fmt.Errorf("invalid price: %s", priceStr)
	}
	var q big.Rat
	// 把数量转有理数：n = qty
	q.SetInt64(int64(qty))
	r.Mul(&r, &q)                // r = price * qty
	return r.FloatString(2), nil // 小数点后2位，四舍五入
}

func addMoney(a, b string) (string, error) {
	var r1, r2 big.Rat
	if _, ok := r1.SetString(a); !ok {
		return "", fmt.Errorf("invalid amount: %s", a)
	}
	if _, ok := r2.SetString(b); !ok {
		return "", fmt.Errorf("invalid amount: %s", b)
	}
	r1.Add(&r1, &r2)
	return r1.FloatString(2), nil
}

func CreateOrderHandler(ctx *gin.Context) {
	var req CreateOrderValidationReq
	// 直接 BindJSON 到结构体（Gin 会读 Body 并反序列化）
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "无效的JSON体",
			"err":  err.Error(),
		})
		return
	}
	// 结构体校验
	if err := ValidateEnter(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 422,
			"msg":  "参数校验失败",
			"errs": Translate(err), // 字段 -> 中文错误
		})
		return
	}
	resp := CreateOrderResp{
		Code:       0,
		Msg:        "ok",
		OrderID:    req.OrderID,
		Amount:     req.Amount,
		Currency:   req.Currency,
		CustomerID: req.CustomerID,       // 如需脱敏可自定义
		Email:      maskEmail(req.Email), // 不回传明文邮箱
		Source:     req.Source,
		Tags:       req.Tags,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		ServerTime: time.Now().Format(time.RFC3339),
	}
	resp.Items = make([]OrderItemResp, 0, len(req.Items))
	sum := "0.00"
	for _, it := range req.Items {
		line, err := mulPrice(it.Qty, it.Price)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 422, "msg": "明细金额非法", "detail": err.Error()})
			return
		}
		sum, err = addMoney(sum, line)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 422, "msg": "汇总失败", "detail": err.Error()})
			return
		}
		resp.Items = append(resp.Items, OrderItemResp{
			SKU:       it.SKU,
			Qty:       it.Qty,
			Price:     it.Price,
			LineTotal: line,
			Note:      it.Note,
		})
	}
	resp.ItemsTotal = sum
	// 一致性提示（领域校验的轻量回显，不阻断）
	eq := sum == req.Amount
	resp.Checks.AmountEqualsItemsTotal = eq

	// TODO: 业务处理...
	ctx.JSON(http.StatusOK, resp)
}

func RegisterValidationRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		api.POST("/validation/orders", CreateOrderHandler)
	}
}
