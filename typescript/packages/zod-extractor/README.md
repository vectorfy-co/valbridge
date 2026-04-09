<div align="center">

# ![valbridge-zod-extractor](https://img.shields.io/static/v1?label=&message=%40vectorfyco%2Fvalbridge-zod-extractor&color=3E67B1&style=for-the-badge&logo=zod&logoColor=white)

Extract valbridge-compatible schema data from existing Zod schemas for high-fidelity Pydantic generation.

<a href="https://www.npmjs.com/package/@vectorfyco/valbridge-zod-extractor"><img src="https://img.shields.io/npm/v/@vectorfyco/valbridge-zod-extractor?style=flat&logo=npm&logoColor=white" alt="npm" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

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

Output is JSON with:
- `schema` -- the extracted JSON Schema document
- `diagnostics` -- compatibility diagnostics for unsupported Zod features or versions

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

## Key behaviors

- Validates the supported Zod version before extraction
- Preserves `x-valbridge` annotations needed by downstream adapters (Pydantic, future targets)
- Emits diagnostics for Zod features that cannot be represented in JSON Schema

## Related packages

| Package | Purpose |
| --- | --- |
| [`@vectorfyco/valbridge-zod`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod) | Zod adapter (JSON Schema to Zod) |
| [`@vectorfyco/valbridge-core`](https://www.npmjs.com/package/@vectorfyco/valbridge-core) | Core IR and JSON Schema parser |
| [`valbridge-pydantic-extractor`](https://pypi.org/project/valbridge-pydantic-extractor/) | Python equivalent for Pydantic models |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
