/*
 * File: 07-maps.go
 * Date: 2025/6/26 15:44
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import "fmt"

func demoMap() {
	// 预估容量: 通常能减少扩容与 rehash。估小一点没关系，最多多扩一两次；避免估得特别大导致长期占用内存
	m := make(map[string]int, 5)
	m["a"] = 1
	m["b"] = 2
	fmt.Printf("m: %v\n", m)

	var m1 map[string]int                            // 声明了一个nil map
	fmt.Printf("m1: %v  IsNil=%t \n", m1, m1 == nil) // map[]  IsNil=true

	// 如果key 不存在返回对应value类型的零值
	v, ok := m["a"]
	fmt.Printf("v: %v  IsNil=%t \n", v, ok)
}

// SumInts 在不使用泛型适配不同类型的示例 type switch方式
func SumInts(xs any) (int64, error) {
	// 检查 xs（必须是接口类型，如 any）里动态类型是什么，然后跳到匹配的 case T
	switch v := xs.(type) { // type 是固定关键字
	case []int:
		fmt.Printf("case []int: v 类型=%T, len=%d\n", v, len(v)) // case []int: v 类型=[]int, len=3
		var s int64
		for _, x := range v {
			s += int64(x)
		}
		return s, nil
	case []int8:
		var s int64
		for _, x := range v {
			s += int64(x)
		}
		return s, nil
	case []int16:
		var s int64
		for _, x := range v {
			s += int64(x)
		}
		return s, nil
	case []int32:
		var s int64
		for _, x := range v {
			s += int64(x)
		}
		return s, nil
	case []int64:
		var s int64
		for _, x := range v {
			s += x
		}
		return s, nil
	}
	return 0, fmt.Errorf("unsupported: %T", xs)
}

func main() {
	demoMap()
	sum, err := SumInts([]int{1, 2, 3})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(sum)
	/*
		Map它是引用类型，用于存储键值对的无序集合，map中的值可以是任何类型，包括内建类型、结构体、指针、函数、切片、甚至是另一个map
		map的键必须是可以比较相等的类型
	*/
}
