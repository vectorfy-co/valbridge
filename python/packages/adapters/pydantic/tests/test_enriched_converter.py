from valbridge_pydantic.converter import convert


def test_convert_surfaces_enriched_parse_diagnostics():
    result = convert(
        {
            "namespace": "test",
            "id": "DisplayName",
            "varName": "display_name",
            "schema": {
                "type": "string",
                "x-valbridge": {
                    "version": "1.0",
                    "coercionMode": "strict",
                    "unexpectedKey": True,
                },
            },
        }
    )

    assert result["diagnostics"] == [
        {
            "severity": "warning",
            "code": "valbridge.unknown_extension_key",
            "message": "Unknown x-valbridge key 'unexpectedKey' was preserved but has no runtime effect.",
            "path": "x-valbridge.unexpectedKey",
            "source": "valbridge",
            "suggestion": "Remove the key or add runtime support before depending on it.",
        }
    ]
