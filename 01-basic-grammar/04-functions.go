/*
 * File: 04-functions.go
 * Date: 2025/6/25 21:25
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import "fmt"

// 带名字的返回值
// 相当于你声明了一个作用域在整个方法的局部变量
func doSomething(success bool) (result string, err error) {
	// (result string, err error) 是 带名字的返回值,Go 会在函数体里自动帮你声明
	/**
	它们在整个函数作用域内都是可见的局部变量
	var result string
	var err error
	*/
	// defer 它在函数返回前执行
	// 如果你用多个 defer，按照最后一个先执行的顺序来执行
	defer func() {
		if err != nil {
			// 如果 err 有值，说明发生错误，可以顺带改 result
			result = "something went wrong"
		}
	}() // 匿名方法直接调用
	if success {
		result = "OK: Operation succeeded"
		return // 相当于 return result, err
	}
	err = fmt.Errorf("oops! Something failed")
	return // 相当于return name, err
}

// 递归  一个函数自己调用自己，并且在合适的时机停下来返回结果
func factorial(n int) int {
	// 阶乘 5! = 5*4*3*2*1
	if n <= 1 {
		return n
	}
	// 大问题：n! = n * (n-1)!
	// 5 * factorial(4) → 4 * factorial(3) → 3 * factorial(2) → 2 * factorial(1) → 1
	// 终止点：n == 1 从最深的 1 一层层往外乘回来
	return n * factorial(n-1)
}

// 尾递归 在函数的最后一步，只做一次递归调用，后面没有额外运算，当前函数调用完成后，不需要保留当前栈帧（调用完立即返回）
/**
在Go 里尾递归是没有意义的，Go 的编译器（gc）默认不会把尾递归转成循环，所以在 Go 里，尾递归写得再优雅，
也照样一层层调用栈跑，不节省内存，不变成循环，想省得手动改写成 for
*/

// 函数式编程：用函数把一切行为当作值来传递，
// 闭包: 函数里套函数，里层函数能用到外层函数的局部变量，并且这些变量不会被销毁！
// 闭包 = 堆内存的保存：外层函数的局部变量，正常是在栈上分配的。当它被闭包捕获后，Go 编译器会把这块变量提升到堆上，让闭包可以在外层函数返回后继续访问
// 重点：闭包会引起内存泄漏，如果闭包捕获了一个大对象/敏感资源（大切片、连接、文件句柄），又长时间没释放，就会造成内存一直占用
// 最佳实践：闭包本身没啥魔法，也不会自己泄漏。只要外部变量持有它，它就跟着活着；没人用了，就让 GC 干活。挂在全局的，自己记得 = nil 或删除容器引用，就是最佳实践
func createCounter() func() int {
	counter := 0 // 局部变量，外面访问不到
	// 返回的匿名函数形成闭包，记住了 count
	return func() int {
		counter++
		return counter
	}
}

// 不定参数：类似python的args,在go中只能放在最后一个参数，在方法内部可以当作切片来使用
func printAll(prefix string, items ...string) {
	if len(items) == 0 {
		fmt.Println(prefix, "(nothing to print)")
	}
	for _, item := range items {
		fmt.Println(prefix, item)
	}
}

// defer 的参数值什么时候确定，跟闭包什么时候确定
// 1 defer 语句的参数是直接传入的 -> 在执行 defer 定义那一行时就把值拷贝了
// 2 defer 语句里用的是闭包引用外部变量 -> 在真正执行 defer 对应函数时才去读当前值
func deepDefer1() int {
	x := 10
	// println 本身就是函数，调用就是传参,传入参数 ("A:", x),所有x 的值在这一行就被拷贝了（当时是 10）
	// 不会等到 defer 执行时再去读外面的 x
	defer fmt.Println("deepDefer1 A:", x)
	defer func(v int) {
		fmt.Println("deepDefer1 B:", v)
	}(x)
	x = 20
	return x
}
func deepDefer2() int {
	x := 10
	// 参数是闭包引用，执行时才确定，里面用到的 x 是捕获的外部变量，等执行时才读值
	defer func() {
		fmt.Println("deepDefer2 B:", x)
	}()
	x = 20
	return x
}

// defer 修改返回值 关键点：命名返回值 vs 匿名返回值
// return执行流程： 先赋值 → 再执行 defer → 再返回
func deferReturn1() int {
	// 如果函数是 匿名返回值. 那么return x 时会先把x的值复制出来，存到一个临时返回槽里
	// 之后defer修改的是局部变量x，而不是那个临时槽，所以返回值不会变
	x := 10
	defer func() {
		x = 20
	}()
	return x
}

func deferReturn2() (x int) {
	// 如果函数有 命名返回值. 那么 return 返回的是那个 变量本身，所以 defer 里改这个变量能影响最终结果
	x = 10
	defer func() {
		x = 20
	}()
	return x
}

type TestStruct struct {
	Name string
}

func deferReturn3() *TestStruct {
	// 会把 val 的值（其实是一个指针地址）复制到返回槽里，此时返回槽里保存的是：指针地址
	val := &TestStruct{
		Name: "test",
	}
	defer func() {
		// 也就是通过指针找到堆上的结构体，把它的字段改了，这一步 没有改变返回槽里的指针地址，而是改了地址指向的对象内容
		val.Name = "修改了val指向的结构体"
	}()
	return val // 返回槽里依旧是val指针地址，但这个地址指向的对象内容已经被改了
}

func main() {
	res, _ := doSomething(false)
	fmt.Println("带名字的返回值:", res)
	fmt.Println("递归-阶乘:", factorial(5))

	// 函数可以作为返回值返回给外部
	c := createCounter()
	// 多次调用，发现状态一直记得，counter 这个变量的值是存在堆上,所以每次 c() 都改的是同一块内存
	fmt.Println("函数可以作为返回值返回给外部:", c()) // 1
	fmt.Println("函数可以作为返回值返回给外部:", c()) // 2
	fmt.Println("函数可以作为返回值返回给外部:", c()) // 3

	// 不定参数
	printAll("[Info]", "A", "B", "C")
	fmt.Println("defer 传值，定义就拍板", deepDefer1())
	fmt.Println("defer 闭包，执行才看数", deepDefer2())
	fmt.Println("defer 闭包，匿名返回值", deferReturn1()) // 10
	fmt.Println("defer 闭包，命名返回值", deferReturn2()) // 20
	fmt.Println("defer 闭包，匿名返回值 + 引用类型（指针、slice、map、chan 等）", deferReturn3())
}
