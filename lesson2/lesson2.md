# Go中的数据结构

## 工具

```go
//计算传入值占用的字节大小
unsafe.Sizeof()
```
## 特殊的数据类型

### 指针

指针的长度是根据系统确定的，64位系统为8个字节
### 空结构体

空结构体类型的定义`struct{}`，空结构体的值的指针都是指向zerobase的，zerobase 的定义如下：

```go
// base address for all 0-byte allocations  
var zerobase uintptr
```

空结构体的长度是0，所有位0长度的值的地址都是固定的，是zerobase的地址。

```go
a := K{}
b := 1
c := K{}
fmt.Println(unsafe.Sizeof(a))
fmt.Printf("%p\n", &a) //空结构体的地址是固定的，zerobase，所有长度是0字节的地址
fmt.Printf("%p\n", &b)
fmt.Printf("%p\n", &c)


-----output---------
0
0x104dec5e0
0x1400000e3b8
0x104dec5e0

```

虽然空结构体的地址是固定的，但是如果空结构体嵌套在其他非空结构体内，地址就不再是zerobase地址了。

```go
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

-------output---------
0
0x14000116280
0x1045245e0
```

空结构体的一个重要用途，可以节省内存，例如下代码

```go
func Test_05NilOjectUse(t *testing.T) {
m := map[int]struct{}{}
m[1] = struct{}{}

nc := make(chan struct{}, 1)
nc <- struct{}{}

}
```

可以看到实例中,map和chan中都是存放了空结构体，但是空结构体的大小是0，所以都不占用空间。实例中的map实现了set的功能，chan实现了不需要传递数据，只是通知的功能。

### 字符串

字符串的长度都是16个字节，因为字符串的结构是一个指针和一个int组成的，可以通过`runntime/string.go`文件中的`stringStruct`
结构体看到。
字符串结构体中的str指针指向一个字节数组。
len表示的是字节数。
如果想要遍历出每个字符，需要使用for range的形式遍历。

```go
//runntime/string.go

type stringStruct struct {
str unsafe.Pointer
len int
}
```

字符串的转换？？带补充

### 切片

切片的定义如下

```go
//runtime/slice.go
type slice struct {
array unsafe.Pointer //数组指针
len   int            //长度
cap   int             //容量
}
```

结构如图

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_093038_107_0.png)

切片创建的方法

```go
//根据数组创建
slice := arr[0:3]

//根据字面量创建,编译时插入创建数组代码
slice := []int{1, 2, 3}

//make创建，运行时调用makeslice创建切片
slice := make([]int, 10)
```

make函数创建切片的时候，是运行时调用`runtime/slice.go`的`makeslice`函数来创建切片。

切片的追加

不需要扩容的时候，只调整len+1

需要扩容的时候

调用runtime.growslice()函数给切片扩容，大概逻辑。

当前切片容量小于1024的时候

默认扩容为原来的2倍，如果期望扩容的大小大于原来的2倍，那么直接使用期望的容量。

当切片大于1024的时候，每次扩充25%。

### map

#### Map的底层数据结构

map在go底层代码中的数据结构定义如下

```go
//runtime/map.go

type hmap struct {
// Note: the format of the hmap is also encoded in cmd/compile/internal/reflectdata/reflect.go.
// Make sure this stays in sync with the compiler's definition.
count     int // # live cells == size of map.  Must be first (used by len() builtin)
flags     uint8
B         uint8 // log_2 of # of buckets (can hold up to loadFactor * 2^B items)
noverflow uint16 // approximate number of overflow buckets; see incrnoverflow for details
hash0     uint32 // hash seed

buckets    unsafe.Pointer // array of 2^B Buckets. may be nil if count==0.
oldbuckets unsafe.Pointer // previous bucket array of half the size, non-nil only when growing
nevacuate  uintptr        // progress counter for evacuation (buckets less than this have been evacuated)

extra *mapextra // optional fields
}
```

| 属性         | 说明                 |
|------------|--------------------|
| count      | map的大小             |
| buckets    | 数组的长度， Buckets=2^B |
| B          | B=log_2 Buckets    |
| hash0      | hash种子             |
| oldbuckets |                    |

buckets是一个数组，每个bucket就是一个bmap，bmap结构如下

```go
// A bucket for a Go map.
type bmap struct {
	// tophash generally contains the top byte of the hash value
	// for each key in this bucket. If tophash[0] < minTopHash,
	// tophash[0] is a bucket evacuation state instead.
	tophash [bucketCnt]uint8
	// Followed by bucketCnt keys and then bucketCnt elems.
	// NOTE: packing all the keys together and then all the elems together makes the
	// code a bit more complicated than alternating key/elem/key/elem/... but it allows
	// us to eliminate padding which would be needed for, e.g., map[int64]int8.
	// Followed by an overflow pointer.
}
```
bmap中的tophash是一个8个长度的数组，里面存储的是hash值的前8位，因为key和value的类型是不确定的，所以key和value是在编译的时候，动态插入的。map的整体结构如下：

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_103952_702_0.png)



#### map 的扩容

#### map为什么需要扩容

在哈希碰撞严重的情况，一个`bmap`中存满8个key value，再向`bmap`中插入数据，就需要通过`overflow`指向一个溢出桶，把数据放到溢出桶内，如果继续碰撞，极端情况溢出桶会指向下一个溢出桶，以此类推，最终形成一个桶的链表，效率会严重下降。

#### map的扩容时机

- 当前不在扩充之中
- 超过了装在系数(平均每个槽里有6.5个key)
- 出现了太多的溢出桶(溢出桶超过普通桶的个数)

#### map扩容的逻辑

扩容的基本逻辑：

- 创建一组新桶
- oldbuckets指向原有的桶数组
- buckets指向新的桶数组
- map标记为扩容状态

扩容的代码在
```go
//runtime/map.go
func hashGrow(t *maptype, h *hmap) {  
    // If we've hit the load factor, get bigger.  
    // Otherwise, there are too many overflow buckets,    // so keep the same number of buckets and "grow" laterally.    bigger := uint8(1)  
    if !overLoadFactor(h.count+1, h.B) {  
       bigger = 0  
       h.flags |= sameSizeGrow  
    }  
    oldbuckets := h.buckets  
    newbuckets, nextOverflow := makeBucketArray(t, h.B+bigger, nil)  
  
    flags := h.flags &^ (iterator | oldIterator)  
    if h.flags&iterator != 0 {  
       flags |= oldIterator  
    }  
    // commit the grow (atomic wrt gc)  
    h.B += bigger  
    h.flags = flags  
    h.oldbuckets = oldbuckets  
    h.buckets = newbuckets  
    h.nevacuate = 0  
    h.noverflow = 0  
  
    if h.extra != nil && h.extra.overflow != nil {  
       // Promote current overflow buckets to the old generation.  
       if h.extra.oldoverflow != nil {  
          throw("oldoverflow is not nil")  
       }  
       h.extra.oldoverflow = h.extra.overflow  
       h.extra.overflow = nil  
    }  
    if nextOverflow != nil {  
       if h.extra == nil {  
          h.extra = new(mapextra)  
       }  
       h.extra.nextOverflow = nextOverflow  
    }  
  
    // the actual copying of the hash table data is done incrementally  
    // by growWork() and evacuate().}
```

makeBucketArray
flags
B
oldbuckets
buckets
extra

扩容的时候，只创建新的桶数组，然后更新相关的属性，不进行数据迁移，go是渐进式的迁移数据，每次操作旧桶的时候把旧桶的数据迁移到新桶，读取数据的时候不进行迁移，只判断是在新桶读取，还是在旧桶里读取。

在修改map旧桶里的数据的时候，在修改值的同时，还要把key所在桶的所有数据迁移到新桶中。

假设原来的B=2，那么根据哈希值低2位确定桶的位置，扩容的时候B=B+1，那么根据哈希值的低3位确定桶的位置，假如原来的后3位为`101`那么迁移之前在1号桶，迁移后在5号桶。

### sync.Map 并发安全map

sync.Map的源码位置
```go
//sync/map.go
type Map struct {
	mu Mutex                      //锁
	read atomic.Pointer[readOnly] //数据
	dirty map[any]*entry          //存储
	misses int                    //
}

type readOnly struct {
	m       map[any]*entry   //数据
	amended bool             //修改的标志
}
type entry struct {
	p atomic.Pointer[any]    //实际value存储的位置
}
```

数据结构如图所示：
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_174540_741_0.png)

sync.Map读写流程

{"a":"AAA"}读写的时候，先找到sync.Map中的read.m，然后根据key找到m中的 \*entry，在根据entry的Pointer找到具体的值

sync.Map追加

先按照读取的流程读取数据，发现没有，那么找到syn.Map的dirty，给dirty加锁，然后给dirty中增加一个\*entry，\*entry的Pointer存储新追加的值，最后标记sync.Map的amended标记为true

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_214756_956_0.png)


sync.Map追加以后得读取

先按照正常的读取操作读取，但是从read中没有读到信息，那么从dirty中读取信息，同时misses++。
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_194409_476_0.png)

sync.Map dirty提升

当misses=len(dirty)的时候，说明越来越多的读操作走到的dirty，所以要用dirty替换read.m，read.m指向dirty，amended=false,misses=0，dirty=nil
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_195024_063_0.png)

sync.Map 追加

dirty刚刚提升为read.m的时候，dirty=nil，当有新的追加操作的时候，重建dirty
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_200030_732_0.png)

sync.Map 正常删除

如果删除的k在read中，那么直接把在read.m中找到k对应的\*entry，把\*entry中的Pointer指针置为空，然后gc会把pointer指向的内容回收掉。
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_214152_316_0.png)


sync.Map 追加后又删除，最后提升dirty

map追加了一个kv后先放在了dirty上，然后又删除掉了，这个时候在dirty里有有这个k的，但是k对应的\*entry中的指针是nil，提升dirty后，变成了read.m里有被删除的k，这个时候如果有追加，那么会重建dirty，为了防止重建的时候，把已经删除的k也重建到dirty上，在已经删除k对应的\*entry中的pointer标记为`expunged`表示已经删除
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230904_213454_083_0.png)
### 接口interface

接口在go代码底层的定义如下

```go
//runtime/runtime2.go

type iface struct {  
    tab  *itab           //
    data unsafe.Pointer  //这是实现类对象的地址
}
type itab struct {  
    inter *interfacetype //接口的类型
    _type *_type         //接口内值的类型
    hash  uint32         // copy of _type.hash. Used for type switches.  
    _     [4]byte  
    fun   [1]uintptr    // variable sized. fun[0]==0 means _type does not implement inter.  
}
```

iface这个结构体就是interface类型内部结构，记录了实现类对象的地址，接口类型，实现类的类型等信息

#### 类型断言

```go
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

output------
true
true
true
true
true
true
false

```

断言就是检查一个接口变量是否实现了某个接口，或者是否是某个具体的类型。断言的对象不能是一个具体的类型
- 断言只针对接口变量，是某个接口或者是空接口
- 可以断言是否实现了某个接口
- 可以断言是否是某个具体的实验接口的类型
- 使用结构体实现接口的时候，在编译的时候会自动让指针也实现接口
- 使用结构体指针实现接口的时候，编译器不会自动让结构体实现接口

#### 空接口

空接口底层是一个独立的类型，结构是runtime.eface，定义空接口的时候，其实就是定义了一个eface类型的变量，\_type存储了赋值语句右侧的值的类型，data存储了值的地址。

```go
//runtime/runtime2.go

type eface struct {  
    _type *_type  
    data  unsafe.Pointer  
}
```

### nil，空接口，空结构体的区别

nil在go源码中是这样定义的

```go
//builtin/builtin.go

// nil is a predeclared identifier representing the zero value for a  
// pointer, channel, func, interface, map, or slice type.  
var nil Type // Type must be a pointer, channel, func, interface, map, or slice type
```

所以nil是一个变量，nil可能是pointer,channel ,func,interface,map,slice中的一种，是这6种类型的0值。nil是有类型的，不同类型的nil值无法比较。

```go
func Test_08Nil(t *testing.T) {  
    var a *int  
    fmt.Println(a == nil)  
  
    var b map[int]int  
    fmt.Println(b == nil)  
  
    //fmt.Println(a == b)  //a和b不能比较，因为类型不一样  
  
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
```

### 内存对齐

#### 对齐系数

```go
//查看一个数据类型的对齐系数
unsafe.Alignof(true)
```

对齐系数表示要存储这个类型的数据，那么存储的地址必须能够被系数整除。例如：当要分配一个存储int32的内存空间时，如果前面已经存储了一个bool类型，这个时候空闲内存的起始地址是从1个字节开始，因为int32的对齐系数是4字节，那么就会跳过2，3，4字节存储空间，直接从第4个字节开始分配。

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230906_084906_340_0.png)

**基本类型对齐系数**

基本类型的对齐系数跟类型占用的内存大小一样

```go
func Test_08Alignof(t *testing.T) {  
    //基本类型的sizeof和对齐系数是一样的。  
    fmt.Println("bool size:", unsafe.Sizeof(true), "Alignof:", unsafe.Alignof(true))  
    fmt.Println("int size:", unsafe.Sizeof(1), "Alignof:", unsafe.Alignof(1))  
    fmt.Println("float64 size:", unsafe.Sizeof(1.1), "Alignof:", unsafe.Alignof(1.1))  
}


=== RUN   Test_08Alignof
bool size: 1 Alignof: 1
int size: 8 Alignof: 8
float size: 8 Alignof: 8
--- PASS: Test_08Alignof (0.00s)
```

**结构体的对齐系数**

结构体的对齐系数其实就是结构体内部成员中对齐系数最大的值。

```go
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

=== RUN   Test_09StructAlignof
AlignofS1 size: 8 Alignof: 4
AlignofS2 size: 12 Alignof: 4
AlignofS3 size: 16 Alignof: 8
--- PASS: Test_09StructAlignof (0.00s)
```

可以看出AlignofS1结构体成员变量中最大的对齐系数是4，那么结构体的对齐系数就是4。AlignofS2中对齐系数最大的是AlignofS1的对齐系数4，这个时候不管AlignofS1的大小具体是多少，只看对齐系数就可以。因为只要按照最大的对齐系数去对齐，再加上内部偏移量的控制，那么比它小的属性也一定能对齐。

#### 结构体大小

结构体大小计算，其实就是按照内存对齐的规则，模拟从0地址开始的内存上分配内存后的大小再加上填充位的大小。

从0开始，依次为成员变量分配内存，当内存地址不能被成员变量的对齐系数整除的时候，那么就向后偏移，找第一个能整除的位置进行分配。

长度填充，就是要让整个结构体的长度是当前结构体对齐系数的整数倍，如果不是，就需要在后面不空

```go
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

=== RUN   Test_10StructSize
Size1 size: 1 Alignof: 1
Size2 size: 4 Alignof: 2
Size3 size: 12 Alignof: 4
--- PASS: Test_10StructSize (0.00s)
```

其中Size3的结构体内部结构如下图：
![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230906_102323_521_0.png)
[截图地址](https://e7ydxchinm.feishu.cn/docx/Ing6do82no7QjTx8DVZcUv6Kngg?openbrd=1#doxcnjB6ubEcDiM8fbMAh59vvMc)
- 第一个属性int32直接从开始分配
- 第二个属性，以为属性的对齐系数是1，所以可以紧接着int32分配
- 第三个属性，int16的对齐系数是2,所以需要偏移1位，到下一个可以被2整除的位置分配
- 第四个属性，又是bool，可以紧接着分配。
- 当前结构体最大的对齐系数是int32，也就是4，所以结构体的对齐系数也是4，分配完以后，整体的长度不是4的倍数，所以需要补齐到4的倍数。也就是再补3字节，到12字节。

可以看出我们在定义结构体的时候，有可能根据属性的定义顺序优化内存的使用，例如：

```go
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

=== RUN   Test_10StructSize2
SS1 size: 12 Alignof: 4
SS2 size: 8 Alignof: 4
NS1 size: 2 Alignof: 2
NS2 size: 4 Alignof: 2
--- PASS: Test_10StructSize2 (0.00s)

```

第一个例子中，先把两个bool放在前面，结构体的大小减少了1/3。因为利用到了前面空出来的内存，减少了后面的填充。

第二个例子中，因为struct{}，本身不占用空间，但是在结构体结尾的时候，会导致后面填充。