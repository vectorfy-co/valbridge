<div align="center">

# ![valbridge-zod](https://img.shields.io/static/v1?label=&message=%40vectorfyco%2Fvalbridge-zod&color=3E67B1&style=for-the-badge&logo=zod&logoColor=white)

Zod 4.x adapter for valbridge -- converts JSON Schema into idiomatic Zod validators with full TypeScript type inference.

<a href="https://www.npmjs.com/package/@vectorfyco/valbridge-zod"><img src="https://img.shields.io/npm/v/@vectorfyco/valbridge-zod?style=flat&logo=npm&logoColor=white" alt="npm" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
pnpm add @vectorfyco/valbridge-zod zod
```

Supported Zod range: `^4.0.0`. Verified in CI against `zod 4.3.6`.

## Usage

This adapter is invoked by the valbridge CLI. Define schemas in a config file:

```jsonc
// user.valbridge.jsonc
{
  "$schema": "https://github.com/vectorfy-co/valbridge/schemas/typescript.jsonc",
  "schemas": [
    {
      "id": "User",
      "adapter": "@vectorfyco/valbridge-zod",
      "sourceType": "file",
      "source": "./schemas/user.json"
    }
  ]
}
```

Then generate:

```bash
valbridge generate
```

Use the generated schemas with the [valbridge runtime client](https://www.npmjs.com/package/@vectorfyco/valbridge).

## JSON Schema compliance

See [compliance/results/REPORT.md](./compliance/results/REPORT.md) for detailed test results from the JSON Schema Test Suite.

## Verification

Run from the adapter directory (`typescript/packages/adapters/zod/`):

```bash
# JSON Schema Test Suite compliance (requires Go CLI)
pnpm run compliance

# Unit tests
pnpm run test

# Type checking
pnpm run typecheck

# Type-fidelity harness (checks for any-leakage regressions)
pnpm run type-fidelity
```

## Fallback typing guardrails

The adapter uses `z.unknown()` as a safe fallback for unbounded domains and `z.any()` only where Zod's type system cannot represent the constraint statically. The goal: minimize `any` in inferred TypeScript types.

### Allowed `z.any()` fallbacks

| Construct | Reason |
|-----------|--------|
| IR `any` node | Empty schema `{}` or `true` -- no constraints to express |
| Empty/all-any intersection | Zero effective schemas |
| Tuple base | `z.tuple()` does not support open tuples with rest items |
| Complex const array | Deep equality validated via runtime helper |
| Complex enum values | Values span multiple types |
| Prototype-property objects | `z.object()` cannot handle `__proto__`, `constructor`, etc. |

### Constructs that must NOT use `any`

These are validated by the type-fidelity harness and will fail CI on regression:

- `oneOf`, `not`, `conditional`, `typeGuarded` -- must use `z.unknown()`
- `const` (object) -- must use `z.object({}).passthrough().refine(...)`

## Troubleshooting

- **Type-fidelity shows `IMPROVED`** -- a probe got narrower than expected. Update `expectAny` to `false` to lock in the improvement.
- **Type-fidelity shows `FAIL`** -- a renderer change regressed the type output. Fix the regression.
- **`ZodEffects` method errors** -- array size constraints must come before `.superRefine()`/`.refine()` calls.

## Related packages

| Package | Purpose |
| --- | --- |
| [`@vectorfyco/valbridge`](https://www.npmjs.com/package/@vectorfyco/valbridge) | Runtime client for generated validators |
| [`@vectorfyco/valbridge-core`](https://www.npmjs.com/package/@vectorfyco/valbridge-core) | Core IR and JSON Schema parser |
| [`@vectorfyco/valbridge-zod-bridge`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod-bridge) | Bridge helpers for Zod generation |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
