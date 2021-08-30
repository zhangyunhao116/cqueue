package cqueue

import (
	"testing"
	"unsafe"

	"github.com/zhangyunhao116/fastrand"
)

type pointerqueue interface {
	Enqueue(unsafe.Pointer) bool
	Dequeue() (unsafe.Pointer, bool)
}

type benchTaskPointer struct {
	name string
	New  func() pointerqueue
}

func BenchmarkPointer(b *testing.B) {
	all := []benchTaskPointer{{
		name: "LSCQ", New: func() pointerqueue {
			return NewLSCQPointer()
		}}}
	all = append(all, benchTaskPointer{
		name: "LinkedQueue",
		New: func() pointerqueue {
			return NewLQPointer()
		},
	})
	all = append(all, benchTaskPointer{
		name: "MSQueue",
		New: func() pointerqueue {
			return NewMSQPointer()
		},
	})
	benchEnqueueOnlyPointer(b, all)
	benchDequeueOnlyEmptyPointer(b, all)
	benchPairPointer(b, all)
	bench50Enqueue50DequeuePointer(b, all)
	bench30Enqueue70DequeuePointer(b, all)
	bench70Enqueue30DequeuePointer(b, all)
}

func benchPairPointer(b *testing.B, benchTaskPointers []benchTaskPointer) {
	for _, v := range benchTaskPointers {
		b.Run("Pair/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					x := fastrand.Uint32()
					q.Enqueue(unsafe.Pointer(&x))
					q.Dequeue()
				}
			})
		})
	}
}

func bench50Enqueue50DequeuePointer(b *testing.B, benchTaskPointers []benchTaskPointer) {
	for _, v := range benchTaskPointers {
		b.Run("50Enqueue50Dequeue/"+v.name, func(b *testing.B) {
			q := v.New()
			b.ResetTimer()
			reportalloc(b)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if fastrand.Uint32n(2) == 0 {
						x := fastrand.Uint32()
						q.Enqueue(unsafe.Pointer(&x))
					} else {
						q.Dequeue()
					}
				}
			})
		})
	}
}

func bench70Enqueue30DequeuePointer(b *testing.B, benchTaskPointers []benchTaskPointer) {
	for _, v := range benchTaskPointers {
		b.Run("70Enqueue30Dequeue/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if fastrand.Uint32n(10) > 2 {
						x := fastrand.Uint32()
						q.Enqueue(unsafe.Pointer(&x))
					} else {
						q.Dequeue()
					}
				}
			})
		})
	}
}

func bench30Enqueue70DequeuePointer(b *testing.B, benchTaskPointers []benchTaskPointer) {
	for _, v := range benchTaskPointers {
		b.Run("30Enqueue70Dequeue/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if fastrand.Uint32n(10) <= 2 {
						x := fastrand.Uint32()
						q.Enqueue(unsafe.Pointer(&x))
					} else {
						q.Dequeue()
					}
				}
			})
		})
	}
}

func benchEnqueueOnlyPointer(b *testing.B, benchTaskPointers []benchTaskPointer) {
	for _, v := range benchTaskPointers {
		b.Run("EnqueueOnly/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					x := fastrand.Uint32()
					q.Enqueue(unsafe.Pointer(&x))
				}
			})
		})
	}
}

func benchDequeueOnlyEmptyPointer(b *testing.B, benchTaskPointers []benchTaskPointer) {
	for _, v := range benchTaskPointers {
		b.Run("DequeueOnlyEmpty/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					q.Dequeue()
				}
			})
		})
	}
}
