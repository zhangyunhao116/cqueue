package cqueue

import (
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/zhangyunhao116/fastrand"
	"github.com/zhangyunhao116/skipset"
)

func TestRange(t *testing.T) {
	q := NewLSCQUint64()

	// Test traverse static queue.
	for _, count := range []int{10, scqsize, scqsize*2 + 100} {
		for i := 1; i < count; i++ {
			q.Enqueue(uint64(i))
		}
		// Traverse all items.
		var tmp int
		i := uint64(1)
		q.Range(func(data uint64) bool {
			tmp++
			if i != data {
				t.Fatal("times:", tmp, "testlen:", count, "want:", i, "got:", data)
			}
			i++
			return true
		})
		// Dequeue all items.
		for i := 1; i < count; i++ {
			data, ok := q.Dequeue()
			if !ok || uint64(i) != data {
				t.Fatal()
			}
		}
		// Traverse empty LSCQ.
		q.Range(func(_ uint64) bool {
			t.Fatal()
			return true
		})
	}

	// Test traverse dynamic queue(Range+enqueue).
	for _, count := range []int{10, scqsize, scqsize*2 + 100} {
		for i := 1; i <= count; i++ {
			q.Enqueue(uint64(i))
		}
		var (
			i uint64 = 1 // values saver
			j uint64     // items counter
		)
		q.Range(func(data uint64) bool {
			if i != data {
				t.Fatal(count, i, data)
			}
			if i == uint64(count) {
				q.Dequeue() // meaningless, because we have traverse the first item
				q.Enqueue(uint64(count) + 1)
			}
			i++
			j++
			return true
		})
		if j != uint64(count)+1 {
			t.Fatal()
		}
		// Clear the LSCQ.
		for {
			_, ok := q.Dequeue()
			if !ok {
				break
			}
		}
	}

	// Test traverse dynamic queue(Range+dequeue).
	for _, count := range []int{10, scqsize, scqsize*2 + 100} {
		for i := 1; i <= count; i++ {
			q.Enqueue(uint64(i))
		}
		var (
			clear           bool
			countAfterClear int
		)
		q.Range(func(_ uint64) bool {
			if !clear {
				for i := 0; i < count-1; i++ {
					q.Dequeue()
				}
				clear = true
				return true
			}
			if clear {
				countAfterClear++
			}
			return true
		})
		if countAfterClear != 1 {
			t.Fatal(count, countAfterClear)
		}

		// Clear the LSCQ.
		for {
			_, ok := q.Dequeue()
			if !ok {
				break
			}
		}
	}

	// Test traverse dynamic queue(Range+enqueue+dequeue).
	for i := 0; i < scqsize-3; i++ {
		q.Enqueue(uint64(10000 + i))
	}
	var s []uint64
	q.Enqueue(10)
	q.Enqueue(11)
	q.Enqueue(12)
	q.Enqueue(13)
	q.Range(func(data uint64) bool {
		if data >= 10000 {
			q.Dequeue()
			return true
		}
		s = append(s, data)
		if data == 10 {
			q.Dequeue() // 10
			q.Dequeue() // 11
			q.Dequeue() // 12
			return true
		}
		if data == 13 {
			q.Enqueue(14)
			q.Enqueue(15)
			return true
		}
		if data == 14 {
			q.Dequeue() // 15
		}
		return true
	})
	for i, v := range []uint64{10, 13, 14} {
		if s[i] != v {
			t.Fatal(i, s[i], v)
		}
	}
}

func TestLength(t *testing.T) {
	const count = 100000
	q := NewLSCQUint64()
	for i := 0; i < count; i++ {
		q.Enqueue(uint64(i))
	}
	if q.Len() != count {
		t.Fatal()
	}
	for i := 0; i < count; i++ {
		q.Dequeue()
	}
	if q.Len() != 0 {
		t.Fatal()
	}
}

func TestBoundedQueue(t *testing.T) {
	q := NewSCQUint64()
	s := skipset.NewUint64()

	// Dequeue empty queue.
	val, ok := q.Dequeue()
	if ok {
		t.Fatal(val)
	}
	if q.Len() != 0 {
		t.Fatal()
	}

	// Single goroutine correctness.
	for i := 0; i < scqsize; i++ {
		if !q.Enqueue(uint64(i)) {
			t.Fatal(i)
		}
		s.Add(uint64(i))
	}

	if q.Len() != s.Len() {
		t.Fatal()
	}

	if q.Enqueue(20) { // queue is full
		t.Fatal()
	}

	s.Range(func(value uint64) bool {
		if val, ok := q.Dequeue(); !ok || val != value {
			t.Fatal(val, ok, value)
		}
		return true
	})

	// Dequeue empty queue after previous loop.
	val, ok = q.Dequeue()
	if ok {
		t.Fatal(val)
	}
	if q.Len() != 0 {
		t.Fatal()
	}
	// ---------- MULTIPLE TEST BEGIN ----------.
	for j := 0; j < 10; j++ {
		s = skipset.NewUint64()

		// Dequeue empty queue.
		val, ok = q.Dequeue()
		if ok {
			t.Fatal(val)
		}

		// Single goroutine correctness.
		for i := 0; i < scqsize; i++ {
			if !q.Enqueue(uint64(i)) {
				t.Fatal()
			}
			s.Add(uint64(i))
		}

		if q.Enqueue(20) { // queue is full
			t.Fatal()
		}

		s.Range(func(value uint64) bool {
			if val, ok := q.Dequeue(); !ok || val != value {
				t.Fatal(val, ok, value)
			}
			return true
		})

		// Dequeue empty queue after previous loop.
		val, ok = q.Dequeue()
		if ok {
			t.Fatal(val)
		}
	}
	// ---------- MULTIPLE TEST END ----------.

	// MPMC correctness.
	var wg sync.WaitGroup
	s1 := skipset.NewUint64()
	s2 := skipset.NewUint64()
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func() {
			if fastrand.Uint32n(2) == 0 {
				r := fastrand.Uint64()
				if q.Enqueue(r) {
					s1.Add(r)
				}
			} else {
				val, ok := q.Dequeue()
				if ok {
					s2.Add(uint64(val))
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if s1.Len() > s2.Len() {
		if s1.Len()-s2.Len() != q.Len() {
			t.Fatal()
		}
	} else {
		if q.Len() != 0 {
			t.Fatal()
		}
	}

	for {
		val, ok := q.Dequeue()
		if !ok {
			break
		}
		s2.Add(uint64(val))
	}

	s1.Range(func(value uint64) bool {
		if !s2.Contains(value) {
			t.Fatal(value)
		}
		return true
	})

	if s1.Len() != s2.Len() {
		t.Fatal("invalid")
	}
}

func TestUnboundedQueue(t *testing.T) {
	// MPMC correctness.
	q := NewLSCQUint64()
	var wg sync.WaitGroup
	s1 := skipset.NewUint64()
	s2 := skipset.NewUint64()
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func() {
			if fastrand.Uint32n(2) == 0 {
				r := fastrand.Uint64()
				if !s1.Add(r) || !q.Enqueue(r) {
					panic("invalid")
				}
			} else {
				val, ok := q.Dequeue()
				if ok {
					s2.Add(uint64(val))
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if s1.Len() > s2.Len() {
		if s1.Len()-s2.Len() != q.Len() {
			t.Fatal()
		}
	} else {
		if q.Len() != 0 {
			t.Fatal()
		}
	}

	for {
		val, ok := q.Dequeue()
		if !ok {
			break
		}
		s2.Add(uint64(val))
	}

	s1.Range(func(value uint64) bool {
		if !s2.Contains(value) {
			t.Fatal(value)
		}
		return true
	})

	if s1.Len() != s2.Len() {
		t.Fatal("invalid")
	}
}

func TestUniqueUint64(t *testing.T) {
	const (
		cpucount = 16
	)
	var (
		wg        sync.WaitGroup
		shared    [cpucount * cacheLineSize]uint64
		sharedset [cpucount]*skipset.Uint64Set
	)
	q := NewLSCQUint64()
	for i := 0; i < cpucount; i++ {
		shared[i*int(cacheLineSize)] = uint64(i) * 1 << 35
		sharedset[i] = skipset.NewUint64()
	}
	for i := 0; i < cpucount; i++ {
		wg.Add(1)
		cpuid := uint64(i)
		go func() {
			for j := 0; j < scqsize*6; j++ {
				if !q.Enqueue(atomic.AddUint64(&shared[cpuid*uint64(cacheLineSize)], 1)) {
					panic("invalid")
				}
				data, ok := q.Dequeue()
				if !ok {
					panic("invalid")
				}
				datashared := data >> 35
				if !sharedset[datashared].Add(data) {
					panic("invalid")
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestUniquePointer(t *testing.T) {
	type dummy struct {
		v uint64
	}
	const (
		cpucount = 16
	)
	var (
		wg        sync.WaitGroup
		shared    [cpucount * cacheLineSize]uint64
		sharedset [cpucount]*skipset.Uint64Set
	)
	q := NewLSCQPointer()
	for i := 0; i < cpucount; i++ {
		shared[i*int(cacheLineSize)] = uint64(i) * 1 << 35
		sharedset[i] = skipset.NewUint64()
	}
	for i := 0; i < cpucount; i++ {
		wg.Add(1)
		cpuid := uint64(i)
		go func() {
			for j := 0; j < scqsize*6; j++ {
				tmp := new(dummy)
				tmp.v = atomic.AddUint64(&shared[cpuid*uint64(cacheLineSize)], 1)
				if !q.Enqueue(unsafe.Pointer(tmp)) {
					panic("invalid")
				}
				datap, ok := q.Dequeue()
				if !ok {
					panic("invalid")
				}
				data := (*dummy)(datap).v
				datashared := data >> 35
				if !sharedset[datashared].Add(data) {
					panic("invalid")
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
