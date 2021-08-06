// +build amd64,!gccgo,!appengine

#include "textflag.h"

TEXT ·compareAndSwapUint128(SB),NOSPLIT,$0
	MOVQ addr+0(FP), R8
	MOVQ old+8(FP), AX
	MOVQ old+16(FP), DX
	MOVQ new+24(FP), BX
	MOVQ new+32(FP), CX
	LOCK
	CMPXCHG16B (R8)
	SETEQ swapped+40(FP)
	RET

TEXT ·loadUint128(SB),NOSPLIT,$0
	MOVQ addr+0(FP), R8
	XORQ AX, AX
	XORQ DX, DX
	XORQ BX, BX
	XORQ CX, CX
	LOCK
	CMPXCHG16B (R8)
	MOVQ AX, val+8(FP)
	MOVQ DX, val+16(FP)
	RET

TEXT ·loadSCQNodeUint64(SB),NOSPLIT,$0
	JMP ·loadUint128(SB)

TEXT ·loadCRQNodeUint64(SB),NOSPLIT,$0
	JMP ·loadUint128(SB)

TEXT ·loadSCQNodePointer(SB),NOSPLIT,$0
	JMP ·loadUint128(SB)

TEXT ·compareAndSwapSCQNodePointerBase(SB),NOSPLIT,$0
	JMP ·compareAndSwapUint128(SB)

TEXT ·compareAndSwapSCQNodeUint64(SB),NOSPLIT,$0
	JMP ·compareAndSwapUint128(SB)

TEXT ·compareAndSwapCRQNodeUint64(SB),NOSPLIT,$0
	JMP ·compareAndSwapUint128(SB)

TEXT ·resetNode(SB),NOSPLIT,$0
	MOVQ addr+0(FP), DX
	LOCK
	BTSQ $62, (DX)
	MOVQ $0, 8(DX)
	RET

TEXT ·runtimeEnableWriteBarrier(SB),NOSPLIT,$0
	MOVL runtime·writeBarrier(SB), AX
	MOVB AX, res+0(FP)
	RET
