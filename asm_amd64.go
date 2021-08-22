// +build amd64,!gccgo,!appengine

package cqueue

import "unsafe"

type uint128 [2]uint64

func loadUint128(addr *uint128) (val uint128)

func loadSCQNodePointer(addr unsafe.Pointer) (val scqNodePointer)

func loadSCQNodeUint64(addr unsafe.Pointer) (val scqNodeUint64)

func resetNode(addr unsafe.Pointer)
