# @vectorfyco/valbridge

Runtime client for generated `valbridge` TypeScript validators with full type inference.

## installation

```bash
npm install @vectorfyco/valbridge
# or
pnpm add @vectorfyco/valbridge
```

## usage

### basic client setup

```typescript
import { createValbridgeClient } from "@vectorfyco/valbridge";
import { schemas } from "./.valbridge/valbridge.gen.js";

const valbridge = createValbridgeClient({ schemas });

// lookup a schema by full namespace:id key
const userSchema = valbridge("user:Profile");
const tsConfigSchema = valbridge("another:TSConfig");

// use the schema (with zod in the current product)
const result = userSchema.parse({ id: "123", name: "john" });
```

### default namespace

set a default namespace to omit the namespace prefix for that namespace:

```typescript
const valbridge = createValbridgeClient({ 
  schemas, 
  defaultNamespace: "user" 
});

// shorthand for "user:Profile"
const profile = valbridge("Profile");

// still need full key for other namespaces
const tsConfig = valbridge("another:TSConfig");
```

### type helpers

use `ValbridgeType` to extract typescript types from your schemas:

```typescript
import type { ValbridgeType } from "@vectorfyco/valbridge";

// extract type from any schema
type UserProfile = ValbridgeType<"user:Profile">;
type TSConfig = ValbridgeType<"another:TSConfig">;

// use in function signatures
function validateUser(data: unknown): UserProfile {
  const schema = valbridge("user:Profile");
  return schema.parse(data);
}
```

## schema + type output

Today, the maintained TypeScript path is Zod-based. Generated artifacts expose both runtime schemas and extracted types:

```typescript
// generated code
const user_User = z.object({ id: z.string(), name: z.string() });
type user_UserType = z.infer<typeof user_User>;

// in schemas object (runtime lookup available)
export const schemas = {
  "user:User": user_User,
} as const;

// in SchemaTypes (type extraction available)
export type SchemaTypes = {
  "user:User": user_UserType;
};
```

Both `valbridge("user:User")` and `ValbridgeType<"user:User">` work.

## key behaviors

- **runtime lookup**: only entries with a runtime schema (`.Code` in generated output) appear in the `schemas` object
- **type extraction**: all entries appear in `SchemaTypes`, regardless of whether they have runtime validators
- **compile-time safety**: typescript prevents using type-only keys with `valbridge()` and schema-only/type-only keys where not applicable

## examples

### using with zod

```typescript
import { createValbridgeClient } from "@vectorfyco/valbridge";
import type { ValbridgeType } from "@vectorfyco/valbridge";
import { schemas } from "./.valbridge/valbridge.gen.js";

const valbridge = createValbridgeClient({ schemas, defaultNamespace: "user" });

// validate data
const userSchema = valbridge("Profile");
const validatedUser = userSchema.parse(unknownData);

// extract types
type User = ValbridgeType<"user:Profile">;

function createUser(data: User): void {
  // ...
}
```

## error handling

if you try to look up a schema that doesn't exist:

```typescript
const schema = valbridge("nonexistent:Key");
// Error: Unknown schema: nonexistent:Key. Run `valbridge generate`.
```

typescript will catch this at compile time with full autocomplete support.

## TypeScript features

- full autocomplete for all registered schema keys
- compile-time errors for invalid keys
- type inference for schema validators
- works cleanly with the current Zod renderer path

## Learn more

- [valbridge documentation](https://github.com/vectorfy-co/valbridge/docs)
- [github repository](https://github.com/vectorfy-co/valbridge)
