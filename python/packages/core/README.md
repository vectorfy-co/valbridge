<div align="center">

# ![valbridge-core](https://img.shields.io/static/v1?label=&message=valbridge-core&color=3776AB&style=for-the-badge&logo=python&logoColor=white)

Core intermediate-representation types, diagnostics, and JSON Schema parsing utilities for the valbridge Python toolchain.

<a href="https://pypi.org/project/valbridge-core/"><img src="https://img.shields.io/pypi/v/valbridge-core?style=flat&logo=pypi&logoColor=white" alt="PyPI" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
pip install valbridge-core
# or
uv add valbridge-core
```

## What it provides

- **JSON Schema parser** -- `parse`, `parse_schema`, and `parse_enriched` convert JSON Schema into the valbridge IR
- **IR node types** -- shared intermediate-representation types used by all Python adapters
- **Diagnostics** -- structured diagnostic codes matching the cross-runtime fidelity contract
- **Adapter CLI helpers** -- stdin/stdout protocol utilities for adapter processes

## Usage

```python
from valbridge_core import parse_schema

schema = {
    "type": "object",
    "properties": {
        "id": {"type": "string", "format": "uuid"},
        "enabled": {"type": "boolean"}
    },
    "required": ["id"]
}

node = parse_schema(schema)
```

## Related packages

| Package | Purpose |
| --- | --- |
| [`valbridge`](https://pypi.org/project/valbridge/) | Runtime client for generated Python validators |
| [`valbridge-pydantic`](https://pypi.org/project/valbridge-pydantic/) | Render valbridge IR into Pydantic models |
| [`@vectorfyco/valbridge-core`](https://www.npmjs.com/package/@vectorfyco/valbridge-core) | TypeScript equivalent |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
