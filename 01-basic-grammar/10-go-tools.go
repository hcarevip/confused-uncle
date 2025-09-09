/*
 * File: 10-go-tools.go
 * Date: 2025/9/8 17:02
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package main

import (
	"container/heap"
	"fmt"
)

// Ordered 常用约束
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
	~float32 | ~float64 | ~string
}

// MapTool 切片辅助方法（删除/查找/过滤/Map/Reduce/集合运算）
// 把传过来挨个处理一遍，处理成什么由你给的函数说了算，处理完装进一个新的切片里
func MapTool[A any, B any](in []A, f func(A) B) []B {
	out := make([]B, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// FilterTool 判断某个元素是否满足某个条件的函数保留元素。返回新切片。
func FilterTool[T any](in []T, pred func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// ReduceTool 把切片里的元素一个个累计起来，得到最终结果
func ReduceTool[T any, R any](in []T, init R, f func(R, T) R) R {
	acc := init
	for _, v := range in {
		acc = f(acc, v)
	}
	return acc
}

// IndexOf 返回首个等于 target 的下标，未找到返回 -1。
// 需要 T 可比较（==）
func IndexOf[T comparable](in []T, target T) int {
	for i, v := range in {
		if v == target {
			return i
		}
	}
	return -1
}

// Contains 是否包含 target。
func Contains[T comparable](in []T, target T) bool { return IndexOf(in, target) >= 0 }

// DeleteAt 删除下标 index 的元素（越界 panic）。返回新切片。
func DeleteAt[T any](in []T, index int) []T {
	if index < 0 || index >= len(in) {
		panic("DeleteAt: index out of range")
	}
	// [0,index) + (index,len)
	return append(in[:index], in[index+1:]...)
}

// DeleteRange 删除 [from, to) 半开区间。返回新切片。
func DeleteRange[T any](in []T, from, to int) []T {
	// []int{1, 2, 3, 4, 5}  2  4
	if from < 0 || to > len(in) || from > to {
		panic("DeleteRange: invalid range")
	}
	return append(in[:from], in[to:]...)
}

// DeleteFunc 删除所有满足谓词的元素。返回新切片（原顺序保持）
func DeleteFunc[T any](in []T, pred func(T) bool) []T {
	// 创建一个 空切片（长度为 0），准备往里装要保留的元素, 长度清零，但不丢掉底层数组，相当于清空切片，准备重用
	// 但因为它底层还是用的 in 的数组，所以 append 时直接覆盖原来的数据，避免重新分配内存
	keep := in[:0]
	for _, v := range in {
		if !pred(v) {
			keep = append(keep, v)
		}
	}
	return keep
}

// Distinct 给一个切片，返回一个新切片，里面的元素不重复，顺序和第一次出现的一致，需要可比较
func Distinct[T comparable](in []T) []T {
	// map[T]struct{}：key 是元素，value 用空结构体 struct{}（零字节，节省内存）用来存已经出现过的元素
	seen := make(map[T]struct{}, len(in))
	// 准备一个结果切片 out，初始长度 0，容量设为输入长度
	out := make([]T, 0, len(in))
	for _, v := range in {
		// 如果 v 还没有出现在 seen 里
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			// 把这个新值追加到结果切片
			out = append(out, v)
		}
	}
	return out
}

// --- 集合运算（基于 comparable） ---

// Union 并集（去重）。
func Union[T comparable](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	seen := make(map[T]struct{}, len(a)+len(b))
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	for _, v := range b {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Intersect 交集（不重复）。
func Intersect[T comparable](a, b []T) []T {
	// 交换，让 a 始终是较短的那个，减少遍历次数（后面只遍历 a）
	if len(a) > len(b) {
		a, b = b, a
	}
	// sb 是b的成员表：判断 v 是否在 b 里
	sb := make(map[T]struct{}, len(b))
	for _, v := range b {
		sb[v] = struct{}{}
	}
	// sb: map[2:{} 3:{} 4:{} 5:{} 6:{}]
	//  a = []int{1, 2, 3, 5
	out := make([]T, 0, len(a))
	seen := make(map[T]struct{}, len(a))
	for _, v := range a {
		if _, ok := sb[v]; ok {
			if _, dup := seen[v]; !dup {
				seen[v] = struct{}{}
				out = append(out, v)
			}
		}
	}
	return out
}

// Difference 差集 a-b（出现在 a、不在 b）。
func Difference[T comparable](a, b []T) []T {
	sb := make(map[T]struct{}, len(b))
	for _, v := range b {
		sb[v] = struct{}{}
	}
	out := make([]T, 0, len(a))
	for _, v := range a {
		if _, ok := sb[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

// 栈

// StackTool 用切片实现的后进先出（LIFO）栈。
type StackTool[T any] struct {
	data []T
}

// StackToolPush 压栈（尾部追加）
func (s *StackTool[T]) StackToolPush(v T) { s.data = append(s.data, v) }

// StackToolPop 弹栈。空栈 panic。
func (s *StackTool[T]) StackToolPop() T {
	if len(s.data) == 0 {
		panic("StackToolPop: empty stack")
	}
	i := len(s.data) - 1
	v := s.data[i]
	s.data = s.data[:i]
	return v
}

// StackToolPeek 查看栈顶（不弹出）。空栈返回零值和 false。
func (s *StackTool[T]) StackToolPeek() (T, bool) {
	if len(s.data) == 0 {
		var z T
		return z, false
	}
	return s.data[len(s.data)-1], true
}

// StackToolLen 当前元素个数。
func (s *StackTool[T]) StackToolLen() int { return len(s.data) }

// StackToolInsert 在任意位置插入。index ∈ [0,len]，否则 panic。
// 等于 len 相当于 Push。
func (s *StackTool[T]) StackToolInsert(index int, value T) {
	if index < 0 || index > len(s.data) {
		panic("StackToolLen: index out of range")
	}
	s.data = append(s.data[:index], append([]T{value}, s.data[index:]...)...)
}

// StackToolDeleteAt 删除任意位置元素（保持顺序）。越界 panic。
func (s *StackTool[T]) StackToolDeleteAt(index int) {
	if index < 0 || index >= len(s.data) {
		panic("StackToolDeleteAt: index out of range")
	}
	s.data = append(s.data[:index], s.data[index+1:]...)
}

// 动态数组版 List

// ArrayList ：基于切片的顺序表。
type ArrayList[T any] struct {
	data []T
}

// NewArrayList 以容量 cap 创建。
func NewArrayList[T any](capHint int) *ArrayList[T] {
	if capHint < 0 {
		capHint = 0
	}
	return &ArrayList[T]{data: make([]T, 0, capHint)}
}

func (a *ArrayList[T]) Len() int { return len(a.data) }

// Get 读取 index 处元素。越界 panic。
func (a *ArrayList[T]) Get(index int) T {
	if index < 0 || index >= len(a.data) {
		panic("Get: index out of range")
	}
	return a.data[index]
}

// Set 设置 index 处元素为 v。越界 panic。
func (a *ArrayList[T]) Set(index int, v T) {
	if index < 0 || index >= len(a.data) {
		panic("Set: index out of range")
	}
	a.data[index] = v
}

// Append 末尾追加。
func (a *ArrayList[T]) Append(v T) { a.data = append(a.data, v) }

// Insert 在 index 插入 v。index ∈ [0,len]。
func (a *ArrayList[T]) Insert(index int, v T) {
	if index < 0 || index > len(a.data) {
		panic("Insert: index out of range")
	}
	a.data = append(a.data[:index], append([]T{v}, a.data[index:]...)...)
}

// Remove 删除 index 处元素（保持顺序）。
func (a *ArrayList[T]) Remove(index int) {
	if index < 0 || index >= len(a.data) {
		panic("Remove: index out of range")
	}
	a.data = append(a.data[:index], a.data[index+1:]...)
}

// Slice 返回底层数据的只读视图（注意：修改返回值会影响自身）。
func (a *ArrayList[T]) Slice() []T { return a.data }

// 双向链表版 List

// DoublyLinkedList 双向链表。支持头尾插入/删除及任意位置遍历。
type DoublyLinkedList[T any] struct {
	head *dlNode[T] // 第一个节点
	tail *dlNode[T] // 最后一个节点
	len  int
}
type dlNode[T any] struct {
	val  T          // 当前存储的元素值
	prev *dlNode[T] // 指向链表里的上一个节点（如果是头结点，就是 nil）  head.prev 永远是 nil
	next *dlNode[T] // 指向链表里的下一个节点（如果是尾结点，就是 nil）  tail.next 永远是 nil
}

func (l *DoublyLinkedList[T]) Len() int { return l.len }

// PushFront 头插。
func (l *DoublyLinkedList[T]) PushFront(v T) {
	// 申请新节点，prev/next 默认 nil
	n := &dlNode[T]{val: v}
	if l.head == nil {
		l.head, l.tail = n, n
	} else {
		n.next = l.head // 新节点的 next 指向原 head
		l.head.prev = n // 原 head 的 prev 回指向新节点
		l.head = n      // 更新 head 为新节点
	}
	l.len++
}

// PushBack 尾插。
func (l *DoublyLinkedList[T]) PushBack(v T) {
	n := &dlNode[T]{val: v}
	if l.tail == nil {
		l.head, l.tail = n, n
	} else {
		n.prev = l.tail
		l.tail.next = n
		l.tail = n
	}
	l.len++
}

// PopFront 头删。
func (l *DoublyLinkedList[T]) PopFront() (T, bool) {
	// 空表不能删除；返回类型的零值 z + false
	if l.head == nil {
		var z T
		return z, false
	}
	// 把当前头节点保存到局部变量 n
	n := l.head
	l.head = n.next // 把 l.head 指向下一个节点
	if l.head != nil {
		l.head.prev = nil
	} else {
		// 当删除后变空表（原来只有 1 个）把 tail 也置为 nil,空表状态下必须 head==nil && tail==nil
		l.tail = nil
	}
	l.len--
	return n.val, true
}

// PopBack 尾删。
func (l *DoublyLinkedList[T]) PopBack() (T, bool) {
	if l.tail == nil {
		var z T
		return z, false
	}
	n := l.tail
	l.tail = n.prev
	if l.tail != nil {
		l.tail.next = nil
	} else {
		l.head = nil
	}
	l.len--
	return n.val, true
}

// Iterate 便利迭代（从前到后）。
func (l *DoublyLinkedList[T]) Iterate(fn func(T) bool) {
	for p := l.head; p != nil; p = p.next {
		if !fn(p.val) {
			return
		}
	}
}

// 环形数组队列（O(1) 入队/出队）
/*
	普通队列的问题
	队列（FIFO，先进先出）：像排队，先来的人先服务
	如果用普通数组实现：
		入队（append）很快。
		出队（删除第一个元素）就很慢，因为数组要整体移动
	环形数组队列: 不用每次移动数组，而是让数组首尾相连，形成一个 环
		维护两个指针
			head：下一个要出队的位置
			tail：下一个要入队的位置
			当tail到达数组末尾时，不用扩容移动，直接回绕到 0 位置
*/

// RingQueue 基于环形缓冲区的 FIFO 队列。实现 O(1) 入队/出队
type RingQueue[T any] struct {
	buf        []T // 固定容量数组
	head, tail int // 出队/入队指针
	size       int // 当前元素数
}

// NewRingQueue capacity 必须 > 0。
func NewRingQueue[T any](capacity int) *RingQueue[T] {
	if capacity <= 0 {
		panic("NewRingQueue: capacity must be > 0")
	}
	return &RingQueue[T]{buf: make([]T, capacity)}
}

func (q *RingQueue[T]) Len() int    { return q.size }
func (q *RingQueue[T]) Cap() int    { return len(q.buf) }
func (q *RingQueue[T]) Empty() bool { return q.size == 0 }
func (q *RingQueue[T]) Full() bool  { return q.size == len(q.buf) }

// Enqueue 入队；满时自动扩容为 2 倍容量。
func (q *RingQueue[T]) Enqueue(v T) {
	if q.Full() {
		q.grow()
	}
	q.buf[q.tail] = v
	q.tail = (q.tail + 1) % len(q.buf)
	q.size++
}

// Dequeue 出队。空时返回零值和 false。
func (q *RingQueue[T]) Dequeue() (T, bool) {
	if q.Empty() {
		var z T
		return z, false
	}
	v := q.buf[q.head]
	var z T
	q.buf[q.head] = z
	// 让 head 往后移动一格，指向下一个要出队的位置, 保证 head 永远在 0 ~ len(buf)-1 之间循环
	q.head = (q.head + 1) % len(q.buf)
	q.size--
	return v, true
}

func (q *RingQueue[T]) grow() {
	newBuf := make([]T, len(q.buf)*2)
	// 拷贝 head..end 与 0..tail-1 的两段
	for i := 0; i < q.size; i++ {
		newBuf[i] = q.buf[(q.head+i)%len(q.buf)]
	}
	q.buf = newBuf
	q.head = 0
	q.tail = q.size
}

// 单向链表 实现的队列（FIFO）
/*
	如果用数组实现队列，最简单的做法是
		入队：直接 append 到末尾  O(1)
		出队：取下标 0 的元素，然后把整个数组前移  O(n)
*/
type lqNode[T any] struct {
	val  T
	next *lqNode[T]
}

// LinkedQueue 基于单链表的队列。
type LinkedQueue[T any] struct {
	head, tail *lqNode[T]
	len        int
}

func (q *LinkedQueue[T]) Len() int    { return q.len }
func (q *LinkedQueue[T]) Empty() bool { return q.len == 0 }

// Enqueue 尾入。
func (q *LinkedQueue[T]) Enqueue(v T) {
	n := &lqNode[T]{val: v}
	if q.tail == nil {
		q.head, q.tail = n, n
	} else {
		q.tail.next = n
		q.tail = n
	}
	q.len++
}

// Dequeue 头出。
func (q *LinkedQueue[T]) Dequeue() (T, bool) {
	if q.head == nil {
		var z T
		return z, false
	}
	n := q.head
	q.head = n.next
	if q.head == nil {
		q.tail = nil
	}
	q.len--
	return n.val, true
}

// 优先级队列（可自定义比较器）
/*
	普通队列 (Queue)
		规则：FIFO（先入先出）
		入队：总是在队尾插入
		出队：总是在队头取出
		典型实现：环形数组、链表队列
	优先级队列 (Priority Queue)
		规则：谁的优先级更高，谁先出队
		入队：随便插（放堆里）
		出队：一定是当前优先级最高的元素
		实现方式：通常用 堆 (heap)，保证每次取出都是 O(log n)
*/

// LessFn 比较器：告诉队列：a 是否比 b 优先级低
// 你可通过传入不同 less 实现小顶堆/大顶堆。
type LessFn[T any] func(a, b T) bool

// pq 内部堆实现。
type pq[T any] struct {
	data []T
	less LessFn[T]
}

func (h *pq[T]) Len() int           { return len(h.data) }
func (h *pq[T]) Less(i, j int) bool { return h.less(h.data[i], h.data[j]) }
func (h *pq[T]) Swap(i, j int)      { h.data[i], h.data[j] = h.data[j], h.data[i] }
func (h *pq[T]) Push(x any)         { h.data = append(h.data, x.(T)) }
func (h *pq[T]) Pop() any           { n := len(h.data) - 1; v := h.data[n]; h.data = h.data[:n]; return v }

// PriorityQueue 封装 container/heap 的泛型优先队列。内部实际操作是 heap.Push、heap.Pop
type PriorityQueue[T any] struct{ h *pq[T] }

func NewPriorityQueue[T any](less LessFn[T]) *PriorityQueue[T] {
	h := &pq[T]{less: less}
	heap.Init(h) // 初始化建堆
	return &PriorityQueue[T]{h: h}
}

func (q *PriorityQueue[T]) Len() int { return q.h.Len() }

func (q *PriorityQueue[T]) Push(v T) {
	// 调用 heap.Push
	heap.Push(q.h, v)
}

// Pop 弹出最高优先级元素（按 less 判定）。
func (q *PriorityQueue[T]) Pop() (T, bool) {
	if q.Len() == 0 {
		var z T
		return z, false
	}
	// 调用 heap.Pop
	return heap.Pop(q.h).(T), true
}

// Peek 查看堆顶（不弹出）。
func (q *PriorityQueue[T]) Peek() (T, bool) {
	if q.Len() == 0 {
		var z T
		return z, false
	}
	// data[0] 始终是优先级最高的元素，是因为把pq交给了 container/heap，而它在 Push / Pop 时自动做了堆化（上浮、下沉）
	return q.h.data[0], true
}

// HashSet（基于 map）

// HashSet 基于 map 的集合。
type HashSet[T comparable] map[T]struct{}

func NewHashSet[T comparable]() HashSet[T] { return make(HashSet[T]) }

func (s HashSet[T]) Add(v T)      { s[v] = struct{}{} }
func (s HashSet[T]) Remove(v T)   { delete(s, v) }
func (s HashSet[T]) Has(v T) bool { _, ok := s[v]; return ok }
func (s HashSet[T]) Len() int     { return len(s) }

// ToSlice 导出为切片（遍历无序）。
func (s HashSet[T]) ToSlice() []T {
	out := make([]T, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	return out
}

// 有序集合（基于排序切片 + 二分查找）

// OrderedSet 保持升序且不重复。基于二分插入/查找。
type OrderedSet[T Ordered] struct {
	data []T
}

func NewOrderedSet[T Ordered]() *OrderedSet[T] { return &OrderedSet[T]{data: nil} }
func (s *OrderedSet[T]) Len() int              { return len(s.data) }

// Has 是否包含。
func (s *OrderedSet[T]) Has(v T) bool {
	_, ok := s.search(v)
	return ok
}

// Add 插入：若已存在则忽略。
func (s *OrderedSet[T]) Add(v T) {
	// 找到插入位置 i，ok 表示是否已存在
	i, ok := s.search(v)
	if ok {
		return
	}
	s.data = append(s.data, v)     //  先扩容一个位置
	copy(s.data[i+1:], s.data[i:]) // 在数组中空出插入位置，把后面的元素往右移动一格，避免覆盖，保持有序
	s.data[i] = v                  // 在位置 i 插入新值
}

// Remove 删除：若不存在忽略。
func (s *OrderedSet[T]) Remove(v T) {
	i, ok := s.search(v)
	if !ok {
		return
	}
	s.data = append(s.data[:i], s.data[i+1:]...)
}

// ToSlice 导出（有序）
func (s *OrderedSet[T]) ToSlice() []T {
	out := make([]T, len(s.data))
	copy(out, s.data)
	return out
}

// search 二分：返回插入位置 i 及是否已存在。
func (s *OrderedSet[T]) search(v T) (i int, found bool) {
	// 每次比较中间元素，丢掉一半区间，快速定位目标
	lo, hi := 0, len(s.data) // lo 和 hi 是搜索的左右边界
	for lo < hi {
		mid := int(uint(lo+hi) >> 1) // 取中间位置 mid = (lo+hi)/2 写成 uint(lo+hi) >> 1 是一种避免整数溢出的写法
		if s.data[mid] < v {         // 目标在右半部分 → 舍弃左半部分，lo=mid+1
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(s.data) && s.data[lo] == v {
		return lo, true
	}
	return lo, false
}

// LinkedHashMap（按插入顺序迭代）

// LinkedHashMap 维护插入顺序的 Map。
// 用双向链表保存顺序，用 map 做 O(1) 查找。
type LinkedHashMap[K comparable, V any] struct {
	m    map[K]*lmNode[K, V]
	head *lmNode[K, V]
	tail *lmNode[K, V]
	len  int
}
type lmNode[K comparable, V any] struct {
	k    K
	v    V
	prev *lmNode[K, V]
	next *lmNode[K, V]
}

func NewLinkedHashMap[K comparable, V any]() *LinkedHashMap[K, V] {
	return &LinkedHashMap[K, V]{m: make(map[K]*lmNode[K, V])}
}

func (l *LinkedHashMap[K, V]) Len() int { return l.len }

// Set 插入或更新；新 key 追加到尾部。
func (l *LinkedHashMap[K, V]) Set(k K, v V) {
	if n, ok := l.m[k]; ok {
		n.v = v
		return
	}
	n := &lmNode[K, V]{k: k, v: v}
	l.m[k] = n
	if l.tail == nil {
		l.head, l.tail = n, n
	} else {
		n.prev = l.tail
		l.tail.next = n
		l.tail = n
	}
	l.len++
}

// Get 读取；存在返回值和 true。
func (l *LinkedHashMap[K, V]) Get(k K) (V, bool) {
	if n, ok := l.m[k]; ok {
		return n.v, true
	}
	var z V
	return z, false
}

// Delete 删除 key；不存在忽略。
func (l *LinkedHashMap[K, V]) Delete(k K) {
	n, ok := l.m[k]
	if !ok {
		return
	}
	delete(l.m, k)
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		l.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		l.tail = n.prev
	}
	l.len--
}

// Iterate 按插入顺序遍历（fn 返回 false 提前停止）。
func (l *LinkedHashMap[K, V]) Iterate(fn func(K, V) bool) {
	for p := l.head; p != nil; p = p.next {
		if !fn(p.k, p.v) {
			return
		}
	}
}

func main() {
	// 切片辅助方法
	// （1）把切片里的每个元素按规则逐个变换，形成一个等长的新切片
	mts := MapTool[int, int]([]int{1, 2, 3}, func(x int) int { return x * x })
	fmt.Println("MapTool: =>", mts) // MapTool: => [1 4 9]

	ftl := FilterTool[int]([]int{1, 2, 3}, func(x int) bool { return x%2 == 0 })
	fmt.Println("FilterTool: =>", ftl) // FilterTool: => [2]

	// 接收当前累计值和下一个元素，返回新的累计值
	rtl := ReduceTool[int, int]([]int{1, 2, 3}, 10, func(x int, acc int) int { return x + acc })
	fmt.Println("ReduceTool: =>", rtl) // ReduceTool: => 16

	// 查找
	findIndex := IndexOf[int]([]int{1, 2, 3}, 2)
	fmt.Println("IndexOf: =>", findIndex) // IndexOf: => 1

	// 是否包含
	isContains := Contains[int]([]int{1, 2, 3}, 20)
	fmt.Println("Contains: =>", isContains) // Contains: => fals

	// 根据索引删除元素,返回新的切片
	afterDeleteIndex := DeleteAt[int]([]int{1, 2, 3}, 1)
	fmt.Println("DeleteAt: =>", afterDeleteIndex) // DeleteAt: => [1 3]

	// 根据区间删除元素，返回新的切片
	afterDeleteRange := DeleteRange[int]([]int{1, 2, 3, 4, 5}, 2, 4)
	fmt.Println("DeleteRange: =>", afterDeleteRange) // DeleteRange: => [1 2 5]

	// DeleteFunc 会遍历切片，把所有符合条件的元素删掉，返回一个新切片，返回结果里的元素顺序和原切片保持一致
	afterDeleteFunc := DeleteFunc[string]([]string{"a", "b", "c"}, func(x string) bool { return x == "a" })
	fmt.Println("DeleteFunc: =>", afterDeleteFunc) // DeleteFunc: => [b c]

	// 去重返回新
	distinct := Distinct[int]([]int{1, 2, 3, 2, 3})
	fmt.Println("Distinct: =>", distinct) // Distinct: => [1 2 3]

	// --- 集合运算（基于 comparable） ---
	// Union 并集
	union := Union[int]([]int{1, 2, 3, 5}, []int{3, 4, 5, 2, 6})
	fmt.Println("Union: =>", union) // [1 2 3 5 4 6]

	//  Intersect 交集（不重复）
	intersect := Intersect[int]([]int{3, 4, 5, 2, 6}, []int{1, 2, 3, 5})
	fmt.Println("Intersect: =>", intersect) // Intersect: => [2 3 5]

	//  Difference 差集 a-b（出现在 a、不在 b）
	difference := Difference[int]([]int{3, 4, 5, 2, 6}, []int{1, 2, 3, 5})
	fmt.Println("Difference: =>", difference) // Difference: => [4 6]

	// 栈
	var stackTool = StackTool[int]{data: []int{1, 2, 3}}
	stackTool.StackToolPush(5)
	stackTool.StackToolPush(6)

	// 弹栈(先进后出)
	stackTool.StackToolPop()
	fmt.Println("StackTool: =>", stackTool) // StackTool: => {[1 2 3 5]}

	// StackToolPeek 查看栈顶（不弹出）
	peek, _ := stackTool.StackToolPeek()
	fmt.Println("Peek: =>", peek) // Peek: => 5

	// 在任意位置插入
	stackTool.StackToolInsert(4, 6)
	fmt.Println("StackTool: =>", stackTool) // StackTool: => {[1 2 3 5 6]}

	fmt.Println("StackToolLen: =>", stackTool.StackToolLen())
	// 删除任意位置元素（保持顺序）
	stackTool.StackToolDeleteAt(4)

	// 动态数组版 List
	arrayList := NewArrayList[int](5)
	fmt.Printf("ArrayList: => %v Len=%d\n", arrayList, arrayList.Len())
	// Append 末尾追加
	arrayList.Append(1)
	arrayList.Append(2)
	arrayList.Append(3)
	arrayList.Append(6)
	// 根据指定index获取元素
	fmt.Println("ArrayList: Get =>", arrayList.Get(0)) // ArrayList: Get => 1
	// Set 设置 index 处元素为 v
	arrayList.Set(3, 5)
	// Insert 在 index 插入 v
	arrayList.Insert(4, 6)
	arrayList.Insert(5, 7)
	// Remove 删除 index 处元素
	arrayList.Remove(5)
	// Slice 返回底层数据的只读视图（注意：修改返回值会影响自身）
	fmt.Println("ArrayList: Slice =>", arrayList.Slice()) // ArrayList: Slice => [1 2 3 5 6]

	//  DoublyLinkedList 双向链表。支持头尾插入/删除及任意位置遍历
	doublyLinkedList := DoublyLinkedList[int]{}
	doublyLinkedList.PushFront(1)
	doublyLinkedList.PushBack(2)
	doublyLinkedList.PushBack(3)
	doublyLinkedList.PushBack(6)
	doublyLinkedList.PushBack(9)
	fmt.Println("DoublyLinkedList: Len =>", doublyLinkedList.Len())

	// PopFront 头删 / 尾删
	doublyLinkedList.PopFront()
	doublyLinkedList.PopBack()

	// Iterate 迭代（从前到后）
	doublyLinkedList.Iterate(func(v int) bool {
		fmt.Print(v, " \n")
		return true
	})

	// 基于环形缓冲区的 FIFO 队列。实现 O(1) 入队/出队
	ringQueue := NewRingQueue[int](5)
	// RingQueue: => &{[0 0 0 0 0] 0 0 0} Len=0  Cen=5  Empty=true  Full=false
	fmt.Printf("RingQueue: => %v Len=%d  Cen=%d  Empty=%t  Full=%t\n",
		ringQueue, ringQueue.Len(), ringQueue.Cap(), ringQueue.Empty(), ringQueue.Full())

	// Enqueue 入队；满时自动扩容为 2 倍容量
	ringQueue.Enqueue(1)
	ringQueue.Enqueue(2)
	ringQueue.Enqueue(3)
	ringQueue.Enqueue(4)
	ringQueue.Enqueue(5)
	ringQueue.Enqueue(6)
	fmt.Printf("RingQueue: => %v Len=%d  Cen=%d  Empty=%t  Full=%t\n",
		ringQueue, ringQueue.Len(), ringQueue.Cap(), ringQueue.Empty(), ringQueue.Full())

	// Dequeue 出队。空时返回零值和 false
	dequeue, b := ringQueue.Dequeue()
	fmt.Println("RingQueue Dequeue: =>", dequeue, b) // RingQueue Dequeue: => 1 true

	// LinkedQueue单向链表队列
	// 单向链表队列就是典型的 FIFO（先进先出） 数据结构, (1) 永远在尾部插入 (2) 永远从头部取出
	linkedQueue := LinkedQueue[int]{}
	//  永远在尾部插入
	linkedQueue.Enqueue(1)
	linkedQueue.Enqueue(2)
	linkedQueue.Enqueue(3)
	// 永远从头部取出
	val, ok := linkedQueue.Dequeue()
	fmt.Println("LinkedQueue Dequeue: =>", val, ok) // LinkedQueue Dequeue: => 1 true

	// 优先级队列, 大顶堆：数字大的优先
	priorityQueue := NewPriorityQueue[int](func(a, b int) bool { return a > b })
	// 优先级队列, 小顶堆：数字小的优先
	// priorityQueue := NewPriorityQueue[int](func(a, b int) bool { return a < b })

	priorityQueue.Push(1)
	priorityQueue.Push(20)
	priorityQueue.Push(3)
	priorityQueue.Push(10)
	// Peek 查看堆顶（不弹出）
	priorityVal, _ := priorityQueue.Peek()
	fmt.Println("PriorityQueue Peek: =>", priorityVal) // PriorityQueue Peek: => 20
	// Pop 弹出最高优先级元素（按 less 判定）
	popVal, _ := priorityQueue.Pop()
	fmt.Println("PriorityQueue Pop: =>", popVal) // PriorityQueue Pop: => 20

	// HashSet（基于 map） HashSet 是一种集合结构，保证元素唯一,插入 / 删除 / 查询 平均时间复杂度：O(1)
	hashSet := NewHashSet[string]()
	hashSet.Add("1")
	hashSet.Add("2")
	hashSet.Add("3")
	hashSet.Add("3")
	fmt.Println("HashSet: =>", hashSet) // HashSet: => map[1:{} 2:{} 3:{}]
	fmt.Println("HashSetLen: =>", hashSet.Len())
	fmt.Println("HashSetHas: =>", hashSet.Has("1"))
	// ToSlice 导出为切片（遍历无序）
	fmt.Println("HashSet ToSlice: =>", hashSet.ToSlice()) // HashSet ToSlice: => [1 2 3]

	// 有序集合（基于排序切片 + 二分查找）
	// 集合有序 + 元素唯一，天然支持排序相关操作,快在有序遍历，但单次插入/删除慢
	orderedSet := NewOrderedSet[string]()
	orderedSet.Add("3")
	orderedSet.Add("2")
	orderedSet.Add("1")
	orderedSet.Add("5")

	fmt.Println("OrderSet Has: =>", orderedSet.Has("1"))
	orderedSet.Remove("5")
	fmt.Println("OrderSet ToSlice: =>", orderedSet.ToSlice()) // OrderSet ToSlice: => [1 2 3]
	fmt.Println("OrderSet Len: =>", orderedSet.Len())

	// LinkedHashMap 既能 O(1) 查找，又能 保留插入顺序.
	// LinkedHashMap 的特点是：有序（按插入顺序） 唯一（key 唯一） 高效（查找、插入、删除都是 O(1)）
	linkedHashMap := NewLinkedHashMap[string, int]()
	// Set 插入或更新；新 key 追加到尾部
	linkedHashMap.Set("name", 1)
	linkedHashMap.Set("age", 2)
	linkedHashMap.Set("city", 3)
	fmt.Println("LinkedHashMap Len: =>", linkedHashMap.Len())
	linkedHashMapVal, _ := linkedHashMap.Get("name")
	fmt.Println("LinkedHashMap Get: =>", linkedHashMapVal)
	linkedHashMap.Delete("none")

	// Iterate 按插入顺序遍历（fn 返回 false 提前停止）
	linkedHashMap.Iterate(func(k string, v int) bool {
		fmt.Println("Iterate: =>", k, v)
		return true
	})
}
