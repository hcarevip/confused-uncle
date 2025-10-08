/*
 * File: study-test.go
 * Date: 2025/10/1 18:07
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func main() {
	s := fmt.Sprintf("%.10f", 100.88666601000)
	fmt.Println(strings.TrimRight(s, "0"))
	s = strings.TrimRight("1888.", ".")
	fmt.Println(s)

	fmt.Println(utf8.RuneCountInString("你好200"))

	text := "car@gmail.com"
	at := strings.IndexByte(text, '@')
	name := text[:at]
	domain := text[at:]
	var mid = ""
	if len(name) > 2 {
		mid = strings.Repeat("*", len(name)-2)
	}
	fmt.Println(name[:1] + mid + name[len(name)-1:] + domain)
}
