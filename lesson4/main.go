package main

import "fmt"

func main() {
	ii := 0
	for i := 0; i < 10000; i++ {
		go func() {
			ii++
		}()
	}
	fmt.Println(ii)
}
