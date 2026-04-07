"""Core JSON Schema to IR parsing logic.

This module contains:
- parse(): Main entry point that converts JSON Schema to IR
- _resolve_json_pointer(): JSON pointer resolution (#/$defs/Name)
- _resolve_ref(): $ref resolution with cycle detection
- _parse_with_ctx(): Main recursive parsing function
- _infer_type(): Type inference from keywords
- _detect_type_guards(): Type guard detection for typeless schemas
- _detect_discriminator(): Discriminated union detection for oneOf

The parse function expects schemas to be pre-bundled by the CLI.
All external $refs should be resolved before reaching this parser.
Internal JSON pointer refs (#/$defs/Name) are resolved during parsing.
"""

from dataclasses import replace
from typing import Any, Dict, List, Literal, Optional, Set, Union, cast

from valbridge_core.ir import (
    _UNSET,
    AnyNode,
    ArrayNode,
    BooleanNode,
    CodeStub,
    DefaultBehavior,
    FormatDetail,
    IntersectionNode,
    NeverNode,
    NumberNode,
    NullNode,
    NullableNode,
    ObjectNode,
    RefNode,
    SchemaAnnotations,
    SchemaNode,
    StringNode,
    Transform,
    TupleNode,
    TypeGuard,
    TypeGuardedNode,
    UnionNode,
)
from valbridge_core.parser.primitives import parse_number, parse_string
from valbridge_core.parser.collections import (
    parse_array,
    parse_object,
    parse_tuple,
)
from valbridge_core.parser.composition import (
    parse_all_of,
    parse_any_of,
    parse_conditional,
    parse_not,
    parse_one_of,
)
from valbridge_core.parser.values import parse_enum, parse_literal
from valbridge_core.parser.context import ParseContext, create_context
from valbridge_core.parser.enriched import collect_unknown_extension_diagnostics

_ANNOTATION_KEYS = frozenset(
    {
        "title",
        "description",
        "examples",
        "default",
        "deprecated",
        "readOnly",
        "writeOnly",
        "x-valbridge",
    }
)

_COERCION_MODES = {"strict", "coerce", "passthrough"}
_EXTRA_MODES = {"allow", "ignore", "forbid"}
_UNION_RESOLUTIONS = {"smart", "leftToRight", "allErrors"}


def _freeze_annotation_value(value: Any) -> Any:
    if isinstance(value, list):
        return tuple(_freeze_annotation_value(item) for item in value)
    if isinstance(value, dict):
        return {key: _freeze_annotation_value(item) for key, item in value.items()}
    return value


def _extract_annotations(schema: Dict[str, Any]) -> SchemaAnnotations | None:
    examples = schema.get("examples")
    valbridge = schema.get("x-valbridge")
    registry_meta = None
    code_stubs = None
    default_behavior = None

    if isinstance(valbridge, dict):
        if isinstance(valbridge.get("registryMeta"), dict):
            registry_meta = _freeze_annotation_value(valbridge["registryMeta"])

        raw_stubs = valbridge.get("codeStubs")
        if isinstance(raw_stubs, list):
            normalized_stubs = []
            for stub in raw_stubs:
                if not isinstance(stub, dict):
                    continue
                kind = stub.get("kind")
                name = stub.get("name")
                if not isinstance(kind, str) or not isinstance(name, str):
                    continue
                payload = stub.get("payload")
                normalized_stubs.append(
                    CodeStub(
                        kind=kind,
                        name=name,
                        payload=_freeze_annotation_value(payload)
                        if isinstance(payload, dict)
                        else None,
                    )
                )
            if normalized_stubs:
                code_stubs = tuple(normalized_stubs)

        raw_default = valbridge.get("defaultBehavior")
        if isinstance(raw_default, dict):
            kind = raw_default.get("kind")
            if isinstance(kind, str) and kind in {"default", "prefault", "factory"}:
                default_behavior = DefaultBehavior(
                    kind=cast("Literal['default', 'prefault', 'factory']", kind),
                    value=_freeze_annotation_value(raw_default["value"])
                    if "value" in raw_default
                    else _UNSET,
                    factory=raw_default.get("factory")
                    if isinstance(raw_default.get("factory"), str)
                    else None,
                )

    annotations = SchemaAnnotations(
        title=schema.get("title"),
        description=schema.get("description"),
        examples=(
            tuple(_freeze_annotation_value(example) for example in examples)
            if examples is not None
            else None
        ),
        default=(
            _freeze_annotation_value(schema["default"])
            if "default" in schema
            else _UNSET
        ),
        deprecated=schema.get("deprecated"),
        read_only=schema.get("readOnly"),
        write_only=schema.get("writeOnly"),
        registry_meta=registry_meta,
        code_stubs=code_stubs,
        default_behavior=default_behavior,
    )
    if (
        annotations.title is None
        and annotations.description is None
        and annotations.examples is None
        and annotations.default is _UNSET
        and annotations.deprecated is None
        and annotations.read_only is None
        and annotations.write_only is None
        and annotations.registry_meta is None
        and annotations.code_stubs is None
        and annotations.default_behavior is None
    ):
        return None
    return annotations


def _merge_annotations(
    base: SchemaAnnotations | None, override: SchemaAnnotations | None
) -> SchemaAnnotations | None:
    if base is None and override is None:
        return None

    merged = SchemaAnnotations(
        title=override.title if override and override.title is not None else base.title if base else None,
        description=(
            override.description
            if override and override.description is not None
            else base.description if base else None
        ),
        examples=(
            override.examples
            if override and override.examples is not None
            else base.examples if base else None
        ),
        default=(
            override.default
            if override and override.default is not _UNSET
            else base.default if base else _UNSET
        ),
        deprecated=(
            override.deprecated
            if override and override.deprecated is not None
            else base.deprecated if base else None
        ),
        read_only=(
            override.read_only
            if override and override.read_only is not None
            else base.read_only if base else None
        ),
        write_only=(
            override.write_only
            if override and override.write_only is not None
            else base.write_only if base else None
        ),
        registry_meta=(
            override.registry_meta
            if override and override.registry_meta is not None
            else base.registry_meta if base else None
        ),
        code_stubs=(
            override.code_stubs
            if override and override.code_stubs is not None
            else base.code_stubs if base else None
        ),
        default_behavior=(
            override.default_behavior
            if override and override.default_behavior is not None
            else base.default_behavior if base else None
        ),
    )
    if (
        merged.title is None
        and merged.description is None
        and merged.examples is None
        and merged.default is _UNSET
        and merged.deprecated is None
        and merged.read_only is None
        and merged.write_only is None
        and merged.registry_meta is None
        and merged.code_stubs is None
        and merged.default_behavior is None
    ):
        return None
    return merged


def _normalize_transforms(value: Any) -> tuple[Transform, ...]:
    if not isinstance(value, list):
        return ()

    transforms: list[Transform] = []
    for item in value:
        if isinstance(item, str):
            transforms.append(Transform(kind=item))
            continue
        if isinstance(item, dict) and isinstance(item.get("kind"), str):
            args = item.get("args")
            transforms.append(
                Transform(
                    kind=item["kind"],
                    args=_freeze_annotation_value(args) if isinstance(args, dict) else None,
                )
            )
    return tuple(transforms)


def _normalize_format_detail(value: Any) -> FormatDetail | None:
    if not isinstance(value, dict) or not isinstance(value.get("kind"), str):
        return None

    data = {
        key: _freeze_annotation_value(item)
        for key, item in value.items()
        if key != "kind"
    }
    return FormatDetail(kind=value["kind"], data=data or None)


def _with_annotations(
    node: SchemaNode, schema: Union[Dict[str, Any], bool]
) -> SchemaNode:
    if not isinstance(schema, dict):
        return node
    base_node: SchemaNode
    if isinstance(node, RefNode) and node.resolved is not None:
        base_node = node.resolved
    else:
        base_node = node

    annotations = _merge_annotations(
        getattr(base_node, "annotations", None), _extract_annotations(schema)
    )
    if annotations is None:
        result = base_node
    else:
        result = replace(base_node, annotations=annotations)

    valbridge = schema.get("x-valbridge")
    if not isinstance(valbridge, dict):
        return result

    if isinstance(result, StringNode):
        coercion_mode = valbridge.get("coercionMode")
        return replace(
            result,
            coercion_mode=coercion_mode
            if coercion_mode in _COERCION_MODES
            else result.coercion_mode,
            transforms=_normalize_transforms(valbridge.get("transforms")) or result.transforms,
            format_detail=_normalize_format_detail(valbridge.get("formatDetail"))
            or result.format_detail,
        )
    if isinstance(result, NumberNode):
        coercion_mode = valbridge.get("coercionMode")
        return replace(
            result,
            coercion_mode=coercion_mode
            if coercion_mode in _COERCION_MODES
            else result.coercion_mode,
        )
    if isinstance(result, BooleanNode):
        coercion_mode = valbridge.get("coercionMode")
        return replace(
            result,
            coercion_mode=coercion_mode
            if coercion_mode in _COERCION_MODES
            else result.coercion_mode,
        )
    if isinstance(result, ObjectNode):
        extra_mode = valbridge.get("extraMode")
        discriminator = valbridge.get("discriminator")
        return replace(
            result,
            extra_mode=extra_mode if extra_mode in _EXTRA_MODES else result.extra_mode,
            discriminator=discriminator
            if isinstance(discriminator, str)
            else result.discriminator,
        )
    if isinstance(result, UnionNode):
        resolution = valbridge.get("resolution")
        discriminator = valbridge.get("discriminator")
        return replace(
            result,
            resolution=resolution
            if resolution in _UNION_RESOLUTIONS
            else result.resolution,
            discriminator=discriminator
            if isinstance(discriminator, str)
            else result.discriminator,
        )

    return result


def parse(schema: Union[Dict[str, Any], bool]) -> SchemaNode:
    """Parse a JSON Schema dict into an IR SchemaNode.

    This is the main entry point for converting JSON Schema to IR.

    Args:
        schema: A JSON Schema as a dict, or a boolean schema (true/false).
                Schemas should be pre-bundled by the CLI - all external $refs
                should already be resolved. Internal JSON pointer refs
                (#/$defs/Name) are supported.

    Returns:
        A SchemaNode representing the schema in IR form.

    Example:
        >>> from valbridge_core.parser import parse
        >>> node = parse({"type": "string", "minLength": 1})
        >>> type(node).__name__
        'StringNode'
        >>> node.constraints.min_length
        1
    """
    # Create context with root schema for internal $ref resolution
    ctx = create_context(schema)
    return _parse_with_ctx(schema, ctx)


def _resolve_json_pointer(ref: str, root: dict[str, Any]) -> Any:
    """Resolve a JSON pointer (fragment starting with #/) to a schema node.

    JSON Pointer is defined in RFC 6901. The pointer starts with # and uses
    / to separate path segments. Special characters are escaped:
    - ~0 represents ~
    - ~1 represents /
    - URI percent-encoding (e.g., %25 for %) is decoded first

    Args:
        ref: JSON pointer string like "#/$defs/Name" or "#/properties/foo%2Fbar"
        root: The root schema to resolve against

    Returns:
        The schema node at the pointer location

    Raises:
        ValueError: If the pointer path doesn't exist
    """
    from urllib.parse import unquote

    path_parts = ref[2:].split("/")  # Remove "#/" prefix
    current: Any = root

    for part in path_parts:
        # Handle URI percent-encoding first (e.g., %25 -> %)
        part = unquote(part)
        # Then handle JSON pointer escaping (~1 -> /, ~0 -> ~)
        part = part.replace("~1", "/").replace("~0", "~")

        if isinstance(current, dict) and part in current:
            current = current[part]
        elif isinstance(current, list):
            # Handle list index
            try:
                idx = int(part)
                if 0 <= idx < len(current):
                    current = current[idx]
                else:
                    raise ValueError(f"list index {idx} out of range")
            except ValueError:
                raise ValueError(f"invalid list index '{part}'")
        else:
            raise ValueError(f"path not found at '{part}'")

    return current


def _resolve_ref(ref: str, ctx: ParseContext) -> SchemaNode:
    """Resolve a $ref to a SchemaNode.

    Only handles internal JSON pointer refs (e.g., #/$defs/Name, #/properties/foo).
    External refs should be bundled by the CLI before reaching the adapter.

    Cycle Detection:
        Uses ctx.resolving set to track refs being resolved. When a cycle is
        detected (ref already in resolving set), returns RefNode(resolved=None).
        This allows adapters to handle recursive types appropriately.

    Args:
        ref: The $ref string value
        ctx: Parse context with root schema and cycle tracking

    Returns:
        RefNode with the ref path and resolved schema (or None for cycles)

    Raises:
        ValueError: If ref is external (not starting with #) or unresolvable
    """
    if ctx.root is None:
        raise ValueError(
            f"Encountered $ref '{ref}' - schemas must be bundled by the Go CLI before processing. "
            "Run the schema through valbridge generate to bundle all references."
        )

    # Only handle JSON pointer refs starting with #
    if not ref.startswith("#"):
        raise ValueError(
            f"Encountered external $ref '{ref}' - schemas must be bundled by the Go CLI before processing. "
            "Run the schema through valbridge generate to bundle all references."
        )

    # Check for cycles - return RefNode with resolved=None to break infinite recursion
    if ref in ctx.resolving:
        return RefNode(path=ref, resolved=None)

    # Root ref - recursive to root schema, always cycles
    if ref == "#":
        return RefNode(path=ref, resolved=None)

    # Resolve JSON pointer
    try:
        target_schema = _resolve_json_pointer(ref, ctx.root)
    except ValueError as e:
        raise ValueError(
            f"Failed to resolve $ref '{ref}': {e}. "
            "The schema may be malformed or the CLI bundler may have an issue."
        )

    # Mark as resolving to detect cycles
    ctx.resolving.add(ref)
    try:
        resolved = _parse_with_ctx(target_schema, ctx)
    finally:
        ctx.resolving.discard(ref)

    return RefNode(path=ref, resolved=resolved)


def _parse_with_ctx(
    schema: Union[Dict[str, Any], bool], ctx: ParseContext
) -> SchemaNode:
    """Parse a JSON Schema dict into an IR SchemaNode with context.

    This is the main recursive parsing function. It handles all JSON Schema
    keywords and dispatches to specialized parsers for different types.

    Parsing order matters - keywords are checked in this order:
    1. Boolean schemas (true/false)
    2. Empty schema
    3. $ref (terminates - no sibling keywords processed)
    4. const (literal value)
    5. enum (value set)
    6. not (negation)
    7. if/then/else (conditional)
    8. nullable (OpenAPI extension)
    9. Composition keywords (allOf, anyOf, oneOf)
    10. Type-based parsing
    """
    # Handle boolean schemas
    if schema is True:
        return AnyNode()
    if schema is False:
        return NeverNode()

    # Handle empty schema
    if not schema:
        return AnyNode()

    ctx.diagnostics.extend(collect_unknown_extension_diagnostics(schema))

    # Handle $ref - CLI should pre-bundle all schemas
    if "$ref" in schema:
        ref = schema["$ref"]
        return _with_annotations(_resolve_ref(ref, ctx), schema)

    # Handle const (literal value)
    if "const" in schema:
        return _with_annotations(parse_literal(schema), schema)

    # Handle enum
    if "enum" in schema:
        return _with_annotations(parse_enum(schema), schema)

    # Handle not keyword
    if "not" in schema:
        return _with_annotations(parse_not(schema["not"], ctx, _parse_with_ctx), schema)

    # Handle conditional (if/then/else)
    if "if" in schema:
        return _with_annotations(
            parse_conditional(
                schema["if"],
                schema.get("then"),
                schema.get("else"),
                ctx,
                _parse_with_ctx,
            ),
            schema,
        )

    # Handle nullable (OpenAPI 3.0 style)
    if schema.get("nullable") is True:
        # Remove nullable and parse the rest
        inner_schema = {
            k: v
            for k, v in schema.items()
            if k != "nullable" and k not in _ANNOTATION_KEYS
        }
        inner = _parse_with_ctx(inner_schema, ctx)
        return _with_annotations(NullableNode(inner=inner), schema)

    # Handle composition keywords
    # Check if there are sibling validation keywords that need to be combined
    composition_keys = {"allOf", "anyOf", "oneOf"}
    meta_keys = {
        "$schema",
        "$id",
        "$ref",
        "$defs",
        "definitions",
        "$comment",
        "title",
        "description",
        "examples",
        "default",
        "deprecated",
        "readOnly",
        "writeOnly",
        "$anchor",
    }
    # Keys that are meaningful only with composition (not standalone validation)
    composition_only_keys = composition_keys | meta_keys

    has_composition = bool(composition_keys & set(schema.keys()))
    has_sibling_validation = bool(set(schema.keys()) - composition_only_keys)
    composition_count = len(composition_keys & set(schema.keys()))

    # Multiple composition keywords (allOf + anyOf, etc.) or composition + sibling validation
    # All must be combined with intersection
    if composition_count > 1 or (has_composition and has_sibling_validation):
        composition_nodes: list[SchemaNode] = []
        if "allOf" in schema:
            # Flatten allOf into intersection members
            all_of_node = parse_all_of(schema["allOf"], ctx, _parse_with_ctx)
            composition_nodes.extend(all_of_node.schemas)
        if "anyOf" in schema:
            composition_nodes.append(
                parse_any_of(schema["anyOf"], ctx, _parse_with_ctx)
            )
        if "oneOf" in schema:
            composition_nodes.append(
                parse_one_of(schema["oneOf"], ctx, _parse_with_ctx)
            )

        # Add sibling validation schema if present
        if has_sibling_validation:
            sibling_schema = {
                k: v for k, v in schema.items() if k not in composition_keys
            }
            sibling_node = _parse_with_ctx(sibling_schema, ctx)
            all_schemas = tuple(composition_nodes) + (sibling_node,)
        else:
            all_schemas = tuple(composition_nodes)

        return _with_annotations(IntersectionNode(schemas=all_schemas), schema)

    # Single composition keyword without sibling validation
    if "allOf" in schema:
        return _with_annotations(parse_all_of(schema["allOf"], ctx, _parse_with_ctx), schema)

    if "anyOf" in schema:
        return _with_annotations(parse_any_of(schema["anyOf"], ctx, _parse_with_ctx), schema)

    if "oneOf" in schema:
        return _with_annotations(parse_one_of(schema["oneOf"], ctx, _parse_with_ctx), schema)

    # Get type - can be string or array
    schema_type = schema.get("type")

    # Handle array of types (union)
    if isinstance(schema_type, list):
        variant_list: list[SchemaNode] = []
        for t in schema_type:
            variant_schema = {**schema, "type": t}
            variant_list.append(_parse_with_ctx(variant_schema, ctx))
        return _with_annotations(UnionNode(variants=tuple(variant_list)), schema)

    # Handle prefixItems (tuple) only if type is explicitly "array"
    # If no type is specified, prefixItems should be wrapped in a type guard
    # so it only applies to arrays (not objects that look like arrays)
    if "prefixItems" in schema and schema_type == "array":
        return parse_tuple(schema, ctx)

    # Infer type from keywords if not specified
    if schema_type is None:
        # Check if this is a type-guarded schema (multiple type-specific keywords without type)
        guards = _detect_type_guards(schema, ctx)
        if len(guards) > 1:
            return _with_annotations(TypeGuardedNode(guards=tuple(guards)), schema)

        # Special case: single array guard with prefixItems needs TypeGuardedNode
        # to allow non-arrays through (JSON Schema semantics: prefixItems only applies to arrays)
        if len(guards) == 1 and guards[0].check == "array" and "prefixItems" in schema:
            return _with_annotations(TypeGuardedNode(guards=tuple(guards)), schema)

        schema_type = _infer_type(schema)

    # Parse based on type
    if schema_type == "string":
        return _with_annotations(parse_string(schema), schema)

    if schema_type == "number":
        return _with_annotations(parse_number(schema, integer=False), schema)

    if schema_type == "integer":
        return _with_annotations(parse_number(schema, integer=True), schema)

    if schema_type == "boolean":
        return _with_annotations(BooleanNode(), schema)

    if schema_type == "null":
        return _with_annotations(NullNode(), schema)

    if schema_type == "object":
        return _with_annotations(parse_object(schema, ctx), schema)

    if schema_type == "array":
        return _with_annotations(parse_array(schema, ctx), schema)

    # Unknown type - return AnyNode
    return _with_annotations(AnyNode(), schema)


def _infer_type(schema: Dict[str, Any]) -> Optional[str]:
    """Infer the type from other keywords in the schema.

    JSON Schema allows omitting "type" when other keywords unambiguously
    indicate the type. This function checks for type-specific keywords
    and returns the implied type.

    Returns:
        Inferred type string or None if type cannot be determined
    """
    # String keywords
    if any(k in schema for k in ("minLength", "maxLength", "pattern", "format")):
        return "string"

    # Number keywords
    if any(
        k in schema
        for k in (
            "minimum",
            "maximum",
            "exclusiveMinimum",
            "exclusiveMaximum",
            "multipleOf",
        )
    ):
        return "number"

    # Object keywords
    if any(
        k in schema
        for k in (
            "properties",
            "required",
            "additionalProperties",
            "patternProperties",
            "propertyNames",
            "minProperties",
            "maxProperties",
            "dependentRequired",
            "dependentSchemas",
            "unevaluatedProperties",
        )
    ):
        return "object"

    # Array keywords
    if any(
        k in schema
        for k in (
            "items",
            "minItems",
            "maxItems",
            "uniqueItems",
            "contains",
            "minContains",
            "maxContains",
            "prefixItems",
            "additionalItems",
            "unevaluatedItems",
        )
    ):
        return "array"

    return None


def _detect_discriminator(variants: List[Any]) -> Optional[str]:
    """Detect if oneOf variants form a discriminated union.

    A discriminated union has all variants as objects with a common property
    that has a literal (const) value unique to each variant. This is useful
    for generating more efficient union types in target languages.

    Args:
        variants: List of oneOf variant schemas

    Returns:
        Property name that discriminates the union, or None if not discriminated

    Example:
        >>> variants = [
        ...     {"type": "object", "properties": {"kind": {"const": "a"}}},
        ...     {"type": "object", "properties": {"kind": {"const": "b"}}}
        ... ]
        >>> _detect_discriminator(variants)
        'kind'
    """
    if not variants:
        return None

    # Find common properties with const values across all variants
    common_discriminators: Optional[Set[str]] = None

    for variant in variants:
        if not isinstance(variant, dict):
            return None
        if variant.get("type") != "object" and "properties" not in variant:
            # Could still be an object without explicit type
            if "properties" not in variant:
                return None

        properties = variant.get("properties", {})
        discriminator_props = set()

        for prop_name, prop_schema in properties.items():
            if isinstance(prop_schema, dict) and "const" in prop_schema:
                discriminator_props.add(prop_name)

        if common_discriminators is None:
            common_discriminators = discriminator_props
        else:
            common_discriminators &= discriminator_props

        if not common_discriminators:
            return None

    # Return the first common discriminator property
    if common_discriminators:
        return next(iter(common_discriminators))
    return None


def _detect_type_guards(schema: Dict[str, Any], ctx: ParseContext) -> List[TypeGuard]:
    """Detect type-specific constraints and create type guards.

    For schemas without an explicit type that have type-specific keywords,
    create guards that apply those constraints only to matching runtime types.

    This handles schemas like:
        {"minLength": 5, "minimum": 0}

    Which should validate:
        - Strings must have minLength >= 5
        - Numbers must be >= 0
        - Other types pass through (JSON Schema allows this)

    Args:
        schema: Schema dict to analyze
        ctx: Parse context for recursive parsing

    Returns:
        List of TypeGuard objects, one per detected type
    """
    guards = []

    # Check for string-specific keywords
    if any(k in schema for k in ("minLength", "maxLength", "pattern", "format")):
        string_schema = parse_string(schema)
        guards.append(TypeGuard(check="string", schema=string_schema))

    # Check for number-specific keywords
    if any(
        k in schema
        for k in (
            "minimum",
            "maximum",
            "exclusiveMinimum",
            "exclusiveMaximum",
            "multipleOf",
        )
    ):
        number_schema = parse_number(schema, integer=False)
        guards.append(TypeGuard(check="number", schema=number_schema))

    # Check for object-specific keywords
    if any(
        k in schema
        for k in (
            "properties",
            "required",
            "additionalProperties",
            "patternProperties",
            "propertyNames",
            "minProperties",
            "maxProperties",
            "dependentRequired",
            "dependentSchemas",
        )
    ):
        object_schema = parse_object(schema, ctx)
        guards.append(TypeGuard(check="object", schema=object_schema))

    # Check for array-specific keywords
    if any(
        k in schema
        for k in (
            "items",
            "minItems",
            "maxItems",
            "uniqueItems",
            "contains",
            "minContains",
            "maxContains",
            "prefixItems",
            "additionalItems",
            "unevaluatedItems",
        )
    ):
        # Use parse_tuple for prefixItems, parse_array for other array keywords
        array_schema: Union[ArrayNode, TupleNode]
        if "prefixItems" in schema:
            array_schema = parse_tuple(schema, ctx)
        else:
            array_schema = parse_array(schema, ctx)
        guards.append(TypeGuard(check="array", schema=array_schema))

    return guards
