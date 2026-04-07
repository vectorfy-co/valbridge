<div align="center">

# ![valbridge-zod-bridge](https://img.shields.io/static/v1?label=&message=%40vectorfyco%2Fvalbridge-zod-bridge&color=3E67B1&style=for-the-badge&logo=zod&logoColor=white)

Shared bridge helpers for cross-language patterns that lack a native Zod equivalent.

<a href="https://www.npmjs.com/package/@vectorfyco/valbridge-zod-bridge"><img src="https://img.shields.io/npm/v/@vectorfyco/valbridge-zod-bridge?style=flat&logo=npm&logoColor=white" alt="npm" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
npm install @vectorfyco/valbridge-zod-bridge
# or
pnpm add @vectorfyco/valbridge-zod-bridge
```

## Purpose

This package is intentionally small. It holds bridge-level helper types and constants shared across generated output and adapter code, without pulling in larger runtime dependencies.

```ts
import type { TemporalBridgeKind } from "@vectorfyco/valbridge-zod-bridge";

const mode: TemporalBridgeKind = "future";
```

## When bridge helpers are used

The valbridge [fidelity system](https://github.com/vectorfy-co/valbridge#how-it-works) classifies every cross-language mapping. When a pattern is tagged `bridge_helper`, the generated code imports from this package instead of approximating behavior with a lossy native construct.

Examples: temporal past/future predicates, prefault default timing, XOR object semantics.

## Related packages

| Package | Purpose |
| --- | --- |
| [`@vectorfyco/valbridge-zod`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod) | Zod adapter for code generation |
| [`valbridge-pydantic-bridge`](https://pypi.org/project/valbridge-pydantic-bridge/) | Python equivalent for Pydantic |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
