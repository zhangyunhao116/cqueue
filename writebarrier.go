package cqueue

import "unsafe"

// For LSCQPointer only.
func atomicWriteBarrier(ptr *unsafe.Pointer) {
	if runtimeEnableWriteBarrier() {
		runtimeatomicwb(ptr, nil)
	}
}
