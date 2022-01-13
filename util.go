// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "errors"
    "reflect"
    "strings"
)

type DebugfProvider interface {
    Debugf(format string, a ...interface{})
}

type NullDebugfProvider struct {}

func (n NullDebugfProvider) Debugf(format string, a ...interface{}) {
    // nothing
}

func GetDebugfProvider(i interface{}) DebugfProvider {
    if i != nil {
        dp, hasDp := i.(DebugfProvider)
        if hasDp {
            return dp
        }
    }
    return NullDebugfProvider{}
}

type Field struct {
    name, fieldName string
    idx int
    omitempty, ignored, asString bool
}

type TypeInfo struct {
    t reflect.Type
    primed bool
    tagToField map[string]*Field    // when a match from tag to field was found
    fields []*Field
}

func (ti *TypeInfo) String() string {
    return fmt.Sprintf("%v", ti.t.String())
}

type StructCache struct {
    dp DebugfProvider
    types map[reflect.Type]*TypeInfo
}

func NewStructCache(i interface{}) *StructCache {
    return &StructCache{
        dp: GetDebugfProvider(i),
        types: make(map[reflect.Type]*TypeInfo),
    }
}

func (sc *StructCache) LookupType(t reflect.Type) *TypeInfo {
    if ti, ok := sc.types[t]; ok {
        return ti
    }
    return nil
}

func (sc *StructCache) NewType(t reflect.Type) *TypeInfo {
    ti := &TypeInfo {
        t: t,
        tagToField: make(map[string]*Field),
        fields: make([]*Field, t.NumField()),
    }
    sc.types[t] = ti
    ti.PrimeFieldCache()
    return ti
}

func (sc *StructCache) LookupOrNewType(t reflect.Type) *TypeInfo {
    ti := sc.LookupType(t)
    if ti != nil {
        return ti
    }
    return sc.NewType(t)
}

func (ti *TypeInfo) PrimeFieldCache() {

    if ti.primed {
        return
    }

    for i := 0; i < ti.t.NumField(); i++ {
        field := ti.t.Field(i)

        name := field.Name
        omitempty := false
        ignored := false
        asString := false

        if json, ok := field.Tag.Lookup("json"); ok {
            jsplit := strings.Split(json, ",")
            first := jsplit[0]
            if first == "-" && len(jsplit) == 1 {
                ignored = true
            } else {
                // use this name for match
                name = first

                for _, keyword := range(jsplit[1:]) {
                    switch keyword {
                    case "omitempty":
                        omitempty = true
                    case "string":
                        asString = true
                    }
                }
            }
        }

        f := &Field{
            name: name,
            fieldName: field.Name,
            idx: i,
            omitempty: omitempty,
            ignored: ignored,
            asString: asString,
        }

        // insert to the field cache
        ti.tagToField[name] = f
        ti.fields[i] = f
    }

    ti.primed = true
}

func (ti *TypeInfo) FieldByName(name string, rv *reflect.Value) (*reflect.Value, *Field) {

    // some sanity checks
    if rv == nil || !(*rv).IsValid() || (*rv).Kind() != reflect.Struct {
        panic(fmt.Sprintf("bad arguments on FieldByName() %s\n", name))
    }

    if !ti.primed {
        ti.PrimeFieldCache()
    }

    // we have a field info, use it (first try exact match)
    uf, ok := ti.tagToField[name]

    // not found? try with a capital first letter
    if !ok {
        tname := strings.Title(name)
        uf, ok = ti.tagToField[tname]
        // found? insert it so that we hit it next time
        if ok {
            ti.tagToField[tname] = uf
        }
    }

    // not found, or ignored
    if !ok || uf.ignored {
        return nil, nil
    }

    // OK, this is the one
    rvf := rv.Field(uf.idx)
    return &rvf, uf
}

func SettableValueOf(i interface{}) reflect.Value {
	v := reflect.ValueOf(i)
	sv := reflect.New(v.Type()).Elem()
	sv.Set(v)
	return sv
}

func IndirectPointer(rv *reflect.Value) (*reflect.Value, error) {

    if rv != nil && rv.Kind() == reflect.Ptr {
        // nil pointer, allocate a new one
        if rv.IsNil() {
            if !rv.CanSet() {
                return nil, errors.New(fmt.Sprintf("cannot allocate new pointer value: %v", rv.Kind()))
            }
            rv.Set(reflect.New(rv.Type().Elem()))
        }
        rvt := rv.Elem()
        if !rvt.IsValid() {
            return nil, errors.New(fmt.Sprintf("deref pointer value is invalid: %v"))
        }
        rv = &rvt
    }

    return rv, nil
}

func IsHashable(rv reflect.Value) bool {

    for {
        k := rv.Kind()
        if (k > reflect.Invalid && k < reflect.Array) ||
            k == reflect.Ptr || k == reflect.UnsafePointer || k == reflect.String {
            return true
        }
        if k != reflect.Interface {
            break
        }
        rv = rv.Elem()
    }

    return false
}

