// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "unsafe"
)

// #include <stdlib.h>
import "C"

type CMemTrackerAllocator interface {
    Allocate(size int) unsafe.Pointer
    CString(str string) *C.char
    IsTracked(ptr unsafe.Pointer) bool
    Free(ptr unsafe.Pointer)
    FreeAll()
}

// void structs are efficient sets
type CMemTrackerVoid struct {}
type CMemTracker map[unsafe.Pointer]CMemTrackerVoid

func CMemTrackerCreate() *CMemTracker {
    cmtv := make(CMemTracker)
    return &cmtv
}

func (cmt *CMemTracker) Destroy() {
    cmt.FreeAll()
    *cmt = nil  // make it crash if used after this
}

func (cmt *CMemTracker) Allocate(size int) unsafe.Pointer {
    ptr := C.calloc(1, C.size_t(size))
    if ptr == nil {
        panic(fmt.Sprintf("calloc wrapper failed to allocated %d bytes", size))
    }
    (*cmt)[ptr] = CMemTrackerVoid{}
    return ptr
}

func (cmt *CMemTracker) CString(str string) *C.char {
    ptr := C.CString(str)
    (*cmt)[unsafe.Pointer(ptr)] = CMemTrackerVoid{}
    return ptr
}

func (cmt *CMemTracker) IsTracked(ptr unsafe.Pointer) bool {
    _, exists := (*cmt)[ptr]
    return exists
}

func (cmt *CMemTracker) Free(ptr unsafe.Pointer) {

    // if we have to crash, crash with style
    if !cmt.IsTracked(ptr) {
        panic("double free of tracked pointer\n")
    }

    // delete from the tracker
    delete(*cmt, ptr)

    // and finally free
    C.free(ptr)
}

func (cmt *CMemTracker) FreeAll() {
    // iterate and remove all tracked pointers
    for ptr, _ := range *cmt {
        cmt.Free(ptr)
    }
}
