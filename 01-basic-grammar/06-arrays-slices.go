/*
 * File: 06-slices.go
 * Date: 2025/6/25 21:37
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"fmt"
)

func demoArray() {
	// 数组（或切片、map、struct 等）在定义时，未显式赋值的元素，会自动初始化为该类型的零值
	// len 实际多少元素，cap 可以放多少元素
	ta1 := [3]int{1, 2}
	fmt.Printf("ta1: %v - len: %d - cap: %d\n", ta1, len(ta1), cap(ta1))

	// 初始化后内存已经分配好了，没有赋值也是零值
	var ta2 [3]string
	fmt.Printf("ta2: %v - len: %d - cap: %d\n", ta2, len(ta2), cap(ta2))

	// 数组不支持append, 因为数组是值类型（value type）声明时长度是类型的一部分，例如 [3]string 和 [4]string 是两种不同的类型
	// 所以数组是固定长度的，长度一旦确定就不可变
	// ta2 = append(ta2, []string{"a", "b", "c", "d"}...)
}

// 打印切片的 header 信息
func printSliceHeader(name string, sl []int) {
	// 切片头（header）本身的地址（每个变量一份拷贝，所以都不同）
	fmt.Printf("%s: 切片头地址=%p  底层数组起点=%p  长度=%d  容量=%d\n",
		name, &sl, &sl[0], len(sl), cap(sl))

	for i := range sl {
		fmt.Printf("    元素[%d] 地址=%p  值=%d\n", i, &sl[i], sl[i])
	}
	/**
	类似输出
	s : 切片头地址=0xc0000a2090  底层数组起点=0xc0000b6030  长度=5  容量=5
	    元素[0] 地址=0xc0000b6030  值=10
	    元素[1] 地址=0xc0000b6038  值=20
	    元素[2] 地址=0xc0000b6040  值=30
	    元素[3] 地址=0xc0000b6048  值=40
	    元素[4] 地址=0xc0000b6050  值=50
	*/
}

func demoSlice() {
	// arr本身不是数组，而是一个结构体值,arr里的第一个字段ptr指向底层数组的第一个元素
	arr := []string{"A", "B", "C"}
	// 里面的ptr永远指向 底层数组的某个元素，切片header 中的 ptr 默认指向切片的第一个元素
	fmt.Printf("arr指向底层数组的指针：%p  底层数组第一个值的指针：%p \n", arr, &arr[0])

	// 切片header是值复制，所有切片变量共享同一个底层数组，所以修改一个切片的内容会影响其它切片
	s := []int{10, 20, 30, 40, 50}
	s1 := s[:]             // 切片 s1 从索引 0 开始（等于 s）
	s2 := s[2:]            // 切片 s2 从索引 2 开始
	s3 := s[1:4]           // 切片 s3 从索引 1 到 4
	fmt.Println("s :", s)  //  [10 20 30 40 50]
	fmt.Println("s1:", s1) //  [10 20 30 40 50]
	fmt.Println("s2:", s2) // [30 40 50]
	fmt.Println("s3:", s3) //  [20 30 40]

	// 切片内部 header 信息
	printSliceHeader("s ", s)
	printSliceHeader("s1", s1)
	printSliceHeader("s2", s2)
	printSliceHeader("s3", s3)

	// 修改 s2[0]，会影响 s 和 s3
	s2[0] = 300
	fmt.Println("\n修改 s2[0] = 300 之后:")
	fmt.Println("s :", s)
	fmt.Println("s2:", s2)
	fmt.Println("s3:", s3)

	// 扩容
	ss := []int{10, 20, 30, 40, 50}
	fmt.Printf("指向底层数组的指针=%p  底层数组起点=%p 长度=%d  容量=%d\n", ss, &ss[0], len(ss), cap(ss))
	// append 触发了扩容，运行时分配了新的底层数组，把旧数据拷贝过去，然后把 ss.ptr 改为指向新数组首元素，并更新 len/cap
	ss = append(ss, 60)
	fmt.Printf("指向底层数组的指针=%p  底层数组起点=%p 长度=%d  容量=%d\n", ss, &ss[0], len(ss), cap(ss))

	// 当多个切片共享同一底层数组时,扩容发生：a、b 不再共享
	base := []int{10, 20, 30, 40, 50} // len=5 cap=5
	a := base[:]                      // 与 base 共享底层数组（起点相同）
	b := base[:]                      // 也共享
	// 情况 1：a 先 append，触发扩容（cap 满了）
	a = append(a, 60)                                      // a 现在指向“新数组”，b 仍指向旧数组
	a[0] = 999                                             // 改 a 的数据只影响 a（新数组）
	fmt.Printf("a=%v  len=%d cap=%d\n", a, len(a), cap(a)) // a=[999 20 30 40 50 60]  len=6 cap=10
	fmt.Printf("b=%v  len=%d cap=%d\n", b, len(b), cap(b)) // b=[10 20 30 40 50]  len=5 cap=5

	// 在初始化切片的时候预估容量（如果清楚最终会有 N 个元素：直接一次到位 优点：0 次扩容；最省 GC 与拷贝）

}
func main() {
	// demoArray()
	demoSlice()

	/**
	切片的底层结构
	一个 切片（slice） 并不是直接存数据，而是一个小的描述符结构体
	type Slice struct {
	    array unsafe.Point  指向底层数组的指针也就是指向切片当前可见部分的底层数组的第一个元素的指针
	    len  int            切片的长度
	    cap  int            切片的容量
	}
	*/
}
