package lesson5

import (
	"fmt"
	"testing"
)

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
