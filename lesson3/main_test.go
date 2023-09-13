package main

import (
	"fmt"
	"runtime"
	"testing"
)

func TestLoopSchedule(t *testing.T) {
	runtime.GOMAXPROCS(1)
	i := 1
	go func() {
		for {
			i = i + 1
		}
	}()

	go func() {
		for {
			fmt.Println(i)
		}
	}()

	select {}
}
