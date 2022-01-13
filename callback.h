#ifndef CALLBACK_H
#define CALLBACK_H

#include <stdlib.h>
#include <libfyaml.h>

extern enum fy_composer_return
compose_process_event(struct fy_parser *fyp, struct fy_event *fye, struct fy_path *path, void *userdata);

extern struct fy_event *
fy_emit_event_create_simple(struct fy_emitter *emit, enum fy_event_type type);

extern struct fy_event *
fy_emit_event_create_document_start(struct fy_emitter *emit, int implicit, const struct fy_version *vers, const struct fy_tag * const *tags);

extern struct fy_event *
fy_emit_event_create_document_end(struct fy_emitter *emit, int implicit);

extern struct fy_event *
fy_emit_event_create_collection_start(struct fy_emitter *emit, enum fy_event_type type, enum fy_node_style ns, const char *anchor, const char *tag);

extern struct fy_event *
fy_emit_event_create_scalar(struct fy_emitter *emit, enum fy_scalar_style ss, const char *value, size_t size, const char *anchor, const char *tag);

extern struct fy_event *
fy_emit_event_create_alias(struct fy_emitter *emit, const char *value);

#endif
