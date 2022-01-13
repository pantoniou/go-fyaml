// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "unsafe"
    // "errors"
    // gopointer "github.com/mattn/go-pointer"
)

/*
#cgo pkg-config: libfyaml
#include "callback.h"
*/
import "C"

type Encoder struct {
    opts *Options
    cmt *CMemTracker
    si SchemaImplementer
    sc *StructCache         // the cache of the decoded structs
    root interface{}
    err error               // error in case of abnormal termination
}

// just forward to the internal cmem tracker
func (enc *Encoder) Allocate(size int) unsafe.Pointer {
    return enc.cmt.Allocate(size)
}

func (enc *Encoder) CString(str string) *C.char {
    return enc.cmt.CString(str)
}

func (enc *Encoder) Free(ptr unsafe.Pointer) {
    enc.cmt.Free(ptr)
}

func (enc *Encoder) IsTracked(ptr unsafe.Pointer) bool {
    return enc.cmt.IsTracked(ptr)
}

func (enc *Encoder) FreeAll() {
    enc.cmt.FreeAll()
}

func NewEncoder(opts...interface{}) (*Encoder, error) {

    var err error = nil

    // get the options if any
    o, err := GetOptions(opts); if err != nil {
        return nil, err
    }

    enc := &Encoder{
        opts: o,
        cmt: CMemTrackerCreate(),
    }

    enc.sc = NewStructCache(enc)

    return enc, nil
}

func (enc *Encoder) Destroy() {
    if enc == nil {
        return
    }

    // and the tracker
    enc.cmt.Destroy()
}

func (enc *Encoder) Debugf(format string, a ...interface{}) {
    if enc.opts.Debug {
        print(fmt.Sprintf(format, a...))
    }
}

