package cqueue

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

var msqPointerPool *sync.Pool = &sync.Pool{New: func() interface{} { return new(msqNodePointer) }}

type MSQPointer struct {
	head unsafe.Pointer // *msqNode
	tail unsafe.Pointer // *msqNode
}

type msqNodePointer struct {
	value unsafe.Pointer
	next  unsafe.Pointer // *msqNode
}

func NewMSQPointer() *MSQPointer {
	node := unsafe.Pointer(new(msqNodePointer))
	return &MSQPointer{head: node, tail: node}
}

func (q *MSQPointer) Enqueue(value unsafe.Pointer) bool {
	node := &msqNodePointer{value: value}
	for {
		tail := atomic.LoadPointer(&q.tail)
		tailstruct := (*msqNodePointer)(tail)
		next := atomic.LoadPointer(&tailstruct.next)
		if tail == atomic.LoadPointer(&q.tail) {
			if next == nil {
				// tail.next is empty, inset new node.
				if atomic.CompareAndSwapPointer(&tailstruct.next, next, unsafe.Pointer(node)) {
					atomic.CompareAndSwapPointer(&q.tail, tail, unsafe.Pointer(node))
					break
				}
			} else {
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			}
		}
	}
	return true
}

func (q *MSQPointer) Dequeue() (value unsafe.Pointer, ok bool) {
	for {
		head := atomic.LoadPointer(&q.head)
		tail := atomic.LoadPointer(&q.tail)
		headstruct := (*msqNodePointer)(head)
		next := atomic.LoadPointer(&headstruct.next)
		if head == atomic.LoadPointer(&q.head) {
			if head == tail {
				if next == nil {
					return nil, false
				}
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			} else {
				value = ((*msqNodePointer)(next)).value
				if atomic.CompareAndSwapPointer(&q.head, head, next) {
					return value, true
				}
			}
		}
	}
}

var msqUint64Pool *sync.Pool = &sync.Pool{New: func() interface{} { return new(msqNodeUint64) }}

type MSQUint64 struct {
	head unsafe.Pointer // *msqNode
	tail unsafe.Pointer // *msqNode
}

type msqNodeUint64 struct {
	value uint64
	next  unsafe.Pointer // *msqNode
}

func NewMSQUint64() *MSQUint64 {
	node := unsafe.Pointer(new(msqNodeUint64))
	return &MSQUint64{head: node, tail: node}
}

func loadMSQPointer(p *unsafe.Pointer) *msqNodeUint64 {
	return (*msqNodeUint64)(atomic.LoadPointer(p))
}

func (q *MSQUint64) Enqueue(value uint64) bool {
	node := &msqNodeUint64{value: value}
	for {
		tail := atomic.LoadPointer(&q.tail)
		tailstruct := (*msqNodeUint64)(tail)
		next := atomic.LoadPointer(&tailstruct.next)
		if tail == atomic.LoadPointer(&q.tail) {
			if next == nil {
				// tail.next is empty, inset new node.
				if atomic.CompareAndSwapPointer(&tailstruct.next, next, unsafe.Pointer(node)) {
					atomic.CompareAndSwapPointer(&q.tail, tail, unsafe.Pointer(node))
					break
				}
			} else {
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			}
		}
	}
	return true
}

func (q *MSQUint64) Dequeue() (value uint64, ok bool) {
	for {
		head := atomic.LoadPointer(&q.head)
		tail := atomic.LoadPointer(&q.tail)
		headstruct := (*msqNodeUint64)(head)
		next := atomic.LoadPointer(&headstruct.next)
		if head == atomic.LoadPointer(&q.head) {
			if head == tail {
				if next == nil {
					return 0, false
				}
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			} else {
				value = ((*msqNodeUint64)(next)).value
				if atomic.CompareAndSwapPointer(&q.head, head, next) {
					return value, true
				}
			}
		}
	}
}
