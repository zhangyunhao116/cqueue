package cqueue

import "unsafe"

// func sync_atomic_CompareAndSwapPointer(ptr *unsafe.Pointer, old, new unsafe.Pointer) bool {
// 	if writeBarrier.enabled {
// 		atomicwb(ptr, new)
// 	}
// 	return sync_atomic_CompareAndSwapUintptr((*uintptr)(noescape(unsafe.Pointer(ptr))), uintptr(old), uintptr(new))
// }
func compareAndSwapSCQNodePointer(addr *scqNodePointer, old, new scqNodePointer) (swapped bool) {
	if runtimeEnableWriteBarrier() {
		runtimeatomicwb((*unsafe.Pointer)(&addr.data), new.data)
	}
	return compareAndSwapSCQNodePointerBase((*scqNodePointer)(runtimenoescape(unsafe.Pointer(addr))), old, new)
}

func runtimeEnableWriteBarrier() bool

//go:linkname runtimeatomicwb runtime.atomicwb
func runtimeatomicwb(ptr *unsafe.Pointer, new unsafe.Pointer)

//go:linkname runtimenoescape runtime.noescape
func runtimenoescape(p unsafe.Pointer) unsafe.Pointer
