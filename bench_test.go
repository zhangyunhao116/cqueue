package cqueue

import (
	"testing"

	"github.com/zhangyunhao116/fastrand"
)

type uint64queue interface {
	Enqueue(uint64) bool
	Dequeue() (uint64, bool)
}

type benchTask struct {
	name string
	New  func() uint64queue
}

func BenchmarkUint64(b *testing.B) {
	all := []benchTask{{
		name: "LSCQ", New: func() uint64queue {
			return NewLSCQUint64()
		}}}
	all = append(all, benchTask{
		name: "LinkedQueue",
		New: func() uint64queue {
			return NewLQUint64()
		},
	})
	all = append(all, benchTask{
		name: "MSQueue",
		New: func() uint64queue {
			return NewMSQUint64()
		},
	})
	benchEnqueueOnly(b, all)
	benchDequeueOnlyEmpty(b, all)
	benchPair(b, all)
	bench50Enqueue50Dequeue(b, all)
	bench30Enqueue70Dequeue(b, all)
	bench70Enqueue30Dequeue(b, all)
}

func reportalloc(b *testing.B) {
	// b.SetBytes(8)
	// b.ReportAllocs()
}

func benchPair(b *testing.B, benchTasks []benchTask) {
	for _, v := range benchTasks {
		b.Run("Pair/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					q.Enqueue(uint64(fastrand.Uint32()))
					q.Dequeue()
				}
			})
		})
	}
}

func bench50Enqueue50Dequeue(b *testing.B, benchTasks []benchTask) {
	for _, v := range benchTasks {
		b.Run("50Enqueue50Dequeue/"+v.name, func(b *testing.B) {
			q := v.New()
			b.ResetTimer()
			reportalloc(b)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if fastrand.Uint32n(2) == 0 {
						q.Enqueue(uint64(fastrand.Uint32()))
					} else {
						q.Dequeue()
					}
				}
			})
		})
	}
}

func bench70Enqueue30Dequeue(b *testing.B, benchTasks []benchTask) {
	for _, v := range benchTasks {
		b.Run("70Enqueue30Dequeue/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if fastrand.Uint32n(10) > 2 {
						q.Enqueue(uint64(fastrand.Uint32()))
					} else {
						q.Dequeue()
					}
				}
			})
		})
	}
}

func bench30Enqueue70Dequeue(b *testing.B, benchTasks []benchTask) {
	for _, v := range benchTasks {
		b.Run("30Enqueue70Dequeue/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if fastrand.Uint32n(10) <= 2 {
						q.Enqueue(uint64(fastrand.Uint32()))
					} else {
						q.Dequeue()
					}
				}
			})
		})
	}
}

func benchEnqueueOnly(b *testing.B, benchTasks []benchTask) {
	for _, v := range benchTasks {
		b.Run("EnqueueOnly/"+v.name, func(b *testing.B) {
			q := v.New()
			reportalloc(b)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					q.Enqueue(uint64(fastrand.Uint32()))
				}
			})
		})
	}
}

func benchDequeueOnlyEmpty(b *testing.B, benchTasks []benchTask) {
	for _, v := range benchTasks {
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
