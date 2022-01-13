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

func LibraryVersion() string {
    return C.GoString(C.fy_library_version())
}

type PathComponent C.struct_fy_path_component

func (pc *PathComponent) C() *C.struct_fy_path_component {
    return (*C.struct_fy_path_component)(pc)
}

func (pc *PathComponent) String() string {
    cstr := C.fy_path_component_get_text(pc.C())
    defer C.free(unsafe.Pointer(cstr))

    return C.GoString(cstr)
}

func (pc *PathComponent) IsSequence() bool {
    return bool(C.fy_path_component_is_sequence(pc.C()))
}

func (pc *PathComponent) IsMapping() bool {
    return bool(C.fy_path_component_is_mapping(pc.C()))
}

func (pc *PathComponent) SequenceIndex() int {
    return int(C.fy_path_component_sequence_get_index(pc.C()))
}

func (pc *PathComponent) MappingScalarKey() *Token {
    return (*Token)(C.fy_path_component_mapping_get_scalar_key(pc.C()))
}

func (pc *PathComponent) MappingScalarKeyTag() *Token {
    return (*Token)(C.fy_path_component_mapping_get_scalar_key_tag(pc.C()))
}

func (pc *PathComponent) MappingComplexKey() *Document {
    return (*Document)(C.fy_path_component_mapping_get_complex_key(pc.C()))
}

func (pc *PathComponent) MappingUserData() interface{} {
    ptr := C.fy_path_component_get_mapping_user_data(pc.C())
    if ptr == nil {
        return nil
    }
    return gopointer.Restore(ptr)
}

func (pc *PathComponent) SetMappingUserData(v interface{}) {
    ptr := C.fy_path_component_get_mapping_user_data(pc.C())
    if ptr != nil {
        gopointer.Unref(ptr)
    }
    if v != nil {
        C.fy_path_component_set_mapping_user_data(pc.C(), gopointer.Save(v))
    } else {
        C.fy_path_component_set_mapping_user_data(pc.C(), C.NULL)
    }
}

func (pc *PathComponent) MappingKeyUserData() interface{} {
    ptr := C.fy_path_component_get_mapping_key_user_data(pc.C())
    if ptr == nil {
        return nil
    }
    return gopointer.Restore(ptr)
}

func (pc *PathComponent) SetMappingKeyUserData(v interface{}) {
    ptr := C.fy_path_component_get_mapping_key_user_data(pc.C())
    if ptr != nil {
        gopointer.Unref(ptr)
    }
    if v != nil {
        C.fy_path_component_set_mapping_key_user_data(pc.C(), gopointer.Save(v))
    } else {
        C.fy_path_component_set_mapping_key_user_data(pc.C(), C.NULL)
    }
}

func (pc *PathComponent) SequenceUserData() interface{} {
    ptr := C.fy_path_component_get_sequence_user_data(pc.C())
    if ptr == nil {
        return nil
    }
    return gopointer.Restore(ptr)
}

func (pc *PathComponent) SetSequenceUserData(v interface{}) {
    ptr := C.fy_path_component_get_sequence_user_data(pc.C())
    if ptr != nil {
        gopointer.Unref(ptr)
    }
    if v != nil {
        C.fy_path_component_set_sequence_user_data(pc.C(), gopointer.Save(v))
    } else {
        C.fy_path_component_set_sequence_user_data(pc.C(), C.NULL)
    }
}

func (pc *PathComponent) CollectionUserData() interface{} {
    if pc.IsMapping() {
        return pc.MappingUserData()
    } else {
        return pc.SequenceUserData()
    }
}

func (pc *PathComponent) SetCollectionUserData(v interface{}) {
    if pc.IsMapping() {
        pc.SetMappingUserData(v)
    } else {
        pc.SetSequenceUserData(v)
    }
}

type Path C.struct_fy_path

func (path *Path) C() *C.struct_fy_path {
    return (*C.struct_fy_path)(path)
}

func (path *Path) String() string {
    cpathstr := C.fy_path_get_text(path.C())
    defer C.free(unsafe.Pointer(cpathstr))

    return C.GoString(cpathstr)
}

func (path *Path) InRoot() bool {
    return bool(C.fy_path_in_root(path.C()))
}

func (path *Path) InMapping() bool {
    return bool(C.fy_path_in_mapping(path.C()))
}

func (path *Path) InSequence() bool {
    return bool(C.fy_path_in_sequence(path.C()))
}

func (path *Path) InMappingKey() bool {
    return bool(C.fy_path_in_mapping_key(path.C()))
}

func (path *Path) InMappingValue() bool {
    return bool(C.fy_path_in_mapping_value(path.C()))
}

func (path *Path) Depth() int {
    return int(C.fy_path_depth(path.C()))
}

func (path *Path) InCollectionRoot() bool {
    return bool(C.fy_path_in_collection_root(path.C()))
}

func (path *Path) LastComponent() *PathComponent {
    return (*PathComponent)(C.fy_path_last_component(path.C()))
}

func (path *Path) LastNotCollectionRootComponent() *PathComponent {
    return (*PathComponent)(C.fy_path_last_not_collection_root_component(path.C()))
}

func (p *Path) RootUserData() interface{} {
    ptr := C.fy_path_get_root_user_data(p.C())
    if ptr == nil {
        return nil
    }
    return gopointer.Restore(ptr)
}

func (p *Path) SetRootUserData(v interface{}) {
    ptr := C.fy_path_get_root_user_data(p.C())
    if ptr != nil {
        gopointer.Unref(ptr)
    }
    if v != nil {
        C.fy_path_set_root_user_data(p.C(), gopointer.Save(v))
    } else {
        C.fy_path_set_root_user_data(p.C(), C.NULL)
    }
}

func (p *Path) ParentUserData() interface{} {
    if p.InRoot() {
        return p.RootUserData()
    }
    parent := p.LastNotCollectionRootComponent()
    if p.InSequence() {
        return parent.SequenceUserData()
    } else {
        return parent.MappingUserData()
    }
}

func (p *Path) SetParentUserData(v interface{}) {
    if p.InRoot() {
        p.SetRootUserData(v)
    } else {
        parent := p.LastNotCollectionRootComponent()
        if p.InSequence() {
            parent.SetSequenceUserData(v)
        } else {
            parent.SetMappingUserData(v)
        }
    }
}

func (p *Path) LastUserData() interface{} {
    last := p.LastComponent()
    if last == nil {
        return p.RootUserData()
    } else if last.IsSequence() {
        return last.SequenceUserData()
    } else {
        return last.MappingUserData()
    }
}

func (p *Path) SetLastUserData(v interface{}) {
    last := p.LastComponent()
    if last == nil {
        p.SetRootUserData(v)
    } else if last.IsSequence() {
        last.SetSequenceUserData(v)
    } else {
        last.SetMappingUserData(v)
    }
}

type Token C.struct_fy_token

func (t *Token) C() *C.struct_fy_token {
    return (*C.struct_fy_token)(t)
}

func (t *Token) Text() string {
    return C.GoString(C.fy_token_get_text0(t.C()))
}

func (t *Token) ScalarStyle() ScalarStyle {
    return ScalarStyle(C.fy_token_scalar_style(t.C()))
}

type Version C.struct_fy_version

func (v *Version) C() *C.struct_fy_version {
    return (*C.struct_fy_version)(v)
}

func (v *Version) String() string {
    return fmt.Sprintf("%d.%d", v.major, v.minor)
}

func VersionDefault() *Version {
    return (*Version)(C.fy_version_default())
}

func (v *Version) IsSupported() bool {
    return bool(C.fy_version_is_supported(v.C()))
}

func VersionsSupported() []*Version {
    var vp *C.struct_fy_version = nil
    var versions []*Version = nil
    var prevp unsafe.Pointer = nil
    for {
        vp = C.fy_version_supported_iterate(&prevp)
        if vp == nil {
            break
        }
        versions = append(versions, (*Version)(vp))
    }
    return versions
}

type Document C.struct_fy_document

func (d *Document) C() *C.struct_fy_document {
    return (*C.struct_fy_document)(d)
}

type DocumentState C.struct_fy_document_state

func (ds *DocumentState) C() *C.struct_fy_document_state {
    return (*C.struct_fy_document_state)(ds)
}

func (ds *DocumentState) Version() *Version {
    return (*Version)(C.fy_document_state_version(ds.C()))
}

func (ds *DocumentState) JSONMode() bool {
    return (bool)(C.fy_document_state_json_mode(ds.C()))
}

type NodeStyle C.enum_fy_node_style

const (
    AnyStyle NodeStyle  = C.FYNS_ANY
    FlowStyle           = C.FYNS_FLOW
    BlockStyle          = C.FYNS_BLOCK
    PlainStyle          = C.FYNS_PLAIN
    SingleQuotedStyle   = C.FYNS_SINGLE_QUOTED
    DoubleQuotedStyle   = C.FYNS_DOUBLE_QUOTED
    LiteralStyle        = C.FYNS_LITERAL
    FoldedStyle         = C.FYNS_FOLDED
    AliasStyle          = C.FYNS_ALIAS
)

func (ns NodeStyle) C() C.enum_fy_node_style {
    return C.enum_fy_node_style(ns)
}

func (ns NodeStyle) String() string {
    switch ns {
    case FlowStyle:
        return "flow"
    case BlockStyle:
        return "block"
    case AnyStyle:
        return "any"
    case PlainStyle:
        return "plain"
    case SingleQuotedStyle:
        return "single-quoted"
    case DoubleQuotedStyle:
        return "double-quoted"
    case LiteralStyle:
        return "literal"
    case FoldedStyle:
        return "folded"
    case Alias:
        return "alias"
    }
    return ""
}

type EventType C.enum_fy_event_type

const (
    None EventType  = C.FYET_NONE
    StreamStart     = C.FYET_STREAM_START
    StreamEnd       = C.FYET_STREAM_END
    DocumentStart   = C.FYET_DOCUMENT_START
    DocumentEnd     = C.FYET_DOCUMENT_END
    Scalar          = C.FYET_SCALAR
    Alias           = C.FYET_ALIAS
    SequenceStart   = C.FYET_SEQUENCE_START
    SequenceEnd     = C.FYET_SEQUENCE_END
    MappingStart    = C.FYET_MAPPING_START
    MappingEnd      = C.FYET_MAPPING_END
)

func (etype EventType) C() C.enum_fy_event_type {
    return C.enum_fy_event_type(etype)
}

func (etype EventType) String() string {
    return C.GoString(C.fy_event_type_get_text(C.enum_fy_event_type(etype)))
}

type Event C.struct_fy_event

func (e *Event) C() *C.struct_fy_event {
    return (*C.struct_fy_event)(e)
}

func (e *Event) String() string {
    return EventType(e._type).String()
}

func (e *Event) Type() EventType {
    return EventType((*C.struct_fy_event)(e)._type)
}

type StreamStartData C.struct_fy_event_stream_start_data
type StreamEndData C.struct_fy_event_stream_end_data
type DocumentStartData C.struct_fy_event_document_start_data
type DocumentEndData C.struct_fy_event_document_end_data
type ScalarData C.struct_fy_event_scalar_data
type AliasData C.struct_fy_event_alias_data
type SequenceStartData C.struct_fy_event_sequence_start_data
type SequenceEndData C.struct_fy_event_sequence_end_data
type MappingStartData C.struct_fy_event_mapping_start_data
type MappingEndData C.struct_fy_event_mapping_end_data

func (e *Event) Data() interface{} {

    switch e.Type() {
    case StreamStart:
        return (*StreamStartData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case StreamEnd:
        return (*StreamEndData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case DocumentStart:
        return (*DocumentStartData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case DocumentEnd:
        return (*DocumentEndData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case Scalar:
        return (*ScalarData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case Alias:
        return (*AliasData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case SequenceStart:
        return (*SequenceStartData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case SequenceEnd:
        return (*SequenceEndData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case MappingStart:
        return (*MappingStartData)(C.fy_event_data((*C.struct_fy_event)(e)))
    case MappingEnd:
        return (*MappingEndData)(C.fy_event_data((*C.struct_fy_event)(e)))
    }
    return nil
}

// return the main Token of an event
func (e *Event) Token() *Token {
    return (*Token)(C.fy_event_get_token(e.C()))
}

// return the anchor Token of an event or nil if it does not exist
func (e *Event) Anchor() *Token {
    return (*Token)(C.fy_event_get_anchor_token(e.C()))
}

func (e *Event) AnchorString() *string {
    if anchorToken := e.Anchor(); anchorToken != nil {
        text := anchorToken.Text()
        return &text
    }
    return nil
}

// return the tag Token of an event or nil if it does not exist
func (e *Event) Tag() *Token {
    return (*Token)(C.fy_event_get_tag_token(e.C()))
}

func (e *Event) IsImplicit() bool {
    switch e.Type() {
    case DocumentStart:
        return bool(e.Data().(*DocumentStartData).implicit)
    case DocumentEnd:
        return bool(e.Data().(*DocumentEndData).implicit)
    case Scalar:
        return bool(e.Data().(*ScalarData).tag_implicit)
    }
    return false
}

func (e *Event) DocumentState() *DocumentState {
    // only on document start
    if e.Type() != DocumentStart {
        return nil
    }
    return (*DocumentState)(e.Data().(*DocumentStartData).document_state)
}

func (e *Event) ScalarValuePtr() *string {

    // only on scalars
    if e.Type() != Scalar {
        return nil
    }
    scalar := (*C.struct_fy_event_scalar_data)(C.fy_event_data(e.C()))
    if scalar.value == nil {
        return nil  // a nil is OK
    }
    str := C.GoString(C.fy_token_get_text0(scalar.value))
    return &str
}

// wrapper to return "" on null value
func (e *Event) ScalarValue() string {
    strp := e.ScalarValuePtr()
    if strp == nil {
        return ""
    }
    return *strp
}

func (e *Event) IsNullScalar() bool {
    value := e.ScalarValuePtr()
    // if it's not a scalar it's a null
    return value == nil
}

type ScalarStyle C.enum_fy_scalar_style

const (
    Any ScalarStyle = C.FYSS_ANY
    Plain           = C.FYSS_PLAIN
    SingleQuoted    = C.FYSS_SINGLE_QUOTED
    DoubleQuoted    = C.FYSS_DOUBLE_QUOTED
    Literal         = C.FYSS_LITERAL
    Folded          = C.FYSS_FOLDED
)

func (ss ScalarStyle) C() C.enum_fy_scalar_style {
    return C.enum_fy_scalar_style(ss)
}

func (ss ScalarStyle) String() string {
    switch ss {
    case Any:
        return "any"
    case Plain:
        return "plain"
    case SingleQuoted:
        return "single-quoted"
    case DoubleQuoted:
        return "double-quoted"
    case Literal:
        return "literal"
    case Folded:
        return "folded"
    }
    return ""
}

type ParseCfg C.struct_fy_parse_cfg

func (pc *ParseCfg) C() *C.struct_fy_parse_cfg {
    return (*C.struct_fy_parse_cfg)(pc)
}

// does not set user data
func ParseCfgCreate(a CMemTrackerAllocator, opts...interface{}) (*ParseCfg, error) {

    // get the options if any
    o, err := GetOptions(opts)
    if err != nil {
        return nil, err
    }

    var pc *ParseCfg = nil
    pc = (*ParseCfg)(a.Allocate(int(unsafe.Sizeof(*pc))))

    pc.search_path = a.CString("")
    pc.flags = 0    // expect 0, revisit if changes

    if o.Quiet {
        pc.flags |= C.FYPCF_QUIET
    }

    switch o.Version {
    case "auto":
        pc.flags |= C.FYPCF_DEFAULT_VERSION_AUTO
    case "1.1":
        pc.flags |= C.FYPCF_DEFAULT_VERSION_1_1
    case "1.2":
        pc.flags |= C.FYPCF_DEFAULT_VERSION_1_2
    case "1.3":
        pc.flags |= C.FYPCF_DEFAULT_VERSION_1_3
    }

    if o.SloppyFlowIndentation {
        pc.flags |= C.FYPCF_SLOPPY_FLOW_INDENTATION
    }

    switch o.JSON {
    case "auto":
        pc.flags |= C.FYPCF_JSON_AUTO
    case "none":
        pc.flags |= C.FYPCF_JSON_NONE
    case "force":
        pc.flags |= C.FYPCF_JSON_FORCE
    }

    if o.Resolve {
        // we turn on both the resolve and the allow duplicate keys
        // option; we want GO to handle key equality
        pc.flags |= C.FYPCF_RESOLVE_DOCUMENT | C.FYPCF_ALLOW_DUPLICATE_KEYS
    }

    return pc, nil
}

func (pc *ParseCfg) Destroy(a CMemTrackerAllocator) {
    if pc == nil {
        return
    }

    // free the search path if it exists
    if pc.search_path != nil {
        a.Free(unsafe.Pointer(pc.search_path))
    }

    // and the C memory
    a.Free(unsafe.Pointer(pc.C()))
}

type Parser C.struct_fy_parser

func (p *Parser) C() *C.struct_fy_parser {
    return (*C.struct_fy_parser)(p)
}

func ParserCreate(a CMemTrackerAllocator, opts...interface{}) (*Parser, error) {

    // get the options if any
    o, err := GetOptions(opts)
    if err != nil {
        return nil, err
    }

    // create the configuration
    // we need it hanging around on the
    // memory tracker until exit
    cfg, err := ParseCfgCreate(a, o)
    if err != nil {
        return nil, err
    }

    // save the allocator (and associated object)
    cfg.userdata = gopointer.Save(a)

    p := (*Parser)(C.fy_parser_create(cfg.C()))
    if p == nil {
        return nil, errors.New("Failed to create parser\n")
    }

    return p, nil
}

func (p *Parser) CMemTrackerAllocator() CMemTrackerAllocator {
    cfg := C.fy_parser_get_cfg(p.C())
    return gopointer.Restore(cfg.userdata).(CMemTrackerAllocator)
}

// just forward to the configured allocator object
func (p *Parser) Allocate(size int) unsafe.Pointer {
    return p.CMemTrackerAllocator().Allocate(size)
}

func (p *Parser) CString(str string) *C.char {
    return p.CMemTrackerAllocator().CString(str)
}

func (p *Parser) Free(ptr unsafe.Pointer) {
    p.CMemTrackerAllocator().Free(ptr)
}

func (p *Parser) IsTracked(ptr unsafe.Pointer) bool {
    return p.CMemTrackerAllocator().IsTracked(ptr)
}

func (p *Parser) FreeAll() {
    p.CMemTrackerAllocator().FreeAll()
}

func (p *Parser) Destroy() {
    // get the current configuration
    cfg := (*ParseCfg)(C.fy_parser_get_cfg(p.C()))
    gopointer.Unref(cfg.userdata)

    C.fy_parser_destroy(p.C())
}

func (p *Parser) SetInputFile(file string) error {

    rc := C.fy_parser_set_input_file(p.C(), p.CString(file)); if rc != 0 {
        return errors.New(fmt.Sprintf("Failed to set input file: %s", file))
    }

    return nil
}

func (p *Parser) SetInputData(data unsafe.Pointer, size uint) error {

    // point the parser there
    if rc := C.fy_parser_set_string(p.C(), (*C.char)(data), C.size_t(size)); rc != 0 {
        return errors.New("failed to set input to data")
    }
    return nil
}

type EventProcessor interface {
    ProcessEvent(e *Event, path *Path) (stop bool, err error)
    SetError(err error)
    Error() error
}

// possible GO bug; no method with the same name (even with a receiver may be used)
//export FY_ProcessEvent
func FY_ProcessEvent(fyp *C.struct_fy_parser, fye *C.struct_fy_event, path *C.struct_fy_path, userdata *C.void) C.enum_fy_composer_return {

    var data unsafe.Pointer = unsafe.Pointer(userdata)

    if data == nil {
        panic("Userdata nil in FY_ProcessEvent callback")
    }

    // restore the EventProcessor interface object
    processor := gopointer.Restore(data).(EventProcessor)

    // go into GO proper
    stop, err := processor.ProcessEvent((*Event)(fye), (*Path)(path))

    // any error?
    if err != nil {
        processor.SetError(err)
        return C.FYCR_ERROR
    }

    // regular stop?
    if stop {
        return C.FYCR_OK_STOP
    }

    // both errors and stop false
    return C.FYCR_OK_CONTINUE
}

type EmitterCfg C.struct_fy_emitter_cfg

func (pc *EmitterCfg) C() *C.struct_fy_emitter_cfg {
    return (*C.struct_fy_emitter_cfg)(pc)
}

func emitterCfgFlagsFromOptions(o *Options) C.enum_fy_emitter_cfg_flags {

    var flags C.enum_fy_emitter_cfg_flags

    flags = C.FYECF_DEFAULT

    if o.Indent != 0 {
        flags &= ^C.enum_fy_emitter_cfg_flags(C.FYECF_INDENT_MASK << C.FYECF_INDENT_SHIFT)
        flags |=  C.enum_fy_emitter_cfg_flags((uint(o.Indent) & C.FYECF_INDENT_MASK) << C.FYECF_INDENT_SHIFT)
    }

    if o.Width != 0 {
        w := o.Width
        if w < 0 {
            w = 255
        } else if w > 254 {
            w = 254
        }
        flags &= ^C.enum_fy_emitter_cfg_flags(C.FYECF_WIDTH_MASK << C.FYECF_WIDTH_SHIFT)
        flags |=  C.enum_fy_emitter_cfg_flags((uint(w) & C.FYECF_WIDTH_MASK) << C.FYECF_WIDTH_SHIFT)
    }

    if o.SortKeys {
        flags |= C.FYECF_SORT_KEYS
    }

    if o.StripLabels {
        flags |= C.FYECF_STRIP_LABELS
    }

    if o.StripTags {
        flags |= C.FYECF_STRIP_TAGS
    }

    if o.StripDocIndicators {
        flags |= C.FYECF_STRIP_DOC
    }

    if o.NoEndingNewline {
        flags |= C.FYECF_NO_ENDING_NEWLINE
    }

    if o.OutputMode != "" {
        flags &= ^C.enum_fy_emitter_cfg_flags(C.FYECF_MODE_MASK << C.FYECF_MODE_SHIFT)
        switch o.OutputMode {
        default:
        case "original":
            flags |= C.FYECF_MODE_ORIGINAL
        case "block":
            flags |= C.FYECF_MODE_BLOCK
        case "flow":
            flags |= C.FYECF_MODE_FLOW
        case "flow-oneline":
            flags |= C.FYECF_MODE_FLOW_ONELINE
        case "json":
            flags |= C.FYECF_MODE_JSON
        case "json-oneline":
            flags |= C.FYECF_MODE_JSON_ONELINE
        case "dejson":
            flags |= C.FYECF_MODE_DEJSON
        case "pretty", "yamlfmt":
            flags |= C.FYECF_MODE_PRETTY
        }
    }

    if o.VersionDirectives != "" {
        flags &= ^C.enum_fy_emitter_cfg_flags(C.FYECF_VERSION_DIR_MASK << C.FYECF_VERSION_DIR_SHIFT)
        switch o.VersionDirectives {
        case "auto":
            flags |= C.FYECF_VERSION_DIR_AUTO
        case "off":
            flags |= C.FYECF_VERSION_DIR_OFF
        case "on":
            flags |= C.FYECF_VERSION_DIR_ON
        }
    }

    if o.TagDirectives != "" {
        flags &= ^C.enum_fy_emitter_cfg_flags(C.FYECF_TAG_DIR_MASK << C.FYECF_TAG_DIR_SHIFT)
        switch o.TagDirectives {
        case "auto":
            flags |= C.FYECF_TAG_DIR_AUTO
        case "off":
            flags |= C.FYECF_TAG_DIR_OFF
        case "on":
            flags |= C.FYECF_TAG_DIR_ON
        }
    }

    return flags
}

// does not set user data
func EmitterCfgCreate(a CMemTrackerAllocator, opts...interface{}) (*EmitterCfg, error) {

    // get the options if any
    o, err := GetOptions(opts)
    if err != nil {
        return nil, err
    }

    var ec *EmitterCfg = nil
    ec = (*EmitterCfg)(a.Allocate(int(unsafe.Sizeof(*ec))))

    ec.flags = emitterCfgFlagsFromOptions(o)

    return ec, nil
}

func (ec *EmitterCfg) Destroy(a CMemTrackerAllocator) {
    if ec == nil {
        return
    }

    // and the C memory
    a.Free(unsafe.Pointer(ec.C()))
}

type Emitter C.struct_fy_emitter

func (e *Emitter) C() *C.struct_fy_emitter {
    return (*C.struct_fy_emitter)(e)
}

func EmitterCreate(a CMemTrackerAllocator, opts...interface{}) (*Emitter, error) {

    // get the options if any
    o, err := GetOptions(opts)
    if err != nil {
        return nil, err
    }

    // create the configuration
    // we need it hanging around on the
    // memory tracker until exit
    cfg, err := EmitterCfgCreate(a, o)
    if err != nil {
        return nil, err
    }

    // save the allocator (and associated object)
    cfg.userdata = gopointer.Save(a)

    e := (*Emitter)(C.fy_emitter_create(cfg.C()))
    if e == nil {
        return nil, errors.New("Failed to create emitter\n")
    }

    return e, nil
}

func (e *Emitter) CMemTrackerAllocator() CMemTrackerAllocator {
    cfg := C.fy_emitter_get_cfg(e.C())
    return gopointer.Restore(cfg.userdata).(CMemTrackerAllocator)
}

// just forward to the configured allocator object
func (e *Emitter) Allocate(size int) unsafe.Pointer {
    return e.CMemTrackerAllocator().Allocate(size)
}

func (e *Emitter) CString(str string) *C.char {
    return e.CMemTrackerAllocator().CString(str)
}

func (e *Emitter) Free(ptr unsafe.Pointer) {
    e.CMemTrackerAllocator().Free(ptr)
}

func (e *Emitter) IsTracked(ptr unsafe.Pointer) bool {
    return e.CMemTrackerAllocator().IsTracked(ptr)
}

func (e *Emitter) FreeAll() {
    e.CMemTrackerAllocator().FreeAll()
}

func (e *Emitter) Destroy() {
    // get the current configuration
    cfg := (*EmitterCfg)(C.fy_emitter_get_cfg(e.C()))
    gopointer.Unref(cfg.userdata)

    C.fy_emitter_destroy(e.C())
}

func EmitToString(a CMemTrackerAllocator, opts...interface{}) (*Emitter, error) {

    // get the options if any
    o, err := GetOptions(opts)
    if err != nil {
        return nil, err
    }

    // we will not need the emitter to associate with the
    // golang object, so no need to set userdata

    e := (*Emitter)(C.fy_emit_to_string(emitterCfgFlagsFromOptions(o)))
    if e == nil {
        return nil, errors.New("Failed to create emitter\n")
    }

    return e, nil
}

func (e *Emitter) CollectStringAndDestroy() string {
    var size C.size_t

    /* get the string result */
    cstr := C.fy_emit_to_string_collect(e.C(), &size)
    /* convert to go string */
    str := C.GoString(cstr)
    /* free the C string result */
    C.free(unsafe.Pointer(cstr))

    /* and destroy the emitter */
    C.fy_emitter_destroy(e.C())

    return str
}

func (e *Emitter) CollectByteDataAndDestroy() []byte {
    return []byte(e.CollectStringAndDestroy())
}

func (e *Emitter) EmitEvent(etype EventType, args...interface{}) error {

    var ev *C.struct_fy_event = nil

    var implicit C.int = 0
    var vers *C.struct_fy_version = (*C.struct_fy_version)(C.NULL)
    var tags **C.struct_fy_tag = (**C.struct_fy_tag)(C.NULL)
    var ns C.enum_fy_node_style = C.FYNS_ANY
    var ss C.enum_fy_scalar_style = C.FYSS_ANY
    var value *C.char = (*C.char)(C.NULL)
    var size C.size_t = 0
    var anchor *C.char = (*C.char)(C.NULL)
    var tag *C.char = (*C.char)(C.NULL)
    var rc C.int = 0

    switch etype {
    case StreamStart, StreamEnd, MappingEnd, SequenceEnd:
        ev = C.fy_emit_event_create_simple(e.C(), etype.C())

    case DocumentStart:
        if len(args) < 3 {
            goto err_args
        }
        if v, isB := args[0].(bool); isB {
            if v {
                implicit = 1
            } else {
                implicit = 0
            }
        } else {
            goto err_inval_args
        }
        ev = C.fy_emit_event_create_document_start(e.C(), implicit, vers, tags)

    case DocumentEnd:
        if len(args) < 1 {
            goto err_args
        }

        if v, isB := args[0].(bool); isB {
            if v {
                implicit = 1
            } else {
                implicit = 0
            }
        } else {
            goto err_inval_args
        }
        ev = C.fy_emit_event_create_document_end(e.C(), implicit)

    case MappingStart, SequenceStart:
        if len(args) < 3 {
            goto err_args
        }
        switch v := args[0].(type) {
        case NodeStyle:
            ns = v.C()
        case int:
            ns = C.enum_fy_node_style(v)
        default:
            goto err_inval_args
        }
        if v, isS := args[1].(string); isS {
            if v != "" {
                anchor = C.CString(v)
                defer C.free(unsafe.Pointer(anchor))
            }
        } else if args[1] != nil {
            goto err_inval_args
        }
        if v, isS := args[2].(string); isS {
            if v != "" {
                tag = C.CString(v)
                defer C.free(unsafe.Pointer(tag))
            }
        } else if args[2] != nil {
            goto err_inval_args
        }
        ev = C.fy_emit_event_create_collection_start(e.C(), etype.C(), ns, anchor, tag)

    case Scalar:
        if len(args) < 4 {
            goto err_args
        }
        switch v := args[0].(type) {
        case ScalarStyle:
            ss = v.C()
        case int:
            ss = C.enum_fy_scalar_style(v)
        default:
            goto err_inval_args
        }
        if v, isS := args[1].(string); isS {
            value = C.CString(v)
            size = C.FY_NT
            defer C.free(unsafe.Pointer(value))
        } else if args[1] == nil {
            /* nothing; this is a nil value */
        }
        if v, isS := args[2].(string); isS {
            if v != "" {
                anchor = C.CString(v)
                defer C.free(unsafe.Pointer(anchor))
            }
        } else if args[2] != nil {
            goto err_inval_args
        }
        if v, isS := args[3].(string); isS {
            if v != "" {
                tag = C.CString(v)
                defer C.free(unsafe.Pointer(tag))
            }
        } else if args[3] != nil {
            goto err_inval_args
        }

        ev = C.fy_emit_event_create_scalar(e.C(), ss, value, size, anchor, tag)

    case Alias:
        if len(args) < 1 {
            goto err_args
        }
        if v, isS := args[0].(string); isS {
            value = C.CString(v)
            defer C.free(unsafe.Pointer(value))
        } else if args[0] != nil {
            goto err_args
        }

        ev = C.fy_emit_event_create_alias(e.C(), value)
    }

    if ev == nil {
        return errors.New(fmt.Sprintf("EmitEvent %s: unable to create event", etype))
    }

    rc = C.fy_emit_event(e.C(), ev)
    if rc != 0 {
        return errors.New(fmt.Sprintf("EmitEvent %s: unable to emit event", etype))
    }

    return nil

err_args:
    return errors.New(fmt.Sprintf("EmitEvent %s: not enough arguments", etype))

err_inval_args:
    return errors.New(fmt.Sprintf("EmitEvent %s: invalid arguments", etype))
}

func init() {
    // register the default schemas, in sequence
    err := RegisterYAMLSchemas()
    if err != nil {
        panic(fmt.Sprintf("Failed to RegisterYAMLSchemas(): %s", err.Error()))
    }
}
