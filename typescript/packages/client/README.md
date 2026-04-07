<div align="center">

# ![valbridge](https://img.shields.io/static/v1?label=&message=%40vectorfyco%2Fvalbridge&color=3178C6&style=for-the-badge&logo=typescript&logoColor=white)

Runtime client for valbridge-generated TypeScript validators with full type inference and compile-time safety.

<a href="https://www.npmjs.com/package/@vectorfyco/valbridge"><img src="https://img.shields.io/npm/v/@vectorfyco/valbridge?style=flat&logo=npm&logoColor=white" alt="npm" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
npm install @vectorfyco/valbridge
# or
pnpm add @vectorfyco/valbridge
```

## Quick start

```typescript
import { createValbridgeClient } from "@vectorfyco/valbridge";
import { schemas } from "./.valbridge/valbridge.gen.js";

const valbridge = createValbridgeClient({ schemas });

// Lookup a schema by namespace:id key (with full autocomplete)
const userSchema = valbridge("user:Profile");
const result = userSchema.parse({ id: "123", name: "john" });
```

## Default namespace

Set a default namespace to omit the prefix for that namespace:

```typescript
const valbridge = createValbridgeClient({
  schemas,
  defaultNamespace: "user"
});

// Shorthand for "user:Profile"
const profile = valbridge("Profile");

// Full key still works for other namespaces
const tsConfig = valbridge("another:TSConfig");
```

## Type extraction

Use `ValbridgeType` to extract TypeScript types from registered schemas at zero runtime cost:

```typescript
import type { ValbridgeType } from "@vectorfyco/valbridge";

type UserProfile = ValbridgeType<"user:Profile">;

function validateUser(data: unknown): UserProfile {
  return valbridge("user:Profile").parse(data);
}
```

## Generated output structure

The maintained TypeScript path uses Zod. Generated artifacts expose both runtime schemas and extracted types:

```typescript
// Generated code
const user_User = z.object({ id: z.string(), name: z.string() });
type user_UserType = z.infer<typeof user_User>;

export const schemas = { "user:User": user_User } as const;
export type SchemaTypes = { "user:User": user_UserType };
```

Both `valbridge("user:User")` and `ValbridgeType<"user:User">` work with full type inference.

## Key behaviors

- **Runtime lookup** -- only entries with a runtime schema appear in the `schemas` object
- **Type extraction** -- all entries appear in `SchemaTypes`, regardless of runtime validators
- **Compile-time safety** -- TypeScript prevents using invalid keys; full autocomplete for all registered schemas
- **Error messages** -- `valbridge("nonexistent:Key")` throws `Error: Unknown schema: nonexistent:Key. Run valbridge generate.`

## Related packages

| Package | Purpose |
| --- | --- |
| [`@vectorfyco/valbridge-core`](https://www.npmjs.com/package/@vectorfyco/valbridge-core) | Core IR and JSON Schema parser |
| [`@vectorfyco/valbridge-zod`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod) | Zod adapter for code generation |
| [`@vectorfyco/valbridge-cli`](https://www.npmjs.com/package/@vectorfyco/valbridge-cli) | CLI to generate validators |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
