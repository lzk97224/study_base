多路复用

epoll相关函数

- epoll_create()
- epoll_ctl()
- epoll_wait()

go对epoll的抽象函数

- netpollinit() -> epoll_create()
- netpollopen() ->epoll_ctl()
- netpoll() - epoll_wait() 
- 

netpollopen 注册可读、可写、断开

netpoll 返回的是协程列表

pollDesc结构
### network poller 初始化

poll_runtime_pollServerInit()
一个go程序只会初始化一个network poller

netpollGenericInit()

netpollinit()


pollCache

```go
type pollCache struct {
	lock  mutex
	first *pollDesc
	// PollDesc objects must be type-stable,
	// because we can get ready notification from epoll/kqueue
	// after the descriptor is closed/reused.
	// Stale notifications are detected using seq variable,
	// seq is incremented when deadlines are changed or descriptor is reused.
}
```

pollDesc结构

```go
//runtime/netpoll.go:73

// Network poller descriptor.  
//  
// No heap pointers.
type pollDesc struct {
	_     sys.NotInHeap
	link  *pollDesc      // in pollcache, protected by pollcache.lock
	fd    uintptr        // constant for pollDesc usage lifetime
	fdseq atomic.Uintptr // protects against stale pollDesc

	// atomicInfo holds bits from closing, rd, and wd,
	// which are only ever written while holding the lock,
	// summarized for use by netpollcheckerr,
	// which cannot acquire the lock.
	// After writing these fields under lock in a way that
	// might change the summary, code must call publishInfo
	// before releasing the lock.
	// Code that changes fields and then calls netpollunblock
	// (while still holding the lock) must call publishInfo
	// before calling netpollunblock, because publishInfo is what
	// stops netpollblock from blocking anew
	// (by changing the result of netpollcheckerr).
	// atomicInfo also holds the eventErr bit,
	// recording whether a poll event on the fd got an error;
	// atomicInfo is the only source of truth for that bit.
	atomicInfo atomic.Uint32 // atomic pollInfo

	// rg, wg are accessed atomically and hold g pointers.
	// (Using atomic.Uintptr here is similar to using guintptr elsewhere.)
	rg atomic.Uintptr // pdReady, pdWait, G waiting for read or pdNil
	wg atomic.Uintptr // pdReady, pdWait, G waiting for write or pdNil

	lock    mutex // protects the following fields
	closing bool
	user    uint32    // user settable cookie
	rseq    uintptr   // protects from stale read timers
	rt      timer     // read deadline timer (set if rt.f != nil)
	rd      int64     // read deadline (a nanotime in the future, -1 when expired)
	wseq    uintptr   // protects from stale write timers
	wt      timer     // write deadline timer
	wd      int64     // write deadline (a nanotime in the future, -1 when expired)
	self    *pollDesc // storage for indirect interface. See (*pollDesc).makeArg.
}
```


### poll_runtime_pollOpen

pollcache.alloc()
在pollcache链表中分配一个节点pollDesc

pollDesc 的rg,wg 都是0

netpollopen()



### netpoll()

runtime会循环调用netpoll()函数，g0协程

gcStart()

netpoll()

epollwait()

mode+='r' 或者 mode+='w'

netpollreday()

netpollunblock(pd,'r',true)

检查pd.wg或者pd.rg的状态

如果当前pd.wg或者pd.rg的状态是pdReady状态直接返回

如果当前pd.wg或者pd.rg的状态是初始状态，那么pd.wg或者pd.rg 赋值 pdReady

如果当前pd.wg或者pd.rg中存储的是协程地址，那么pd.wg或者pd.rg 赋值 pdReady，那么返回协程地址

toRun.push(rg)，把写成放到返回的协程列表中。



### poll_runtime_pollWait()

业务协程调用poll_runtime_pollWait()

netpollblock()

判断协程rg或者wg是否是pdReady(1)

如果是，把状态改回0，返回true。`gpp.CompareAndSwap(pdReady, pdNil)`

如果不是，把状态改为pdWait，协程休眠。 `gpp.CompareAndSwap(pdNil, pdWait)`

休眠 `gopark(netpollblockcommit, unsafe.Pointer(gpp), waitReasonIOWait, traceBlockNet, 5)`

休眠的时候调用了 `netpollblockcommit`

`atomic.Casuintptr((*uintptr)(gpp), pdWait, uintptr(unsafe.Pointer(gp)))` 把pdWait标识改成协程的地址。



协程需要收发数据时，Socket已经可读写

协程需要发送数据时，Socket暂时无法读写




