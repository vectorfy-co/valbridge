<div align="center">

# ![valbridge-pydantic-extractor](https://img.shields.io/static/v1?label=&message=valbridge-pydantic-extractor&color=E92063&style=for-the-badge&logo=pydantic&logoColor=white)

Extract JSON Schema from existing Pydantic models, enriched with `x-valbridge` annotations for high-fidelity cross-language generation.

<a href="https://pypi.org/project/valbridge-pydantic-extractor/"><img src="https://img.shields.io/pypi/v/valbridge-pydantic-extractor?style=flat&logo=pypi&logoColor=white" alt="PyPI" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
pip install valbridge-pydantic-extractor
# or
uv add valbridge-pydantic-extractor
```

## CLI usage

```bash
valbridge-pydantic-extractor app.models:User --python-path .
```

Output is JSON with:
- `schema` -- the extracted JSON Schema document
- `diagnostics` -- import or extraction diagnostics when the target cannot be resolved cleanly

## Options

| Flag | Description |
| --- | --- |
| `--python-path <path>` | Prepend an import path before loading the target module |
| `--module-root <path>` | Add module roots for project-local imports |
| `--stub-module <module>` | Install placeholder modules for optional imports |
| `--env KEY=VALUE` | Inject environment variables before importing the model |

## Key behaviors

- Target must use `module:Class` format
- Target class must inherit from `pydantic.BaseModel`
- Extracted output preserves `x-valbridge` annotations needed by downstream code generation
- Emits diagnostics for Pydantic features that cannot be represented in JSON Schema

## Related packages

| Package | Purpose |
| --- | --- |
| [`valbridge-pydantic`](https://pypi.org/project/valbridge-pydantic/) | Pydantic adapter (JSON Schema to Pydantic) |
| [`valbridge-core`](https://pypi.org/project/valbridge-core/) | Core IR and JSON Schema parser |
| [`@vectorfyco/valbridge-zod-extractor`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod-extractor) | TypeScript equivalent for Zod schemas |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
