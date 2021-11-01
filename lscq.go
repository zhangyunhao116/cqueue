package cqueue

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/zhangyunhao116/atomicx"
)

var pointerSCQPool = sync.Pool{
	New: func() interface{} {
		return NewSCQPointer()
	},
}

type LSCQPointer struct {
	head *SCQPointer
	_    [cacheLineSize - unsafe.Sizeof(new(uintptr))]byte
	tail *SCQPointer
}

func NewLSCQPointer() *LSCQPointer {
	q := NewSCQPointer()
	return &LSCQPointer{head: q, tail: q}
}

func (q *LSCQPointer) Dequeue() (data unsafe.Pointer, ok bool) {
	for {
		cq := (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head))))
		data, ok = cq.Dequeue()
		if ok {
			return
		}
		// cq does not have enough entries.
		nex := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next)))
		if nex == nil {
			// We don't have next SCQ.
			return
		}
		// cq.next is not empty, subsequent entry will be insert into cq.next instead of cq.
		// So if cq is empty, we can move it into ncqpool.
		atomic.StoreInt64(&cq.threshold, int64(scqsize*2)-1)
		data, ok = cq.Dequeue()
		if ok {
			return
		}
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)), (unsafe.Pointer(cq)), nex) {
			// We can't ensure no other goroutines will access cq.
			// The cq can still be previous dequeue's cq.
			cq = nil
		}
	}
}

func (q *LSCQPointer) Enqueue(data unsafe.Pointer) bool {
	for {
		cq := (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail))))
		nex := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next)))
		if nex != nil {
			// Help move cq.next into tail.
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), (unsafe.Pointer(cq)), nex)
			continue
		}
		if cq.Enqueue(data) {
			return true
		}
		// Concurrent cq is full.
		atomicx.OrUint64(&cq.tail, 1<<63) // close cq, subsequent enqueue will fail
		cq.mu.Lock()
		if atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next))) != nil {
			cq.mu.Unlock()
			continue
		}
		ncq := pointerSCQPool.Get().(*SCQPointer) // create a new queue
		ncq.Enqueue(data)
		// Try Add this queue into cq.next.
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next)), nil, unsafe.Pointer(ncq)) {
			// Success.
			// Try move cq.next into tail (we don't need to recheck since other enqueuer will help).
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(cq), unsafe.Pointer(ncq))
			cq.mu.Unlock()
			return true
		}
		// CAS failed, put this new SCQ into scqpool.
		// No other goroutines will access this queue.
		ncq.Dequeue()
		pointerSCQPool.Put(ncq)
		cq.mu.Unlock()
	}
}

// Len returns an *estimated* length of the LSCQ.
// WARNING: the returned length could be inaccurate even when
// none of the goroutines use it.
func (q *LSCQPointer) Len() int {
	var l int
	cq := (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head))))
	for {
		l += cq.Len()
		// Equal to `cq = cq.next`.
		cq = (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next))))
		if cq == nil {
			break
		}
	}
	return l
}

func (q *LSCQPointer) Range(f func(data unsafe.Pointer) bool) {
	cq := (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head))))
	for {
		cq.Range(f)
		// Equal to `cq = cq.next`.
		cq = (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next))))
		if cq == nil {
			break
		}
	}
}

func (q *LSCQPointer) Peek() (data unsafe.Pointer, ok bool) {
	cq := (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head))))
	for {
		data, ok = cq.Peek()
		if ok {
			return
		}
		// Equal to `cq = cq.next`.
		cq = (*SCQPointer)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cq.next))))
		if cq == nil {
			return
		}
	}
}

func NewSCQPointer() *SCQPointer {
	ring := new([scqsize]scqNodePointer)
	for i := range ring {
		ring[i].flags = 1<<63 + 1<<62 // newSCQFlags(true, true, 0)
	}
	return &SCQPointer{
		head:      scqsize,
		tail:      scqsize,
		threshold: -1,
		ring:      ring,
	}
}

type SCQPointer struct {
	_         [cacheLineSize]byte
	head      uint64
	_         [cacheLineSize - unsafe.Sizeof(new(uint64))]byte
	tail      uint64 // 1-bit finalize + 63-bit tail
	_         [cacheLineSize - unsafe.Sizeof(new(uint64))]byte
	threshold int64
	_         [cacheLineSize - unsafe.Sizeof(new(uint64))]byte
	next      *SCQPointer
	ring      *[scqsize]scqNodePointer
	mu        sync.Mutex
}

type scqNodePointer struct {
	flags uint64 // isSafe 1-bit + isEmpty 1-bit + cycle 62-bit
	data  unsafe.Pointer
}

func (q *SCQPointer) Enqueue(data unsafe.Pointer) bool {
	for {
		// Increment the TAIL, try to occupy an entry.
		tailvalue := atomic.AddUint64(&q.tail, 1)
		tailvalue -= 1 // we need previous value
		T := uint64Get63(tailvalue)
		if uint64Get1(tailvalue) {
			// The queue is closed, return false, so following enqueuer
			// will insert this data into next SCQ.
			return false
		}
		entAddr := &q.ring[cacheRemap16Byte(T)]
		cycleT := T / scqsize
	eqretry:
		// Enqueue do not need data, if this entry is empty, we can assume the data is also empty.
		entFlags := atomic.LoadUint64(&entAddr.flags)
		isSafe, isEmpty, cycleEnt := loadSCQFlags(entFlags)
		if cycleEnt < cycleT && isEmpty && (isSafe || atomic.LoadUint64(&q.head) <= T) {
			// We can use this entry for adding new data if
			// 1. Tail's cycle is bigger than entry's cycle.
			// 2. It is empty.
			// 3. It is safe or tail >= head (There is enough space for this data)
			ent := scqNodePointer{flags: entFlags}
			newEnt := scqNodePointer{flags: newSCQFlags(true, false, cycleT), data: data}
			// Save input data into this entry.
			if !compareAndSwapSCQNodePointer(entAddr, ent, newEnt) {
				// Failed, do next retry.
				goto eqretry
			}
			// Success.
			if atomic.LoadInt64(&q.threshold) != (int64(scqsize)*2)-1 {
				atomic.StoreInt64(&q.threshold, (int64(scqsize)*2)-1)
			}
			return true
		}
		// Add a full queue check in the loop(CAS2).
		if T+1 >= atomic.LoadUint64(&q.head)+scqsize {
			// T is tail's value before FAA(1), latest tail is T+1.
			return false
		}
	}
}

func (q *SCQPointer) Dequeue() (data unsafe.Pointer, ok bool) {
	if atomic.LoadInt64(&q.threshold) < 0 {
		// Empty queue.
		return
	}

	for {
		// Decrement HEAD, try to release an entry.
		H := atomic.AddUint64(&q.head, 1)
		H -= 1 // we need previous value
		entAddr := &q.ring[cacheRemap16Byte(H)]
		cycleH := H / scqsize
	dqretry:
		ent := loadSCQNodePointer(unsafe.Pointer(entAddr))
		isSafe, isEmpty, cycleEnt := loadSCQFlags(ent.flags)
		if cycleEnt == cycleH { // same cycle, return this entry directly
			// 1. Clear the data in this slot.
			// 2. Set `isEmpty` to 1
			atomicWriteBarrier(&entAddr.data)
			resetNode(unsafe.Pointer(entAddr))
			return ent.data, true
		}
		if cycleEnt < cycleH {
			var newEnt scqNodePointer
			if isEmpty {
				newEnt = scqNodePointer{flags: newSCQFlags(isSafe, true, cycleH)}
			} else {
				newEnt = scqNodePointer{flags: newSCQFlags(false, false, cycleEnt), data: ent.data}
			}
			if !compareAndSwapSCQNodePointer(entAddr, ent, newEnt) {
				goto dqretry
			}
		}
		// Check if the queue is empty.
		tailvalue := atomic.LoadUint64(&q.tail)
		T := uint64Get63(tailvalue)
		if T <= H+1 {
			// Invalid state.
			q.fixstate(H + 1)
			atomic.AddInt64(&q.threshold, -1)
			return
		}
		if atomic.AddInt64(&q.threshold, -1)+1 <= 0 {
			return
		}
	}
}

func (q *SCQPointer) fixstate(originalHead uint64) {
	for {
		head := atomic.LoadUint64(&q.head)
		if originalHead < head {
			// The last dequeuer will be responsible for fixstate.
			return
		}
		tailvalue := atomic.LoadUint64(&q.tail)
		if tailvalue >= head {
			// The queue has been closed, or in normal state.
			return
		}
		if atomic.CompareAndSwapUint64(&q.tail, tailvalue, head) {
			return
		}
	}
}

// Len returns an *estimated* length of the SCQ.
// WARNING: the returned length could be inaccurate even when
// only one goroutine uses it.
func (q *SCQPointer) Len() int {
	_, tail := uint64GetAll(atomic.LoadUint64(&q.tail))
	head := atomic.LoadUint64(&q.head)
	if head >= tail {
		return 0
	}
	l := int(tail - head)
	if l > scqsize {
		return scqsize
	}
	return l
}

// Range !! Experimental API !!
func (q *SCQPointer) Range(f func(data unsafe.Pointer) bool) {
	head := atomic.LoadUint64(&q.head)
	_, tail := uint64GetAll(atomic.LoadUint64(&q.tail))
	if head >= tail {
		return
	}
	// Traverse until arriving latest tail.
	// [head]   is the first one.
	// [tail-1] is the last one.
rangeTraverse:
	traverseTimes := tail - head
	if traverseTimes > scqsize {
		traverseTimes = scqsize
	}
	cycleH := head / scqsize
	for i := uint64(0); i < traverseTimes; i++ {
		entAddr := &q.ring[cacheRemap16Byte(head+i)]
		ent := loadSCQNodePointer(unsafe.Pointer(entAddr))
		_, isEmpty, cycle := loadSCQFlags(ent.flags)
		if isEmpty {
			// This item has been dequeued.
			continue
		}
		if cycleH > cycle {
			nowhead := atomic.LoadUint64(&q.head)
			if nowhead >= tail {
				break
			}
			head = nowhead
			goto rangeTraverse
		}
		if !f(ent.data) {
			return
		}
	}
	closed, newtail := uint64GetAll(atomic.LoadUint64(&q.tail))
	if closed || newtail == tail {
		return
	} else {
		head = tail
		tail = newtail
		goto rangeTraverse
	}
}

// Peek !! Experimental API !!
func (q *SCQPointer) Peek() (data unsafe.Pointer, ok bool) {
	for {
		head := atomic.LoadUint64(&q.head)
		entAddr := &q.ring[cacheRemap16Byte(head)]
		ent := loadSCQNodePointer(unsafe.Pointer(entAddr))
		_, isEmpty, _ := loadSCQFlags(ent.flags)
		if !isEmpty {
			return ent.data, true
		}
		_, tail := uint64GetAll(atomic.LoadUint64(&q.tail))
		if head >= tail {
			return
		}
	}
}
