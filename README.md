# valbridge

`valbridge` bridges validation engines between TypeScript and Python.

The current product surface is intentionally narrow:

- `Zod -> Pydantic`
- `Pydantic -> Zod`
- runtime bridge helpers for live conversion workflows
- code generation for CI/CD and checked-in artifacts

The repo is intentionally focused on the current Zod/Pydantic bridge. Anything outside that path should be treated as legacy until it is either removed or rewritten.

## Packages

- npm:
  - `@vectorfyco/valbridge`
  - `@vectorfyco/valbridge-core`
  - `@vectorfyco/valbridge-zod`
  - `@vectorfyco/valbridge-zod-extractor`
  - `@vectorfyco/valbridge-zod-bridge`
- PyPI / workspace packages:
  - `valbridge`
  - `valbridge-core`
  - `valbridge-pydantic`
  - `valbridge-pydantic-extractor`
  - `valbridge-pydantic-bridge`

## Status

The clean repo target is:

- GitHub: `vectorfy-co/valbridge`
- npm scope: `@vectorfyco/valbridge`

## Origin

This repo was originally forked from [`xschemadev/xschema`](https://github.com/xschemadev/xschema).

Thanks to the original `xschema` work for the foundation this refactor and rebrand were built from.

## Development

```bash
cd typescript && pnpm install
cd ../python && uv sync
cd ../cli && go test ./...
```
