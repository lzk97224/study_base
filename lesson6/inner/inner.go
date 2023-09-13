package inner

import (
	"fmt"
	_ "unsafe"
)

//go:linkname testnamelinke lesson6/out.Testnamelinke
func testnamelinke() {
	fmt.Println("哈哈")
}
