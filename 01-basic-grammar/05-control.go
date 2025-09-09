/*
 * File: 05-control.go
 * Date: 2025/6/25 21:37
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */
package main

import "fmt"

// Go 支持在 if 语句里提前声明并初始化一个局部变量，这个变量的作用域只在 if/else 块里,作用域仅限于上面的 if-else 语句块
func demoIfElseScopedVar(start, end int) string {
	fmt.Println("=== Go if-else 局部变量示例 ===")

	// 在 if 语句里定义一个临时变量 distance
	if distance := end - start; distance > 0 {
		// distance 只在 if-else 内可见
		return fmt.Sprintf("条件成立，end 比 start 大，差值 = %d", distance)
	} else {
		return fmt.Sprintf("条件不成立，end <= start，差值 = %d", distance)
	}

	// 编译错误: 这里不能访问 distance
	// fmt.Println(distance)
}

func demoForLoop() {
	i := 0
	for i < 10 {
		fmt.Println(i)
		i++
	}
}

func demoForRange() {
	arr := [10]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for idx, val := range arr {
		fmt.Println(idx, val)
	}
	for idx := range arr {
		fmt.Println(idx, arr[idx])
	}
}

// Go 1.22 新特性：for 的两大改动
// 1- 循环变量不再共享  2-for 支持整数范围
func demoForLoopBehavior() {
	fmt.Println("新语义 (>= Go 1.22):")
	func() {
		var fs []func()
		// 每次循环都会生成 i 的新副本
		for i := 0; i < 3; i++ {
			// 老版本闭包捕获同一个变量，新版本闭包捕获独立副本
			fs = append(fs, func() { fmt.Println(i) })
		}
		for _, f := range fs {
			f()
		}
	}()

	// 新增了 整数区间写法，类似 Python
	for i := range 5 {
		fmt.Println(i)
	}
}

// User 坑点
type User struct {
	Name string
}

func demoForLoopFeature() {
	u := []User{
		{Name: "Tom"},
		{Name: "Jack"},
		{Name: "John"},
	}
	um := make(map[string]*User, len(u))
	for idx := range u {
		// 对 range 变量 v 取地址, &v每轮地址不同，不再像 Go ≤1.21 那样所有指针都一样
		um[u[idx].Name] = &u[idx]
	}
	um["Jack"].Name = "Changed" // 取出map中的值，类型是*User
	// Go对(*ptr).Field 提供了简写规则
	// 完整写法 (*um["Jack"]).Name = "Changed"  *um["Jack"] → 解引用，得到 User
	for key, v := range u {
		fmt.Println("key:", key, "value:", v.Name)
	}
}

func demoSwitch1(score int) {
	// switch expr { case v1: ... } → 只能匹配值
	switch score {
	case 100, 200, 300: // 编译器会拿 score == caseValue 来对比
		fmt.Println("A")
	case 500:
		fmt.Println("B")
	default:
		fmt.Println("C")
	}
}

func demoSwitch2(score int) {
	// switch { case cond: ... } → 可以写任意条件
	switch {
	case score == 100, score == 200, score == 300:
		fmt.Println("A")
	case score == 500:
		fmt.Println("B")
	case score > 600 && score < 1000:
		fmt.Println("C")
	default:
		fmt.Println("D")
	}
}

func main() {
	fmt.Println("在if语句里提前声明并初始化一个局部变量:", demoIfElseScopedVar(1, 2))
	demoForLoopBehavior()
	demoForLoopFeature()
	demoSwitch2(100)
}
