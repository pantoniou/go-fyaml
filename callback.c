#include <stdlib.h>
#include <libfyaml.h>

extern enum fy_composer_return
FY_ProcessEvent(struct fy_parser *fyp, struct fy_event *fye, struct fy_path *path, void *userdata);

enum fy_composer_return
compose_process_event(struct fy_parser *fyp, struct fy_event *fye, struct fy_path *path, void *userdata)
{
	return FY_ProcessEvent(fyp, fye, path, userdata);
}

struct fy_event *
fy_emit_event_create_simple(struct fy_emitter *emit, enum fy_event_type type)
{
    return fy_emit_event_create(emit, type);
}

struct fy_event *
fy_emit_event_create_document_start(struct fy_emitter *emit, int implicit, const struct fy_version *vers, const struct fy_tag * const *tags)
{
    return fy_emit_event_create(emit, FYET_DOCUMENT_START, implicit, vers, tags);
}

struct fy_event *
fy_emit_event_create_document_end(struct fy_emitter *emit, int implicit)
{
    return fy_emit_event_create(emit, FYET_DOCUMENT_END, implicit);
}

struct fy_event *
fy_emit_event_create_collection_start(struct fy_emitter *emit, enum fy_event_type type, enum fy_node_style ns, const char *anchor, const char *tag)
{
    return fy_emit_event_create(emit, type, ns, anchor, tag);
}

struct fy_event *
fy_emit_event_create_scalar(struct fy_emitter *emit, enum fy_scalar_style ss, const char *value, size_t size, const char *anchor, const char *tag)
{
    return fy_emit_event_create(emit, FYET_SCALAR, ss, value, size, anchor, tag);
}

struct fy_event *
fy_emit_event_create_alias(struct fy_emitter *emit, const char *value)
{
    return fy_emit_event_create(emit, FYET_ALIAS, value);
}
