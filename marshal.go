// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "reflect"
    "errors"
    "strconv"
    "fmt"
)

func (enc *Encoder) emitMarshalMap(e *Emitter, rv reflect.Value) error {
    if err := e.EmitEvent(MappingStart, AnyStyle, "", ""); err != nil {
        return err
    }
    for _, key := range rv.MapKeys() {
        if err := enc.emitMarshal(e, key); err != nil {
            return err
        }
        if err := enc.emitMarshal(e, rv.MapIndex(key)); err != nil {
            return err
        }
    }
    return e.EmitEvent(MappingEnd)
}

func (enc *Encoder) emitMarshalStruct(e *Emitter, rv reflect.Value) error {

    // lookup the type info
    ti := enc.sc.LookupOrNewType(rv.Type())
    if ti == nil {
        return errors.New(fmt.Sprintf("%s: could not lookup type %s\n", rv.Type()))
    }
    if err := e.EmitEvent(MappingStart, AnyStyle, "", ""); err != nil {
        return err
    }
    for i, f := range ti.fields {
        if err := enc.emitMarshal(e, reflect.ValueOf(f.name)); err != nil {
            return err
        }
        if err := enc.emitMarshal(e, rv.Field(i)); err != nil {
            return err
        }
    }
    return e.EmitEvent(MappingEnd)
}

func (enc *Encoder) emitMarshalSlice(e *Emitter, rv reflect.Value) error {
    if err := e.EmitEvent(SequenceStart, AnyStyle, "", ""); err != nil {
        return err
    }
    for i := 0; i < rv.Len(); i++ {
        if err := enc.emitMarshal(e, rv.Index(i)); err != nil {
            return err
        }
    }
    return e.EmitEvent(SequenceEnd)
}

func (enc *Encoder) emitMarshalString(e *Emitter, rv reflect.Value) error {
    // XXX schema string
    str := rv.String()
    return e.EmitEvent(Scalar, Any, str, "", "")
}

func (enc *Encoder) emitMarshalInt(e *Emitter, rv reflect.Value) error {
    str := strconv.FormatInt(rv.Int(), 10)
    return e.EmitEvent(Scalar, Plain, str, "", "")
}

func (enc *Encoder) emitMarshalUint(e *Emitter, rv reflect.Value) error {
    str := strconv.FormatUint(rv.Uint(), 10)
    return e.EmitEvent(Scalar, Plain, str, "", "")
}

func (enc *Encoder) emitMarshalFloat(e *Emitter, rv reflect.Value) error {
	p := 64
	if rv.Kind() == reflect.Float32 {
		p = 32
	}

	str := strconv.FormatFloat(rv.Float(), 'g', -1, p)
	switch str {
	case "+Inf":
		str = ".inf"
	case "-Inf":
		str = "-.inf"
	case "NaN":
		str = ".nan"
	}
    return e.EmitEvent(Scalar, Plain, str, "", "")
}

func (enc *Encoder) emitMarshalBool(e *Emitter, rv reflect.Value) error {
    var str string
    if rv.Bool() {
        str = "true"
    } else {
        str = "false"
    }
    return e.EmitEvent(Scalar, Plain, str, "", "")
}

func (enc *Encoder) emitMarshalNull(e *Emitter, rv reflect.Value) error {
    // XXX schema null
    var str string
    if !enc.jsonOutput {
        str = "~"       // by default emit ~
    } else {
        str = "null"    // or the JSON null
    }
    return e.EmitEvent(Scalar, Plain, str, "", "")
}

func (enc *Encoder) emitMarshal(e *Emitter, rv reflect.Value) error {

    if !rv.IsValid() || rv.Kind() == reflect.Ptr && rv.IsNil() {
        return enc.emitMarshalNull(e, rv)
    }

    // placeholder for interface
    iface := rv.Interface()
    switch v := iface.(type) {
    default:
        _ = v
    }

    switch rv.Kind() {
    case reflect.Interface, reflect.Ptr:
        return enc.emitMarshal(e, rv.Elem())
    case reflect.Map:
        return enc.emitMarshalMap(e, rv)
    case reflect.Struct:
        return enc.emitMarshalStruct(e, rv)
    case reflect.Slice, reflect.Array:
        return enc.emitMarshalSlice(e, rv)
    case reflect.String:
        return enc.emitMarshalString(e, rv)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return enc.emitMarshalInt(e, rv)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
        return enc.emitMarshalUint(e, rv)
	case reflect.Float32, reflect.Float64:
        return enc.emitMarshalFloat(e, rv)
    case reflect.Bool:
        return enc.emitMarshalBool(e, rv)
    }

    return errors.New("emit marshal can't handle type: " + rv.Type().String())
}

func (enc *Encoder) Marshal(v interface{}) ([]byte, error) {

    // create the emitter object to string
    e, err := EmitToString(enc, enc.opts)
    if err != nil {
        return nil, err
    }

    str := ""

    // stream start
    err = e.EmitEvent(StreamStart)
    if err != nil {
        goto err_out
    }

    // document start
    err = e.EmitEvent(DocumentStart, true, "", nil)
    if err != nil {
        goto err_out
    }

    // emit the document contents
    err = enc.emitMarshal(e, reflect.ValueOf(v))
    if err != nil {
        goto err_out
    }

    // document end
    err = e.EmitEvent(DocumentEnd, true)
    if err != nil {
        goto err_out
    }

    err = e.EmitEvent(StreamEnd)
    if err != nil {
        goto err_out
    }

    str = e.CollectStringAndDestroy()

    return []byte(str), nil

err_out:
    _ = e.CollectStringAndDestroy()
    return nil, err
}

func Marshal(v interface{}, opts...interface{}) ([]byte, error) {

    var err error
    var o *Options
    var enc *Encoder

    // get the options if any
    if o, err = GetOptions(opts); err != nil {
        return nil, err
    }

    // new encoder with options
    if enc, err = NewEncoder(o); err != nil {
        return nil, err
    }
    defer enc.Destroy()

    return enc.Marshal(v)
}
