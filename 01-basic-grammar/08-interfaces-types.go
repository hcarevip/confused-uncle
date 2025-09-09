/*
 * File: 08-interfaces-types.go
 * Date: 2025/6/28 19:33
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
	"unsafe"
)

type Book struct {
	// 结构体和结构体字段也都是遵循大小写控制访问原则
	Title string
}

// SetByPtr 指针接收者,能用 *T 调；也能用 T 调（前提：这个 T 是可取地址的变量，编译器会自动取地址）
// 指针接收者减少复制但引入共享状态，要考虑并发安全
func (b *Book) SetByPtr(t string) {
	b.Title = t
}

type Reader interface {
	read() string
}

func (b *Book) read() string {
	// 安全处理 typed-nil
	if b == nil {
		return "<nil book>"
	}
	return "reading"
}

// 验证只有当动态类型和动态值都为空时，接口变量才等于 nil
func testCase1() {
	var r Reader // 接口里什么都没有时
	// 接口变量 r 包含：动态类型 和 动态值，只有都为空时，接口变量才等于 nil
	fmt.Printf("testCase1: r==nil? %v, type=%T, value=%#v\n", r == nil, r, r)
	//  r==nil? true, type=<nil>, value=<nil>
}

func testCase2() {
	// 接口实现检查的规则：只看方法集，不做自动取址,指针方法 ⇒ 只有 *T 实现；T 不实现
	// 接口变量 r 包含：动态类型 和 动态值
	var r Reader = &Book{Title: "一千零二夜"} // 是指把指针（如 &Book{…}  类型是 *Book）赋给接口变量
	fmt.Printf("testCase2: r==nil? %v, type=%T, value=%#v\n", r == nil, r, r)
	// r==nil? false, type=*main.Book, value=&main.Book{Title:"一千零二夜"}
	// 动态类型：*main.Book 接口变量里装着的具体类型是谁，这里就是指向 Book 的指针
	// 动态值： &main.Book{Title:"一千零二夜"} 这个具体类型的实际内容（这里是一根非空指针，指向那本书）
	fmt.Println("testCase2 call:", r.read())
}

func testCase3() {
	var pb *Book = nil
	var r Reader = pb
	fmt.Printf("C4: r==nil? %v, type=%T, value=%#v\n", r == nil, r, r)
	//  r==nil? false, type=*main.Book, value=(*main.Book)(nil)
	// 此时 r != nil，因为类型已经是 *main.Book, 了，即使值是 nil
}

func h(s *[]int) {
	fmt.Printf("函数内：形参 s（= 调用方 &slice 变量）=%p\n", s)

	var oldBase, newBase unsafe.Pointer
	if len(*s) > 0 {
		oldBase = unsafe.Pointer(&(*s)[0]) // 旧底层数组首址
	}

	*s = append(*s, 9) // 可能触发扩容，换新数组
	if len(*s) > 0 {
		newBase = unsafe.Pointer(&(*s)[0]) // 新底层数组首址
	}

	fmt.Printf("函数内：旧底层首址=%p  新底层首址=%p  s=%v\n", oldBase, newBase, *s)
}

// JSONTime 基于 TypeB 的底层表示，定义了一个全新的类型 TypeA
// 它和别名类型（type A = B）完全不同——衍生类型是新类型，别名不是新类型
type JSONTime time.Time // 衍生类型：全新类型
// FormatYMD 给 time.Time 扩展一个自定义格式化方法；你不能给 time.Time 直接加方法（非本包类型），但可以衍生一个新类型再加
func (t JSONTime) FormatYMD() string { // 给新类型加方法
	// t.FormatYMD("2006-01-02") 语法报错
	return time.Time(t).Format("2006-01-02") // 注意显式转换回 time.Time 使用其能力
}

// ReaderWriter 1 接口组合接口
// 在 struct 或 interface 里匿名放入另一个类型，让它的字段/方法提升（promote）到外层，像是拼装能力
type ReaderWriter interface {
	io.Reader
	io.Writer
}

type MyCloser interface {
	ReaderWriter  // 组合已有接口
	Close() error // 再加自己特有的方法
}

// Counter 2 结构体组合结构体 方法/字段被提升，外层可直接用
type Counter struct{ n int }

func (c *Counter) Inc()     { c.n++ } // 指针接收者方法
func (c *Counter) Val() int { return c.n }

type S1 struct {
	Counter // 以“值”组合
}

// S2 3 结构体组合结构体指针
type S2 struct {
	*Counter // 以“指针”组合
}

// Logger 4 结构体组合接口（可插拔）
type Logger interface {
	Printf(string, ...any)
}
type StdLogger struct{}

func (StdLogger) Printf(f string, a ...any) { fmt.Printf(f+"\n", a...) }

type JSONLogger struct{}

func (JSONLogger) Printf(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	b, _ := json.Marshal(msg)
	fmt.Println("[JSONLogger]", string(b))
}

type Service struct {
	// Service 上可以直接使用 Logger 的方法名
	/*
		结构体嵌入接口，本体并不实现接口方法，但可以像有这个方法一样调用

	*/
	Logger // 组合接口
}

func NewService(l Logger) *Service {
	if l == nil {
		panic("logger required")
	}
	s := &Service{Logger: l}
	fmt.Printf("NewService: Logger 动态类型=%T  动态值=%#v  isNil=%v\n", s.Logger, s.Logger, s.Logger == nil)
	return s
}

func (s Service) Work() {
	fmt.Printf("Work:     s.Logger 动态类型=%T  动态值=%#v  isNil=%v\n", s.Logger, s.Logger, s.Logger == nil)

	// 注意：s.Printf(...) 只是语法糖，等价于 s.Logger.Printf(...)
	fmt.Println("→ 通过方法提升调用：")
	s.Printf("start (promoted)")

	fmt.Println("→ 显式写字段调用：")
	s.Logger.Printf("start (explicit)")
}
func main() {
	// 能否改到外部取决于方法接收者是否为指针，与你调用时传 T 还是 *T 无关
	// 假设方法接收者是否为值接收，这里使用b := &Book{Title: "一千零一夜"} 也是改不了的
	b := Book{Title: "一千零一夜"}
	b.SetByPtr("指针接收者+值变量") // 可取地址，编译器自动取地址 (&b).SetByPtr()
	fmt.Println("A3:", b.Title)

	// 接口值 = 动态类型 + 动态值
	testCase1()
	testCase2()
	testCase3()

	// Go 只有值传递,指针类型（*T）做参数时，拷贝的是地址这个值
	s := []int{1, 2, 3}
	fmt.Printf("调用前：slice变量地址=&s=%p  底层首址=&s[0]=%p  s=%v\n", &s, &s[0], s)
	h(&s) // 0xc000008030
	fmt.Printf("调用后：slice变量地址=&s=%p  底层首址=&s[0]=%p  s=%v\n", &s, &s[0], s)

	// 新定义类型 / defined type  语法：type TypeA TypeB  基于 TypeB 的底层表示，定义了一个全新的类型 TypeA
	// TypeA 是一个全新的类型，即使它的底层类型和 TypeB 一样
	// 需要显式转换：var a TypeA; var b TypeB; a = TypeA(b)
	// 方法不会自动继承：TypeB 上有的方法，TypeA 不会自动拥有
	// 接口实现不传递：TypeB 实现了某接口，并不意味着 TypeA 也实现了；除非你在 TypeA 上把那些方法重新实现一遍
	d := JSONTime(time.Now())  // 显式转换
	fmt.Println(d.FormatYMD()) // 2025-09-0

	// 组合（embedding）
	// 组合 不等于 继承。外层只是转发/提升内层的方法与字段
	s1 := S1{Counter: Counter{10}}
	s1.Inc()
	fmt.Println(s1.Val()) // 等价 s.Counter.Val() → 11

	s2 := S2{Counter: &Counter{10}}
	s2.Inc()
	fmt.Println(s2.Val()) // 11

	fmt.Println("-------------------------------------")
	s3 := NewService(StdLogger{})
	// 方法提升不等于复制方法：Service 并没有真的长出 Printf；只是选择器解析时，编译器把 s.Printf解析为 s.Logger.Printf
	// 而接口本身没有实现,真正执行的是 s.Logger 里动态类型的方法, 运行时根据 type 跳到对应实现的方法
	// 真正执行哪个实现，取决于接口值的动态类型（这里是 StdLogger 或 JSONLogger）
	s3.Work()
	fmt.Println("--- 切换实现为 JSONLogger ---")
	// 运行时切换为另一个实现（动态分派到新实现）
	// 切换后 s.Logger 动态类型=main.JSONLogger  动态值=main.JSONLogger{}  isNil=false
	s3.Logger = JSONLogger{}
	s3.Work()

	// 演示 nil 接口会 panic（故意触发，捕获后打印）
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic 捕获：", r)
		}
	}()
	var sNil Service // 零值：嵌入的接口字段为 nil
	fmt.Println("--- 调用零值 Service.Work（会 panic） ---")
	sNil.Work()
}
