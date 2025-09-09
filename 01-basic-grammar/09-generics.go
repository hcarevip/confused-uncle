/*
 * File: 09-generics.go
 * Date: 2025/9/1 21:50
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import "fmt"

/**
在 Go 里，泛型的语法是通过 类型参数（type parameter） 实现的,让代码能处理多种类型，而不用为每个类型单独写一遍
1 同一份逻辑可以应用到多种类型
2 编译期检查，而不是运行时报错
3 性能更好：避免反射，编译器会为实际类型生成对应的代码
*/

// Max 泛型函数：返回更大的那个
func Max[T ~int | ~int64 | ~float64 | ~string](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Box 泛型类型
type Box[T any] struct {
	V T
}

func (b *Box[T]) Set(v T) T {
	b.V = v
	return b.V
}

// Map 把 []A 映射成 []B
func Map[A any, B any](in []A, f func(A) B) []B {
	out := make([]B, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// Filter  按谓词保留元素
func Filter[T any](in []T, pred func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce  聚合
func Reduce[T any, R any](in []T, init R, f func(R, T) R) R {
	acc := init
	for _, v := range in {
		acc = f(acc, v)
	}
	return acc
}

type Stack[T any] struct {
	data []T
}

func (s *Stack[T]) Push(v T) {
	s.data = append(s.data, v)
}
func (s *Stack[T]) Insert(index int, value T) {
	// index 必须在 [0, len(s.data)] 范围内
	if index < 0 || index > len(s.data) {
		panic("index out of range")
	}
	// 假设：[]int{10, 20, 30}  index = 1
	// s.data[:index] =  前半部分[10]   s.data[index:] = 后半部分 [20, 30]
	// 先把value + 后半部分 append([]T{value}, s.data[index:]...) 拼起来 = append([99], [20, 30]...) = [99, 20, 30]
	// 再把前半部分拼上去 append(s.data[:index], 上一步结果...) = append([10], [99, 20, 30]...) = [10, 99, 20, 30]
	s.data = append(s.data[:index], append([]T{value}, s.data[index:]...)...)
}
func (s *Stack[T]) Pop() T {
	if len(s.data) == 0 {
		panic("stack is empty")
	}
	//(*s).data = s.data[:len(s.data)-1]
	v := s.data[len(s.data)-1] // 取最后一个元素
	s.data = s.data[:len(s.data)-1]
	return v
}
func (s *Stack[T]) Len() int {
	return len(s.data)
}

type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}
func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }
func (s Set[T]) Delete(v T)   { delete(s, v) }

type Cache[K comparable, V any] struct {
	m map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{m: make(map[K]V)}
}
func (c *Cache[K, V]) Get(k K) (V, bool) { v, ok := c.m[k]; return v, ok }
func (c *Cache[K, V]) Put(k K, v V)      { c.m[k] = v }

type Addable interface {
	// 除了方法，还可以写 类型集（type set）.用途是作为 类型约束（constraint），用在泛型类型参数声明里
	~int | ~int64 | ~float32 | ~float64 | ~string
}

func Sum[T Addable](xs ...T) T {
	var z T
	for _, v := range xs {
		z += v
	}
	return z
}

type ReaderLike interface{ Read([]byte) (int, error) }

func ReadN[T ReaderLike](r T, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := r.Read(buf)
	return buf, err
}
func main() {
	// 核心概念
	fmt.Println(Max(3, 5)) // 类型推断 -> int
	fmt.Println(Max(3.14, 2.71))
	fmt.Println(Max("go", "golang")) // -> string

	b1 := Box[int]{V: 1}
	b1.Set(10)
	fmt.Println(b1)

	b2 := Box[string]{V: "php"}
	b2.Set("golang")
	fmt.Println(b2.V)

	// 约束（constraints）与类型集（type set）
	// 编译器知道 T 能做什么，从而编译期检查而不是运行时报错
	/*
		(1) any：interface{} 的别名，表示无限制
		(2) comparable：可做 ==、!= 的类型（如map 的 key 就要求这个）
		(3) 联合：| 列出允许的具体类型或近似
		(4) ~（近似符）：表示底层类型为某个类型（含别名/自定义类型）例如 ~int 包含 type MyInt int
			如 int | int64 不包含 type MyInt int；写 ~int 才包含
	*/
	xs := []int{1, 2, 3, 4, 5}
	evens := Filter(xs, func(x int) bool { return x%2 == 0 })
	squares := Map(evens, func(x int) int { return x * x })
	sum := Reduce(squares, 0, func(acc, x int) int { return acc + x })
	fmt.Println(evens, squares, sum)

	// 泛型类型：Stack
	stack := Stack[int]{data: []int{1, 2, 3, 4, 5}}
	stack.Push(6)
	fmt.Printf("Push %#v len=%d\n", stack, stack.Len())

	stack.Insert(2, 20)
	fmt.Printf("Insert %#v len=%d\n", stack, stack.Len())
	stack.Pop()
	fmt.Printf("Pop %#v len=%d\n", stack, stack.Len())

	// 泛型类型：Set
	set := NewSet[string]()
	set.Add("1")
	set.Add("2")
	fmt.Printf("%#v\n", set)
	fmt.Println(set.Has("2"))
	set.Delete("1")

	// 泛型类型：Cache
	cache := NewCache[int, int]()
	cache.Put(1, 2)
	fmt.Printf("%#v\n", cache)
	v, ok := cache.Get(1)
	fmt.Printf("%#v  %t\n", v, ok) // 2 true

	// 进阶约束技巧与可维护性
	res := Sum(1, 2, 3)
	fmt.Println(res)
}
