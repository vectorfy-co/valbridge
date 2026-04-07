# @vectorfyco/valbridge-zod-bridge

Shared bridge helpers used by the `valbridge` TypeScript toolchain when working with Zod-backed generation flows.

## Installation

```bash
npm install @vectorfyco/valbridge-zod-bridge
# or
pnpm add @vectorfyco/valbridge-zod-bridge
```

## Usage

```ts
import type { TemporalBridgeKind } from "@vectorfyco/valbridge-zod-bridge";

const mode: TemporalBridgeKind = "future";
```

## Notes

This package is intentionally small. It exists to hold bridge-level helper types and constants that are shared across generated output and adapter code without pulling in larger runtime dependencies.
