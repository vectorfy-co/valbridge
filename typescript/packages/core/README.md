# @vectorfyco/valbridge-core

Core intermediate-representation types, diagnostics, and JSON Schema parsing utilities for `valbridge`.

## Installation

```bash
npm install @vectorfyco/valbridge-core
# or
pnpm add @vectorfyco/valbridge-core
```

## What it provides

- JSON Schema parser entry points such as `parse`, `parseSchema`, and `parseEnriched`
- shared IR node and annotation types used by adapters
- diagnostics and adapter CLI helpers used across the `valbridge` toolchain

## Usage

```ts
import { parseSchema } from "@vectorfyco/valbridge-core";

const schema = {
  type: "object",
  properties: {
    id: { type: "string" },
    enabled: { type: "boolean" }
  },
  required: ["id"]
};

const node = parseSchema(schema);
```

## Related packages

- `@vectorfyco/valbridge` for runtime schema lookup in generated TypeScript output
- `@vectorfyco/valbridge-zod` for rendering `valbridge` IR into Zod validators
