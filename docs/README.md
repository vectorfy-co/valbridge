# valbridge Documentation

Detailed technical documentation for the valbridge toolchain.

## Type Support

| Document | Description |
| --- | --- |
| [Type Support Reference](./type-support.md) | Complete mapping of every type and construct between Zod v4 and Pydantic v2, with compliance results |

## Architecture

| Document | Description |
| --- | --- |
| [CLI Architecture](./cli-current-flow.md) | Pipeline stages, data types, concurrency model, module dependencies |
| [CLI Analysis](./cli-analysis-and-improvements.md) | Known issues, performance analysis, improvement roadmap |

## Direct Converter

| Document | Description |
| --- | --- |
| [Feature Matrix](./direct-converter/feature-matrix.md) | Zod 4.x / Pydantic 2.x cross-language mapping table with fidelity classes |
| [Native vs Bridge](./direct-converter/native-vs-bridge.md) | Decision rules for when to use native APIs versus bridge helpers |
| [Enriched Format](./direct-converter/enriched-format.md) | `x-valbridge` JSON Schema extension specification |
| [Implementation Log](./direct-converter/implementation-log.md) | Development progress notes |

## Quick links

- [Main README](../README.md) -- project overview, quick start, installation
- [Language Package](../cli/language/README.md) -- adding new language targets
