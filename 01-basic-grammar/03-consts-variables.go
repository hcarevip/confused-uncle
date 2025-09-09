/*
 * File: 03-variables.go
 * Date: 2025/6/25 21:25
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */
package main

import "fmt"

// Name 包级变量（package-level variable）对这个包里所有文件有效
var (
	Name    string = "张三" // 变量名首字母大写可以跨包使用，但是不到逼不得已的地步不要这样使用
	address string = "澳门" // 小写只在本包可见
)

func main() {
	// 常量 iota 控制常量初始化
	const (
		A = iota // 同一个 const 块里每换一行自动 +1
		B
		C
		D = 100
		E
		F
		G    = iota
		Bit1 = 1 << iota // 当前行是第 7 行，iota=7, 左移 7 位 = 乘 2⁷ = 128
		Bit2
		Bit3
	)
	fmt.Println(A, B, C, D, E, F, G, Bit1, Bit2, Bit3) // 0 1 2 100 100 100 6 128 256 512

}
