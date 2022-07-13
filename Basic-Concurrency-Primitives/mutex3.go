package main

import (
	"fmt"
	"github.com/petermattis/goid"
	"sync"
	"sync/atomic"
)

type RecursiveMutex struct {
	sync.Mutex
	owner     int64 //记录当前锁的拥有者goroutine的id
	recursion int32 //辅助字段，记录重入的次数
}

func (m *RecursiveMutex) Lock() {
	gid := goid.Get()
	//如果当前持有锁的goroutine就是这次调用的goroutine，说明是重入
	if atomic.LoadInt64(&m.owner) == gid {
		m.recursion++
		return
	}
	m.Mutex.Lock()
	// 获得锁的goroutine第一次调用，记录下他的goroutine id，调用次数加1
	// 原子操作的存储过程，将gid赋值给owner
	atomic.StoreInt64(&m.owner, gid)
	m.recursion = 1
}
func (m *RecursiveMutex) Unlock() {
	gid := goid.Get()
	//非持有锁的goroutine尝试释放锁，错误的使用
	if atomic.LoadInt64(&m.owner) != gid {
		panic(fmt.Sprintf("wrong the owner(%d):%d!", m.owner, gid))
	}
	//调用次数减1
	m.recursion--
	if m.recursion != 0 {
		//如果这个goroutine还没有完全释放，则直接返回
		return
	}
	// 此goroutine最后一次调用，需要释放锁
	atomic.StoreInt64(&m.owner, -1)
	m.Mutex.Unlock()
}
func main() {
	var c count
	c.Lock()
	defer c.Unlock()
	c.c++
	foo(c)
	// 重入锁失败
}

// 没有成对出现是不行的
func double() {
	var mu sync.Mutex
	m := 10
	mu.Lock()
	//mu.Lock()导致死锁
	m++
	fmt.Println(m)
}

// 重入锁失败
func chongRu() {
	var m sync.Mutex
	m.Lock()
	m.Lock()
	m.Unlock()
	m.Unlock()
	// 重入锁失败
}

func foo(c count) {
	c.Lock()
	defer c.Unlock()
	fmt.Println("in foo")
}

type count struct {
	sync.Mutex
	c int
}

type TokenRecursiveMutex struct {
	sync.Mutex
	token     int64
	recursion int32
}

func (m *TokenRecursiveMutex) Lock(token int64) {
	if atomic.LoadInt64(&m.token) == token {
		m.recursion++
		return
	}
	m.Mutex.Lock() //传入的token不一致，说明不是递归调用
	//抢到锁之后记录这个token
	atomic.StoreInt64(&m.token, token)
	m.recursion = 1
}
func (m *TokenRecursiveMutex) Unlock(token int64) {
	if atomic.LoadInt64(&m.token) != token {
		panic(fmt.Sprintf("wrong the owner(%d):%d!", m.token, token))
	}

	m.recursion-- //当前持有这个锁的token释放锁
	if m.recursion != 0 {
		//如果这个goroutine还没有完全释放，则直接返回
		return
	}
	// 此goroutine最后一次调用，需要释放锁
	atomic.StoreInt64(&m.token, 0)
	m.Mutex.Unlock()
}
