// vim: tabstop=4 shiftwidth=4 expandtab
package fyaml

import (
    "fmt"
    "errors"
    "strings"
    "strconv"
)

type Options struct {
    Quiet bool                  // FYPCF_QUIET
    Version string              // FYPCF_VERSION auto, 1.1, 1.2, 1.3
    SloppyFlowIndentation bool  // FYPCF_SLOW_FLOW_INDENTATION
    Resolve bool                // FYPCF_RESOLVE_DOCUMENT
    JSON string                 // FYPCF_JSON auto, none, force
    SearchPath string           // parser search path

    MemCopy bool                // always copy in memory
    Lazy, Verbose, Debug bool   // parser options
    Strict, Custom bool         // unmarshal options
    Schema string               // auto, failsafe, yaml, json, 1.1, 1.2, 1.3

    Indent int                  // emitter indent - 1 >= i <= 9 set, 0 default
    Width int                   // 0 = default, 80 >= w < 255 set, < 0 inf
    SortKeys bool               // FYECF_SORT_KEYS
    StripLabels bool            // FYECF_STRIP_LABELS
    StripTags bool              // FYECF_STRIP_TAGS
    StripDocIndicators bool     // FYECF_STRIP_DOC
    NoEndingNewline bool        // FYECF_NO_ENDING_NEWLINE
    OutputMode string           // original, block, flow, flow-oneline, json, json-oneline, dejson, pretty
    VersionDirectives string    // auto, off, on
    TagDirectives string        // auto, off, on
}

var OptionsDefault = Options {
    Quiet: true,                // by default we are quiet
    Version: "auto",            // by default it's auto
    SloppyFlowIndentation: false, // by default it's false
    Resolve: true,              // by default resolution is enabled
    JSON: "auto",               // by default it's auto

    MemCopy: false,             // use the file directly if possible
    Lazy: false,                // by default we are not lazy
    Verbose: false,             // by default we are not verbose
    Debug: false,               // by default debug is off
    Strict: false,              // by default we are not strict
    Custom: true,               // by default we have custom unmarshalers
    SearchPath: "",             // by default just the current dir
    Schema: "auto",             // by default autodetect

    Indent: 0,                  // use the library default,
    Width: 0,                   // use the library default,
    SortKeys: false,            // by default we don't sort keys
    StripLabels: false,         // by default we don't strip labels
    StripTags: false,           // by default we don't strip tags
    StripDocIndicators: false,  // by default we don't strip document indicators
    NoEndingNewline: false,     // by default we will emit an ending newline
    OutputMode: "",             // use the library default
    VersionDirectives: "auto",  // use the library default
    TagDirectives: "auto",      // use the library default
}

func GetOptions(opts []interface{}) (*Options, error) {

    // start with the defaults
    o := OptionsDefault

    // start with the set of the options if passed
    for _, opt := range opts {
        if v, isO := opt.(Options); isO {
            o = v
        } else if vp, isOp := opt.(*Options); isOp {
            o = *vp
        }
    }

    // make a string only option slice
    sopts := make([]string, 0)

    for _, opt := range opts {
        if str, isStr := opt.(string); isStr {
            if str != "" {
                sopts = append(sopts, str)
            }
        } else if sstr, isSStr := opt.([]string); isSStr {
            if len(sstr) > 0 {
                sopts = append(sopts, sstr...)
            }
        } else if _, isO := opt.(Options); isO {
            /* nothing */
        } else if _, isOp := opt.(*Options); isOp {
            /* nothing */
        } else {
            return nil, errors.New(fmt.Sprintf("Bad type of option argument %T", opt))
        }
    }

    for _, opt := range sopts {

        var key, value string

        neg := false
        set := true
        kv := strings.Split(opt, "=")
        key = kv[0]

        switch len(kv) {
        case 1:
            // reverse set if prefix is set to that
            if strings.HasPrefix(key, "no") {
                key = strings.TrimPrefix(key, "no")
                set = false
                neg = true
            }

        case 2:
            value = kv[1]
            set = value == "true" || value == "1"

        default:
            value = strings.Join(kv[1:],"=")
        }

        // simple booleans
        if strings.EqualFold(key, "quiet") {
            o.Quiet = set
        } else if strings.EqualFold(key, "slow-flow-indentation") {
            o.SloppyFlowIndentation = set
        } else if strings.EqualFold(key, "resolve") {
            o.Resolve = set
        } else if strings.EqualFold(key, "memcopy") {
            o.MemCopy = set
        } else if strings.EqualFold(key, "lazy") {
            o.Lazy = set
        } else if strings.EqualFold(key, "verbose") {
            o.Verbose = set
        } else if strings.EqualFold(key, "debug") {
            o.Debug = set
        } else if strings.EqualFold(key, "strict") {
            o.Strict = set
        } else if strings.EqualFold(key, "custom") {
            o.Custom = set

        } else if !neg && strings.EqualFold(key, "version") {

            switch value {
            case "auto", "1.1", "1.2", "1.3":
                o.Version = value
            default:
                return nil, errors.New(fmt.Sprintf("Bad version %s (must be one of auto, 1.1, 1.2, 1.3)", value))
            }

        } else if !neg && strings.EqualFold(key, "json") {

            switch value {
            case "auto", "none", "force":
                o.JSON = value
            default:
                return nil, errors.New(fmt.Sprintf("Bad JSON %s (must be one of auto, none, force)", value))
            }

        } else if !neg && strings.EqualFold(key, "searchpath") {

            o.SearchPath = value

        } else if !neg && strings.EqualFold(key, "schema") {

            switch value {
            case "auto", "failsafe", "core", "json", "1.1", "1.2", "1.3":
                o.Schema = value
            default:
                return nil, errors.New(fmt.Sprintf("Bad schema %s (must be one of auto, failsafe, core, json, 1.1, 1.2, 1.3)", value))
            }

        } else if !neg && strings.EqualFold(key, "indent") {
            i, err := strconv.ParseInt(value, 10, 64)
            if err != nil || i < 2 || i > 9 {
                return nil, errors.New(fmt.Sprintf("Bad indent %s format", value))
            }
            o.Indent = int(i)

        } else if !neg && strings.EqualFold(key, "width") {
            if value == "infinite" || value == "inf" {
                o.Width = 255;  // infinite marker
            } else {
                i, err := strconv.ParseInt(value, 10, 64)
                if err != nil || i < 20 || i > 254 {
                    return nil, errors.New(fmt.Sprintf("Bad width %s format", value))
                }
                o.Width = int(i)
            }
        } else if strings.EqualFold(key, "sort-keys") {
            o.SortKeys = set
        } else if strings.EqualFold(key, "strip-labels") {
            o.StripLabels = set
        } else if strings.EqualFold(key, "strip-tags") {
            o.StripTags = set
        } else if strings.EqualFold(key, "strip-doc-indicators") {
            o.StripDocIndicators = set
        } else if strings.EqualFold(key, "no-ending-newline") {
            o.NoEndingNewline = set
        } else if !neg && strings.EqualFold(key, "output-mode") {
            switch value {
            case "original", "block", "flow", "flow-oneline", "json", "json-oneline", "dejson", "pretty":
                o.OutputMode = value
            default:
                return nil, errors.New(fmt.Sprintf("Bad output-mode %s (must be one of original, block, flow, flow-oneline, json, json-oneline, dejson, pretty)", value))
            }
        } else if !neg && strings.EqualFold(key, "version-directives") {
            switch value {
            case "auto", "off", "on":
                o.VersionDirectives = value
            default:
                return nil, errors.New(fmt.Sprintf("Bad version-directives %s (must be one of auto, off, on)", value))
            }
        } else if !neg && strings.EqualFold(key, "tag-directives") {
            switch value {
            case "auto", "off", "on":
                o.TagDirectives = value
            default:
                return nil, errors.New(fmt.Sprintf("Bad tag-directives %s (must be one of auto, off, on)", value))
            }
        } else {
            return nil, errors.New(fmt.Sprintf("Unknown Option %s", opt))
        }
    }
    return &o, nil
}
