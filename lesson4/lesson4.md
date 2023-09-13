# 锁

锁工具有

- sync.Mutex互斥锁
- sync.RWMutex读写锁
- sync.WaitGroup等待组
- sync.Once 只执行一次，用于初始化

锁的基础，原子操作和信号锁
## 原子操作

```go
atomic.AddInt32(p, 1)  
atomic.CompareAndSwapInt32(p,1,2)
```

## 信号量锁 sema

```go
type Mutex struct {  
    state int32  
    sema  uint32  //信号量锁
}
```

```go
runtime/sema.go

// Asynchronous semaphore for sync.Mutex.  
  
// A semaRoot holds a balanced tree of sudog with distinct addresses (s.elem).// Each of those sudog may in turn point (through s.waitlink) to a list// of other sudogs waiting on the same address.  
// The operations on the inner lists of sudogs with the same address  
// are all O(1). The scanning of the top-level semaRoot list is O(log n),// where n is the number of distinct addresses with goroutines blocked  
// on them that hash to the given semaRoot.  
// See golang.org/issue/17953 for a program that worked badly  
// before we introduced the second level of list, and  
// BenchmarkSemTable/OneAddrCollision/* for a benchmark that exercises this.  
type semaRoot struct {  
    lock  mutex  
    treap *sudog        // root of balanced tree of unique waiters.  
    nwait atomic.Uint32 // Number of waiters. Read w/o the lock.  //记录了等待协程的数量
}

// sudog represents a g in a wait list, such as for sending/receiving// on a channel.  
//  
// sudog is necessary because the g ↔ synchronization object relation// is many-to-many. A g can be on many wait lists, so there may be// many sudogs for one g; and many gs may be waiting on the same  
// synchronization object, so there may be many sudogs for one object.  
//  
// sudogs are allocated from a special pool. Use acquireSudog and// releaseSudog to allocate and free them.type sudog struct {  
    // The following fields are protected by the hchan.lock of the  
    // channel this sudog is blocking on. shrinkstack depends on    // this for sudogs involved in channel ops.  
    g *g  
  
    next *sudog  
    prev *sudog  
    elem unsafe.Pointer // data element (may point to stack)  
  
    // The following fields are never accessed concurrently.    // For channels, waitlink is only accessed by g.    // For semaphores, all fields (including the ones above)    // are only accessed when holding a semaRoot lock.  
    acquiretime int64  
    releasetime int64  
    ticket      uint32  
  
    // isSelect indicates g is participating in a select, so    // g.selectDone must be CAS'd to win the wake-up race.    isSelect bool  
  
    // success indicates whether communication over channel c  
    // succeeded. It is true if the goroutine was awoken because a  
    // value was delivered over channel c, and false if awoken  
    // because c was closed.    success bool  
  
    parent   *sudog // semaRoot binary tree    waitlink *sudog // g.waiting list or semaRoot  
    waittail *sudog // semaRoot  
    c        *hchan // channel  
}

```

sema unint32 都对应着一个 semaRoot结构体

结构体中treap 属性是一个sudog结构体指针，sudog中的 g 指向一个协程，一个sudog 有next,prev都是sudog指针。这样treap就形成了一棵树，其实他就是一个平衡二叉树。

treap存储了再sema变量上等待的协程。

nwait记录等待协程的数量

![xx|600](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230907_090700_461_0.png)

获取锁是调用的函数semacquire，源码如下

```go
// Called from runtime.  
func semacquire(addr *uint32) {  
    semacquire1(addr, false, 0, 0, waitReasonSemacquire)  
}
  
func semacquire1(addr *uint32, lifo bool, profile semaProfileFlags, skipframes int, reason waitReason) {  
    gp := getg()  
    if gp != gp.m.curg {  
       throw("semacquire not on the G stack")  
    }  
  
    // Easy case.  
    if cansemacquire(addr) {  
       return  
    }  
  
    // Harder case:  
    // increment waiter count    
    // try cansemacquire one more time, return if succeeded    
    // enqueue itself as a waiter    
    // sleep    
    // (waiter descriptor is dequeued by signaler)    s := acquireSudog()  
    root := semtable.rootFor(addr)  
    t0 := int64(0)  
    s.releasetime = 0  
    s.acquiretime = 0  
    s.ticket = 0  
    if profile&semaBlockProfile != 0 && blockprofilerate > 0 {  
       t0 = cputicks()  
       s.releasetime = -1  
    }  
    if profile&semaMutexProfile != 0 && mutexprofilerate > 0 {  
       if t0 == 0 {  
          t0 = cputicks()  
       }  
       s.acquiretime = t0  
    }  
    for {  
       lockWithRank(&root.lock, lockRankRoot)  
       // Add ourselves to nwait to disable "easy case" in semrelease.  
       root.nwait.Add(1)  
       // Check cansemacquire to avoid missed wakeup.  
       if cansemacquire(addr) {  
          root.nwait.Add(-1)  
          unlock(&root.lock)  
          break  
       }  
       // Any semrelease after the cansemacquire knows we're waiting  
       // (we set nwait above), so go to sleep.       
       root.queue(addr, s, lifo)  
       goparkunlock(&root.lock, reason, traceBlockSync, 4+skipframes)  
       if s.ticket != 0 || cansemacquire(addr) {  
          break  
       }  
    }  
    if s.releasetime > 0 {  
       blockevent(s.releasetime-t0, 3+skipframes)  
    }  
    releaseSudog(s)  
}


func cansemacquire(addr *uint32) bool {  
    for {  
       v := atomic.Load(addr)  
       if v == 0 {  
          return false  
       }  
       if atomic.Cas(addr, v, v-1) {  
          return true  
       }  
    }  
}
```

semacquire又调用了semacquire1函数，semacquire1中 `Easy case.  `调用了 cansemacquire函数

cansemacquire函数中对于sema变量分两种情况，一种是等于0，一种是大于0。当大于0的时候直接用cas操作，将sema减1，并返回true。

当sema等于0的时候，会执行`root.nwait.Add(1)`把等待协程数加1，同时`root.queue(addr, s, lifo)  `把协程加入到平衡二叉树中，最后`goparkunlock(&root.lock, reason, traceBlockSync, 4+skipframes)`把协程休眠。


释放锁的时候，调用了函数semrelease，源码如下：

```go
func semrelease(addr *uint32) {
	semrelease1(addr, false, 0)
}

func semrelease1(addr *uint32, handoff bool, skipframes int) {
	root := semtable.rootFor(addr)
	atomic.Xadd(addr, 1)

	// Easy case: no waiters?
	// This check must happen after the xadd, to avoid a missed wakeup
	// (see loop in semacquire).
	if root.nwait.Load() == 0 {
		return
	}

	// Harder case: search for a waiter and wake it.
	lockWithRank(&root.lock, lockRankRoot)
	if root.nwait.Load() == 0 {
		// The count is already consumed by another goroutine,
		// so no need to wake up another goroutine.
		unlock(&root.lock)
		return
	}
	s, t0 := root.dequeue(addr)
	if s != nil {
		root.nwait.Add(-1)
	}
	unlock(&root.lock)
	if s != nil { // May be slow or even yield, so unlock first
		acquiretime := s.acquiretime
		if acquiretime != 0 {
			mutexevent(t0-acquiretime, 3+skipframes)
		}
		if s.ticket != 0 {
			throw("corrupted semaphore ticket")
		}
		if handoff && cansemacquire(addr) {
			s.ticket = 1
		}
		readyWithTime(s, 5+skipframes)
		if s.ticket == 1 && getg().m.locks == 0 {
			// Direct G handoff
			// readyWithTime has added the waiter G as runnext in the
			// current P; we now call the scheduler so that we start running
			// the waiter G immediately.
			// Note that waiter inherits our time slice: this is desirable
			// to avoid having a highly contended semaphore hog the P
			// indefinitely. goyield is like Gosched, but it emits a
			// "preempted" trace event instead and, more importantly, puts
			// the current G on the local runq instead of the global one.
			// We only do this in the starving regime (handoff=true), as in
			// the non-starving case it is possible for a different waiter
			// to acquire the semaphore while we are yielding/scheduling,
			// and this would be wasteful. We wait instead to enter starving
			// regime, and then we start to do direct handoffs of ticket and
			// P.
			// See issue 33747 for discussion.
			goyield()
		}
	}
}

```

semrelease函数调用了semrelease1，semrelease1中对addr加1，`nwait==0`的时候，也就是没有协程等待的时候直接退出程序。

如果`nwait>0`的时候，`root.dequeue(addr)`从平衡二叉树取出一个协程，`root.nwait.Add(-1)`等待协程数量减1，然后唤醒取出的协程。


总的来说，sema的数值表示可以同时获取锁的数量，加锁的时候，减1，释放锁的时候加1

## sync.Mutex互斥锁

sync.Mutex的结构如下：

```go
type Mutex struct {  
    state int32  
    sema  uint32  
}
```

state的标识如下 ：

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230907_124320_020_0.png)

| 标识        | 描述             |
| ----------- | ---------------- |
| Locked      | 是否被锁         |
| Woken       | 唤醒             |
| Starving    | 饥饿状态         |
| WaiterShift | 等待锁的协程数量 |

### 正常模式

mutex正常模式加锁的流程，先通过cas原子操作尝试加锁，如果没有成功，开始自旋检查标志位是否释放锁，如果释放了，那么尝试加锁，如果加锁成功，那么直接返回。如果尝试多次依然没有加锁成功，那么把WaiterShift标志位+1，然后休眠当前的协程。

源码如下：

```go
func (m *Mutex) Lock() {  
    // Fast path: grab unlocked mutex.  
    if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {  
       if race.Enabled {  
          race.Acquire(unsafe.Pointer(m))  
       }  
       return  
    }  
    // Slow path (outlined so that the fast path can be inlined)  
    m.lockSlow()  
}
```

```go
func (m *Mutex) lockSlow() {
	var waitStartTime int64
	starving := false
	awoke := false
	iter := 0
	old := m.state
	for {
		// Don't spin in starvation mode, ownership is handed off to waiters
		// so we won't be able to acquire the mutex anyway.
		if old&(mutexLocked|mutexStarving) == mutexLocked && runtime_canSpin(iter) {
			// Active spinning makes sense.
			// Try to set mutexWoken flag to inform Unlock
			// to not wake other blocked goroutines.
			if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
				atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
				awoke = true
			}
			runtime_doSpin()
			iter++
			old = m.state
			continue
		}
		new := old
		// Don't try to acquire starving mutex, new arriving goroutines must queue.
		if old&mutexStarving == 0 {
			new |= mutexLocked
		}
		if old&(mutexLocked|mutexStarving) != 0 {
			new += 1 << mutexWaiterShift
		}
		// The current goroutine switches mutex to starvation mode.
		// But if the mutex is currently unlocked, don't do the switch.
		// Unlock expects that starving mutex has waiters, which will not
		// be true in this case.
		if starving && old&mutexLocked != 0 {
			new |= mutexStarving
		}
		if awoke {
			// The goroutine has been woken from sleep,
			// so we need to reset the flag in either case.
			if new&mutexWoken == 0 {
				throw("sync: inconsistent mutex state")
			}
			new &^= mutexWoken
		}
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
			if old&(mutexLocked|mutexStarving) == 0 {
				break // locked the mutex with CAS
			}
			// If we were already waiting before, queue at the front of the queue.
			queueLifo := waitStartTime != 0
			if waitStartTime == 0 {
				waitStartTime = runtime_nanotime()
			}
			runtime_SemacquireMutex(&m.sema, queueLifo, 1)
			starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
			old = m.state
			if old&mutexStarving != 0 {
				// If this goroutine was woken and mutex is in starvation mode,
				// ownership was handed off to us but mutex is in somewhat
				// inconsistent state: mutexLocked is not set and we are still
				// accounted as waiter. Fix that.
				if old&(mutexLocked|mutexWoken) != 0 || old>>mutexWaiterShift == 0 {
					throw("sync: inconsistent mutex state")
				}
				delta := int32(mutexLocked - 1<<mutexWaiterShift)
				if !starving || old>>mutexWaiterShift == 1 {
					// Exit starvation mode.
					// Critical to do it here and consider wait time.
					// Starvation mode is so inefficient, that two goroutines
					// can go lock-step infinitely once they switch mutex
					// to starvation mode.
					delta -= mutexStarving
				}
				atomic.AddInt32(&m.state, delta)
				break
			}
			awoke = true
			iter = 0
		} else {
			old = m.state
		}
	}

	if race.Enabled {
		race.Acquire(unsafe.Pointer(m))
	}
}

```

Lock函数中的`atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) ` 尝试加锁，成功直接返回；如果加锁失败的时候，会调用`m.lockSlow()`函数

`m.lockSlow()`函数中是一个死循环`old&(mutexLocked|mutexStarving) == mutexLocked`是判断当前状态是否为正常模式并且被加锁的状态，如果是，那么执行空逻辑继续自旋，如果没有加锁，那么通过`new |= mutexLocked`和`atomic.CompareAndSwapInt32(&m.state, old, new)`两个语句尝试加锁，如果加锁失败，重新回到循环开头。

`m.lockSlow()`函数的循环中，通过`runtime_canSpin(iter)`语句判断是否可以继续自旋等待，如果不可以了，那么执行`new += 1 << mutexWaiterShift`把等待锁的协程数量+1，同时执行`atomic.CompareAndSwapInt32(&m.state, old, new) `修改state，把WaiterShift字段更新，最后
执行`runtime_SemacquireMutex(&m.sema, queueLifo, 1)`休眠当前线程。

### 饥饿模式

当一个在休眠等待锁的协程被唤醒以后，会判断自己等待锁时间是否超过1ms，如果超过，把锁切换到饥饿模式

饥饿模式中，不自旋，新来的协程直接休眠

饥饿模式中，被唤醒的协程直接获取锁。

没有协程在队列中继续等待时，回到正常模式。




## sync.RWMutex读写锁

sync.RWMutex的结构如下：

```go
// There is a modified copy of this file in runtime/rwmutex.go.
// If you make any changes here, see if you should make them there.

// A RWMutex is a reader/writer mutual exclusion lock.
// The lock can be held by an arbitrary number of readers or a single writer.
// The zero value for a RWMutex is an unlocked mutex.
//
// A RWMutex must not be copied after first use.
//
// If a goroutine holds a RWMutex for reading and another goroutine might
// call Lock, no goroutine should expect to be able to acquire a read lock
// until the initial read lock is released. In particular, this prohibits
// recursive read locking. This is to ensure that the lock eventually becomes
// available; a blocked Lock call excludes new readers from acquiring the
// lock.
//
// In the terminology of the Go memory model,
// the n'th call to Unlock “synchronizes before” the m'th call to Lock
// for any n < m, just as for Mutex.
// For any call to RLock, there exists an n such that
// the n'th call to Unlock “synchronizes before” that call to RLock,
// and the corresponding call to RUnlock “synchronizes before”
// the n+1'th call to Lock.
type RWMutex struct {
	w           Mutex        // held if there are pending writers 互斥锁，用作写锁
	writerSem   uint32       // semaphore for writers to wait for completing readers  作为写协程等待队列
	readerSem   uint32       // semaphore for readers to wait for completing writers  读协程等待队列
	readerCount atomic.Int32 // number of pending readers  正值：正在读的协程；负值：加了写锁
	readerWait  atomic.Int32 // number of departing readers  写锁应该等待读协程个数
}

```

### Lock()加写锁

**情况一**

writerSem=空
readerSem=空
readerCount=0
readerWait=0
先调用w.Lock()，对互斥锁加锁，然后修改readerCount为readerCount-rwmutexMaxReaders，加写锁的过程结束。

**情况二**

writerSem=空
readerSem=空
readerCount=3
readerWait=0

先调用w.Lock()，对互斥锁加锁，然后readerWait修改为readerCount的值，然后readerCount修改为3-rwmutexMaxReaders，当前写写成加入writerSem的队列

## sync.WaitGroup

sync.WaitGroup结构

```go
// A WaitGroup waits for a collection of goroutines to finish.
// The main goroutine calls Add to set the number of
// goroutines to wait for. Then each of the goroutines
// runs and calls Done when finished. At the same time,
// Wait can be used to block until all goroutines have finished.
//
// A WaitGroup must not be copied after first use.
//
// In the terminology of the Go memory model, a call to Done
// “synchronizes before” the return of any Wait call that it unblocks.
type WaitGroup struct {
	noCopy noCopy

	state atomic.Uint64 // high 32 bits are counter, low 32 bits are waiter count. //包含两部分内容 counter和waiter 
	sema  uint32
}
```

Wait()函数，如果`waiter==0`，直接返回，否则协程休眠

Done()函数就是给counter减1，减完以后如果`counter==0`，那么循环把休眠的线程都唤醒。

## sync.Once

sync.Once 结构

```go
// Once is an object that will perform exactly one action.
//
// A Once must not be copied after first use.
//
// In the terminology of the Go memory model,
// the return from f “synchronizes before”
// the return from any call of once.Do(f).
type Once struct {
	// done indicates whether the action has been performed.
	// It is first in the struct because it is used in the hot path.
	// The hot path is inlined at every call site.
	// Placing done first allows more compact instructions on some architectures (amd64/386),
	// and fewer instructions (to calculate offset) on other architectures.
	done uint32
	m    Mutex
}
```

## 常见问题

### 锁拷贝问题

拷贝锁有可能会出现死锁的现象，例如：

```go
func TestLockCopy2(t *testing.T) {  
    m1 := sync.Mutex{}  
    go func() {  
       m1.Lock()  
       time.Sleep(10 * time.Second)  
    }()  
  
    time.Sleep(1 * time.Second)  
  
    m2 := m1  
    m1.Unlock()  
  
    m2.Lock()  
  
    fmt.Println("结束")  
}


=== RUN   TestLockCopy
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan receive]:
testing.(*T).Run(0x140000829c0, {0x10257d698?, 0x1b06def4eb692?}, 0x1025df5d0)
	/usr/local/go/src/testing/testing.go:1649 +0x350
testing.runTests.func1(0x140000a0420?)
	/usr/local/go/src/testing/testing.go:2054 +0x48
testing.tRunner(0x140000829c0, 0x140000ddc28)
	/usr/local/go/src/testing/testing.go:1595 +0xe8

```

可以通过命令 `go vet main.go` 来检测问题。

### RACE竞争检测

```go
func main() {  
    ii := 0  
    for i := 0; i < 10000; i++ {  
       go func() {  
          ii++  
       }()  
    }  
    fmt.Println(ii)  
}

```

以上代码不加锁的情况下，并发10000个协程对ii++。

编译以上代码的时候使用如下命令 `go build -race `,然后执行编译好的可执行文件。会报一下警告：

```go
WARNING: DATA RACE
Read at 0x00c00011e028 by goroutine 8:
  main.main.func1()
      /Users/lizhikui/Documents/study/workspaces/go/study_base/lesson4/main.go:9 +0x2c

Previous write at 0x00c00011e028 by goroutine 6:
  main.main.func1()
      /Users/lizhikui/Documents/study/workspaces/go/study_base/lesson4/main.go:9 +0x3c

Goroutine 8 (running) created at:
  main.main()
      /Users/lizhikui/Documents/study/workspaces/go/study_base/lesson4/main.go:8 +0x48

Goroutine 6 (finished) created at:
  main.main()
      /Users/lizhikui/Documents/study/workspaces/go/study_base/lesson4/main.go:8 +0x48
==================
9946
Found 1 data race(s)

```

### 死锁检查

go-deadlock一个第三方的go 包，替换原生包中的Mutex，出现死锁的时候会打印出错误信息。