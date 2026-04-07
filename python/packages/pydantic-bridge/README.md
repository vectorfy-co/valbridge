<div align="center">

# ![valbridge-pydantic-bridge](https://img.shields.io/static/v1?label=&message=valbridge-pydantic-bridge&color=E92063&style=for-the-badge&logo=pydantic&logoColor=white)

Shared bridge helpers for cross-language patterns that lack a native Pydantic equivalent.

<a href="https://pypi.org/project/valbridge-pydantic-bridge/"><img src="https://img.shields.io/pypi/v/valbridge-pydantic-bridge?style=flat&logo=pypi&logoColor=white" alt="PyPI" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
pip install valbridge-pydantic-bridge
# or
uv add valbridge-pydantic-bridge
```

## Purpose

This package is intentionally lightweight. It holds bridge-level helpers shared across Pydantic-facing valbridge components without forcing consumers to depend on larger adapter packages.

## When bridge helpers are used

The valbridge [fidelity system](https://github.com/vectorfy-co/valbridge#how-it-works) classifies every cross-language mapping. When a pattern is tagged `bridge_helper`, the generated code imports from this package instead of approximating behavior with a lossy native construct.

Examples: temporal past/future predicates, prefault default timing, XOR object semantics.

## Related packages

| Package | Purpose |
| --- | --- |
| [`valbridge-pydantic`](https://pypi.org/project/valbridge-pydantic/) | Pydantic adapter for code generation |
| [`@vectorfyco/valbridge-zod-bridge`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod-bridge) | TypeScript equivalent for Zod |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
