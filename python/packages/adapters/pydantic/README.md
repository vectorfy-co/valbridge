<div align="center">

# ![valbridge-pydantic](https://img.shields.io/static/v1?label=&message=valbridge-pydantic&color=E92063&style=for-the-badge&logo=pydantic&logoColor=white)

Pydantic v2 adapter for valbridge -- converts JSON Schema into native Pydantic BaseModel classes with full type safety.

<a href="https://pypi.org/project/valbridge-pydantic/"><img src="https://img.shields.io/pypi/v/valbridge-pydantic?style=flat&logo=pypi&logoColor=white" alt="PyPI" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
pip install valbridge-pydantic pydantic
# or
uv add valbridge-pydantic pydantic
```

Verified against `pydantic 2.12.5`.

## Usage

This adapter is invoked by the valbridge CLI. Define schemas in a config file:

```jsonc
// user.valbridge.jsonc
{
  "$schema": "https://github.com/vectorfy-co/valbridge/schemas/python.jsonc",
  "schemas": [
    {
      "id": "User",
      "adapter": "vectorfyco/valbridge-pydantic",
      "sourceType": "file",
      "source": "./schemas/user.json"
    }
  ]
}
```

Then generate:

```bash
valbridge generate
```

Use the generated models with the [valbridge runtime client](https://pypi.org/project/valbridge/).

## Verification

Run from the adapter directory (`python/packages/adapters/pydantic/`):

```bash
# JSON Schema Test Suite compliance (requires Go CLI)
cd ../../cli && go build -o valbridge . && \
  ./valbridge compliance --lang python --adapter-path ../python/packages/adapters/pydantic

# Unit tests
uv run pytest

# Type checking
uv run pyright src/
```

## Fallback typing guardrails

The adapter uses `Annotated[Any, BeforeValidator(...)]` only for constructs where the validated domain is unbounded. Every other construct must produce a narrower type.

### Allowed `Any` fallbacks

| Construct | Reason |
|-----------|--------|
| `not` | Negation is "everything except X" -- no union can express this |
| `conditional` (single branch) | Unmatched values pass through unconstrained |
| `typeGuarded` (heterogeneous) | Unmatched types pass through the guard |
| Open tuple | Extra elements beyond prefix items are untyped |
| Recursive refs | Self-referencing `$ref` cycles |

### Constructs that must NOT use `Any`

- `oneOf` -- must produce `T1 | T2 | ...`
- `conditional` (if/then/else) -- must produce `ThenType | ElseType`
- Closed tuples -- must produce `tuple[T1, T2, ...]`
- `const` / `enum` -- must produce the narrowest applicable type
- `allOf` (object merges) -- must produce a BaseModel subclass

### Disallowed patterns

- Unknown IR node kind -- must raise `ConversionError`
- Unresolved external refs -- must raise `ConversionError`
- New renderer branches must not silently return `Any` without documented justification

## Troubleshooting

- **pyright errors** -- adapter source must pass `uv run pyright src/` with zero errors
- **Compliance failures** -- run full compliance and compare against baseline; check that validator lambdas are functionally identical

## Related packages

| Package | Purpose |
| --- | --- |
| [`valbridge`](https://pypi.org/project/valbridge/) | Runtime client for generated validators |
| [`valbridge-core`](https://pypi.org/project/valbridge-core/) | Core IR and JSON Schema parser |
| [`valbridge-pydantic-bridge`](https://pypi.org/project/valbridge-pydantic-bridge/) | Bridge helpers for Pydantic generation |
| [`@vectorfyco/valbridge-zod`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod) | TypeScript equivalent (Zod adapter) |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
