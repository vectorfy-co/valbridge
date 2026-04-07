<div align="center">

# ![valbridge-core](https://img.shields.io/static/v1?label=&message=%40vectorfyco%2Fvalbridge-core&color=3178C6&style=for-the-badge&logo=typescript&logoColor=white)

Core intermediate-representation types, diagnostics, and JSON Schema parsing utilities for the valbridge toolchain.

<a href="https://www.npmjs.com/package/@vectorfyco/valbridge-core"><img src="https://img.shields.io/npm/v/@vectorfyco/valbridge-core?style=flat&logo=npm&logoColor=white" alt="npm" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
npm install @vectorfyco/valbridge-core
# or
pnpm add @vectorfyco/valbridge-core
```

## What it provides

- **JSON Schema parser** -- entry points `parse`, `parseSchema`, and `parseEnriched` convert JSON Schema into the valbridge IR
- **IR node types** -- shared intermediate-representation types used by all adapters (Zod, Pydantic, and future targets)
- **Diagnostics** -- structured diagnostic codes for cross-runtime fidelity reporting
- **Adapter CLI helpers** -- stdin/stdout protocol utilities for adapter processes

## Usage

```ts
import { parseSchema } from "@vectorfyco/valbridge-core";

const schema = {
  type: "object",
  properties: {
    id: { type: "string", format: "uuid" },
    enabled: { type: "boolean" }
  },
  required: ["id"]
};

const node = parseSchema(schema);
```

### Enriched parsing (with `x-valbridge` metadata)

```ts
import { parseEnriched } from "@vectorfyco/valbridge-core";

const schema = {
  type: "string",
  "x-valbridge": {
    version: "1.0",
    coercionMode: "strict",
    transforms: ["trim"]
  }
};

const node = parseEnriched(schema);
```

## Related packages

| Package | Purpose |
| --- | --- |
| [`@vectorfyco/valbridge`](https://www.npmjs.com/package/@vectorfyco/valbridge) | Runtime client for generated TypeScript validators |
| [`@vectorfyco/valbridge-zod`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod) | Render valbridge IR into Zod validators |
| [`@vectorfyco/valbridge-zod-extractor`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod-extractor) | Extract JSON Schema from existing Zod schemas |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
