package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func testDeferRunWithGo() {
	go func() {
		defer func() {
			fmt.Println("testDeferRunWithGo") //会打印
		}()
		testDeferBeCcalled()
		fmt.Println("1111111")
	}()
}

func testDeferBeCcalled() {
	defer func() {
		fmt.Println("testDeferBeCcalled") //会打印
	}()
	panic("xxx") //会把协程中的defer都执行一次
}

func TestDefer(t *testing.T) {
	defer func() {
		fmt.Println("TestDefer main") //不会打印，子协程中panic后，主协程的defer也不会执行
	}()
	testDeferRunWithGo()
	time.Sleep(time.Second)
	fmt.Println("TestDefer main") //不会打印，子协程中panic后，主协程中后续程序不会执行
	time.Sleep(time.Second)
}

// 打印各种类型的名字
func TestType(t *testing.T) {
	fmt.Println(reflect.TypeOf(int8(1)), reflect.TypeOf(int8(1)).Name())
	fmt.Println(reflect.TypeOf(int64(1)), reflect.TypeOf(int64(1)).Name())
	fmt.Println(reflect.TypeOf(int(1)), reflect.TypeOf(int(1)).Name())
	fmt.Println(reflect.TypeOf("sdf"), reflect.TypeOf("sdf").Name())

	s := struct {
		Name string
	}{}

	to := reflect.TypeOf(s)
	fmt.Println(to, to.Name())

	tp := reflect.TypeOf(&s)
	fmt.Println(tp, tp.PkgPath())

	ts := reflect.TypeOf([]string{})
	fmt.Println(ts, ts.Name())

	tm := reflect.TypeOf(map[string]string{})
	fmt.Println(tm, tm.Name())
}

func ld(c1 chan struct{}, index int) {
	for {
		_, ok := <-c1
		if ok {
			fmt.Println(index, "ok")
			break
		} else {
			fmt.Println(index, "wait")
			time.Sleep(time.Second)
		}
	}
}

func TestContext1(t *testing.T) {
	c1 := make(chan struct{}, 1)

	go ld(c1, 1)
	go ld(c1, 2)

	time.Sleep(2 * time.Second)

	c1 <- struct{}{} //发送消息的时候，只有一个协程会处理消息
	//close(c1) //close的时候，所有的协程都会被唤醒处理，只是接收不到可处理的消息，context就是利用了这个特性

	time.Sleep(time.Hour)

}
