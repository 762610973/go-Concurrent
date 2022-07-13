package main

import (
	"fmt"
	"sync"
)

//第一个阶段：先来先得，性能不是最优的，如果能够把锁交给正在占用CPU时间片的goroutine的话，就不需要做上下文的额切换
//第二个节点：给新人机会，新来的goroutine也有机会先获得锁，甚至一个goroutine可能连续获取到锁，打破了先来先得的逻辑
//第三个阶段：多给些机会，如果新来的goroutine或者是被环形的goroutine首次获取不到锁，就会通过一定次数的自旋
//第四个阶段：解决饥饿（等待中的goroutine可能会一直获取不到锁）：加入饥饿模式，可以避免把机会全部留给新来的goroutine，保证请求锁的goroutine获取锁的公平性
//mutex绝不容忍一个goroutine被落下，永远没有机会获取锁，尽可能让等待较长的goroutine更有机会获取到锁
//正常模式+饥饿模式
func main5() {
	var wg sync.WaitGroup
	wg.Add(2)
	var mu sync.Mutex
	x := int64(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		x = 3
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		mu.Lock()
		x = 1000
		mu.Unlock()
	}()
	wg.Wait()
	println(x)
}
func main() {
	a := []int{1, 2, 3}
	b := a
	change2(&b)
	fmt.Println(a)
	fmt.Println(b)
	fmt.Printf("%p", a)
}
func change1(s []int) {
	s[0] = -1
	s = append(s, 4, 5)
}
func change2(s *[]int) {
	(*s)[0] = 0
	*s = append(*s, 7, 8)
}
