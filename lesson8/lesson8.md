# cgo

```go
package main  
  
/*  
int sum(int a,int b){  
    return a+b;}  
*/  
import "C"  
  
import (  
    "fmt"  
)  
  
/*  
cgo使用  
1. import "C"，导入C  
2. 在import "C" 上面紧临的地方通过注释的方式写c函数  
3. 通过C.函数名,调用c的函数  
*/  
func testCGO() {  
    fmt.Println(C.sum(1, 1))  
}  
  
func main() {  
    testCGO()  
}
```


```shell
# 使用如下命令可以查看编译过程中的中间文件
go tool cgo main.go
```

# defer 的原理

## 实现方案

- 1.12之前，堆上分配
- 1.13之后，栈上分配
- 1.14之后，开放编码

## 堆上分配

在p结构体里有个deferpoll，记录了所有的defer语句。
函数返回的时候，调用runtime.deferreturn()，取出deferpool中的defer执行

## 栈上分配

只能保存一个defer

## 开放编码

如果defer语句在编译的时候可以固定，直接修改用户代码，defer语句放入函数末尾。

## 执行流程

在编译defer语句时

实际上是插入了`deferproc`函数，函数中创建了新的defer结构体，把栈帧，计数器函数指针等付给新的defer，然后把让协程中的_defer属性指向新创建的defer，同时defer.link指向协程中原来_defer属性的值。

创建defer的时候，会记录在p中的deferpool属性中

函数结束的时候会调用 `deferreturn` 函数

deferreturn函数中会根据协程中的_defer属性，把这个链表中所有的defer根据链表顺序都执行一次。

# panic 和 recover

panic以后会执行gopanic函数，函数会执行当前协程的所有defer

主要看`gopanic`函数实现

# 反射


