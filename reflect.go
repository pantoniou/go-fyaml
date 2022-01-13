// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "errors"
    "reflect"
)

// type of []interface{}
var genericIfaceType = reflect.TypeOf((*(interface{}))(nil)).Elem()
var genericSeqType = reflect.SliceOf(genericIfaceType)
var genericMapType = reflect.MapOf(genericIfaceType, genericIfaceType)

type TypeServicesProvider interface {
    LookupType(t reflect.Type) *TypeInfo
    NewType(t reflect.Type) *TypeInfo
    LookupOrNewType(t reflect.Type) *TypeInfo
}

type Resolver interface {
    RegisterAnchor(anchor string, path *Path, ow ObjectWrapper, cw CollectionWrapper, aw AddressWrapper) error
    FindReference(anchor string) (ObjectWrapper, CollectionWrapper, AddressWrapper, error)
    // RegisterReference(reference string, path *Path) (bool, error)
    // ResolveReference(reference string, path *Path) error
}

type ResolverEntry struct {
    ow ObjectWrapper
    cw CollectionWrapper
    aw AddressWrapper
}

type RootState struct {
    dp DebugfProvider       // the debug logger provider
    root interface{}        // the root
    startRvt reflect.Value  // the actual root reflect value
    startRv *reflect.Value  // the start reflection value
    ow ObjectWrapper        // the root object
    ri *reflect.Value       // the interface of the root value
    rv *reflect.Value       // root reflect value
    sc *StructCache         // the cache of the decoded structs
    si SchemaImplementer    // our schema (if it exists)

    anchors map[string]*ResolverEntry
}

func NewRootState(event *Event, path *Path, root interface{}, si SchemaImplementer, dp DebugfProvider) (CollectionWrapper, error) {
    s := &RootState {
        root: root,
        startRvt: reflect.ValueOf(root),
        sc: NewStructCache(dp),
        si: si,
        dp: dp,
        anchors: make(map[string]*ResolverEntry),
    }
    s.startRv = &s.startRvt
    return s, nil
}

func NewSequenceStateDefault(event *Event, path *Path, startRv *reflect.Value, t TagHandler) (CollectionWrapper, error) {
    return &SequenceState {
        startRv: startRv,
        t: t,
        anchor: event.AnchorString(),
    }, nil
}

func NewMappingStateDefault(event *Event, path *Path, startRv *reflect.Value, t TagHandler) (CollectionWrapper, error) {
    return &MappingState {
        startRv: startRv,
        t: t,
        anchor: event.AnchorString(),
        dupf: make(map[*Field]uvoid),
    }, nil
}

func NewScalarStateDefault(event *Event, path *Path, startRv *reflect.Value, t TagHandler) (ScalarWrapper, error) {
    return &ScalarState {
        startRv: startRv,
        t: t,
        anchor: event.AnchorString(),
    }, nil
}

// implement the SchemaObjectCreator (from the si member if it exists)
func (s *RootState) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    return s.si.NewSchemaObject(event, path, startRv)
}

// implement the SchemaResolver (from the si member if it exists)
func (s *RootState) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    return s.si.ResolveScalar(tag, value, kind)
}

// implement the TypeServicesProvider interface
func (s *RootState) LookupType(t reflect.Type) *TypeInfo {
    return s.sc.LookupType(t)
}

func (s *RootState) NewType(t reflect.Type) *TypeInfo {
    return s.sc.NewType(t)
}

func (s *RootState) LookupOrNewType(t reflect.Type) *TypeInfo {
    return s.sc.LookupOrNewType(t)
}

// implement the Resolver interface
func (s *RootState) RegisterAnchor(anchor string, path *Path, ow ObjectWrapper, cw CollectionWrapper, aw AddressWrapper) error {

    s.dp.Debugf("%s: Anchor %s\n", path, anchor)

    s.anchors[anchor] = &ResolverEntry{
        ow: ow,
        cw: cw,
        aw: aw,
    }
    return nil
}

func (s *RootState) FindReference(anchor string) (ObjectWrapper, CollectionWrapper, AddressWrapper, error) {
    re, hasRe := s.anchors[anchor]
    if hasRe {
        return re.ow, re.cw, re.aw, nil
    }
    return nil, nil, nil, nil
}

// implement the DebugfProvider interface
func (s *RootState) Debugf(format string, a ...interface{}) {
    s.dp.Debugf(format, a...)
}

// the ObjectWrapper interface
func (s *RootState) StartRV() *reflect.Value {
    return s.startRv
}

func (s *RootState) Anchor() *string {
    return nil
}

// the root does not have a tag handler
func (s *RootState) TagHandler() TagHandler {
    return nil
}

// the root has an schema implementer
func (s *RootState) SchemaImplementer() SchemaImplementer {
    return s.si
}

// the CollectionWrapper interface
func (s *RootState) ObjStartIn(event *Event, path *Path) (ObjectWrapper, error) {
    if s.rv == nil || !s.rv.IsValid() {
        return nil, errors.New("Invalid root reflect value")
    }

    soc := path.RootUserData().(SchemaObjectCreator)

    ow, err := soc.NewSchemaObject(event, path, s.rv)
    if err != nil {
        return nil, err
    }

    s.ow = ow

    return s.ow, nil
}

func (s *RootState) ObjEndIn(event *Event, path *Path, ow ObjectWrapper) error {

    if ow != s.ow {
        panic("root: mismatch on object start/end")
    }
    s.ow = nil

    if anchor := ow.Anchor(); anchor != nil {

        if r, hasR := path.RootUserData().(Resolver); hasR {
            if err := r.RegisterAnchor(*anchor, path, ow, s, nil); err != nil {
                return err
            }
        }
        // schema does not provide resolver services
    }

    // perhaps optimize here
    return nil
}

func (s *RootState) CollectionStart(event *Event, path *Path) error {

    rv := s.startRv

    s.dp.Debugf("ReflectionRootStart: %s\n", path)

    kind := rv.Kind()
    if kind != reflect.Ptr || rv.IsNil() || !rv.IsValid() {
        return errors.New(fmt.Sprintf("%s: illegal value type for root", path))
    }

    // get the pointer of the value 
    rvt := rv.Elem()
    if !rvt.IsValid() {
        return errors.New(fmt.Sprintf("%s: Unable to retrieve ptr context", path))
    }
    if !rvt.CanSet() {
        return errors.New(fmt.Sprintf("%s: cannot set the value", path))
    }

    // update s
    s.ri = rv
    s.rv = &rvt

    return nil
}

func (s *RootState) CollectionEnd(event *Event, path *Path) error {

    dp := s.dp

    dp.Debugf("CollectionEnd %s\n", path)

    return nil
}

// the RootAddress contains nothing
func (s *RootState) CurrentAddress(path *Path) AddressWrapper {
    return nil
}

type SequenceState struct {
    startRv *reflect.Value  // the start reflection value
    t TagHandler
    anchor *string
    pc *PathComponent       // the path component
    ri *reflect.Value       // the interface (if generic)
    rv *reflect.Value       // the reflect value of the sequence
    rvi *reflect.Value      // the reflect value of the item
    idx int                 // item index (<0 if not in item)
    ow ObjectWrapper        // the current addressed objected 
}

// the ObjectWrapper interface
func (s *SequenceState) StartRV() *reflect.Value {
    return s.startRv
}

func (s *SequenceState) Anchor() *string {
    return s.anchor
}

func (s *SequenceState) TagHandler() TagHandler {
    return s.t
}

func (s *SequenceState) SchemaImplementer() SchemaImplementer {
    return s.t.SchemaImplementer()
}

// the CollectionWrapper interface
func (s *SequenceState) ObjStartIn(event *Event, path *Path) (ObjectWrapper, error) {

    // create a new value for the sequence item to store the value to
    et := s.rv.Type().Elem()
    rvt := reflect.New(et).Elem()

    // save the sequence index for the end of the object
    s.idx = s.pc.SequenceIndex()

    s.rvi = &rvt

    rv, err := IndirectPointer(s.rvi)
    if err != nil {
        return nil, err
    }

    soc := path.RootUserData().(SchemaObjectCreator)

    ow, err := soc.NewSchemaObject(event, path, rv)
    if err != nil {
        return nil, err
    }

    s.ow = ow

    return s.ow, nil
}

func (s *SequenceState) ObjEndIn(event *Event, path *Path, ow ObjectWrapper) error {

    if ow != s.ow {
        panic("sequence: mismatch on object start/end")
    }

    switch s.rv.Kind() {
    case reflect.Slice:

        // grow the slice if we're over capacity
        if s.idx >= s.rv.Cap() {
            newcap := s.rv.Cap()
            for {
                newcap = newcap + newcap/2
                if newcap < 4 {
                    newcap = 4
                }
                if newcap >= s.idx + 1 {
                    break
                }
            }
            newv := reflect.MakeSlice(s.rv.Type(), s.rv.Len(), newcap)

            reflect.Copy(newv, *s.rv)

            s.rv.Set(newv)
        }

        // we are under cap now, set the length if over the current length
        if s.idx >= s.rv.Len() {
            s.rv.SetLen(s.idx + 1)
        }

        // address the given index now
        rvt := s.rv.Index(s.idx)
        if !rvt.IsValid() {
            return errors.New(fmt.Sprintf("%v: illegal index %d - %v", path, s.idx, s.rv))
        }

        // and set it
        // rvt.Set(*s.rvi)
        rvt.Set(*s.ow.StartRV())

    case reflect.Interface:
        panic("")

    default:
        return errors.New(fmt.Sprintf("%v: illegal value type for sequence: rv=%v kind=%v", path, s.rv, s.rv.Kind()))
    }

    if anchor := ow.Anchor(); anchor != nil {
        if r, hasR := path.RootUserData().(Resolver); hasR {
            if err := r.RegisterAnchor(*anchor, path, ow, s, nil); err != nil {
                return err
            }
        }
        // schema does not provide resolver services
    }

    s.ow = nil

    return nil
}

func (s *SequenceState) CollectionStart(event *Event, path *Path) error {

    dp := path.RootUserData().(DebugfProvider)

    rv := s.startRv

    dp.Debugf("ReflectionSequenceStart %s\n", path)

    var ri *reflect.Value

    kind := rv.Kind()
    switch kind {
    case reflect.Slice:
        // start by resetting the slice length to zero
        rv.SetLen(0)

    case reflect.Interface:
        // save interface
        ri = rv

        // needs to be with New because we need it to be settable
        sv := reflect.New(genericSeqType).Elem()
        sv.Set(reflect.MakeSlice(genericSeqType, 0, 0))

        // point to this from now on
        rv = &sv

        dp.Debugf("%s: generic sequence kind %s: %v - iface=%v\n", path, rv.Kind(), *rv, *ri)

    default:
        return errors.New(fmt.Sprintf("%v: illegal value type sequence: %v", path, kind))
    }

    s.pc = path.LastComponent()
    s.ri = ri
    s.rv = rv

    return nil
}

func (s *SequenceState) CollectionEnd(event *Event, path *Path) error {

    dp := path.RootUserData().(DebugfProvider)

    dp.Debugf("ReflectionSequenceEnd %s\n", path)

    // if we're on an interface, set it (should be settable)
    if s.ri != nil {

        if s.rv == nil || !s.rv.IsValid() {
            return errors.New(fmt.Sprintf("%v: invalid sequence end interface", path))
        }
        if !s.rv.CanSet() {
           return errors.New(fmt.Sprintf("%v: unsettable sequence end interface", path))
        }

        var itemType, thisItemType reflect.Type
        uniformItemTypes := false

        length := s.rv.Len()
        for idx := 0; idx < length; idx++ {
            item := s.rv.Index(idx)

            if item.IsValid() && !item.IsNil() {
                thisItemType = item.Elem().Type()
            } else {
                thisItemType = nil
            }

            if idx == 0 {
                itemType = thisItemType
                uniformItemTypes = true
                continue
            }
            if thisItemType != itemType {
                uniformItemTypes = false
                break
            }
        }

        // if the uniform type is the generic (or nil) sequence item type
        if uniformItemTypes && (itemType == genericSeqType || itemType == nil) {
            uniformItemTypes = false
        }

        if uniformItemTypes {

            dp.Debugf("uniform items of type %v\n", itemType)

            sv := reflect.MakeSlice(reflect.SliceOf(itemType), length, length)

            // set unwrapping the item
            for idx := 0; idx < length; idx++ {
                sv.Index(idx).Set(s.rv.Index(idx).Elem())
            }

            // set it to the new more specific sequence
            s.ri.Set(sv)

        } else {

            // set it to the generic sequence
            s.ri.Set(*s.rv)
        }
    }

    return nil
}

// the sequence address is an index
type SequenceAddress struct {
    idx int
}

func (a *SequenceAddress) String() string {
    return fmt.Sprintf("%d", a.idx)
}

func (s *SequenceState) CurrentAddress(path *Path) AddressWrapper {
    return &SequenceAddress{
        idx: s.idx,
    }
}

type MappingState struct {
    startRv *reflect.Value  // the start reflection value
    t TagHandler
    anchor *string

    pc *PathComponent       // the path component
    ri *reflect.Value       // the interface (if generic)
    rv *reflect.Value       // the reflect value of the mapping
    rvk *reflect.Value      // the reflect value of the key of the mapping
    rvv *reflect.Value      // the reflect value of the value
    ti *TypeInfo            // the type-info of the struct (if is a struct)
    uf *Field               // the unmarshaler field if on struct
    dupf map[*Field]uvoid   // duplicate fields check

    ow ObjectWrapper        // the current object addressed
    owk ObjectWrapper       // the key object wrapper
    owv ObjectWrapper       // the value object wrapper

}

func (s *MappingState) ObjStartInMapKeyTyped(path *Path) (*reflect.Value, error) {

    scalarKey := s.pc.MappingScalarKey()
    if scalarKey == nil {
        panic("Mapping scalar key is NULL, can't handle complex key yet\n")
    }

    // in typed mode, the tag is ignored...
    // TODO perhaps spit out a warning or something

    strkey := scalarKey.Text()

    // typed mapping
    rvv, uf := s.ti.FieldByName(strkey, s.rv)
    if rvv == nil {
        return nil, errors.New(fmt.Sprintf("%v: illegal key field %s", path, strkey))
    }

    // check for duplicate
    if _, exists := s.dupf[uf]; exists {
        return nil, errors.New(fmt.Sprintf("%v: duplicate key %s", path, strkey))
    }
    // mark it
    s.dupf[uf]=uvoid{}

    // get the pointer to the reflect value of the string key
    rvt := reflect.ValueOf(&strkey).Elem()

    if !rvt.IsValid() {
        return nil, errors.New(fmt.Sprintf("%v: Unable to retrieve ptr context", path))
    }
    if !rvt.CanSet() {
        return nil, errors.New(fmt.Sprintf("%v: cannot set the value", path))
    }

    s.rvk = &rvt
    // save the value info for later
    s.uf = uf
    s.rvv = rvv

    return IndirectPointer(s.rvk)
}

func (s *MappingState) ObjEndInMapKeyTyped(event *Event, path *Path) error {
    return nil
}

func (s *MappingState) ObjStartInMapValueTyped(event *Event, path *Path) (*reflect.Value, error) {

    return IndirectPointer(s.rvv)
}

func (s *MappingState) ObjEndInMapValueTyped(event *Event, path *Path) error {
    return nil
}

func (s *MappingState) ObjStartInMapKeyGeneric(path *Path) (*reflect.Value, error) {

    dp := path.RootUserData().(DebugfProvider)

    // generic interface mapping
    dp.Debugf("%s: generic interface mapping key\n", path)

    // fill in an interface{} as a key
    var iv interface{}
    rvt := reflect.ValueOf(&iv).Elem()
    // rvt = reflect.ValueOf(&strkey).Elem()

    if !rvt.IsValid() {
        return nil, errors.New(fmt.Sprintf("%v: Unable to retrieve ptr context", path))
    }
    if !rvt.CanSet() {
        return nil, errors.New(fmt.Sprintf("%v: cannot set the value", path))
    }

    s.rvk = &rvt
    // no field info, nor value
    s.uf = nil
    s.rvv = nil

    return IndirectPointer(s.rvk)
}

func (s *MappingState) ObjEndInMapKeyGeneric(event *Event, path *Path) error {

    // TODO optimize key storage... check for duplicates etc
    return nil
}

func (s *MappingState) ObjStartInMapValueGeneric(event *Event, path *Path) (*reflect.Value, error) {

    dp := path.RootUserData().(DebugfProvider)

    dp.Debugf("generic interface mapping key %s\n", path)

    // create a new value for the map to store the value to
    et := s.rv.Type().Elem()
    rvt := reflect.New(et).Elem()

    s.rvv = &rvt

    return IndirectPointer(s.rvv)
}

func (s *MappingState) ObjEndInMapValueGeneric(event *Event, path *Path) error {

    rvk := s.owk.StartRV()
    rvv := s.owv.StartRV()

    // find out if the type is hashable
    hashable := IsHashable(*rvk)

    var key, value reflect.Value

    if hashable {
        // commit to the map
        key, value = *rvk, *rvv
    } else {
        // make a string from the last component
        str := path.LastComponent().String()
        rvt := reflect.ValueOf(&str).Elem()
        key, value = rvt, *rvv
    }

    chk := s.rv.MapIndex(key)

    if chk.IsValid() {
       return errors.New(fmt.Sprintf("%v: duplicate key %v on mapping", path, key))
   }

    s.rv.SetMapIndex(key, value)

    return nil
}

func (s *MappingState) ObjStartInMapKey(event *Event, path *Path) (*reflect.Value, error) {

    if s.ri == nil {
        return s.ObjStartInMapKeyTyped(path)
    } else {
        return s.ObjStartInMapKeyGeneric(path)
    }
}

func (s *MappingState) ObjStartInMapValue(event *Event, path *Path) (*reflect.Value, error) {

    if s.ri == nil {
        return s.ObjStartInMapValueTyped(event, path)
    } else {
        return s.ObjStartInMapValueGeneric(event, path)
    }
}

// the ObjectWrapper interface
func (s *MappingState) StartRV() *reflect.Value {
    return s.startRv
}

func (s *MappingState) Anchor() *string {
    return s.anchor
}

func (s *MappingState) TagHandler() TagHandler {
    return s.t
}

func (s *MappingState) SchemaImplementer() SchemaImplementer {
    return s.t.SchemaImplementer()
}

// the CollectionWrapper interface
func (s *MappingState) ObjStartIn(event *Event, path *Path) (ObjectWrapper, error) {

    var rv *reflect.Value
    var err error

    inKey := path.InMappingKey()
    if inKey {
        rv, err = s.ObjStartInMapKey(event, path)
    } else {
        rv, err = s.ObjStartInMapValue(event, path)
    }

    if err != nil {
        return nil, err
    }

    soc := path.RootUserData().(SchemaObjectCreator)

    ow, err := soc.NewSchemaObject(event, path, rv)
    if err != nil {
        return nil, err
    }

    // save the currently address object
    s.ow = ow

    // and associate with the key/value
    if inKey {
        s.owk = ow
    } else {
        s.owv = ow
    }

    return ow, nil
}

func (s *MappingState) ObjEndInMapKey(event *Event, path *Path) error {
    if s.ri == nil {
        return s.ObjEndInMapKeyTyped(event, path)
    } else {
        return s.ObjEndInMapKeyGeneric(event, path)
    }
}

func (s *MappingState) ObjEndInMapValue(event *Event, path *Path) error {
    if s.ri == nil {
        return s.ObjEndInMapValueTyped(event, path)
    } else {
        return s.ObjEndInMapValueGeneric(event, path)
    }
}

func (s *MappingState) ObjEndIn(event *Event, path *Path, ow ObjectWrapper) error {

    if ow != s.ow {
        panic("mapping: mismatch on object start/end")
    }

    var err error

    inKey := path.InMappingKey()
    if inKey {
        err = s.ObjEndInMapKey(event, path)
    } else {
        err = s.ObjEndInMapValue(event, path)
    }

    if err != nil {
        return err
    }

    // when the value is done, clear both
    if !inKey {
        s.owk = nil
        s.owv = nil
    }

    s.ow = nil

    dp := path.RootUserData().(DebugfProvider)
    if anchor := ow.Anchor(); anchor != nil {
        dp.Debugf("%s: Anchor %s\n", path, *anchor)
    }

    return nil
}

func (s *MappingState) CollectionStart(event *Event, path *Path) error {

    dp := path.RootUserData().(DebugfProvider)
    rv := s.startRv

    dp.Debugf("ReflectionMappingStart %s\n", path)

    var ti *TypeInfo = nil
    var ri *reflect.Value = nil

    kind := rv.Kind()
    switch kind {
    case reflect.Struct:

        // get the type services provider from the root object
        tsp, hasTsp := path.RootUserData().(TypeServicesProvider)
        if !hasTsp {
            return errors.New(fmt.Sprintf("%s: The root object does not provide type service for type %s\n", path, rv.Type()))
        }

        // lookup the type info
        ti = tsp.LookupOrNewType(rv.Type())
        if ti == nil {
            return errors.New(fmt.Sprintf("%s: could not lookup type %s\n", path, rv.Type()))
        }

    case reflect.Interface:
        // save interface
        ri = rv

        // create an empty [interface{}]interface{} value
        sv := reflect.New(genericMapType).Elem()
        sv.Set(reflect.MakeMap(genericMapType))

        // point to this from now on
        rv = &sv

        dp.Debugf("%s: generic mapping kind=%s type=%s %v - iface=%v\n", path, rv.Kind(), rv.Type(), *rv, *ri)

    default:
        return errors.New(fmt.Sprintf("%v: illegal value type for mapping: %v", path, kind))
    }

    // update the state
    s.pc = path.LastComponent()
    s.ri = ri
    s.rv = rv
    s.ti = ti

    return nil
}

func (s *MappingState) CollectionEnd(event *Event, path *Path) error {

    dp := path.RootUserData().(DebugfProvider)

    dp.Debugf("ReflectionMappingEnd %s\n", path)

    // if we're on an interface, set it (should be settable)
    if s.ri != nil {

        dp.Debugf("on interface\n")
        if s.rv == nil || !s.rv.IsValid() {
            return errors.New(fmt.Sprintf("%v: invalid mapping end interface", path))
        }
        if !s.rv.CanSet() {
           return errors.New(fmt.Sprintf("%v: unsettable mapping end interface", path))
        }

        // we now have a generic map[interface{}]interface{}
        // we will try to restrict the types of the keys/values
        // converting to something like map[string]string

        var keyType, valueType, thisKeyType, thisValueType reflect.Type
        uniformKeyTypes, uniformValueTypes := false, false

        keyType = nil
        valueType = nil
        for idx, key := range s.rv.MapKeys() {
            if key.IsValid() && !key.IsNil() {
                thisKeyType = key.Elem().Type()
            } else {
                thisKeyType = nil
            }
            value := s.rv.MapIndex(key)
            if value.IsValid() && !value.IsNil() {
                thisValueType = value.Elem().Type()
            } else {
                thisValueType = nil
            }
            if idx == 0 {
                keyType = thisKeyType
                valueType = thisValueType
                uniformKeyTypes = true
                uniformValueTypes = true
                continue
            }
            if uniformKeyTypes && thisKeyType != keyType {
                uniformKeyTypes = false
            }
            if uniformValueTypes && thisValueType != valueType {
                uniformValueTypes = false
            }

            // if both are non-uniform, no point in continuing
            if !uniformKeyTypes && !uniformValueTypes {
                break
            }
        }

        // if the types are uniform but generic (or null), do not bother
        if uniformKeyTypes && (keyType == genericIfaceType || keyType == nil) {
            uniformKeyTypes = false
        }

        if uniformValueTypes && (valueType == genericIfaceType || valueType == nil) {
            uniformValueTypes = false
        }

        if uniformKeyTypes {
            dp.Debugf("uniform keys of type %v\n", keyType)
        }

        if uniformValueTypes {
            dp.Debugf("uniform values of type %v\n", valueType)
        }

        // any kind of uniformity makes us tighten the map type
        if uniformKeyTypes || uniformValueTypes {

            // make the non-uniform type generic
            if !uniformKeyTypes {
                keyType = genericIfaceType
            }
            if !uniformValueTypes {
                valueType = genericIfaceType
            }

            // make a new map with the right size
            sv := reflect.MakeMapWithSize(reflect.MapOf(keyType, valueType), s.rv.Len())

            for _, key := range s.rv.MapKeys() {

                value := s.rv.MapIndex(key)

                var k, v reflect.Value

                if uniformKeyTypes {
                    if uniformValueTypes {
                        // key specific, value specific
                        k, v = key.Elem(), value.Elem()
                    } else {
                        // key specific, value generic
                        k, v = key.Elem(), value
                    }
                } else {
                    if uniformValueTypes {
                        // key generic, value specific
                        k, v = key, value.Elem()
                    } else {
                        // key generic, value generic
                        // should never happen, but do it anyway
                        k, v = key, value
                    }
                }

                chk := sv.MapIndex(k)

                if chk.IsValid() {
                    dp.Debugf("> key=%v value=%v - chk=%v duplicate\n", key, value, chk)
                   return errors.New(fmt.Sprintf("%v: duplicate key %v on mapping", path, k))
                } else {
                    dp.Debugf("> key=%v value=%v - chk=%v\n", key, value, chk)
                }

                dp.Debugf("> key=%v value=%v\n", key, value)
                sv.SetMapIndex(k, v)
            }

            // set it to the new more specific map
            s.ri.Set(sv)

        } else {

            // set it to the generic map
            s.ri.Set(*s.rv)
        }

    } else {

        dp.Debugf("NOT on interface\n")
    }

    return nil
}

// the mapping address is either a key or a value
type MappingAddress struct {
    isKey bool
    key ObjectWrapper
}

func (a *MappingAddress) String() string {
    if a.isKey {
        return fmt.Sprintf(".key(%s)", "xxx")
    } else {
        return fmt.Sprintf("%s", "xxx")
    }
}

func (s *MappingState) CurrentAddress(path *Path) AddressWrapper {
    return &MappingAddress{
        isKey: path.InMappingKey(),
        key: s.owk,
    }
}

type ScalarState struct {
    startRv *reflect.Value  // start reflection value
    t TagHandler
    anchor *string
}

// the ObjectWrapper interface
func (s *ScalarState) StartRV() *reflect.Value {
    return s.startRv
}

func (s *ScalarState) Anchor() *string {
    return s.anchor
}

func (s *ScalarState) TagHandler() TagHandler {
    return s.t
}

func (s *ScalarState) SchemaImplementer() SchemaImplementer {
    return s.t.SchemaImplementer()
}

func (s *ScalarState) SetScalar(event *Event, path *Path) error {

    dp := path.RootUserData().(DebugfProvider)

    rv := s.startRv

    dp.Debugf("SetScalar %s - tag=%s\n", path, event.Tag().Text())

    value := event.ScalarValuePtr()

    if value != nil {
        dp.Debugf("> %q - %s\n", *value, event.Token().ScalarStyle())
    } else {
        dp.Debugf("> <null>\n")
    }

    dp.Debugf("> setting: kind is %v\n", rv.Kind())

    kind := rv.Kind()

    switch kind {

    case reflect.Ptr:
        // nil value, set to nil
        if value == nil {
            rv.Set(reflect.Zero(rv.Type()))
        } else {
            // interface{} ?
        }

    case reflect.Bool:

        if value == nil {
            return errors.New(fmt.Sprintf("%v: cannot set bool to null", path))
        }

        // check the scalar style (must be plain)
        ss := event.Token().ScalarStyle()
        if ss != Plain {
            return errors.New(fmt.Sprintf("%v: bool scalar style is %s instead of plain", path, ss))
        }

        if *value == "true" {
            rv.SetBool(true)
        } else if  *value == "false" {
            rv.SetBool(false)
        } else {
            return errors.New(fmt.Sprintf("%v: cannot set bool to %s", path, *value))
        }

    case reflect.String:

        if value == nil {
            return errors.New(fmt.Sprintf("%v: cannot set string to null", path))
        }

        rv.SetString(*value)

    case reflect.Interface:
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store scalar", path))
        }
        if value == nil {
            rv.Set(reflect.Zero(rv.Type()))
        } else {
            rv.Set(reflect.ValueOf(*value))
        }

    default:
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    return nil
}
