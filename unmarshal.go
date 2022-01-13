// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "unsafe"
    "errors"
    gopointer "github.com/mattn/go-pointer"
)

/*
#cgo pkg-config: libfyaml
#include "callback.h"
*/
import "C"

func (dec *Decoder) unmarshalInternal(data []byte, filename string, v interface{}) error {

    // create the parser object
    p, err := ParserCreate(dec, dec.opts)
    if err != nil {
        return err
    }
    defer p.Destroy()

    if data != nil {
        // allocate the slice memory on the C side
        dataCopyC := dec.Allocate(len(data))
        defer dec.Free(dataCopyC)

        // make a go slice and copy the data there
        dataCopy := unsafe.Slice((*byte)(dataCopyC), len(data))
        copy(dataCopy, data)

        // and let it rip with the slice copy at the C side
        if err := p.SetInputData(dataCopyC, uint(len(data))); err != nil {
            return err
        }
    } else if filename != "" {

        // use the file
        if err := p.SetInputFile(filename); err != nil {
            return err
        }

    } else {
        return errors.New(fmt.Sprintf("failed to find unarshal method"))
    }

    dec.err = nil
    dec.si = nil
    dec.root = v

    // get a pointer for the unmarshaler object
    cp := gopointer.Save(dec)
    defer gopointer.Unref(cp)

    // note we don't abstract this internal parser interface
    rc := C.fy_parse_compose(p.C(), C.fy_parse_composer_cb(C.compose_process_event), cp)

    // is there a processor error get it and clear
    err = dec.err
    dec.err = nil

    if err != nil {
        return err
    }

    // no processor error, parser error?
    if !bool(C.fy_composer_return_is_ok(C.enum_fy_composer_return(rc))) {
        return errors.New(fmt.Sprintf("Failed on compose"))
    }

    dec.Debugf("return root: %T\n", v)

    return nil
}

func (dec *Decoder) Unmarshal(data []byte, v interface{}) error {
    return dec.unmarshalInternal(data, "", v)
}

func (dec *Decoder) UnmarshalFile(filename string, v interface{}) error {
    return dec.unmarshalInternal(nil, filename, v)
}

func Unmarshal(data []byte, v interface{}, opts...interface{}) error {

    var err error
    var o *Options
    var dec *Decoder

    // get the options if any
    if o, err = GetOptions(opts); err != nil {
        return err
    }

    // new decoder with options
    if dec, err = NewDecoder(o); err != nil {
        return err
    }
    defer dec.Destroy()

    // and let it rip
    return dec.Unmarshal(data, v)
}

func UnmarshalFile(filename string, v interface{}, opts...interface{}) error {

    var err error
    var o *Options
    var dec *Decoder

    // get the options if any
    if o, err = GetOptions(opts); err != nil {
        return err
    }

    // new decoder with options
    if dec, err = NewDecoder(o); err != nil {
        return err
    }
    defer dec.Destroy()

    // and let it rip
    return dec.UnmarshalFile(filename, v)
}

func (dec *Decoder) CollectionCreate(event *Event, path *Path) error {

    var err error

    var ow ObjectWrapper
    var cw CollectionWrapper

    et := event.Type()
    switch et {
    case DocumentStart:

        schema := dec.opts.Schema

        // select the schema
        dec.si, err = SelectSchema(schema, event, path)
        if err != nil {
            return err
        }

        // and start the unmarshaling
        cw, err = dec.si.DocumentStartUnmarshal(dec, dec.root, event, path)
        if err != nil {
            return err
        }

    case SequenceStart, MappingStart:
        // the prolog for the object

        cw = path.ParentUserData().(CollectionWrapper)
        ow, err = cw.ObjStartIn(event, path)
        if err != nil {
            return err
        }

        // the object is a collection
        cw = ow.(CollectionWrapper)

    default:
        panic(fmt.Sprintf("CollectionCreate() with %s event not allowed", event.Type()))
    }

    // and go process it
    err = cw.CollectionStart(event, path)
    if err != nil {
        return err
    }

        // save it for later use
    switch et {
    case DocumentStart:
        path.SetRootUserData(cw)

    case SequenceStart:
        path.LastComponent().SetSequenceUserData(cw)

    case MappingStart:
        path.LastComponent().SetMappingUserData(cw)
    }

    return nil
}

func (dec *Decoder) CollectionDestroy(event *Event, path *Path) error {

    var cw CollectionWrapper

    et := event.Type()
    switch et {
    case DocumentEnd:
        cw = path.RootUserData().(CollectionWrapper)

    case SequenceEnd:
        cw = path.LastComponent().SequenceUserData().(CollectionWrapper)

    case MappingEnd:
        cw = path.LastComponent().MappingUserData().(CollectionWrapper)

    default:
        panic(fmt.Sprintf("CollectionDestroy() with %s event not allowed", event.Type()))
    }

    if err := cw.CollectionEnd(event, path); err != nil {
        return err
    }

    switch et {
    case DocumentEnd:

        // no object end in for root
        if err := dec.si.DocumentEndUnmarshal(dec, event, path); err != nil {
            return err
        }
        dec.si = nil
        path.SetRootUserData(nil)

    case SequenceEnd:
        pcw := path.ParentUserData().(CollectionWrapper)
        if err := pcw.ObjEndIn(event, path, cw); err != nil {
            return err
        }

        path.LastComponent().SetSequenceUserData(nil)

    case MappingEnd:
        pcw := path.ParentUserData().(CollectionWrapper)
        if err := pcw.ObjEndIn(event, path, cw); err != nil {
            return err
        }

        path.LastComponent().SetMappingUserData(nil)

    default:
        panic(fmt.Sprintf("CollectionDestroy() with %s event not allowed", event.Type()))
    }

    return nil
}

func (dec *Decoder) Scalar(event *Event, path *Path) error {

    var err error
    var ow ObjectWrapper
    var sw ScalarWrapper

    cw := path.ParentUserData().(CollectionWrapper)

    // the prolog for the object
    ow, err = cw.ObjStartIn(event, path)
    if err != nil {
        return err
    }

    // the object is a scalar
    sw = ow.(ScalarWrapper)

    // set
    err = sw.SetScalar(event, path)
    if err != nil {
        return err
    }

    // scalars/alias end the reflection object
    return cw.ObjEndIn(event, path, ow)
}

func (dec *Decoder) ProcessEvent(event *Event, path *Path) (bool, error) {

    dec.Debugf("%v: %v\n", event, path)

    var err error = nil

    switch et := event.Type(); et {
    case StreamStart, StreamEnd:
        // nothing for now
        return false, nil

    case Scalar, Alias:
        err = dec.Scalar(event, path)

    case DocumentStart, SequenceStart, MappingStart:
        err = dec.CollectionCreate(event, path)

    case DocumentEnd, SequenceEnd, MappingEnd:
        err = dec.CollectionDestroy(event, path)

        // for document end, stop now, no multi documents yet
        if et == DocumentEnd && err == nil {
            return true, nil
        }
    }

    if err != nil {
        return true, err
    }

    return false, nil
}
