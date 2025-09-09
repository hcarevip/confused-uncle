/*
 * File: 02-types.go
 * Date: 2025/6/25 21:25
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"bytes"
	"fmt"
	"math"
	"strings"
)

func main() {
	/**
	基本类型
	(1) int 家族：int8、int16、int32、int64、int再内存分别使用 1 2 4 8 个字节来表示
	    而int使用几个字节取决与CPU，目前大多数int都是8个字节
	(2) uint 家族：uint8、uint16、uint32、uint64、uint 无符号整数,优先使用uint
	(3) float家族：float32、float64 优先使用float64
	*/

	// math 包下复杂运算例子
	fmt.Println(math.Ceil(1.08))
	fmt.Println(math.Floor(1.08)) // 向下取整
	fmt.Println(math.Abs(-3.52))
	fmt.Println(math.Round(-3.52))
	fmt.Println(math.Max(3.52, 1.08)) // 比大小
	fmt.Println(math.Min(3.52, 1.08))

	fmt.Println(math.NaN()) // NaN  Not a Number

	// 幂运算 Pow(x, y) 表示 x 的 y 次方
	result := math.Pow(2, 3)     // 2 的 3 次方
	fmt.Println("2^3 =", result) // → 8

	principal := 1000.0
	rate := 1.05
	years := 5.0
	future := principal * math.Pow(rate, years)
	if !math.IsNaN(future) {
		fmt.Println("Future value:", future)
	}
	// Sqrt 平方根
	fmt.Println("Sqrt(9):", math.Sqrt(9))   // 3
	fmt.Println("Sqrt(16):", math.Sqrt(16)) // 4
	fmt.Println("Sqrt(3):", math.Sqrt(3))   // 1.7320508075688772

	// int 和 uint都有最大值和最小值，float32 和 float64只有最大值和最小正数，没有最小值
	fmt.Println("float64 最大值:", math.MaxFloat64)
	fmt.Println("float64 最小正数:", math.SmallestNonzeroFloat64)

	// 字符串相关操作 strings 包下
	// 对字符串中的每个字符（rune）执行一个函数，然后把返回的 rune 组成一个新字符串
	// 常用场景: 过滤非法字符、替换某些字符、去除空白字符等等
	input := "你好，世界！" // 中文标点替换为英文标点
	res := strings.Map(func(r rune) rune {
		switch r {
		case '，':
			return ','
		case '！':
			return '!'
		default:
			return r
		}
	}, input)
	fmt.Println("Output:", res)

	// byte相关操作 bytes 包下
	// Clone 返回原 []byte 的一份 新拷贝,两个切片底层内存完全分离，后续对新切片修改不影响原切片
	original := []byte("hello world")
	copyData := bytes.Clone(original)
	copyData[0] = 'H'
	fmt.Println("Original:", string(original)) // hello world
	fmt.Println("Clone   :", string(copyData)) // Hello world

	// []byte（字节切片） []byte 和 字符串互相转换
	// string 是不可变的字节序列  []byte 是可变的字节切片
	s := "uu-coming"
	b := []byte(s)         // 会 拷贝 s 的所有底层字节，生成一个新的 []byte 切片
	fmt.Println(string(b)) // 也会 拷贝 b 的所有字节，生成一个新的只读 string
	// 如果高频场景 用 bytes.Buffer替代，减少内存开销
	var buffer bytes.Buffer
	buffer.WriteString(s + " ")
	buffer.Write(b)
	fmt.Println(buffer.String()) // 只在最后一次转

	// bool: &&  ||  !
	// !(a && b) 等价于 !a || !b 括号中的 && 变成 ||   德摩根律
	// !(a || b) 等价于 !a && !b 括号中的 || 变成 &&
}
