# channel管道

## 管道的基本使用

```go
func Test01SimpleChan(t *testing.T) {

	/*c1 := make(chan string) //无缓冲区的管道
	c1 <- "1"               //无缓冲区的管道直接向管道中发数据，会造成死锁
	fmt.Println(<-c1)*/

	c2 := make(chan string) //无缓冲区管道
	go func() {             //启动一个协程向管道中发送数据
		c2 <- "1"
	}()
	fmt.Println(<-c2) //主协程从管道中接收数据，这样是可以正常运行的。

	c3 := make(chan string, 1) //有缓冲区的管道，有缓冲区的管道可以在一个协程中先发数据，再接收数据
	c3 <- "1"
	//c3 <- "1" //但是发送的数据不能超过缓冲区的大小，否则也会报错
	fmt.Println(<-c3)

	c4 := make(chan string, 1)
	close(c4)
	//c4 <- "1"         //已经关闭的管道不能再向其中发送数据
	fmt.Println(<-c4) //已经关闭的管道可以从中获取信息，但是获取到的都是空

}
```

## Channel的结构设计

 channel的结构源码定义：

```go
//runtime\chan.go

type hchan struct {
	//下面的5个属性组成了一个环形的缓冲区
	qcount   uint           // total data in the queue 总共缓存了数据的数量
	dataqsiz uint           // size of the circular queue  缓冲区的大小
	buf      unsafe.Pointer // points to an array of dataqsiz elements //缓冲区里的第一个缓存的数据，  
	elemsize uint16 //每个数据的大小
	elemtype *_type // element type  每个数据的类型

	//下面4个属性，组成了发送队列和接收队列
	sendx    uint   // send index
	recvx    uint   // receive index
	recvq    waitq  // list of recv waiters
	sendq    waitq  // list of send waiters


	//关闭状态
	closed   uint32


	//锁
	// lock protects all fields in hchan, as well as several
	// fields in sudogs blocked on this channel.
	//
	// Do not change another G's status while holding this lock
	// (in particular, do not ready a G), as this can deadlock
	// with stack shrinking.
	lock mutex
}
```

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230907_173520_096_0.png)

channel的缓冲区是一个环形结构，由上面代码的前5个属性组成，环形缓存复用内存空间，不需要回收内存。

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230907_234146_153_0.png)

sendx,sendq组成了发送数据协程的队列

recvx,recvq组成了接收数据协程队列

![channel|300](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230908_000321_908_0.png)

## channel发送数据的原理

发送数据的语法`c<-` 在编译的时候会转成`runtime.chansend1()`函数调用。

```go
//runtime/chan.go

// entry point for c <- x from compiled code.  
//  
//go:nosplit  
func chansend1(c *hchan, elem unsafe.Pointer) {  
    chansend(c, elem, true, getcallerpc())  
}
```

发送的场景

- 直接发送
- 放入缓存
- 休眠等待

### 直接发送

- 发送数据前，已经有G在休眠等待接收
- 此时缓存肯定是空的，不用考虑缓存。
- 将数据直接拷贝给G的接收变量，唤醒G。
[](https://boardmix.cn/app/share/CAE.CMDZkQwgASoQZgcjfOhzsvEu0i2oT-K8fzAGQAE/S7I2Ge)

## channel接收数据

channel接收数据有两种使用方式

```go
i:=<-c    //方法一
i,ok:=<-c //方法二
```

第一种方法调用编译时替换成 `chanrecv1`
第二种方法调用编译时替换成 `chanrecv2`
