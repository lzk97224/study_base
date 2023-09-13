package main

/*
int sum(int a,int b){
	return a+b;
}
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
