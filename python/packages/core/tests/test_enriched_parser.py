import json
from pathlib import Path

from valbridge_core import parse
from valbridge_core import parse_enriched


def test_parse_reads_x_valbridge_string_enrichment():
    result = parse(
        {
            "type": "string",
            "x-valbridge": {
                "version": "1.0",
                "coercionMode": "strict",
                "transforms": ["trim", {"kind": "toLowerCase"}],
                "formatDetail": {"kind": "uuid", "version": "v4"},
                "registryMeta": {"source": "zod"},
                "codeStubs": [{"kind": "validator", "name": "slugify"}],
                "defaultBehavior": {"kind": "prefault", "value": "guest"},
            },
        }
    )

    assert result.kind == "string"
    assert result.coercion_mode == "strict"
    assert tuple(transform.kind for transform in result.transforms) == (
        "trim",
        "toLowerCase",
    )
    assert result.format_detail is not None
    assert result.format_detail.kind == "uuid"
    assert result.format_detail.data["version"] == "v4"
    assert result.annotations is not None
    assert result.annotations.registry_meta == {"source": "zod"}
    assert result.annotations.code_stubs is not None
    assert result.annotations.code_stubs[0].name == "slugify"
    assert result.annotations.default_behavior is not None
    assert result.annotations.default_behavior.kind == "prefault"


def test_parse_stores_property_alias_info_on_object_properties():
    result = parse(
        {
            "type": "object",
            "properties": {
                "displayName": {
                    "type": "string",
                    "x-valbridge": {
                        "version": "1.0",
                        "aliasInfo": {
                            "validationAlias": "display_name",
                            "serializationAlias": "displayName",
                        },
                    },
                }
            },
        }
    )

    assert result.kind == "object"
    prop = dict(result.properties)["displayName"]
    assert prop.alias_info is not None
    assert prop.alias_info.validation_alias == "display_name"
    assert prop.alias_info.serialization_alias == "displayName"


def test_parse_enriched_merges_ref_site_enrichment_and_reports_unknown_keys():
    fixture_path = (
        Path(__file__).parent / "fixtures" / "enriched" / "string-ref-merge.json"
    )
    schema = json.loads(fixture_path.read_text())

    result = parse_enriched(schema)

    assert result.node.kind == "string"
    assert result.node.coercion_mode == "strict"
    assert tuple(transform.kind for transform in result.node.transforms) == ("trim",)
    assert [diagnostic.code for diagnostic in result.diagnostics] == [
        "valbridge.unknown_extension_key"
    ]
    assert result.diagnostics[0].path == "x-valbridge.unexpectedKey"
