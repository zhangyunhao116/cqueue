// +build ignore

package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	f, err := os.Open("lscq.go")
	if err != nil {
		panic(err)
	}
	filedata, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}

	w := new(bytes.Buffer)
	w.WriteString(`// Code generated by go run types_gen.go; DO NOT EDIT.` + "\r\n")
	w.WriteString(string(filedata)[strings.Index(string(filedata), "package cqueue") : strings.Index(string(filedata), ")\n")+1])
	// ts := []string{"Float32", "Float64", "Int64", "Int32", "Int16", "Int", "Uint64", "Uint32", "Uint16", "Uint"} // all types need to be converted
	ts := []string{"Uint64"} // all types need to be converted
	for _, upper := range ts {
		lower := strings.ToLower(upper)
		data := string(filedata)
		// Remove header(imported packages).
		data = data[strings.Index(data, ")\n")+1:]
		// Common cases.
		data = strings.Replace(data, "atomic.StorePointer((*unsafe.Pointer)(ent.data), nil)", "", -1)
		data = strings.Replace(data, "NewPointer", "New"+upper, -1)
		data = strings.Replace(data, "data unsafe.Pointer", "data "+lower, -1)
		data = strings.Replace(data, "data  unsafe.Pointer", "data "+lower, -1)
		data = strings.Replace(data, "pointerSCQ", lower+"SCQ", -1)
		data = strings.Replace(data, "PointerSCQ", upper+"SCQ", -1)
		data = strings.Replace(data, "pointerQueue", lower+"Queue", -1)
		data = strings.Replace(data, "PointerQueue", upper+"Queue", -1)
		data = strings.Replace(data, "scqNodePointer", "scqNode"+upper, -1)
		data = strings.Replace(data, "compareAndSwapSCQNodePointer", "compareAndSwapSCQNode"+upper, -1)
		data = strings.Replace(data, "loadSCQNodePointer", "loadSCQNode"+upper, -1)
		// // Add the special case.
		// data = data + strings.Replace(lengthFunction, "Int64Set", upper+"Set", 1)
		w.WriteString(data)
		w.WriteString("\r\n")
	}

	out, err := format.Source(w.Bytes())
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("types.go", out, 0660); err != nil {
		panic(err)
	}
}

func lowerSlice(s []string) []string {
	n := make([]string, len(s))
	for i, v := range s {
		n[i] = strings.ToLower(v)
	}
	return n
}

func inSlice(s []string, val string) bool {
	for _, v := range s {
		if v == val {
			return true
		}
	}
	return false
}
