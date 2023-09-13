# # 认识Golang
## C VS Java VS Golang 特性对比

| C                          | Java              | Javascript                 | Golang                                   |
| -------------------------- | ----------------- | -------------------------- | ---------------------------------------- |
| 编译成机器码               | 编译成中间码      | 不需要编译，<br />解释执行 | 编译成机器码                             |
| 不需要执行环境             | 需要执行环境(JVM) | 需要执行环境(V8)           | 自带执行环境，<br />在编译好的执行文件中 |
| 一次编码只能适用于一种平台 | 一次编译多处执行  | 一次编写多处执行           | 一次编写多处执行                         |
| 自己管理内存地方           | GC自动回收垃圾    | 自动垃圾回收               | 自动垃圾回收                             |

- Go综合了多种语言的优势
- Go天生支持高并发场景
- Go目前已经在业界有了广泛的应用

## Runtime

Runtime，运行时，就是支撑Go运行的执行环境。类似Java的JVM，JavaScript的V8引擎。

Go没有虚拟机的概念，Runtime作为程序的一部分，最终编译打包进二进制文件里，调用Runtime里功能和调用自己的代码是一样的。

Runtime提供了功能有：

- 内存管理能力
- 垃圾回收能力
- 协程调度
- 屏蔽系统调用能力(做了不同系统调用之间的差异)
- 很多关键字就是在调用Runtime的函数 ^21c280
### 部分关键字和Runtime函数的对应关系

| 关键字 | 函数                        |
| ------ | --------------------------- |
| go     | newproc                     |
| new    | newobject                   |
| make   | makeslice,makechain,makemap |
| <-     | chansend1,chanrecv1         |
## Go的编译过程

go 编译的过程大概分为：
- 词法分析
- 句法分析
- 语义分析
- 中间码生成(SSA)
- 代码优化
- 机器码生成
- 链接

```bash
go build -n
```

这个命令可以把编译的过程打印出来。
如果想更具体的看到每步操作后生成的文件，可以通过下面的命令看到。

```bash
export GOSSAFUNC=main
go build
```

`GOSSAFUNC`环境变量指定一个函数，通过`go build`可以生成html形式的go编译过程，最终可以看到平台无关的汇编SSA代码。

生成SSA最终会生成平台相关的汇编指令，可以通过如下命令看到：

```bash
go build -gcflags -S main.go
```
## Go程序怎么跑起来的
go执行的第一个行代码是`/usr/local/go/src/runtime/rt0_xxx.s`文件开始，这是汇编文件，xxx省略的是具体的cpu和系统，例如：rt0_linux_amd64.s。以64位linux文件为例，执行的主要步骤如下：
- 先执行文件中的`_rt0_amd64_linux`函数，然后跳转到`asm_amd64.s`文件的`_rt0_amd64`函数
- `_rt0_amd64`函数中处理了argc和argv,然后调用了`runtime·rt0_go`函数
- `runtime·rt0_go`函数也在`asm_amd64.s`文件中，函数中也是先处理了argc和argv
- 然后调用`runtime·g0`，启动了g0协程，这个g0协程不受调度器的调度，然后创建栈信息
- 然后调用了`runtime·check`函数，这是一个go写的函数，在runtime.runtime1.go中的`check`函数，主要做了各种检查，包括各种类型的长度，cas
- 调用`runtime·args`函数
- 调用`runtime·osinit`函数
- 调用`runtime·schedinit`函数，初始化调度器
- 通过`runtime·mainPC`获取到`runtime.main`函数的地址
- 调用`runtime·newproc`函数创建一个新的协程
- 调用`runtime·mstart`创建一个m，也就是系统线程