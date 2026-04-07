# Direct Converter Feature Matrix

Scope in this document is intentionally limited to:

- Zod `4.3.x`
- Pydantic `2.12.x`

## Fidelity Classes

- `native_exact`: direct target-library construct with materially equivalent behavior
- `native_approximate`: target-library construct exists but semantics drift in a known way
- `bridge_helper`: parity requires a small repo-local helper runtime
- `unsupported_stub`: cannot be emitted safely; generation must surface a diagnostic or fail in strict mode

## Core Mapping Table

| Source Feature | Pydantic Source | Zod Target | Fidelity | Diagnostic |
| --- | --- | --- | --- | --- |
| Strict string | `StrictStr` | `z.string()` | `native_exact` | `native.exact.strict_string` |
| String trim | `StringConstraints(strip_whitespace=True)` | `.trim()` | `native_exact` | `native.exact.string_trim` |
| Object forbid extras | `ConfigDict(extra='forbid')` | strict object behavior | `native_exact` | `native.exact.object_extra_forbid` |
| UUID v4 | `UUID4` | `z.uuidv4()` | `native_exact` | `native.exact.uuid_v4` |
| URL | `AnyUrl` / `HttpUrl` | `z.url()` plus optional refinement | `native_approximate` | `native.approx.url` |
| Datetime ISO | `AwareDatetime` / `NaiveDatetime` / datetime JSON schema | `z.iso.datetime(...)` where possible | `native_approximate` | `native.approx.iso_datetime` |
| Field alias | `Field(alias=...)` | metadata only in schema plus renderer policy | `native_approximate` | `native.approx.alias` |
| Field discriminator | `Field(discriminator=...)` | `z.discriminatedUnion(...)` when sound | `native_exact` | `native.exact.discriminator` |
| Left-to-right union mode | `Field(union_mode='left_to_right')` | no direct equivalent | `bridge_helper` or diagnostic | `bridge.union.left_to_right` |
| Past/future temporal predicates | `PastDate`, `FutureDatetime`, etc. | refinement/helper | `bridge_helper` | `bridge.temporal.bound` |
| Custom validator / serializer intent | `field_validator`, `field_serializer`, `model_validator` | comment + hook/helper only | `unsupported_stub` unless helper-backed | `unsupported.custom_validator` |
| Zod transform chain | `.trim()`, `.toLowerCase()`, `.transform(...)` | Pydantic constraints or validators | mixed | `native.exact.transform.*` / `bridge.transform.custom` |
| Zod `.default()` | instance default | Pydantic field default | `native_exact` | `native.exact.default` |
| Zod `.prefault()` | instance prefault | no direct Pydantic primitive | `bridge_helper` or approximation | `bridge.default.prefault` |
| Zod metadata | `.describe()`, `.meta()` | `Field(...)`, `json_schema_extra` | `native_exact` | `native.exact.metadata` |
| XOR object semantics | refined unions | no native single API in either target | `bridge_helper` | `bridge.object.xor` |

## Hard API Corrections

- Use `z.url()`, not `z.httpUrl()`.
- Use instance `.prefault(value)`, not a top-level Zod helper.
- Do not emit `z.xor()`.
- Treat Pydantic `union_mode='left_to_right'` as field-level only.
- Never emit always-failing runtime placeholders for unsupported validators.

## Generation Policy

- Prefer native target constructs first.
- Introduce a helper runtime only after a repeated parity gap is proven with tests.
- Emit explicit diagnostics for every non-exact mapping.
- Fail generation in strict mode instead of producing unsound runtime stubs.
