package main

import (
	"fmt"
	"sync"
)

// main1 未使用并发原语，会出现竞争的问题
func main1() {
	count := 0
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				count++
			}
		}()
	}
	wg.Wait() //等待10个goroutine完成
	fmt.Println(count)
}

// main2 使用Mutex，可以解决并发问题
func main2() {
	count := 0
	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				mu.Lock()
				count++
				mu.Unlock()
			}
		}()
	}
	wg.Wait() //等待10个goroutine完成
	fmt.Println(count)
}

// main3 进一步封装
func main3() {
	var count counter

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				count.Lock()
				count.count++
				count.Unlock()
			}
		}()
	}
	wg.Wait() //等待10个goroutine完成
	fmt.Println(count.count)
}

type counter struct {
	sync.Mutex //这里可以匿名添加，然后可以直接使用
	count      uint64
}

// Counter 线程安全的计数器类型
type Counter struct {
	//counterType int
	name  string
	mu    sync.Mutex
	count uint64
}

// incr 加1 的方法，内部使用互斥锁保护
func (c *Counter) incr() {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}

// Count 得到计数器的值，也需要锁保护
func (c *Counter) Count() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// main 最终实现版本
func main() {
	var counter Counter
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				counter.incr() //受到锁保护的方法
			}
		}()
	}
	wg.Wait()
	fmt.Println(counter.Count())

}
