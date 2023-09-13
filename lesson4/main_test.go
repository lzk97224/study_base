package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAtomic(t *testing.T) {
	var add = func(p *int32) {
		//*p++
		atomic.AddInt32(p, 1)
		atomic.CompareAndSwapInt32(p, 1, 2)
	}
	c := int32(0)
	for i := 0; i < 1000; i++ {
		go add(&c)
	}
	time.Sleep(time.Second)
	fmt.Println(c)
}

func TestAtomic1(t *testing.T) {

}

func TestLockCopy(t *testing.T) {
	m := sync.Mutex{}

	m.Lock()
	n := m
	m.Unlock()

	n.Lock()

	fmt.Println("结束")
}

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

func TestLock(t *testing.T) {
	i := 0
	for i := 0; i < 10000; i++ {
		i++
	}
	fmt.Println(i)
}
