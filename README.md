# 极客时间Go语言并发编程实战笔记

## 基本并发原语

### 同步原语的适用场景

1. 共享资源。并发地读写共享资源，会出现数据竞争的问题，所以需要Mutex、RWMutex这样的并发原语来保护
2. 任务编排。需要goroutine按照一定的规律执行，而goroutine之间有相互等待或者依赖的顺序关系，常常使用WaitGroup或者Channel来实现
3. 消息传递。信息交流以及不同的goroutine之间的线程安全的数据交流，常常使用Channel来实现

### 第一节：mutex：解决资源并发访问问题

- 使用`race detector`检测并发访问共享资源是否有问题，Google基于C/C++的`sanitizers`技术实现 ，编译器通过探测所有的内存访问，加入代码能监视对这些内存地址的访问（读/写），在运行代码的时候，此工具能监控到对共享变量的非同步访问，出现race的时候，就会打印出警告信息
- 可以在编译、测试或者运行的时候加入race参数：`go run -race main.go`
- 缺点：无法在编译时检查出race问题,在运行时加入指令才能检测出来
- Mutex的零值是还没有goroutine等待的未加锁的状态，所以不需要额外的初始化

### 第二节：mutex：庖丁解牛看实现

> ![四大阶段](/assets/IMG_0056.JPG)

- 第一个阶段：先来先得
  - 请求锁的goroutine会排队等候获取互斥锁。虽然貌似公平，但是从性能上来说，并不是最优的，因为如果能够把锁交给正在占用CPU时间片的goroutine的话，就不需要做上下文的额切换
  ![初版](/assets/IMG_0057.JPG)
- 第二个阶段：给新人机会
  - 新来的goroutine也有机会先获得锁，甚至一个goroutine可能连续获取到锁，打破了先来先得的逻辑
  - 请求锁的goroutine有两类，一类是新来得请求锁的goroutine，另一类是被唤醒的等待请求锁的goroutine
  - ![](/assets/IMG_0059.JPG)
- 第三个阶段：多给些机会
  - 如果新来的goroutine或者是被唤醒的goroutine首次获取不到锁，就会通过自旋，尝试检查锁是否被释放。在尝试一定自旋次数后，再执行原来的逻辑
  - 对于临界区代码执行非常短的场景来说，这是一个非常好的优化。因为临界区的代码耗时很短，锁很快就能释放，而抢夺锁的goroutine不用通过休眠唤醒方式等待调度，直接spin几次，可能就获得了锁
- 第四个阶段：解决饥饿（等待中的goroutine可能会一直获取不到锁）
  - **加入饥饿模式，可以避免把机会全部留给新来的goroutine，保证请求锁的goroutine获取锁的公平性；mutex绝不容忍一个goroutine被落下，永远没有机会获取锁，尽可能让等待较长的goroutine更有机会获取到锁；**
  - 新来得goroutine参与竞争，有可能每次都会被新来的goroutine抢到获取锁的机会，在极端情况下，等待中的goroutine可能会一直获取不到锁，这就是饥饿问题
  - 增加了饥饿模式，将饥饿模式的最大等待时间阈值设置成了1毫秒，意味着一旦等待者等待的时间超过了这个阈值，mutex的处理就可能进入饥饿模式，优先让等待者先获取到锁
  - 通过加入饥饿模式，可以避免把机会全部留给新来的goroutine，保证了请求锁的goroutine获取锁公平性
  - 正常模式下，waiter都是进入先进先出队列，被唤醒的waiter并不会直接持有锁，而是要和新来的goroutine进行竞争。新来的goroutine有先天的优势，他们正在CPU中运行，可能数量还不少，所以，在高并发情况下，被唤醒的waiter可能比较悲剧的获取不到锁，这是会被插入到队列的前面。如果waiter获取不到锁的时间超过阈值1ms，那么mutex就进入了饥饿模式
  - 饥饿模式下，mutex的拥有者将直接把锁交给队列最前面的waiter。新来的goroutine不会尝试获取锁，即使看起来锁没有被持有，他也不会去抢，也不会spin，而是加入等待队列的尾部
  - 转入正常模式：
    - 此waiter已经是队列中的最后一个waiter了，没有其他的等待锁的goroutine了
    - 此waiter的等待时间小于1ms
  
  ![](/assets/IMG_0060.JPG)
- 主要是正常模式（性能更好）+饥饿模式（是一种公平性和性能的一种平衡，优先对待的是那些一直在等待的waiter）
- 如果新来的goroutine或者是被唤醒的

### 第三节：mutex：4种易错场景大盘点

1. Lock/Unlock不是成对出现
2. Copy已使用的mutex
   - Package sync的同步原语在使用后是不能复制的
   - mutex是一个有状态的对象，他的state字段记录这个锁的状态
   - 使用vet工具检测是否出现复制的
3. 重入
   - 不支持可重入锁（递归锁）
   - 可重入锁（递归锁）解决了代码冲入或者递归调用带来的死锁问题，同时也可以要求只有持有锁的goroutine才能unlock这个锁
   - 实现可重入锁
     - 获取goroutine id

       ```go
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
       ```
     - token
     
       ```go
       /*
       - Go开发者不希望利用goroutine id做一些不确定的东西，所以没有暴露获取goroutine id的方法
       - 调用者自己提供一个token，获取所得时候把这个token传入，释放所得时候也需要把这个token传入，通过用户传入的token替换方案1中goroutine id，其他逻辑和方案一致
       */
       import (
       	"sync"
       	"sync/atomic"
       )
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
       ```
4. 死锁

### 第四节：mutex：如何拓展额外功能
1. 如果互斥锁被某个goroutine获取了， 而且没有被释放，阿么其他请求这把锁的goroutine就会阻塞等待
2. 有些情况下，获取不到锁并不需要一直等待，一直等待会导致业务处理能力下降
3. **锁是性能下降的“罪魁祸首”之一，有效的降低锁的竞争，就能够很好的提高性能。**
4. TryLock
   - 尝试获取锁，当一个goroutine调用这个TryLock方法请求锁的时候，如果这把锁没有被其他goroutine持有，这个goroutine就持有了这把锁，并返回true
   - 如果这把锁已经被其他goroutine持有，或者是正在准备交给某个唤醒的goroutine，那么请求锁的goroutine就直接返回false，不会阻塞在方法调用上

