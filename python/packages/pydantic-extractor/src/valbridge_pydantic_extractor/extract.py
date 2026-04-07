from __future__ import annotations

from copy import deepcopy
from dataclasses import dataclass
from typing import Annotated, Any, get_args, get_origin

from pydantic import BaseModel

from valbridge_core import Diagnostic


@dataclass(frozen=True)
class ExtractedSchema:
    schema: dict[str, Any]
    diagnostics: tuple[Diagnostic, ...]


def extract_model(model: type[BaseModel]) -> ExtractedSchema:
    schema = deepcopy(model.model_json_schema())
    diagnostics: list[Diagnostic] = []

    defs = schema.get("$defs")
    if not isinstance(defs, dict):
        defs = {}
        schema["$defs"] = defs

    _augment_model_schema(model, schema)

    for nested_model in _collect_nested_models(model):
        def_schema = defs.get(nested_model.__name__)
        if isinstance(def_schema, dict):
            _augment_model_schema(nested_model, def_schema)

    return ExtractedSchema(schema=schema, diagnostics=tuple(diagnostics))


def _augment_model_schema(model: type[BaseModel], schema: dict[str, Any]) -> None:
    x_schema = _ensure_valbridge(schema)
    extra_mode = _map_extra_mode(getattr(model, "model_config", {}).get("extra"))
    if extra_mode is not None:
        x_schema["extraMode"] = extra_mode

    properties = schema.get("properties")
    if not isinstance(properties, dict):
        return

    required_names = list(schema.get("required", []))
    renamed_properties: dict[str, Any] = {}

    for field_name, field in model.model_fields.items():
        property_key = _resolve_property_key(properties, field_name, field)
        property_schema = deepcopy(properties.get(property_key, {}))

        _apply_field_enrichment(model, field_name, field, property_schema)
        renamed_properties[field_name] = property_schema
        required_names = [
            field_name if required_name == property_key else required_name
            for required_name in required_names
        ]

    for property_key, property_schema in properties.items():
        if property_key not in renamed_properties:
            renamed_properties[property_key] = property_schema

    schema["properties"] = renamed_properties
    if required_names:
        schema["required"] = required_names


def _apply_field_enrichment(
    model: type[BaseModel],
    field_name: str,
    field: Any,
    schema: dict[str, Any],
) -> None:
    x_schema = _ensure_valbridge(schema)

    alias_info: dict[str, Any] = {}
    alias = getattr(field, "alias", None)
    validation_alias = getattr(field, "validation_alias", None)
    serialization_alias = getattr(field, "serialization_alias", None)

    if isinstance(validation_alias, str) and validation_alias != field_name:
        alias_info["validationAlias"] = validation_alias
    elif isinstance(alias, str) and alias != field_name:
        alias_info["validationAlias"] = alias

    if isinstance(serialization_alias, str) and serialization_alias != field_name:
        alias_info["serializationAlias"] = serialization_alias
    elif isinstance(alias, str) and alias != field_name:
        alias_info["serializationAlias"] = alias

    if alias_info:
        x_schema["aliasInfo"] = alias_info

    transforms = _collect_transforms(field)
    if transforms:
        x_schema["transforms"] = transforms

    format_detail = _collect_format_detail(field)
    if format_detail is not None:
        x_schema["formatDetail"] = format_detail

    registry_meta: dict[str, Any] = {}
    if getattr(field, "validate_default", None) is True:
        registry_meta["validateDefault"] = True
    if isinstance(getattr(field, "json_schema_extra", None), dict):
        registry_meta["jsonSchemaExtra"] = deepcopy(field.json_schema_extra)
    if registry_meta:
        x_schema["registryMeta"] = registry_meta

    default_factory = getattr(field, "default_factory", None)
    if callable(default_factory):
        x_schema["defaultBehavior"] = {
            "kind": "factory",
            "factory": f"{default_factory.__module__}.{default_factory.__qualname__}",
        }
    elif hasattr(field, "is_required") and not field.is_required():
        x_schema["defaultBehavior"] = {
            "kind": "default",
            "value": deepcopy(getattr(field, "default", None)),
        }

    union_resolution = _collect_union_resolution(field)
    if union_resolution is not None:
        x_schema["resolution"] = union_resolution

    discriminator = getattr(field, "discriminator", None)
    if isinstance(discriminator, str):
        x_schema["discriminator"] = discriminator

    code_stubs = _collect_code_stubs(model, field_name)
    if code_stubs:
        x_schema["codeStubs"] = code_stubs


def _ensure_valbridge(schema: dict[str, Any]) -> dict[str, Any]:
    valbridge = schema.get("x-valbridge")
    if isinstance(valbridge, dict):
        return valbridge

    valbridge = {}
    schema["x-valbridge"] = valbridge
    return valbridge


def _map_extra_mode(value: Any) -> str | None:
    if value in {"allow", "ignore", "forbid"}:
        return value
    return None


def _resolve_property_key(
    properties: dict[str, Any], field_name: str, field: Any
) -> str:
    candidates = [
        getattr(field, "validation_alias", None),
        getattr(field, "alias", None),
        getattr(field, "serialization_alias", None),
        field_name,
    ]

    for candidate in candidates:
        if isinstance(candidate, str) and candidate in properties:
            return candidate

    return field_name


def _collect_transforms(field: Any) -> list[dict[str, Any]]:
    transforms: list[dict[str, Any]] = []
    for metadata in getattr(field, "metadata", ()):
        if metadata.__class__.__name__ != "StringConstraints":
            continue
        if getattr(metadata, "strip_whitespace", False):
            transforms.append({"kind": "trim"})
        if getattr(metadata, "to_lower", False):
            transforms.append({"kind": "toLowerCase"})
        if getattr(metadata, "to_upper", False):
            transforms.append({"kind": "toUpperCase"})
    return transforms


def _collect_format_detail(field: Any) -> dict[str, Any] | None:
    for metadata in getattr(field, "metadata", ()):
        uuid_version = getattr(metadata, "uuid_version", None)
        if isinstance(uuid_version, int):
            return {"kind": "uuid", "version": f"v{uuid_version}"}
    return None


def _collect_union_resolution(field: Any) -> str | None:
    for metadata in getattr(field, "metadata", ()):
        union_mode = getattr(metadata, "union_mode", None)
        if union_mode == "left_to_right":
            return "leftToRight"
        if union_mode == "smart":
            return "smart"
    return None


def _collect_code_stubs(model: type[BaseModel], field_name: str) -> list[dict[str, Any]]:
    decorators = getattr(model, "__pydantic_decorators__", None)
    if decorators is None:
        return []

    stubs: list[dict[str, Any]] = []

    for decorator in getattr(decorators, "field_validators", {}).values():
        if field_name not in decorator.info.fields:
            continue
        stubs.append(
            {
                "kind": "validator",
                "name": decorator.cls_var_name,
                "payload": {"mode": decorator.info.mode},
            }
        )

    for decorator in getattr(decorators, "field_serializers", {}).values():
        if field_name not in decorator.info.fields:
            continue
        stubs.append(
            {
                "kind": "serializer",
                "name": decorator.cls_var_name,
                "payload": {"mode": decorator.info.mode},
            }
        )

    return stubs


def _collect_nested_models(model: type[BaseModel]) -> set[type[BaseModel]]:
    nested: set[type[BaseModel]] = set()
    for field in model.model_fields.values():
        nested.update(_walk_model_types(field.annotation))
    return nested


def _walk_model_types(annotation: Any) -> set[type[BaseModel]]:
    nested: set[type[BaseModel]] = set()

    if isinstance(annotation, type) and issubclass(annotation, BaseModel):
        nested.add(annotation)
        return nested

    origin = get_origin(annotation)
    if origin is Annotated:
        args = get_args(annotation)
        if args:
            nested.update(_walk_model_types(args[0]))
        return nested

    for arg in get_args(annotation):
        nested.update(_walk_model_types(arg))

    return nested
