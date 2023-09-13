package lesson7

import (
	"fmt"
	"testing"
)

func taoYiPoint() *int {
	result := 1
	return &result
}

func TestTaoYiPoint(t *testing.T) {
	taoYiPoint()
}

func taoYiNilInterface() {
	i := 1
	fmt.Println(i)
}

func TestTaoYiNilInterface(t *testing.T) {
	taoYiPoint()
}
