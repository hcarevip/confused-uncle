/*
 * File: 01-hello.go
 * Date: 2025/6/25 21:25
 * Author: 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import "fmt"

/*
*
1 main 方法必须要再mian包里
2 同一个目录下go package声明一致,但是同一个目录下go 文件和 test.go 声明可以不一致(如go 文件的package 交demo那么测试文件可以声明为 demo_test)
*/
func main() {
	fmt.Println("Hello World!")
	fmt.Println("God help those who stand in our way!")
}
