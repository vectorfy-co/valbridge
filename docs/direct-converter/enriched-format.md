# Enriched `x-valbridge` Format

The canonical metadata container is a single `x-valbridge` object on any schema node.

## Shape

```json
{
  "type": "string",
  "x-valbridge": {
    "version": "1.0",
    "coercionMode": "strict",
    "transforms": ["trim"],
    "registryMeta": {
      "source": "pydantic"
    }
  }
}
```

## Rules

- Do not scatter metadata across `x-valbridge-*` keys.
- Root and nested nodes use the same container shape.
- Unknown keys must be preserved for forward compatibility and diagnostics.
- Standard JSON Schema behavior remains unchanged when `x-valbridge` is absent.

## Merge Rule For `$ref`

When both the referenced definition and the `$ref` site include enrichment:

1. start from definition-site metadata
2. overlay ref-site metadata
3. preserve unknown keys unless the same key is overridden locally

This keeps referenced canonical metadata while allowing local specialization.
