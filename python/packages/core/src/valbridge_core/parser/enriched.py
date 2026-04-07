"""Helpers for canonical x-valbridge enrichment."""

from dataclasses import dataclass
from typing import Any

from valbridge_core.diagnostics import Diagnostic
from valbridge_core.ir import AliasInfo
from valbridge_core.parser.context import create_context

_KNOWN_VALBRIDGE_KEYS = frozenset(
    {
        "version",
        "coercionMode",
        "transforms",
        "formatDetail",
        "registryMeta",
        "codeStubs",
        "defaultBehavior",
        "aliasInfo",
        "extraMode",
        "discriminator",
        "resolution",
    }
)


def _extract_valbridge(schema: dict[str, Any] | bool) -> dict[str, Any] | None:
    if not isinstance(schema, dict):
        return None
    extension = schema.get("x-valbridge")
    return extension if isinstance(extension, dict) else None


def extract_alias_info(schema: dict[str, Any] | bool) -> AliasInfo | None:
    extension = _extract_valbridge(schema)
    if extension is None:
        return None

    alias_info = extension.get("aliasInfo")
    if not isinstance(alias_info, dict):
        return None

    validation_alias = alias_info.get("validationAlias")
    serialization_alias = alias_info.get("serializationAlias")
    alias_path = alias_info.get("aliasPath")

    normalized = AliasInfo(
        validation_alias=validation_alias
        if isinstance(validation_alias, str)
        else None,
        serialization_alias=serialization_alias
        if isinstance(serialization_alias, str)
        else None,
        alias_path=tuple(part for part in alias_path if isinstance(part, str))
        if isinstance(alias_path, list)
        else None,
    )

    if (
        normalized.validation_alias is None
        and normalized.serialization_alias is None
        and normalized.alias_path is None
    ):
        return None
    return normalized


@dataclass(frozen=True)
class ParsedEnrichedSchema:
    node: Any
    diagnostics: tuple[Diagnostic, ...]


def collect_unknown_extension_diagnostics(schema: dict[str, Any] | bool) -> tuple[Diagnostic, ...]:
    extension = _extract_valbridge(schema)
    if extension is None:
        return ()

    diagnostics: list[Diagnostic] = []
    for key in extension:
        if key not in _KNOWN_VALBRIDGE_KEYS:
            diagnostics.append(
                Diagnostic(
                    severity="warning",
                    code="valbridge.unknown_extension_key",
                    message=f"Unknown x-valbridge key '{key}' was preserved but has no runtime effect.",
                    path=f"x-valbridge.{key}",
                    source="valbridge",
                    suggestion="Remove the key or add runtime support before depending on it.",
                )
            )
    return tuple(diagnostics)


def parse_enriched(schema: dict[str, Any] | bool) -> ParsedEnrichedSchema:
    from valbridge_core.parser.core import _parse_with_ctx

    ctx = create_context(schema)
    node = _parse_with_ctx(schema, ctx)
    return ParsedEnrichedSchema(node=node, diagnostics=tuple(ctx.diagnostics))
