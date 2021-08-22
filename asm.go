package cqueue

import (
	"unsafe"

	"github.com/zhangyunhao116/atomicx"
)

// TODO: GC write barrier implementation.(DO NOT REMOVE THESE COMMENTS)
// func sync_atomic_CompareAndSwapPointer(ptr *unsafe.Pointer, old, new unsafe.Pointer) bool {
// 	if writeBarrier.enabled {
// 		atomicwb(ptr, new)
// 	}
// 	return sync_atomic_CompareAndSwapUintptr((*uintptr)(noescape(unsafe.Pointer(ptr))), uintptr(old), uintptr(new))
// }

//go:nosplit
func compareAndSwapSCQNodePointer(addr *scqNodePointer, old, new scqNodePointer) (swapped bool) {
	// TODO: Reconsider the GC write barrier.
	// For now, the addr and new will escape to heap.
	if runtimeEnableWriteBarrier() {
		runtimeatomicwb(&addr.data, new.data)
	}
	return atomicx.CompareAndSwapUint128((*atomicx.Uint128)(runtimenoescape(unsafe.Pointer(addr))), old.flags, uint64(uintptr(old.data)), new.flags, uint64(uintptr(new.data)))
}

func compareAndSwapSCQNodeUint64(addr *scqNodeUint64, old, new scqNodeUint64) (swapped bool) {
	return atomicx.CompareAndSwapUint128((*atomicx.Uint128)(unsafe.Pointer(addr)), old.flags, old.data, new.flags, new.data)
}

func runtimeEnableWriteBarrier() bool

//go:linkname runtimeatomicwb runtime.atomicwb
func runtimeatomicwb(ptr *unsafe.Pointer, new unsafe.Pointer)

//go:linkname runtimenoescape runtime.noescape
func runtimenoescape(p unsafe.Pointer) unsafe.Pointer
