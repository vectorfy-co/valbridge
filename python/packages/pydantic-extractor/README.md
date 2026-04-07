# valbridge-pydantic-extractor

Workspace-local extraction utilities for turning Pydantic models into JSON Schema enriched for `valbridge`.

## Installation

```bash
pip install valbridge-pydantic-extractor
```

## CLI usage

```bash
valbridge-pydantic-extractor app.models:User --python-path .
```

The extractor prints JSON with:

- `schema`: the extracted schema document
- `diagnostics`: import or extraction diagnostics when the target cannot be resolved cleanly

## Options

- `--python-path <path>`: prepend an import path before loading the target module
- `--module-root <path>`: add one or more module roots for project-local imports
- `--stub-module <module>`: install placeholder modules for optional imports during extraction
- `--env KEY=VALUE`: inject environment variables before importing the model

## Notes

- the target must use `module:Class` format
- the target class must inherit from `pydantic.BaseModel`
- extracted output preserves `x-valbridge` annotations needed by downstream code generation
