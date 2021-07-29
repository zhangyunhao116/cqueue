package cqueue

import "sync"

type LQUint64 struct {
	head *lqNodeUint64
	tail *lqNodeUint64
	mu   sync.Mutex
}

type lqNodeUint64 struct {
	value uint64
	next  *lqNodeUint64
}

func NewLQUint64() *LQUint64 {
	node := new(lqNodeUint64)
	return &LQUint64{head: node, tail: node}
}

func (q *LQUint64) Enqueue(value uint64) bool {
	q.mu.Lock()
	q.tail.next = &lqNodeUint64{value: value}
	q.tail = q.tail.next
	q.mu.Unlock()
	return true
}

func (q *LQUint64) Dequeue() (uint64, bool) {
	q.mu.Lock()
	if q.head.next == nil {
		q.mu.Unlock()
		return 0, false
	} else {
		value := q.head.next.value
		q.head = q.head.next
		q.mu.Unlock()
		return value, true
	}
}
