package main

import (
	"lesson6/out"
	_ "unsafe"
)

func main() {
	useGoLinkName()
}

// 使用golink
// 1. 定义正常的函数 lesson6/inner.testnamelinke
// 2. 定义函数声明  lesson6/out.Testnamelinke
// 3. 在函数声明文件同级目录创建一个.s文件
// 4. 在函数声明文件中，引入函数定义的包 `import _ "lesson6/inner"`
// 5. 在函数定义的文件中引入unsafe包  `_ "unsafe"`
// 6. 在函数定义的文件中添加注解 `//go:linkname testnamelinke lesson6/out.Testnamelinke`
func useGoLinkName() {
	out.Testnamelinke()
}
