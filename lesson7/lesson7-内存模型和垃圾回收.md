## 栈内存

### Go协程栈的作用

![stack|200](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230911_174047_764_0.png)

- 记录协程的函数调用关系
- 局部变量
- 函数传参
- 函数的返回

### Go协程栈的位置

Go的协程栈位于Go的堆内存上

Go堆内存位于操作系统虚拟内存上

### Go协程的结构

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_084241_152_0.png)

### 协程栈扩容

Go栈的初始空间为2kB

在函数调用前判断栈空间，morestack，判断栈是否足够，必要时进行扩容。

早期的时候用分段栈

现在使用连续栈

#### 分段栈

![分段栈|200](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_090720_571_0.png)

优点：没有空间浪费
缺点：会访问两块内存，有访问内存的开销

#### 连续栈

当空间不足的时候，开辟新的2倍于原来空间的内存，把老空间的内容拷贝过去。

当空间使用不足原来的1/4的时候，缩容为原来的1/2

优点：空间一直连续

缺点：伸缩的时候开销大



### 逃逸分析

不是所有的变量都放在协程栈上，例如栈帧回收后，需要继续使用的变量或者太大的变量。


**指针变量逃逸**，返回的结果是指针，在函数外可以访问，那么定义的返回值会放到堆上。

```go
func taoYiPoint() *int {  
    result := 1  
    return &result  
}
```

**空接口逃逸**，因为出入的参数是空接口，很可能在函数内部使用反射，反射要求被反射的对象在堆上，所以会造成逃逸。

```go
func taoYiNilInterface() {  
    i := 1  
    fmt.Println(i)  
}

// Println formats using the default formats for its operands and writes to standard output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Println(a ...any) (n int, err error) {
	return Fprintln(os.Stdout, a...)
}
```

**大变量逃逸**，过大的变量会导致栈空间不足，64位机器中，一般超过64kB的变量会逃逸

## 堆内存

### 内存分配方式

#### 线性分配

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_095426_348_0.png)

#### 链表分配

#### 分级分配

![分级分配](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_100008_948_0.png) ^19dc6a


### 内存结构

Go每次申请的虚拟内存单元是64MB

最多可以申请2^20个虚拟内存单元

内存单元也叫heapArena

所有的heapArana组成了mheap(堆内存)

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_094859_044_0.png)

heapArena

```go
// A heapArena stores metadata for a heap arena. heapArenas are stored
// outside of the Go heap and accessed via the mheap_.arenas index.
type heapArena struct {
	_ sys.NotInHeap

	// bitmap stores the pointer/scalar bitmap for the words in
	// this arena. See mbitmap.go for a description.
	// This array uses 1 bit per word of heap, or 1.6% of the heap size (for 64-bit).
	bitmap [heapArenaBitmapWords]uintptr

	// If the ith bit of noMorePtrs is true, then there are no more
	// pointers for the object containing the word described by the
	// high bit of bitmap[i].
	// In that case, bitmap[i+1], ... must be zero until the start
	// of the next object.
	// We never operate on these entries using bit-parallel techniques,
	// so it is ok if they are small. Also, they can't be bigger than
	// uint16 because at that size a single noMorePtrs entry
	// represents 8K of memory, the minimum size of a span. Any larger
	// and we'd have to worry about concurrent updates.
	// This array uses 1 bit per word of bitmap, or .024% of the heap size (for 64-bit).
	noMorePtrs [heapArenaBitmapWords / 8]uint8

	// spans maps from virtual address page ID within this arena to *mspan.
	// For allocated spans, their pages map to the span itself.
	// For free spans, only the lowest and highest pages map to the span itself.
	// Internal pages map to an arbitrary span.
	// For pages that have never been allocated, spans entries are nil.
	//
	// Modifications are protected by mheap.lock. Reads can be
	// performed without locking, but ONLY from indexes that are
	// known to contain in-use or stack spans. This means there
	// must not be a safe-point between establishing that an
	// address is live and looking it up in the spans array.
	spans [pagesPerArena]*mspan

	// pageInUse is a bitmap that indicates which spans are in
	// state mSpanInUse. This bitmap is indexed by page number,
	// but only the bit corresponding to the first page in each
	// span is used.
	//
	// Reads and writes are atomic.
	pageInUse [pagesPerArena / 8]uint8

	// pageMarks is a bitmap that indicates which spans have any
	// marked objects on them. Like pageInUse, only the bit
	// corresponding to the first page in each span is used.
	//
	// Writes are done atomically during marking. Reads are
	// non-atomic and lock-free since they only occur during
	// sweeping (and hence never race with writes).
	//
	// This is used to quickly find whole spans that can be freed.
	//
	// TODO(austin): It would be nice if this was uint64 for
	// faster scanning, but we don't have 64-bit atomic bit
	// operations.
	pageMarks [pagesPerArena / 8]uint8

	// pageSpecials is a bitmap that indicates which spans have
	// specials (finalizers or other). Like pageInUse, only the bit
	// corresponding to the first page in each span is used.
	//
	// Writes are done atomically whenever a special is added to
	// a span and whenever the last special is removed from a span.
	// Reads are done atomically to find spans containing specials
	// during marking.
	pageSpecials [pagesPerArena / 8]uint8

	// checkmarks stores the debug.gccheckmark state. It is only
	// used if debug.gccheckmark > 0.
	checkmarks *checkmarksMap

	// zeroedBase marks the first byte of the first page in this
	// arena which hasn't been used yet and is therefore already
	// zero. zeroedBase is relative to the arena base.
	// Increases monotonically until it hits heapArenaBytes.
	//
	// This field is sufficient to determine if an allocation
	// needs to be zeroed because the page allocator follows an
	// address-ordered first-fit policy.
	//
	// Read atomically and written with an atomic CAS.
	zeroedBase uintptr
}
```

mheap

```go

// Main malloc heap.
// The heap itself is the "free" and "scav" treaps,
// but all the other global data is here too.
//
// mheap must not be heap-allocated because it contains mSpanLists,
// which must not be heap-allocated.
type mheap struct {
	_ sys.NotInHeap

	// lock must only be acquired on the system stack, otherwise a g
	// could self-deadlock if its stack grows with the lock held.
	lock mutex

	pages pageAlloc // page allocation data structure

	sweepgen uint32 // sweep generation, see comment in mspan; written during STW

	// allspans is a slice of all mspans ever created. Each mspan
	// appears exactly once.
	//
	// The memory for allspans is manually managed and can be
	// reallocated and move as the heap grows.
	//
	// In general, allspans is protected by mheap_.lock, which
	// prevents concurrent access as well as freeing the backing
	// store. Accesses during STW might not hold the lock, but
	// must ensure that allocation cannot happen around the
	// access (since that may free the backing store).
	allspans []*mspan // all spans out there

	// Proportional sweep
	//
	// These parameters represent a linear function from gcController.heapLive
	// to page sweep count. The proportional sweep system works to
	// stay in the black by keeping the current page sweep count
	// above this line at the current gcController.heapLive.
	//
	// The line has slope sweepPagesPerByte and passes through a
	// basis point at (sweepHeapLiveBasis, pagesSweptBasis). At
	// any given time, the system is at (gcController.heapLive,
	// pagesSwept) in this space.
	//
	// It is important that the line pass through a point we
	// control rather than simply starting at a 0,0 origin
	// because that lets us adjust sweep pacing at any time while
	// accounting for current progress. If we could only adjust
	// the slope, it would create a discontinuity in debt if any
	// progress has already been made.
	pagesInUse         atomic.Uintptr // pages of spans in stats mSpanInUse
	pagesSwept         atomic.Uint64  // pages swept this cycle
	pagesSweptBasis    atomic.Uint64  // pagesSwept to use as the origin of the sweep ratio
	sweepHeapLiveBasis uint64         // value of gcController.heapLive to use as the origin of sweep ratio; written with lock, read without
	sweepPagesPerByte  float64        // proportional sweep ratio; written with lock, read without

	// Page reclaimer state

	// reclaimIndex is the page index in allArenas of next page to
	// reclaim. Specifically, it refers to page (i %
	// pagesPerArena) of arena allArenas[i / pagesPerArena].
	//
	// If this is >= 1<<63, the page reclaimer is done scanning
	// the page marks.
	reclaimIndex atomic.Uint64

	// reclaimCredit is spare credit for extra pages swept. Since
	// the page reclaimer works in large chunks, it may reclaim
	// more than requested. Any spare pages released go to this
	// credit pool.
	reclaimCredit atomic.Uintptr

	// arenas is the heap arena map. It points to the metadata for
	// the heap for every arena frame of the entire usable virtual
	// address space.
	//
	// Use arenaIndex to compute indexes into this array.
	//
	// For regions of the address space that are not backed by the
	// Go heap, the arena map contains nil.
	//
	// Modifications are protected by mheap_.lock. Reads can be
	// performed without locking; however, a given entry can
	// transition from nil to non-nil at any time when the lock
	// isn't held. (Entries never transitions back to nil.)
	//
	// In general, this is a two-level mapping consisting of an L1
	// map and possibly many L2 maps. This saves space when there
	// are a huge number of arena frames. However, on many
	// platforms (even 64-bit), arenaL1Bits is 0, making this
	// effectively a single-level map. In this case, arenas[0]
	// will never be nil.
	arenas [1 << arenaL1Bits]*[1 << arenaL2Bits]*heapArena

	// arenasHugePages indicates whether arenas' L2 entries are eligible
	// to be backed by huge pages.
	arenasHugePages bool

	// heapArenaAlloc is pre-reserved space for allocating heapArena
	// objects. This is only used on 32-bit, where we pre-reserve
	// this space to avoid interleaving it with the heap itself.
	heapArenaAlloc linearAlloc

	// arenaHints is a list of addresses at which to attempt to
	// add more heap arenas. This is initially populated with a
	// set of general hint addresses, and grown with the bounds of
	// actual heap arena ranges.
	arenaHints *arenaHint

	// arena is a pre-reserved space for allocating heap arenas
	// (the actual arenas). This is only used on 32-bit.
	arena linearAlloc

	// allArenas is the arenaIndex of every mapped arena. This can
	// be used to iterate through the address space.
	//
	// Access is protected by mheap_.lock. However, since this is
	// append-only and old backing arrays are never freed, it is
	// safe to acquire mheap_.lock, copy the slice header, and
	// then release mheap_.lock.
	allArenas []arenaIdx

	// sweepArenas is a snapshot of allArenas taken at the
	// beginning of the sweep cycle. This can be read safely by
	// simply blocking GC (by disabling preemption).
	sweepArenas []arenaIdx

	// markArenas is a snapshot of allArenas taken at the beginning
	// of the mark cycle. Because allArenas is append-only, neither
	// this slice nor its contents will change during the mark, so
	// it can be read safely.
	markArenas []arenaIdx

	// curArena is the arena that the heap is currently growing
	// into. This should always be physPageSize-aligned.
	curArena struct {
		base, end uintptr
	}

	// central free lists for small size classes.
	// the padding makes sure that the mcentrals are
	// spaced CacheLinePadSize bytes apart, so that each mcentral.lock
	// gets its own cache line.
	// central is indexed by spanClass.
	central [numSpanClasses]struct {
		mcentral mcentral
		pad      [(cpu.CacheLinePadSize - unsafe.Sizeof(mcentral{})%cpu.CacheLinePadSize) % cpu.CacheLinePadSize]byte
	}

	spanalloc              fixalloc // allocator for span*
	cachealloc             fixalloc // allocator for mcache*
	specialfinalizeralloc  fixalloc // allocator for specialfinalizer*
	specialprofilealloc    fixalloc // allocator for specialprofile*
	specialReachableAlloc  fixalloc // allocator for specialReachable
	specialPinCounterAlloc fixalloc // allocator for specialPinCounter
	speciallock            mutex    // lock for special record allocators.
	arenaHintAlloc         fixalloc // allocator for arenaHints

	// User arena state.
	//
	// Protected by mheap_.lock.
	userArena struct {
		// arenaHints is a list of addresses at which to attempt to
		// add more heap arenas for user arena chunks. This is initially
		// populated with a set of general hint addresses, and grown with
		// the bounds of actual heap arena ranges.
		arenaHints *arenaHint

		// quarantineList is a list of user arena spans that have been set to fault, but
		// are waiting for all pointers into them to go away. Sweeping handles
		// identifying when this is true, and moves the span to the ready list.
		quarantineList mSpanList

		// readyList is a list of empty user arena spans that are ready for reuse.
		readyList mSpanList
	}

	unused *specialfinalizer // never set, just here to force the specialfinalizer type into DWARF
}
```


因为Go采用的是分级分配的方式，所以在heapArena中会按照不同级别进行划分。

Go中一组相同级别大小内存空间的组叫`mspan`，[[#^19dc6a|分级分配]]，第一行为一个`mspan`，第二行为一个`mspan`。

mspan的结构在代码中的定义如下

```go
type mspan struct {
	_    sys.NotInHeap
	next *mspan     // next span in list, or nil if none
	prev *mspan     // previous span in list, or nil if none
	list *mSpanList // For debugging. TODO: Remove.

	startAddr uintptr // address of first byte of span aka s.base()
	npages    uintptr // number of pages in span

	manualFreeList gclinkptr // list of free objects in mSpanManual spans

	// freeindex is the slot index between 0 and nelems at which to begin scanning
	// for the next free object in this span.
	// Each allocation scans allocBits starting at freeindex until it encounters a 0
	// indicating a free object. freeindex is then adjusted so that subsequent scans begin
	// just past the newly discovered free object.
	//
	// If freeindex == nelem, this span has no free objects.
	//
	// allocBits is a bitmap of objects in this span.
	// If n >= freeindex and allocBits[n/8] & (1<<(n%8)) is 0
	// then object n is free;
	// otherwise, object n is allocated. Bits starting at nelem are
	// undefined and should never be referenced.
	//
	// Object n starts at address n*elemsize + (start << pageShift).
	freeindex uintptr
	// TODO: Look up nelems from sizeclass and remove this field if it
	// helps performance.
	nelems uintptr // number of object in the span.

	// Cache of the allocBits at freeindex. allocCache is shifted
	// such that the lowest bit corresponds to the bit freeindex.
	// allocCache holds the complement of allocBits, thus allowing
	// ctz (count trailing zero) to use it directly.
	// allocCache may contain bits beyond s.nelems; the caller must ignore
	// these.
	allocCache uint64

	// allocBits and gcmarkBits hold pointers to a span's mark and
	// allocation bits. The pointers are 8 byte aligned.
	// There are three arenas where this data is held.
	// free: Dirty arenas that are no longer accessed
	//       and can be reused.
	// next: Holds information to be used in the next GC cycle.
	// current: Information being used during this GC cycle.
	// previous: Information being used during the last GC cycle.
	// A new GC cycle starts with the call to finishsweep_m.
	// finishsweep_m moves the previous arena to the free arena,
	// the current arena to the previous arena, and
	// the next arena to the current arena.
	// The next arena is populated as the spans request
	// memory to hold gcmarkBits for the next GC cycle as well
	// as allocBits for newly allocated spans.
	//
	// The pointer arithmetic is done "by hand" instead of using
	// arrays to avoid bounds checks along critical performance
	// paths.
	// The sweep will free the old allocBits and set allocBits to the
	// gcmarkBits. The gcmarkBits are replaced with a fresh zeroed
	// out memory.
	allocBits  *gcBits
	gcmarkBits *gcBits
	pinnerBits *gcBits // bitmap for pinned objects; accessed atomically

	// sweep generation:
	// if sweepgen == h->sweepgen - 2, the span needs sweeping
	// if sweepgen == h->sweepgen - 1, the span is currently being swept
	// if sweepgen == h->sweepgen, the span is swept and ready to use
	// if sweepgen == h->sweepgen + 1, the span was cached before sweep began and is still cached, and needs sweeping
	// if sweepgen == h->sweepgen + 3, the span was swept and then cached and is still cached
	// h->sweepgen is incremented by 2 after every GC

	sweepgen              uint32
	divMul                uint32        // for divide by elemsize
	allocCount            uint16        // number of allocated objects
	spanclass             spanClass     // size class and noscan (uint8)
	state                 mSpanStateBox // mSpanInUse etc; accessed atomically (get/set methods)
	needzero              uint8         // needs to be zeroed before allocation
	isUserArenaChunk      bool          // whether or not this span represents a user arena
	allocCountBeforeCache uint16        // a copy of allocCount that is stored just before this span is cached
	elemsize              uintptr       // computed from sizeclass or from npages
	limit                 uintptr       // end of data in span
	speciallock           mutex         // guards specials list and changes to pinnerBits
	specials              *special      // linked list of special records sorted by offset.
	userArenaChunkFree    addrRange     // interval for managing chunk allocation

	// freeIndexForScan is like freeindex, except that freeindex is
	// used by the allocator whereas freeIndexForScan is used by the
	// GC scanner. They are two fields so that the GC sees the object
	// is allocated only when the object and the heap bits are
	// initialized (see also the assignment of freeIndexForScan in
	// mallocgc, and issue 54596).
	freeIndexForScan uintptr
}
```

Go中一共有67个`mspan`

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_100400_612_0.png)

使用mspan进行分级分配的示意图：

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_100546_278_0.png)

```go
//runtime/sizeclasses.go

// class  bytes/obj  bytes/span  objects  tail waste  max waste  min align
//     1          8        8192     1024           0     87.50%          8
//     2         16        8192      512           0     43.75%         16
//     3         24        8192      341           8     29.24%          8
//     4         32        8192      256           0     21.88%         32
//     5         48        8192      170          32     31.52%         16
//     6         64        8192      128           0     23.44%         64
//     7         80        8192      102          32     19.07%         16
//     8         96        8192       85          32     15.95%         32
//     9        112        8192       73          16     13.56%         16
//    10        128        8192       64           0     11.72%        128
//    11        144        8192       56         128     11.82%         16
//    12        160        8192       51          32      9.73%         32
//    13        176        8192       46          96      9.59%         16
//    14        192        8192       42         128      9.25%         64
//    15        208        8192       39          80      8.12%         16
//    16        224        8192       36         128      8.15%         32
//    17        240        8192       34          32      6.62%         16
//    18        256        8192       32           0      5.86%        256
//    19        288        8192       28         128     12.16%         32
//    20        320        8192       25         192     11.80%         64
//    21        352        8192       23          96      9.88%         32
//    22        384        8192       21         128      9.51%        128
//    23        416        8192       19         288     10.71%         32
//    24        448        8192       18         128      8.37%         64
//    25        480        8192       17          32      6.82%         32
//    26        512        8192       16           0      6.05%        512
//    27        576        8192       14         128     12.33%         64
//    28        640        8192       12         512     15.48%        128
//    29        704        8192       11         448     13.93%         64
//    30        768        8192       10         512     13.94%        256
//    31        896        8192        9         128     15.52%        128
//    32       1024        8192        8           0     12.40%       1024
//    33       1152        8192        7         128     12.41%        128
//    34       1280        8192        6         512     15.55%        256
//    35       1408       16384       11         896     14.00%        128
//    36       1536        8192        5         512     14.00%        512
//    37       1792       16384        9         256     15.57%        256
//    38       2048        8192        4           0     12.45%       2048
//    39       2304       16384        7         256     12.46%        256
//    40       2688        8192        3         128     15.59%        128
//    41       3072       24576        8           0     12.47%       1024
//    42       3200       16384        5         384      6.22%        128
//    43       3456       24576        7         384      8.83%        128
//    44       4096        8192        2           0     15.60%       4096
//    45       4864       24576        5         256     16.65%        256
//    46       5376       16384        3         256     10.92%        256
//    47       6144       24576        4           0     12.48%       2048
//    48       6528       32768        5         128      6.23%        128
//    49       6784       40960        6         256      4.36%        128
//    50       6912       49152        7         768      3.37%        256
//    51       8192        8192        1           0     15.61%       8192
//    52       9472       57344        6         512     14.28%        256
//    53       9728       49152        5         512      3.64%        512
//    54      10240       40960        4           0      4.99%       2048
//    55      10880       32768        3         128      6.24%        128
//    56      12288       24576        2           0     11.45%       4096
//    57      13568       40960        3         256      9.99%        256
//    58      14336       57344        4           0      5.35%       2048
//    59      16384       16384        1           0     12.49%       8192
//    60      18432       73728        4           0     11.11%       2048
//    61      19072       57344        3         128      3.57%        128
//    62      20480       40960        2           0      6.87%       4096
//    63      21760       65536        3         256      6.25%        256
//    64      24576       24576        1           0     11.45%       8192
//    65      27264       81920        3         128     10.00%        128
//    66      28672       57344        2           0      4.91%       4096
//    67      32768       32768        1           0     12.50%       8192

// alignment  bits  min obj size
//         8     3             8
//        16     4            32
//        32     5           256
//        64     6           512
//       128     7           768
//      4096    12         28672
//      8192    13         32768

```

为了记录mspan的分配状态，需要给mspan做索引，索引的结构叫mcentral

mcentral结构如下：

```go

// Central list of free objects of a given size.
type mcentral struct {
	_         sys.NotInHeap
	spanclass spanClass

	// partial and full contain two mspan sets: one of swept in-use
	// spans, and one of unswept in-use spans. These two trade
	// roles on each GC cycle. The unswept set is drained either by
	// allocation or by the background sweeper in every GC cycle,
	// so only two roles are necessary.
	//
	// sweepgen is increased by 2 on each GC cycle, so the swept
	// spans are in partial[sweepgen/2%2] and the unswept spans are in
	// partial[1-sweepgen/2%2]. Sweeping pops spans from the
	// unswept set and pushes spans that are still in-use on the
	// swept set. Likewise, allocating an in-use span pushes it
	// on the swept set.
	//
	// Some parts of the sweeper can sweep arbitrary spans, and hence
	// can't remove them from the unswept set, but will add the span
	// to the appropriate swept list. As a result, the parts of the
	// sweeper and mcentral that do consume from the unswept list may
	// encounter swept spans, and these should be ignored.
	partial [2]spanSet // list of spans with a free object
	full    [2]spanSet // list of spans with no free objects
}
```

一共136个mcentral，其中68个需要gc扫描，68个不需要gc扫描

![](https://gitlab.com/lzk97224/imgs/raw/main/2023/09/20230912_101946_169_0.png)

mcentral的性能问题

mcentral是中心索引，修改的时候需要加互斥锁，在高并发下存在严重的竞争，导致性能问题。

线程缓存mcache

每个p拥有一个mcache

```go
// Per-thread (in Go, per-P) cache for small objects.
// This includes a small object cache and local allocation stats.
// No locking needed because it is per-thread (per-P).
//
// mcaches are allocated from non-GC'd memory, so any heap pointers
// must be specially handled.
type mcache struct {
	_ sys.NotInHeap

	// The following members are accessed on every malloc,
	// so they are grouped here for better caching.
	nextSample uintptr // trigger heap sample after allocating this many bytes
	scanAlloc  uintptr // bytes of scannable heap allocated

	// Allocator cache for tiny objects w/o pointers.
	// See "Tiny allocator" comment in malloc.go.

	// tiny points to the beginning of the current tiny block, or
	// nil if there is no current tiny block.
	//
	// tiny is a heap pointer. Since mcache is in non-GC'd memory,
	// we handle it by clearing it in releaseAll during mark
	// termination.
	//
	// tinyAllocs is the number of tiny allocations performed
	// by the P that owns this mcache.
	tiny       uintptr
	tinyoffset uintptr
	tinyAllocs uintptr

	// The rest is not accessed on every malloc.

	alloc [numSpanClasses]*mspan // spans to allocate from, indexed by spanClass

	stackcache [_NumStackOrders]stackfreelist

	// flushGen indicates the sweepgen during which this mcache
	// was last flushed. If flushGen != mheap_.sweepgen, the spans
	// in this mcache are stale and need to the flushed so they
	// can be swept. This is done in acquirep.
	flushGen atomic.Uint32
}
```

### 内存分配逻辑

#### 对象分级

- 微对像(0,16B)
- 小对象\[16B,32KB\]
- 大对象(32kB,+∞)

微小对象分配到普通mspan

大对象量身定做mspan，就是0级的mspan

#### 微对象分配流程

- 从mcache中拿到2级的mspan
- 把多个微对象合并成一个16B存入

> 分配内存的代码在 runtime/malloc.go 的 mallocgc函数查看



## 垃圾回收

### 垃圾回收分类

- 标记-清除
- 标记-整理
- 复制
- 三色标记

### 标记清除
标记起点-root set
- 被栈上的指针引用
- 被全局变量指针引用
- 被寄存器中的指针引用

### 三色标记法

- 黑色：有用，已经分析扫描
- 灰色：有用，还未分析扫描
- 白色：暂时无用

初始状态都是白色
找到root节点，root节点标记为灰色
分析root节点的下级节点，root节点标记为黑色，root节点的下级节点有用的标记为灰色
依次分析下一级节点
最后剩下的白色节点，就是无用的节点，清除白色的节点。

**删除屏障**

==指针释放的白色，标记为灰色==，保证被释放的白色节点，因为后续又指给黑色的节点，导致最终被误删除。

**插入屏障**

==对指针新指向的节点，标记为灰色== ，保证新插入的节点是黑色节点的子节点，导致无删除。

## GC优化

### GC 触发的时机

- 系统定时触发
- 用户显式触发
- 申请内存时触发

#### 系统定时触发

```go
//runtime/proc.go
func sysmon()   //定时触发的函数

var forcegcperiod int64 = 2 * 60 * 1e9  //定时触发的时间

```

#### 用户显式触发

`runtime.GC`

#### 申请内存的时候触发

mallocgc函数申请内存的时候



### GC优化原则

- 尽量少在堆上产生垃圾
	- 内存池化
	- 减少逃逸
	- 使用空结构体

### GC优化工具

- go tool pprof
- go tool trace
- go build -gcflags = " -m"
- GODEBUG=" gctrace=1"