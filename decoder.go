// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "unsafe"
)

/*
#cgo pkg-config: libfyaml
#include "callback.h"
*/
import "C"

type Decoder struct {
    opts *Options
    cmt *CMemTracker
    si SchemaImplementer
    root interface{}
    err error               // error in case of abnormal termination
}

// just forward to the internal cmem tracker
func (dec *Decoder) Allocate(size int) unsafe.Pointer {
    return dec.cmt.Allocate(size)
}

func (dec *Decoder) CString(str string) *C.char {
    return dec.cmt.CString(str)
}

func (dec *Decoder) Free(ptr unsafe.Pointer) {
    dec.cmt.Free(ptr)
}

func (dec *Decoder) IsTracked(ptr unsafe.Pointer) bool {
    return dec.cmt.IsTracked(ptr)
}

func (dec *Decoder) FreeAll() {
    dec.cmt.FreeAll()
}

func NewDecoder(opts...interface{}) (*Decoder, error) {

    var err error = nil

    // get the options if any
    o, err := GetOptions(opts); if err != nil {
        return nil, err
    }

    // empty parser object
    dec := &Decoder{
        opts: o,
        cmt: CMemTrackerCreate(),
    }

    return dec, nil
}

func (dec *Decoder) Destroy() {
    if dec == nil {
        return
    }

    // and the tracker
    dec.cmt.Destroy()
}

func (dec *Decoder) Debugf(format string, a ...interface{}) {
    if dec.opts.Debug {
        print(fmt.Sprintf(format, a...))
    }
}

func (dec *Decoder) SetError(err error) {
    dec.err = err
}

func (dec *Decoder) Error() error {
    return dec.err
}
