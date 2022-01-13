// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "sync"
    "errors"
    "sort"
    "reflect"
)

// efficient set
type uvoid struct{}

// this is an abstract address wrapper interface
type AddressWrapper interface {
    String() string
}

type ObjectWrapper interface {
    StartRV() *reflect.Value
    Anchor() *string
    TagHandler() TagHandler
    SchemaImplementer() SchemaImplementer
}

// something that can wrap a collection
// it's either a root, sequence or map state
type CollectionWrapper interface {
    ObjectWrapper
    ObjStartIn(event *Event, path *Path) (ObjectWrapper, error)
    ObjEndIn(event *Event, path *Path, ow ObjectWrapper) error
    CollectionStart(event *Event, path *Path) error
    CollectionEnd(event *Event, path *Path) error
    CurrentAddress(path *Path) AddressWrapper
}

// something that can wrap a scalar/alias
type ScalarWrapper interface {
    ObjectWrapper
    SetScalar(event *Event, path *Path) error
}

// per tag type (just the tag name and the type)
type TagHandler interface {
    Tag() string
    NewSchemaObject(event *Event, path *Path, startRv *reflect.Value, si SchemaImplementer) (ObjectWrapper, error)
    SetSchemaImplementer(si SchemaImplementer)
    SchemaImplementer() SchemaImplementer
    Specify(kind reflect.Kind) reflect.Kind
}

// just object creator
type SchemaObjectCreator interface {
    NewSchemaObject(event *Event, path *Path, startRv *reflect.Value) (ObjectWrapper, error)
}

// just resolver
type SchemaResolver interface {
    ResolveScalar(tag, value *string, kind reflect.Kind) (TagHandler, reflect.Kind)
}

// the schema implementer
type SchemaImplementer interface {
    SchemaObjectCreator
    SchemaResolver

    SchemaName() string
    SchemaAliases() []string
    SchemaDescription() string
    SchemaPriority() int
    FindTagHandler(event *Event, path *Path, rv *reflect.Value) (TagHandler, bool, error)
    LookupTagHandler(tag string) (TagHandler, bool)
    Selected(event *Event, path *Path)

    DocumentStartUnmarshal(dec *Decoder, root interface{}, event *Event, path *Path) (CollectionWrapper, error)
    DocumentEndUnmarshal(dec *Decoder, event *Event, path *Path) error
}

type Schema struct {
    genId uint64
    name string
    si SchemaImplementer
}

type SchemaRegistry struct {
    schemas map[string]*Schema
    lock sync.RWMutex
    nextGenId uint64
    init bool
}

// register a schema to the registry
func (r *SchemaRegistry) Register(si SchemaImplementer) error {

    r.lock.Lock()
    if r.schemas == nil {
        r.schemas = make(map[string]*Schema)
    }

    // make a string slice of the main name and the aliases
    names := make([]string, 0)
    names = append(names, si.SchemaName())
    names = append(names, si.SchemaAliases()...)

    for _, name := range names {
        // check if it's already there
        if _, hasSchema := r.schemas[name]; hasSchema {
            r.lock.Unlock()
            return errors.New("schema " + name + " already exists")
        }

        // add it to the map
        r.schemas[name] = &Schema {
            genId: r.nextGenId,
            name: name,
            si: si,
        }
        r.nextGenId++
    }

    r.lock.Unlock()

    return nil
}

// unregister a schema from the registry
func (r *SchemaRegistry) Unregister(si SchemaImplementer) {

    r.lock.Lock()
    if r.schemas == nil {
        r.schemas = make(map[string]*Schema)
    }

    // make a string slice of the main name and the aliases
    names := make([]string, 0)
    names = append(names, si.SchemaName())
    names = append(names, si.SchemaAliases()...)

    for _, name := range names {
        // ignore errors
        if _, hasSchema := r.schemas[name]; hasSchema {
            delete(r.schemas, name)
        }
    }

    r.lock.Unlock()
}

// sorted by insertion order (genId) schema list (names)
func (r *SchemaRegistry) List() []string {

    // read lock
    r.lock.RLock()

    // collect (in random order) the registered schema names
    list := make([]*Schema, len(r.schemas))
    idx := 0
    for _, s := range r.schemas {
        list[idx] = s
        idx++
    }

    // sort by priority and genId
    sort.Slice(list, func(i, j int) bool {
        ip := list[i].si.SchemaPriority()
        jp := list[j].si.SchemaPriority()
        if ip != jp {
            return ip > jp  // highest priority goes in front
        }
        // fallback to genId checking
        return list[i].genId < list[j].genId
    })

    // now create the name slice
    nameList := make([]string, len(list))
    for i, s := range list {
        nameList[i] = s.name
    }

    r.lock.RUnlock()

    return nameList
}

func (r *SchemaRegistry) Lookup(name string) SchemaImplementer {

    var si SchemaImplementer = nil

    r.lock.RLock()
    if s, hasSchema := r.schemas[name]; hasSchema {
        si = s.si
    }
    r.lock.RUnlock()

    return si
}

func (r *SchemaRegistry) Select(name string, event *Event, path *Path) (SchemaImplementer, error) {

    // empty name, go with priority
    if name == "" {
        // could not find directly, get the highest priority one
        schemas := r.List()
        if len(schemas) > 0 {
            name = schemas[0]
        }
    }

    // first try a name lookup
    si := r.Lookup(name)
    if si == nil {
        // not found? impossible but fallback to failsafe
        si = r.Lookup("failsafe")
    }

    if si == nil {
        return nil, errors.New("Unable to select a schema")
    }

    si.Selected(event, path)

    return si, nil
}

// global schemas
var GlobalSchemaRegistry *SchemaRegistry = &SchemaRegistry{}

func RegisterSchema(si SchemaImplementer) error {
    return GlobalSchemaRegistry.Register(si)
}

func UnregisterSchema(si SchemaImplementer) {
    GlobalSchemaRegistry.Unregister(si)
}

func ListSchema() []string {
    return GlobalSchemaRegistry.List()
}

func LookupSchema(name string) SchemaImplementer {
    return GlobalSchemaRegistry.Lookup(name)
}

func SelectSchema(name string, event *Event, path *Path) (SchemaImplementer, error) {
    return GlobalSchemaRegistry.Select(name, event, path)
}
