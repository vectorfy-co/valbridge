# @vectorfyco/valbridge-zod-extractor

Workspace-local extraction utilities for turning Zod schemas back into JSON Schema enriched for `valbridge`.

## Installation

```bash
npm install @vectorfyco/valbridge-zod-extractor zod
# or
pnpm add @vectorfyco/valbridge-zod-extractor zod
```

## CLI usage

```bash
valbridge-zod-extractor \
  --module-path ./schema/user.js \
  --export-name UserSchema
```

The command prints JSON with:

- `schema`: the extracted JSON Schema document
- `diagnostics`: compatibility diagnostics for unsupported Zod features or versions

## Programmatic usage

```ts
import { extractSchema } from "@vectorfyco/valbridge-zod-extractor";
import { z } from "zod";

const UserSchema = z.object({
  id: z.string(),
  email: z.string().email()
});

const result = extractSchema(UserSchema);
console.log(result.schema);
```

## Notes

- built for the current `valbridge` Zod integration path
- validates the supported Zod version before extraction
- preserves `x-valbridge` annotations needed by downstream tooling
