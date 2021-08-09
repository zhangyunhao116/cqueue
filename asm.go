package cqueue

import "unsafe"

// func sync_atomic_CompareAndSwapPointer(ptr *unsafe.Pointer, old, new unsafe.Pointer) bool {
// 	if writeBarrier.enabled {
// 		atomicwb(ptr, new)
// 	}
// 	return sync_atomic_CompareAndSwapUintptr((*uintptr)(noescape(unsafe.Pointer(ptr))), uintptr(old), uintptr(new))
// }
func compareAndSwapSCQNodePointer(addr *scqNodePointer, old, new scqNodePointer) (swapped bool) {
	// TODO: Reconsider the GC write barrier.
	if runtimeEnableWriteBarrier() {
		runtimeatomicwb(&addr.data, new.data)
	}
	// TODO: Pure assembly doesn't need `noescape`? since addr is a leaking param.
	return compareAndSwapSCQNodePointerBase(addr, old, new)
}

func runtimeEnableWriteBarrier() bool

//go:linkname runtimeatomicwb runtime.atomicwb
func runtimeatomicwb(ptr *unsafe.Pointer, new unsafe.Pointer)

//go:linkname runtimenoescape runtime.noescape
func runtimenoescape(p unsafe.Pointer) unsafe.Pointer
