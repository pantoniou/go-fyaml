// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "errors"
    "reflect"
    "fmt"
    "unsafe"
    "strings"
    "strconv"
)

// please note that libfyaml produces full tag forms by default
// so no need to do long/short forms for detection
// if you need to check if a tag is a short form, just compare
// against the longTagPrefix

const DefaultLongTagPrefix = "tag:yaml.org,2002:"

type YAMLSchemaType int

const (
    UnknownSchema YAMLSchemaType = iota
    FailsafeSchema              // failsafe
    JSONSchema                  // json
    YAML11Schema                // 1.1
    CoreSchema                  // 1.2
    YAML13Schema                // 1.3
    YAML12Schema = CoreSchema   // alias of Core
)

type RefState struct {
    ys *YAMLSchema          // the schema we belong to
    explicit bool           // if it's a explicit anchor
    startRv *reflect.Value
    ref string              // the reference
    ow ObjectWrapper        // the object that the anchor was set on
    cw CollectionWrapper    // the collection that the object belongs
    aw AddressWrapper       // the address that the collection uses to address it
}

// the ObjectWrapper interface
func (s *RefState) StartRV() *reflect.Value {
    // XXX
    return s.startRv
}

func (s *RefState) Anchor() *string {
    // never has an anchor
    return nil
}

func (s *RefState) TagHandler() TagHandler {
    return &s.ys.refT
}

func (s *RefState) SchemaImplementer() SchemaImplementer {
    return s.ys.si
}

// the ScalarWrapper interface
func (s *RefState) SetScalar(event *Event, path *Path) error {

    dp := path.RootUserData().(DebugfProvider)

    r, hasR := path.RootUserData().(Resolver)
    if !hasR {
        return errors.New(fmt.Sprintf("%s: Cannot resolve %s", path, s.ref))
    }

    dp.Debugf("%s: Resolving alias %s\n", path, s.ref)

    ow, _, _, err := r.FindReference(s.ref)
    if err != nil {
        return err
    }

    if ow == nil {
        // dp.Debugf("%s: Cannot resolve alias %s\n", path, s.ref)
        return errors.New(fmt.Sprintf("%s: Cannot resolve alias %s\n", path, s.ref))
    }

    dp.Debugf("value %v\n", *ow.StartRV())

    rv := s.startRv
    targetRv := ow.StartRV()

    kind := rv.Kind()
    targetKind := targetRv.Kind()

    // kinds match?
    if kind == targetKind {
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store alias %s", path, s.ref))
        }
        rv.Set(*targetRv)
    } else {
        return errors.New(fmt.Sprintf("%v: cannot store alias %s, mismatched types %s %s", path, s.ref, kind, targetKind))
    }

    return nil
}

type RefTag struct {
    si SchemaImplementer
}

// the reference tag handler does not appear in the schema at all
func (t *RefTag) Tag() string {
    return ""
}

func (t *RefTag) SetSchemaImplementer(si SchemaImplementer) {
    // doesn't do anything
}

func (t *RefTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *RefTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {
    // never use this
    return nil, nil
}

func (t *RefTag) Specify(kind reflect.Kind) reflect.Kind {
    // ref does not implement this
    return reflect.Invalid
}

// this is the common YAML schema implementer
// note that it doesn't support the full schema interface on purpose
// you need to wrap it in another full schema type to get that
type YAMLSchema struct {
    // the configuration
    st YAMLSchemaType                   // the schema type
    si SchemaImplementer                // the implementer

    // internals
    supportedTags map[string]TagHandler // the map of tag to tag-handler

    // the basic supported tags (not all are supported in failsafe)
    strT StrTag
    boolT BoolTag
    nullT NullTag
    intT IntTag
    floatT FloatTag
    seqT SeqTag
    mapT MapTag

    // the invisible internal reference tag
    refT RefTag
}

func NewYAMLSchema(st YAMLSchemaType, si SchemaImplementer) *YAMLSchema {

    ys := &YAMLSchema{
        st: st,
        si: si,
        supportedTags: make(map[string]TagHandler),
    }

    var tags []TagHandler

    switch ys.st {
    case FailsafeSchema:
        tags = []TagHandler{ &ys.strT, &ys.seqT, &ys.mapT }

    default:
        tags = []TagHandler{ &ys.strT, &ys.boolT, &ys.nullT, &ys.intT, &ys.floatT, &ys.seqT, &ys.mapT }
    }

    // initialize the supported tags
    for _, t := range tags {
        t.SetSchemaImplementer(si)
        ys.supportedTags[t.Tag()] = t
    }

    // and the hidden reference tag handler
    t := &ys.refT
    t.SetSchemaImplementer(si)
    // note that there's no textual representation

    return ys
}

func (ys *YAMLSchema) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    return NewRootState(event, path, root, ys.si, dec)
}

func (ys *YAMLSchema) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    return nil
}

func (ys *YAMLSchema) ImplicitResolve(vp *string) TagHandler {

    var i, l int
    var v []rune

    st := ys.st
    switch st {
    case FailsafeSchema:
        // the failsafe is all strings
        return &ys.strT

    case JSONSchema:
        // null
        if vp == nil || *vp == "null" {
            return &ys.nullT
        }

        // empty string
        if *vp == "" {
            return &ys.strT
        }
        // boolean
        if *vp == "true" || *vp == "false" {
            return &ys.boolT
        }

    case CoreSchema, YAML13Schema:
        // null
        if vp == nil ||
           *vp == "null" || *vp == "Null" || *vp == "NULL" ||
           *vp == "~" || *vp == "" {
            return &ys.nullT
        }

        // boolean
        if *vp == "true" || *vp == "True" || *vp == "TRUE" ||
           *vp == "false" || *vp == "False" || *vp == "FALSE" {
            return &ys.boolT
        }

        // float infinities
        if *vp == ".nan" || *vp == ".NaN" || *vp == ".NAN" {
            return &ys.floatT
        }

        // float infinites
        if *vp == ".inf" || *vp == ".Inf" || *vp == ".INF" ||
           *vp == "-.inf" || *vp == "-.Inf" || *vp == "-.INF" ||
           *vp == "+.inf" || *vp == "+.Inf" || *vp == "+.INF" {
            return &ys.floatT
        }

    case YAML11Schema:
        // null
        if vp == nil ||
           *vp == "null" || *vp == "Null" || *vp == "NULL" ||
           *vp == "~" || *vp == "" {
            return &ys.nullT
        }

        // boolean
        if *vp == "true" || *vp == "True" || *vp == "TRUE" ||
           *vp == "false" || *vp == "False" || *vp == "FALSE" ||
           *vp == "y" || *vp == "Y" || *vp == "yes" || *vp == "Yes" || *vp == "YES" ||
           *vp == "n" || *vp == "N" || *vp == "no" || *vp == "No" || *vp == "NO" ||
           *vp == "on" || *vp == "On" || *vp == "ON" ||
           *vp == "off" || *vp == "Off" || *vp == "OFF" {
            return &ys.boolT
        }

        // float infinities
        if *vp == ".nan" || *vp == ".NaN" || *vp == ".NAN" {
            return &ys.floatT
        }

        // float infinites
        if *vp == ".inf" || *vp == ".Inf" || *vp == ".INF" ||
           *vp == "-.inf" || *vp == "-.Inf" || *vp == "-.INF" ||
           *vp == "+.inf" || *vp == "+.Inf" || *vp == "+.INF" {
            return &ys.floatT
        }
    }

    if vp == nil {
        return &ys.nullT
    }

    // 0 integer
    if *vp == "0" {
        return &ys.intT
    }

    // we don't do that big nums
    if len(*vp) > 256 {
        return &ys.strT
    }

    // convert to rune array and go over it
    v = []rune(*vp)
    i = 0
    l = len(v)

    // handle hex and octals
    if st != JSONSchema && l >= 3 && v[0] == '0' && (v[1] == 'o' || v[1] == 'x') {

        if v[1] == 'o' {

            // octals
            i += 2

            if i >= l || !(v[i] >= '0' && v[i] <= '7') {
                return &ys.strT
            }
            i++

            for i < l && v[i] >= '0' && v[i] <= '7' {
                i++
            }

        } else {

            // hex
            i += 2
            if i >= l || !((v[i] >= '0' && v[i] <= '9') ||
                          (v[i] >= 'a' && v[i] <= 'f') ||
                          (v[i] >= 'A' && v[i] <= 'F')) {
                return &ys.strT
            }
            i++

            for i < l && (v[i] >= '0' && v[i] <= '9') ||
                         (v[i] >= 'a' && v[i] <= 'f') ||
                         (v[i] >= 'A' && v[i] <= 'F') {
                i++
            }
        }
        // consumed everything? it's an int
        if i >= l {
            return &ys.intT
        }
    }

    // possible integer or float 

    // integer regex 0 | -? [1-9] [0-9]*

    // sign (JSON schema only allows -)
    if st == JSONSchema {
        if i < l && v[i] == '-' {
            i++
        }
    } else {
        if i < l && (v[i] == '-' || v[i] == '+') {
            i++
        }
    }

    if st == JSONSchema || (i < l && v[i] != '.') {
        // [1-9]
        if i < l && v[i] >= '1' && v[i] <= '9' {
            i++
            // [0-9]*
            for i < l && v[i] >= '0' && v[i] <= '9' {
                i++
            }
        }

        // consumed everything? it's an int
        if i >= l {
            return &ys.intT
        }
    }

    // optional fractional part
    if i < l && v[i] == '.' {
        i++
        if i >= l || v[i] < '0' || v[i] > '9' { // nothing after comma, or not a digit
            return &ys.strT
        }
        i++
        // [0-9]*
        for i < l && v[i] >= '0' && v[i] <= '9' {
            i++
        }
    }

    // all out?
    if i >= l {
        return &ys.floatT
    }

    // no? try scientific part
    if v[i] != 'e' && v[i] <= 'E' {
        return &ys.strT
    }
    i++

    // optional scientific sign
    if i < l && (v[i] == '+' || v[i] == '-') {
        i++
    }

    // at least one digit must exist
    if i >= l || v[i] < '0' || v[i] > '9' {
        return &ys.strT
    }

    // [0-9]*
    for i < l && v[i] >= '0' && v[i] <= '9' {
        i++
    }

    // everything consumed? float with scientific notation
    if i >= l {
        return &ys.floatT
    }

    // it's a string otherwise...
    return &ys.strT
}

func (ys *YAMLSchema) YAMLSchemaType() YAMLSchemaType {
    return ys.st
}

func (ys *YAMLSchema) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {

    if event.Type() == Alias {
        return &RefState {
            ys: ys,
            explicit: true,
            startRv: startRv,
            ref: event.Token().Text(),
        }, nil
    }

    // find the tag handler (note we use the schema callback)
    th, _, err := ys.si.FindTagHandler(event, path, startRv)
    if err != nil {
        return nil, err
    }

    return th.NewSchemaObject(event, path, startRv, ys.si)
}

// find a tag handler to match (whether explicitly or implicitly)
func (ys *YAMLSchema) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {

    // the tags are retrieved only on these 4 events
    // ignore all others
    etype := event.Type()
    switch etype {
    case Alias, Scalar, SequenceStart, MappingStart:
        break
    default:
        return nil, false, nil
    }

    // if there's a tag try to use it
    if event.Tag() != nil {

        // lookup and use it if it's there
        if th, hasTag := ys.si.LookupTagHandler(event.Tag().Text()); hasTag {
            return th, true, nil
        }

        // no tag, switch to implicit mode
    }

    // select something implicitly (if it's sequence or a mapping)
    switch etype {
    case SequenceStart:
        // failsafe safe
        return &ys.seqT, false, nil

    case MappingStart:
        // failsafe safe
        return &ys.mapT, false, nil

    case Scalar:
        // a scalar; if it's anything other than plain style it's a string
        if event.Token().ScalarStyle() != Plain {
            // failsafe safe
            return &ys.strT, false, nil
        }

        // it's a scalar that we need to scan but before that, we
        // might be on typed mode and a type is available
        switch rv.Kind() {
        case reflect.String:
            // failsafe safe
            return &ys.strT, false, nil

        case reflect.Bool:
            if ys.st != FailsafeSchema {
                return &ys.boolT, false, nil
            }

        case reflect.Int, reflect.Uint, reflect.Int8, reflect.Uint8,
             reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32,
             reflect.Int64, reflect.Uint64:
            if ys.st != FailsafeSchema {
                return &ys.intT, false, nil
            }

        case reflect.Float32, reflect.Float64:
            if ys.st != FailsafeSchema {
                return &ys.floatT, false, nil
            }

        case reflect.Interface:
            // everything failed, we have to figure it out from the contents
            if th := ys.ImplicitResolve(event.ScalarValuePtr()); th != nil {
                return th, false, nil
            }
        }

    case Alias:
        // return the 'hidden' reference tag handle
        if ys.st != JSONSchema {
            return &ys.refT, true, nil
        }
        // json does not have aliases (we shouldn't get here anyway)
    }

    return nil, false, errors.New(fmt.Sprintf("%s: cannot infer implicit tag from kind %s", path, rv.Kind()))
}

// common YAML schemas Selected method; empty for now
func (ys *YAMLSchema) Selected(event *Event, path *Path) {
    // nothing yet 
}

// common YAML schemas Selected method; empty for now
func (ys *YAMLSchema) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {

    if tag != nil {
        if th, hasTag := ys.si.LookupTagHandler(*tag); hasTag {
            return th, th.Specify(kind)
        }
    }

    switch kind {
    case reflect.String:
        // failsafe safe
        th := &ys.strT
        return th, th.Specify(kind)

    case reflect.Bool:
        if ys.st != FailsafeSchema {
            th := &ys.boolT
            return th, th.Specify(kind)
        }

    case reflect.Int, reflect.Uint, reflect.Int8, reflect.Uint8,
         reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32,
         reflect.Int64, reflect.Uint64:
        if ys.st != FailsafeSchema {
            th := &ys.intT
            return th, th.Specify(kind)
        }

    case reflect.Float32, reflect.Float64:
        if ys.st != FailsafeSchema {
            th := &ys.floatT
            return th, th.Specify(kind)
        }

    case reflect.Interface:
        // everything failed, we have to figure it out from the contents
        if th := ys.ImplicitResolve(value); th != nil {
            return th, th.Specify(kind)
        }
    }

    // nothing we can do...
    return nil, reflect.Invalid
}

func (ys *YAMLSchema) LookupTagHandler(tag string) (TagHandler, bool) {
    th, hasTag := ys.supportedTags[tag]
    return th, hasTag
}

// the users must provide this
type YAMLSchemaProvider interface {
    YAMLSchema() *YAMLSchema
}

////////////////////////////////////////////////////////

// !!str
type StrState struct {
    sw ScalarWrapper
}

// the ObjectWrapper interface
func (s *StrState) StartRV() *reflect.Value {
    return s.sw.StartRV()
}

func (s *StrState) Anchor() *string {
    return s.sw.Anchor()
}

func (s *StrState) TagHandler() TagHandler {
    return s.sw.TagHandler()
}

func (s *StrState) SchemaImplementer() SchemaImplementer {
    return s.sw.SchemaImplementer()
}

// the ScalarWrapper interface
func (s *StrState) SetScalar(event *Event, path *Path) error {

    rv := s.StartRV()

    value := event.ScalarValue()

    switch kind := rv.Kind(); kind {

    case reflect.String:
        rv.SetString(value)

    case reflect.Interface:
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store string", path))
        }
        rv.Set(reflect.ValueOf(value))

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    return nil
}

type StrTag struct {
    si SchemaImplementer
}

func (t *StrTag) Tag() string {
    return DefaultLongTagPrefix + "str"
}

func (t *StrTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *StrTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *StrTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {

    // we need to descriminate (we don't allow arbitrary types for storage)
    switch kind := startRv.Kind(); kind {
    case reflect.String, reflect.Interface:
        // OK
    default:
        return nil, errors.New(fmt.Sprintf("%s: Cannot store a %s to a %v", path, t.Tag(), kind))
    }

    sw, err := NewScalarStateDefault(event, path, startRv, t)
    if err != nil {
        return nil, err
    }

    return &StrState {
        sw: sw,
    }, nil
}

func (t *StrTag) Specify(kind reflect.Kind) reflect.Kind {
    if kind == reflect.Interface || kind == reflect.String {
        return reflect.String
    }
    return reflect.Invalid
}

// !!bool
type BoolState struct {
    sw ScalarWrapper
}

// the ObjectWrapper interface
func (s *BoolState) StartRV() *reflect.Value {
    return s.sw.StartRV()
}

func (s *BoolState) Anchor() *string {
    return s.sw.Anchor()
}

func (s *BoolState) TagHandler() TagHandler {
    return s.sw.TagHandler()
}

func (s *BoolState) SchemaImplementer() SchemaImplementer {
    return s.sw.SchemaImplementer()
}

// the ScalarWrapper interface
func (s *BoolState) SetScalar(event *Event, path *Path) error {

    rv := s.sw.StartRV()

    // get the scalar value
    str := event.ScalarValue()

    isValid, value := false, false

    // default is the core schema
    st := CoreSchema

    // get the schema type
    if ysp, hasYsp := s.SchemaImplementer().(YAMLSchemaProvider); hasYsp {
        st = ysp.YAMLSchema().YAMLSchemaType()
    }

    switch st {

    case FailsafeSchema:
        // failsafe does not have a bool, how did we get here?

    case JSONSchema:
        // JSON schema is simple
        value = str == "true"
        isValid = value || str == "false"

        // the core schema is the default
    case CoreSchema, YAML13Schema:
        value = str == "true" || str == "True" || str == "TRUE"
        isValid = value || str == "false" || str == "False" || str == "FALSE"

        // here comes the crazy
    case YAML11Schema:
        value = str == "true" || str == "True" || str == "TRUE" ||
                str == "y" || str == "Y" || str == "yes" || str == "Yes" || str == "YES" ||
                str == "on" || str == "On" || str == "ON"
        isValid = value || str == "false" || str == "False" || str == "FALSE" ||
                           str == "n" || str == "N" || str == "no" || str == "No" || str == "NO" ||
                           str == "off" || str == "Off" || str == "OFF"
    }

    if !isValid {
        return errors.New(fmt.Sprintf("%v: invalid scalar %s to store to bool", path, str))
    }

    switch kind := rv.Kind(); kind {

    case reflect.Bool:
        rv.SetBool(value)

    case reflect.Interface:
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store string", path))
        }
        rv.Set(reflect.ValueOf(value))

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    return nil
}

type BoolTag struct {
    si SchemaImplementer
}

func (t *BoolTag) Tag() string {
    return DefaultLongTagPrefix + "bool"
}

func (t *BoolTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *BoolTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *BoolTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {

    // we need to descriminate (we don't allow arbitrary types for storage)
    switch kind := startRv.Kind(); kind {
    case reflect.Bool, reflect.Interface:
        // OK
    default:
        return nil, errors.New(fmt.Sprintf("%s: Cannot store a %s to a %v", path, t.Tag(), kind))
    }

    sw, err := NewScalarStateDefault(event, path, startRv, t)
    if err != nil {
        return nil, err
    }

    return &BoolState {
        sw: sw,
    }, nil
}

func (t *BoolTag) Specify(kind reflect.Kind) reflect.Kind {
    if kind == reflect.Interface || kind == reflect.Bool {
        return reflect.Bool
    }
    return reflect.Invalid
}

// !!null
type NullState struct {
    sw ScalarWrapper
}

// the ObjectWrapper interface
func (s *NullState) StartRV() *reflect.Value {
    return s.sw.StartRV()
}

func (s *NullState) Anchor() *string {
    return s.sw.Anchor()
}

func (s *NullState) TagHandler() TagHandler {
    return s.sw.TagHandler()
}

func (s *NullState) SchemaImplementer() SchemaImplementer {
    return s.sw.SchemaImplementer()
}

// the ScalarWrapper interface
func (s *NullState) SetScalar(event *Event, path *Path) error {

    rv := s.sw.StartRV()

    strp := event.ScalarValuePtr()

    isValid := false

    // default is the core schema
    st := CoreSchema

    // get the schema type
    if ysp, hasYsp := s.SchemaImplementer().(YAMLSchemaProvider); hasYsp {
        st = ysp.YAMLSchema().YAMLSchemaType()
    }

    switch st {

    case FailsafeSchema:
        // failsafe does not have a null, how did we get here?

    case JSONSchema:
        // JSON schema is simple
        isValid = strp != nil && *strp == "null"

        // the core schema is the default
    case CoreSchema, YAML13Schema, YAML11Schema:
        isValid = strp == nil || *strp == "null" || *strp == "Null" || *strp == "NULL" || *strp == "~"

    }

    if !isValid {
        return errors.New(fmt.Sprintf("%v: invalid scalar %s for null", path, *strp))
    }

    switch kind := rv.Kind(); kind {

    case reflect.Ptr:
        rv.Set(reflect.Zero(rv.Type()))

    case reflect.Interface:
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store null", path))
        }
        rv.Set(reflect.Zero(rv.Type()))

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for null", path, kind))
    }

    return nil
}

type NullTag struct {
    si SchemaImplementer
}

func (t *NullTag) Tag() string {
    return DefaultLongTagPrefix + "null"
}

func (t *NullTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *NullTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *NullTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {

    // we need to descriminate (we don't allow arbitrary types for storage)
    switch kind := startRv.Kind(); kind {
    case reflect.Ptr, reflect.Interface:
        // OK
    default:
        return nil, errors.New(fmt.Sprintf("%s: Cannot store a %s to a %v", path, t.Tag(), kind))
    }

    sw, err := NewScalarStateDefault(event, path, startRv, t)
    if err != nil {
        return nil, err
    }

    return &NullState {
        sw: sw,
    }, nil
}

func (t *NullTag) Specify(kind reflect.Kind) reflect.Kind {
    if kind == reflect.Interface || kind == reflect.Ptr {
        return kind
    }
    return reflect.Invalid
}

// !!int
type IntState struct {
    sw ScalarWrapper
}

// the ObjectWrapper interface
func (s *IntState) StartRV() *reflect.Value {
    return s.sw.StartRV()
}

func (s *IntState) Anchor() *string {
    return s.sw.Anchor()
}

func (s *IntState) TagHandler() TagHandler {
    return s.sw.TagHandler()
}

func (s *IntState) SchemaImplementer() SchemaImplementer {
    return s.sw.SchemaImplementer()
}

// the ScalarWrapper interface
func (s *IntState) SetScalar(event *Event, path *Path) error {

    rv := s.sw.StartRV()

    prec := 0
    signed := false

    // get the scalar value
    str := event.ScalarValue()

    base := 10

    // default is the core schema
    st := CoreSchema

    // get the schema type
    if ysp, hasYsp := s.SchemaImplementer().(YAMLSchemaProvider); hasYsp {
        st = ysp.YAMLSchema().YAMLSchemaType()
    }

    switch st {

    case FailsafeSchema:
        // failsafe does not have an int, how did we get here?

    case YAML11Schema, CoreSchema, YAML13Schema:

        if strings.HasPrefix(str, "0o") {
            base = 8
            str = strings.TrimPrefix(str, "0o")
        } else if strings.HasPrefix(str, "0x") {
            base = 16
            str = strings.TrimPrefix(str, "0x")
        }
    }

    var i int
    var ui int

    // two time check
    kind := rv.Kind()
    switch kind {

    case reflect.Interface:
        // for base10 prefer a signed type
        if base == 10 {
            prec = int(unsafe.Sizeof(i)) * 8
            signed = true
        } else {
            prec = int(unsafe.Sizeof(ui)) * 8
            signed = false
        }

    case reflect.Int:
        prec = int(unsafe.Sizeof(i)) * 8
        signed = true

    case reflect.Uint:
        prec = int(unsafe.Sizeof(ui)) * 8
        signed = false

    case reflect.Int8:
        prec = 8
        signed = true

    case reflect.Uint8:
        prec = 8
        signed = false

    case reflect.Int16:
        prec = 16
        signed = true

    case reflect.Uint16:
        prec = 16
        signed = false

    case reflect.Int32:
        prec = 32
        signed = true

    case reflect.Uint32:
        prec = 32
        signed = false

    case reflect.Int64:
        prec = 32
        signed = true

    case reflect.Uint64:
        prec = 32
        signed = false

    case reflect.Float32:
        prec = 24
        signed = true

    case reflect.Float64:
        prec = 53
        signed = true

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    // base 8 and 10 are always unsigned
    if signed && base == 10 {

        value, err := strconv.ParseInt(str, base, prec)

        // special case, range error on an interface, switch to unsigned type
        // if the number would fit in a an unsigned we'll make it work
        if err != nil && kind == reflect.Interface {
            if numError, isNumError := err.(*strconv.NumError); isNumError {
                if numError == strconv.ErrRange {
                    signed = false
                    goto do_unsigned
                }
            }
        }

        if err != nil {
            return errors.New(fmt.Sprintf("%v: cannot convert to signed integer with %d bits of precision: %s", path, prec, err.Error()))
        }

        switch kind {

        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            rv.SetInt(value)

        case reflect.Float32, reflect.Float64:
            rv.SetFloat(float64(value))

        case reflect.Interface:
            if !rv.CanAddr() {
                return errors.New(fmt.Sprintf("%v: cannot address to store int", path))
            }
            ivalue := int(value)
            rv.Set(reflect.ValueOf(ivalue))

        default:
            // should never get here, but, check anyway
            return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
        }
        return nil
    }

do_unsigned:

    // if we're forcing into a signed type, reduce precision by one bit
    if signed {
        prec--
    }

    value, err := strconv.ParseUint(str, base, prec)
    if err != nil {
        return errors.New(fmt.Sprintf("%v: cannot convert to unsigned integer with %d bits of precision: %s", path, prec, err.Error()))
    }

    switch kind {

    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        rv.SetInt(int64(value))

    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        rv.SetUint(value)

    case reflect.Float32, reflect.Float64:
        rv.SetFloat(float64(value))

    case reflect.Interface:
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store int", path))
        }
        uivalue := uint(value)
        rv.Set(reflect.ValueOf(uivalue))

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    return nil
}

type IntTag struct {
    si SchemaImplementer
}

func (t *IntTag) Tag() string {
    return DefaultLongTagPrefix + "int"
}

func (t *IntTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *IntTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *IntTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {

    // we need to descriminate (we don't allow arbitrary types for storage)
    switch kind := startRv.Kind(); kind {
    case reflect.Int, reflect.Uint, reflect.Int8, reflect.Uint8,
         reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32,
         reflect.Int64, reflect.Uint64,
         reflect.Float32, reflect.Float64,  // we allow floats (but with limited precision)
         reflect.Interface:
        // OK
    default:
        return nil, errors.New(fmt.Sprintf("%s: Cannot store a %s to a %v", path, t.Tag(), kind))
    }

    sw, err := NewScalarStateDefault(event, path, startRv, t)
    if err != nil {
        return nil, err
    }

    return &IntState {
        sw: sw,
    }, nil
}

func (t *IntTag) Specify(kind reflect.Kind) reflect.Kind {

    switch kind {
    case reflect.Interface:
        return reflect.Int

    case reflect.Int, reflect.Uint,
         reflect.Int8, reflect.Uint8,
         reflect.Int16, reflect.Uint16,
         reflect.Int32, reflect.Uint32,
         reflect.Int64, reflect.Uint64,
         reflect.Float32, reflect.Float64:

         return kind
    }
    return reflect.Invalid
}

// !!float
type FloatState struct {
    sw ScalarWrapper
}

// the ObjectWrapper interface
func (s *FloatState) StartRV() *reflect.Value {
    return s.sw.StartRV()
}

func (s *FloatState) Anchor() *string {
    return s.sw.Anchor()
}

func (s *FloatState) TagHandler() TagHandler {
    return s.sw.TagHandler()
}

func (s *FloatState) SchemaImplementer() SchemaImplementer {
    return s.sw.SchemaImplementer()
}

// the ScalarWrapper interface
func (s *FloatState) SetScalar(event *Event, path *Path) error {

    rv := s.StartRV()

    prec := 0

    // get the scalar value
    str := event.ScalarValue()

    // two time check
    kind := rv.Kind()
    switch kind {

    case reflect.Float32:
        prec = 32

    case reflect.Float64, reflect.Interface:
        prec = 64

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    value, err := strconv.ParseFloat(str, prec)
    if err != nil {
        return errors.New(fmt.Sprintf("%v: cannot convert to float with %d bits of precision: %s", path, prec, err.Error()))
    }

    switch kind {

    case reflect.Float32, reflect.Float64:
        rv.SetFloat(value)

    case reflect.Interface:
        if !rv.CanAddr() {
            return errors.New(fmt.Sprintf("%v: cannot address to store int", path))
        }
        rv.Set(reflect.ValueOf(value))

    default:
        // should never get here, but, check anyway
        return errors.New(fmt.Sprintf("%v: cannot handle kind %v for scalar", path, kind))
    }

    return nil
}

type FloatTag struct {
    si SchemaImplementer
}

func (t *FloatTag) Tag() string {
    return DefaultLongTagPrefix + "float"
}

func (t *FloatTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *FloatTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *FloatTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {

    // we need to descriminate (we don't allow arbitrary types for storage)
    switch kind := startRv.Kind(); kind {
    case reflect.Float32, reflect.Float64,  // we allow floats (but with limited precision)
         reflect.Interface:
        // OK
    default:
        return nil, errors.New(fmt.Sprintf("%s: Cannot store a %s to a %v", path, t.Tag(), kind))
    }

    sw, err := NewScalarStateDefault(event, path, startRv, t)
    if err != nil {
        return nil, err
    }

    return &FloatState {
        sw: sw,
    }, nil
}

func (t *FloatTag) Specify(kind reflect.Kind) reflect.Kind {

    switch kind {
    case reflect.Interface:
        return reflect.Float64

    case reflect.Float32, reflect.Float64:
         return kind
    }
    return reflect.Invalid
}

// !!seq
type SeqTag struct {
    si SchemaImplementer
}

func (t *SeqTag) Tag() string {
    return DefaultLongTagPrefix + "seq"
}

func (t *SeqTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *SeqTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *SeqTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {
    return NewSequenceStateDefault(event, path, startRv, t)
}

func (t *SeqTag) Specify(kind reflect.Kind) reflect.Kind {

    switch kind {
    case reflect.Interface:
        return reflect.Slice

    case reflect.Slice, reflect.Array:
         return kind
    }
    return reflect.Invalid
}

// !!map
type MapTag struct {
    si SchemaImplementer
}

func (t *MapTag) Tag() string {
    return DefaultLongTagPrefix + "map"
}

func (t *MapTag) SetSchemaImplementer(si SchemaImplementer) {
    t.si = si
}

func (t *MapTag) SchemaImplementer() SchemaImplementer {
    return t.si
}

func (t *MapTag) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error) {
    return NewMappingStateDefault(event, path, startRv, t)
}

func (t *MapTag) Specify(kind reflect.Kind) reflect.Kind {

    switch kind {
    case reflect.Interface:
        return reflect.Map

    case reflect.Struct, reflect.Map:
         return kind
    }
    return reflect.Invalid
}

////////////////////////////////////////////////////////

// the failsafe schema
type FailsafeSI struct {
    ys *YAMLSchema
}

// the SchemaImplementer interface
func (si *FailsafeSI) SchemaName() string {
    return "failsafe"
}

func (si *FailsafeSI) SchemaAliases() []string {
    return nil
}

func (si *FailsafeSI) SchemaDescription() string {
    return "The failsafe YAML schema"
}

func (si *FailsafeSI) SchemaPriority() int {
    // failsafe has a low priority
    return 0
}

func (si *FailsafeSI) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    return si.ys.DocumentStartUnmarshal(dec, root, event, path)
}

func (si *FailsafeSI) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    return si.ys.DocumentEndUnmarshal(dec, event, path)
}

func (si *FailsafeSI) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    return si.ys.NewSchemaObject(event, path, startRv)
}

// find a tag handler to match (whether explicitly or implicitly)
func (si *FailsafeSI) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {
    return si.ys.FindTagHandler(event, path, rv)
}

func (si *FailsafeSI) LookupTagHandler(tag string) (TagHandler, bool) {
    return si.ys.LookupTagHandler(tag)
}

func (si *FailsafeSI) Selected(event *Event, path *Path) {
    si.ys.Selected(event, path)
}

func (si *FailsafeSI) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    return si.ys.ResolveScalar(tag, value, kind)
}

// for the base schemas we can retrieve this (internal only)
func (si *FailsafeSI) YAMLSchema() *YAMLSchema {
    return si.ys
}

func RegisterFailsafeSchema() error {
    si := &FailsafeSI{}
    si.ys = NewYAMLSchema(FailsafeSchema, si)
    return RegisterSchema(si)
}

// the core schema (1.2)
type CoreSI struct {
    ys *YAMLSchema
}

// the SchemaImplementer interface
func (si *CoreSI) SchemaName() string {
    return "core"
}

func (si *CoreSI) SchemaAliases() []string {
    return []string{"1.2"}
}

func (si *CoreSI) SchemaDescription() string {
    return "The core YAML schema (1.2)"
}

func (si *CoreSI) SchemaPriority() int {
    // core has a modest priority
    return 10
}

func (si *CoreSI) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    return si.ys.DocumentStartUnmarshal(dec, root, event, path)
}

func (si *CoreSI) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    return si.ys.DocumentEndUnmarshal(dec, event, path)
}

func (si *CoreSI) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    return si.ys.NewSchemaObject(event, path, startRv)
}

// find a tag handler to match (whether explicitly or implicitly)
func (si *CoreSI) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {
    return si.ys.FindTagHandler(event, path, rv)
}

func (si *CoreSI) LookupTagHandler(tag string) (TagHandler, bool) {
    return si.ys.LookupTagHandler(tag)
}

func (si *CoreSI) Selected(event *Event, path *Path) {
    si.ys.Selected(event, path)
}

func (si *CoreSI) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    return si.ys.ResolveScalar(tag, value, kind)
}

func RegisterCoreSchema() error {
    si := &CoreSI{}
    si.ys = NewYAMLSchema(CoreSchema, si)
    return RegisterSchema(si)
}

// for the base schemas we can retrieve this (internal only)
func (si *CoreSI) YAMLSchema() *YAMLSchema {
    return si.ys
}

// the 1.3 schema
type Yaml13SI struct {
    ys *YAMLSchema
}

// the SchemaImplementer interface
func (si *Yaml13SI) SchemaName() string {
    return "1.3"
}

func (si *Yaml13SI) SchemaAliases() []string {
    return nil
}

func (si *Yaml13SI) SchemaDescription() string {
    return "The 1.3 YAML schema"
}

func (si *Yaml13SI) SchemaPriority() int {
    // 1.3 has a modest priority (but lower than core)
    return 9
}

func (si *Yaml13SI) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    return si.ys.DocumentStartUnmarshal(dec, root, event, path)
}

func (si *Yaml13SI) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    return si.ys.DocumentEndUnmarshal(dec, event, path)
}

func (si *Yaml13SI) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    return si.ys.NewSchemaObject(event, path, startRv)
}

// find a tag handler to match (whether explicitly or implicitly)
func (si *Yaml13SI) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {
    return si.ys.FindTagHandler(event, path, rv)
}

func (si *Yaml13SI) LookupTagHandler(tag string) (TagHandler, bool) {
    return si.ys.LookupTagHandler(tag)
}

func (si *Yaml13SI) Selected(event *Event, path *Path) {
    si.ys.Selected(event, path)
}

func (si *Yaml13SI) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    return si.ys.ResolveScalar(tag, value, kind)
}

// for the base schemas we can retrieve this (internal only)
func (si *Yaml13SI) YAMLSchema() *YAMLSchema {
    return si.ys
}

func RegisterYaml13Schema() error {
    si := &Yaml13SI{}
    si.ys = NewYAMLSchema(YAML13Schema, si)
    return RegisterSchema(si)
}

// the 1.1 schema
type Yaml11SI struct {
    ys *YAMLSchema
}

// the SchemaImplementer interface
func (si *Yaml11SI) SchemaName() string {
    return "1.1"
}

func (si *Yaml11SI) SchemaAliases() []string {
    return nil
}

func (si *Yaml11SI) SchemaDescription() string {
    return "The 1.1 YAML schema"
}

func (si *Yaml11SI) SchemaPriority() int {
    // 1.1 has the lowest priority of normal schemas
    return 1
}

func (si *Yaml11SI) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    return si.ys.DocumentStartUnmarshal(dec, root, event, path)
}

func (si *Yaml11SI) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    return si.ys.DocumentEndUnmarshal(dec, event, path)
}

func (si *Yaml11SI) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    return si.ys.NewSchemaObject(event, path, startRv)
}

// find a tag handler to match (whether explicitly or implicitly)
func (si *Yaml11SI) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {
    return si.ys.FindTagHandler(event, path, rv)
}

func (si *Yaml11SI) LookupTagHandler(tag string) (TagHandler, bool) {
    return si.ys.LookupTagHandler(tag)
}

func (si *Yaml11SI) Selected(event *Event, path *Path) {
    si.ys.Selected(event, path)
}

func (si *Yaml11SI) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    return si.ys.ResolveScalar(tag, value, kind)
}

// for the base schemas we can retrieve this (internal only)
func (si *Yaml11SI) YAMLSchema() *YAMLSchema {
    return si.ys
}

func RegisterYaml11Schema() error {
    si := &Yaml11SI{}
    si.ys = NewYAMLSchema(YAML11Schema, si)
    return RegisterSchema(si)
}

// the JSON schema
type JsonSI struct {
    ys *YAMLSchema
}

// the SchemaImplementer interface
func (si *JsonSI) SchemaName() string {
    return "json"
}

func (si *JsonSI) SchemaAliases() []string {
    return nil
}

func (si *JsonSI) SchemaDescription() string {
    return "The JSON schema"
}

func (si *JsonSI) SchemaPriority() int {
    // JSON can only be selected manually, or when parsing JSON, so low
    return 0
}

func (si *JsonSI) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    return si.ys.DocumentStartUnmarshal(dec, root, event, path)
}

func (si *JsonSI) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    return si.ys.DocumentEndUnmarshal(dec, event, path)
}

func (si *JsonSI) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    return si.ys.NewSchemaObject(event, path, startRv)
}

// find a tag handler to match (whether explicitly or implicitly)
func (si *JsonSI) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {
    return si.ys.FindTagHandler(event, path, rv)
}

func (si *JsonSI) LookupTagHandler(tag string) (TagHandler, bool) {
    return si.ys.LookupTagHandler(tag)
}

func (si *JsonSI) Selected(event *Event, path *Path) {
    si.ys.Selected(event, path)
}

func (si *JsonSI) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    return si.ys.ResolveScalar(tag, value, kind)
}

// for the base schemas we can retrieve this (internal only)
func (si *JsonSI) YAMLSchema() *YAMLSchema {
    return si.ys
}

func RegisterJsonSchema() error {
    si := &JsonSI{}
    si.ys = NewYAMLSchema(JSONSchema, si)
    return RegisterSchema(si)
}

// the auto schema
type AutoSI struct {
    si SchemaImplementer    // the real one
}

// the SchemaImplementer interface
func (si *AutoSI) SchemaName() string {
    return "auto"
}

func (si *AutoSI) SchemaAliases() []string {
    return nil
}

func (si *AutoSI) SchemaDescription() string {

    desc := "The automatic schema"

    // if not selected yet return the 
    if si.si == nil {
        return desc + " (not configured yet)"
    }
    return desc + " forwarding to " + si.si.SchemaDescription()
}

func (si *AutoSI) SchemaPriority() int {
    // Auto has the highest of all regular schemas
    return 20
}

func (si *AutoSI) DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error) {
    // if not selected yet return nil
    if si.si == nil {
        return nil, errors.New(fmt.Sprintf("%v: Cannot DocumentStartUnmarshal(), not configured", path))
    }
    return si.si.DocumentStartUnmarshal(dec, root, event, path)
}

func (si *AutoSI) DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error {
    // if not selected yet return nil
    if si.si == nil {
        return errors.New(fmt.Sprintf("%v: Cannot DocumentEndUnmarshal(), not configured", path))
    }
    return si.si.DocumentEndUnmarshal(dec, event, path)
}

func (si *AutoSI) NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error) {
    // if not selected yet return nil
    if si.si == nil {
        return nil, errors.New(fmt.Sprintf("%v: Cannot create new Schema object, not configured", path))
    }
    return si.si.NewSchemaObject(event, path, startRv)
}

// find a tag handler to match (whether explicitly or implicitly)
func (si *AutoSI) FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error) {
    // if not selected yet return nil
    if si.si == nil {
        return nil, false, errors.New(fmt.Sprintf("%v: Cannot find a handler not configured", path))
    }
    return si.si.FindTagHandler(event, path, rv)
}

func (si *AutoSI) LookupTagHandler(tag string) (TagHandler, bool) {
    // if not selected yet return nil
    if si.si == nil {
        return nil, false
    }
    return si.si.LookupTagHandler(tag)
}

func (si *AutoSI) Selected(event *Event, path *Path) {

    // get the document state
    ds := event.DocumentState()

    schema := ""
    // select according to the version, or json mode
    if !ds.JSONMode() {
        schema = ds.Version().String()
    } else {
        schema = "json"
    }

    // find according to the version/json mode
    si.si = LookupSchema(schema)

    // not found? the failsafe should exist at least
    if si.si == nil {
        si.si = LookupSchema("failsafe")
    }

    // we should at least get the failsafe, if not, panic()
    if si.si == nil {
        panic(fmt.Sprintf("Unabled to select a schema at all for %s", schema))
    }

    // and forward the selected
    si.si.Selected(event, path)
}

func (si *AutoSI) ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind) {
    // if not selected yet return nil
    if si.si == nil {
        return nil, reflect.Invalid
    }
    return si.si.ResolveScalar(tag, value, kind)
}

// for the base schemas we can retrieve this (internal only)
func (si *AutoSI) YAMLSchema() *YAMLSchema {
    // no schema until configuration
    if si.si == nil {
        return nil
    }
    // check if we have that
    if ysp, hasYsp := si.si.(YAMLSchemaProvider); hasYsp {
        return ysp.YAMLSchema()
    }
    return nil
}

func RegisterAutoSchema() error {

    si := &AutoSI{}
    return RegisterSchema(si);
}

// register all the YAML schemas
func RegisterYAMLSchemas() error {

    // start with the failsafe
    if err := RegisterFailsafeSchema(); err != nil {
        return err
    }

    // the core (1.2)
    if err := RegisterCoreSchema(); err != nil {
        return err
    }

    // the 1.3
    if err := RegisterYaml13Schema(); err != nil {
        return err
    }

    // the 1.1
    if err := RegisterYaml11Schema(); err != nil {
        return err
    }

    // the json schema
    if err := RegisterJsonSchema(); err != nil {
        return err
    }

    // the auto schema
    if err := RegisterAutoSchema(); err != nil {
        return err
    }

    return nil
}
