# Native vs Bridge Decisions

This document records the decision rule for the direct-converter path.

## Use Native Target APIs When

- The target library has a first-class construct.
- The construct preserves the important runtime behavior.
- The emitted code remains idiomatic for the target ecosystem.

Examples:

- Pydantic strict scalar types
- Pydantic `Field(...)` metadata
- Zod string transforms like `.trim()`
- Zod discriminated unions when the discriminator is explicit and sound

## Use A Bridge Helper When

- Behavior is reproducible but not directly representable.
- The helper can stay small, local, and explicit.
- The helper can be tested independently.

Examples:

- temporal past/future predicates
- prefault-like behavior with nontrivial default timing semantics
- XOR object semantics
- normalization helpers that must be shared across many emitted schemas

## Do Not Emit Silent Drift

If the converter cannot preserve semantics natively or through a small helper:

- emit a structured diagnostic
- add a generated comment where helpful
- optionally fail generation in strict mode

## Helper Runtime Constraints

- Helper packages must be repo-local.
- Helpers are scoped only to Zod `4.3.x` and Pydantic `2.12.x`.
- Helpers are opt-in dependencies of emitted code, not global runtime assumptions.
