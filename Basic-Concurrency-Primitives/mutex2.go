package main

import (
	"fmt"
	"net/http"
	"unsafe"
)

//第一个阶段：先来先得，性能不是最优的，如果能够把锁交给正在占用CPU时间片的goroutine的话，就不需要做上下文的额切换
//第二个节点：给新人机会，新来的goroutine也有机会先获得锁，甚至一个goroutine可能连续获取到锁，打破了先来先得的逻辑
//第三个阶段：多给些机会，如果新来的goroutine或者是被环形的goroutine首次获取不到锁，就会通过一定次数的自旋
//第四个阶段：解决饥饿（等待中的goroutine可能会一直获取不到锁）：加入饥饿模式，可以避免把机会全部留给新来的goroutine，保证请求锁的goroutine获取锁的公平性
//mutex绝不容忍一个goroutine被落下，永远没有机会获取锁，尽可能让等待较长的goroutine更有机会获取到锁
//正常模式+饥饿模式

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

type nei struct {
	a int8
	b int16
	c int32
	d int64
}

type nei2 struct {
	d int64
	a int8
	c int32
	b int64 //这里不管是多少都是24个字节
}

type nei3 struct {
	d uint64
	a uint8
	c uint32
	b uint64 //这里不管是多少都是24个字节
}

func judge(a, b interface{}) {
	if a.(float64) < b.(float64) {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}
}

func judge1[T int | float64](a, b T) {
	if a < b {
		fmt.Println("1")
	} else {
		fmt.Println("2")
	}
}

func main() {
	judge(2.8, 6.5)
	judge1[int](2, 3)
	n1 := nei{
		a: 1,
		b: 2,
		c: 3,
		d: 4,
	}
	n2 := nei2{
		d: 1,
		a: 2,
		c: 3,
		b: 4,
	}
	n3 := nei3{
		d: 1,
		a: 2,
		c: 3,
		b: 4,
	}
	fmt.Println(unsafe.Sizeof(n1))
	fmt.Println(unsafe.Sizeof(n2))
	fmt.Println(unsafe.Sizeof(n3))
}
