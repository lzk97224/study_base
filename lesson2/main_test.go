package main

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

func Test_01printIntSize(t *testing.T) {
	i := 1
	fmt.Println(unsafe.Sizeof(i))
}

func Test_02printPointSize(t *testing.T) {
	i := 1
	p := &i
	fmt.Println(unsafe.Sizeof(i))
	fmt.Println(unsafe.Sizeof(p))
}

type K struct {
}

func Test_03printNilOjectSize(t *testing.T) {
	a := K{}
	b := 1
	c := K{}
	fmt.Println(unsafe.Sizeof(a))
	fmt.Printf("%p\n", &a) //空结构体的地址是固定的，zerobase，所有长度是0字节的地址
	fmt.Printf("%p\n", &b)
	fmt.Printf("%p\n", &c)
}

type F struct {
	C C
	I int
}
type C struct {
}

func Test_04printNilOjectSize(t *testing.T) {
	f := F{}
	c := C{}
	fmt.Println(unsafe.Sizeof(f.C))
	fmt.Printf("%p\n", &f.C)
	fmt.Printf("%p\n", &c)
}

func Test_05NilOjectUse(t *testing.T) {
	m := map[int]struct{}{}
	m[1] = struct{}{}

	nc := make(chan struct{}, 1)
	nc <- struct{}{}
}

func Test_06StringLen(t *testing.T) {
	ostr := "中国人12😁❀🔟"
	nrunBytes := []rune(ostr)
	fmt.Println(len(ostr))            //字节长度
	fmt.Println(len(nrunBytes))       //字符长度
	fmt.Println(string(nrunBytes[5])) //取一个字符

	//打印字符串各个指针地址
	fmt.Printf("%p\n", &ostr) //字符串的开始地址
	ostrToHeader := (*reflect.StringHeader)(unsafe.Pointer(&ostr))
	fmt.Printf("%p\n", ostrToHeader)       //获取到字符串强转成(reflect.StringHeader)的地址
	fmt.Printf("%p\n", &ostrToHeader.Data) //打印reflect.StringHeader中Data的存储地址
	//reflect.StringHeader的起始地址和reflect.StringHeader中Data的地址是一样的

	//所以可以使用ostrToHeader.Data的地址，直接强转成*string，可以看到也是可以打印出字符串内容的
	tp := *(*string)(unsafe.Pointer(&ostrToHeader.Data))
	fmt.Printf("%p\n", &tp)
	fmt.Println(tp)

	//单独从字符串中把字节数组拿出来打印
	p1 := unsafe.Pointer(ostrToHeader.Data)
	b1 := (*[22]byte)(p1)  //把字节数组拿出来
	s1 := string((*b1)[:]) //字节数组转切片再转字符串
	fmt.Println(s1)

	/*up1 := ostrToHeader.Data
	bs := []byte{}
	for i := 0; i < ostrToHeader.Len; i++ {

		by := (*byte)(unsafe.Pointer(up1))
		bs = append(bs, *by)
		up1 = up1 + 8

	}
	fmt.Println(string(bs))
	tmp := 1
	fmt.Println(unsafe.Offsetof(tmp))*/

}

type Animal interface {
	Bellow()
}

type Bird interface {
	Fly()
}

type Ostrich struct {
}

func (o *Ostrich) Fly() {
}

func (t *Ostrich) Bellow() {
	fmt.Println("Ostrich Bellow")
}

type Sparrow struct {
}

func (s Sparrow) Bellow() {
}

func (s Sparrow) Fly() {
}

func Test_07Interface(t *testing.T) {
	var ok bool
	//var o1 = &Ostrich{}
	//_, ok = o1.(Animal) //这里报错，因为断言的对象必须是接口类型，any也可以
	//fmt.Println(ok)

	var o2 any = &Ostrich{}
	_, ok = o2.(Animal) //空接口any断言是否是实现了某个接口
	fmt.Println(ok)

	var o3 any = &Ostrich{}
	_, ok = o3.(*Ostrich) //空接口any断言是否是某个具体的类型
	fmt.Println(ok)

	var o4 Animal = &Ostrich{}
	_, ok = o4.(*Ostrich) //某个接口类型断言是否为具体的类型
	fmt.Println(ok)

	_, ok = o4.(Bird) //断言是否是另一个接口
	fmt.Println(ok)

	switch o4.(type) {
	case Animal:
	case Bird:
	case *Ostrich:
	}

	var s1 any = Sparrow{}
	_, ok = s1.(Animal) //结构体实现接口，断言结构体是否实现接口，结果是true
	fmt.Println(ok)

	var s2 any = &Sparrow{}
	_, ok = s2.(Animal) //结构体实现接口，断言结构体指针是否实现了接口，结果依然是true，因为结构体实现接口的时候，编译器会自动让结构体指针也实现接口
	fmt.Println(ok)

	var o5 any = Ostrich{}
	_, ok = o5.(Animal) //结构体指针实现了接口，但是结构体断言是否实现了接口，结果是false，
	fmt.Println(ok)

}

func Test_08Nil(t *testing.T) {
	var a *int
	fmt.Println(a == nil)

	var b map[int]int
	fmt.Println(b == nil)

	//fmt.Println(a == b) 	//a和b不能比较，因为类型不一样

	/*var c struct{}
	fmt.Println(c == nil)*/ //无法将 'nil' 转换为类型 'struct{}'

	var a1 *int
	var b1 *int
	fmt.Println(a1 == b1) //相同类型的nil是相等的

	var nullInterface any
	var nullIntPtr *int

	fmt.Println(nullInterface == nil)
	fmt.Println(nullIntPtr == nil)
	fmt.Println(nullInterface == nullIntPtr) //虽然空接口可以跟其他类型比较，但是跟其他类型的nil值依然不相等
	nullInterface = nullIntPtr               //any就是空接口interface{}，它的空值是nil，但是当它被赋值为nullIntPtr以后，虽然nullInterface中还是没有值，但是有了类型信息，所以就已经不是0值了。
	fmt.Println(nullInterface == nil)        //false，
	fmt.Println(nullInterface == nullIntPtr) //在空接口被赋值为其他类型值以后，是可以跟值比较是否相等的。
}

func Test_08Alignof(t *testing.T) {
	//基本类型的sizeof和对齐系数是一样的。
	fmt.Println("bool size:", unsafe.Sizeof(true), "Alignof:", unsafe.Alignof(true))
	fmt.Println("int size:", unsafe.Sizeof(1), "Alignof:", unsafe.Alignof(1))
	fmt.Println("float64 size:", unsafe.Sizeof(1.1), "Alignof:", unsafe.Alignof(1.1))
}

type AlignofS1 struct {
	a bool  //Alignof:1
	b int32 //Alignof:4
}
type AlignofS2 struct {
	c int16     //Alignof:2
	b AlignofS1 //Alignof:4
}

type AlignofS3 struct {
	c int64     //Alignof:8
	b AlignofS1 //Alignof:4
}

func Test_09StructAlignof(t *testing.T) {
	fmt.Println("AlignofS1 size:", unsafe.Sizeof(AlignofS1{}), "Alignof:", unsafe.Alignof(AlignofS1{}))
	fmt.Println("AlignofS2 size:", unsafe.Sizeof(AlignofS2{}), "Alignof:", unsafe.Alignof(AlignofS2{}))
	fmt.Println("AlignofS3 size:", unsafe.Sizeof(AlignofS3{}), "Alignof:", unsafe.Alignof(AlignofS3{}))
}

type Size1 struct {
	a bool
}

type Size2 struct {
	b int16
	a bool
}
type Size3 struct {
	b int32
	a bool
	c int16
	d bool
}

func Test_10StructSize(t *testing.T) {
	fmt.Println("Size1 size:", unsafe.Sizeof(Size1{}), "Alignof:", unsafe.Alignof(Size1{}))
	fmt.Println("Size2 size:", unsafe.Sizeof(Size2{}), "Alignof:", unsafe.Alignof(Size2{}))
	fmt.Println("Size3 size:", unsafe.Sizeof(Size3{}), "Alignof:", unsafe.Alignof(Size3{}))
}

type SS1 struct {
	a bool
	b int32
	c bool
}

type SS2 struct {
	a bool
	c bool
	b int32
}

type NS1 struct {
	z struct{}
	c int16
}
type NS2 struct {
	c int16
	z struct{}
}

func Test_10StructSize2(t *testing.T) {
	//第一个例子
	fmt.Println("SS1 size:", unsafe.Sizeof(SS1{}), "Alignof:", unsafe.Alignof(SS1{}))
	fmt.Println("SS2 size:", unsafe.Sizeof(SS2{}), "Alignof:", unsafe.Alignof(SS2{}))

	//第二个例子
	fmt.Println("NS1 size:", unsafe.Sizeof(NS1{}), "Alignof:", unsafe.Alignof(NS1{}))
	fmt.Println("NS2 size:", unsafe.Sizeof(NS2{}), "Alignof:", unsafe.Alignof(NS2{}))

}
