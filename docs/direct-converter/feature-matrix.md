# Direct Converter Feature Matrix

Cross-language mapping table for the valbridge direct conversion path.

**Scope:** Zod `4.3.x` and Pydantic `2.12.x` only.

---

## Fidelity Classes

| Class | Meaning | Action |
| --- | --- | --- |
| `native_exact` | Direct target-library construct with materially equivalent behavior | Emit native code |
| `native_approximate` | Target construct exists but semantics drift in a known way | Emit native code + diagnostic |
| `bridge_helper` | Parity requires a small repo-local helper runtime | Emit helper import + diagnostic |
| `unsupported_stub` | Cannot be emitted safely | Surface diagnostic or fail in strict mode |

---

## Core Mapping Table

### Pydantic to Zod

```mermaid
flowchart LR
    subgraph Source["Pydantic Source"]
        S1["StrictStr"]
        S2["StringConstraints(strip_whitespace)"]
        S3["ConfigDict(extra='forbid')"]
        S4["UUID4"]
        S5["AnyUrl / HttpUrl"]
        S6["AwareDatetime"]
        S7["Field(alias=...)"]
        S8["Field(discriminator=...)"]
        S9["PastDate / FutureDatetime"]
    end

    subgraph Target["Zod Target"]
        T1["z.string()"]
        T2[".trim()"]
        T3["strict object"]
        T4["z.uuidv4()"]
        T5["z.url() + refinement"]
        T6["z.iso.datetime(...)"]
        T7["metadata / renderer policy"]
        T8["z.discriminatedUnion(...)"]
        T9["refinement / bridge helper"]
    end

    S1 -->|"native_exact"| T1
    S2 -->|"native_exact"| T2
    S3 -->|"native_exact"| T3
    S4 -->|"native_exact"| T4
    S5 -->|"native_approximate"| T5
    S6 -->|"native_approximate"| T6
    S7 -->|"native_approximate"| T7
    S8 -->|"native_exact"| T8
    S9 -->|"bridge_helper"| T9

    style Source fill:#fce4ec,stroke:#E92063
    style Target fill:#e3f2fd,stroke:#3E67B1
```

### Full Mapping Reference

| Source Feature | Pydantic Source | Zod Target | Fidelity | Diagnostic |
| --- | --- | --- | --- | --- |
| Strict string | `StrictStr` | `z.string()` | `native_exact` | `native.exact.strict_string` |
| String trim | `StringConstraints(strip_whitespace=True)` | `.trim()` | `native_exact` | `native.exact.string_trim` |
| Object forbid extras | `ConfigDict(extra='forbid')` | strict object behavior | `native_exact` | `native.exact.object_extra_forbid` |
| UUID v4 | `UUID4` | `z.uuidv4()` | `native_exact` | `native.exact.uuid_v4` |
| URL | `AnyUrl` / `HttpUrl` | `z.url()` plus optional refinement | `native_approximate` | `native.approx.url` |
| Datetime ISO | `AwareDatetime` / `NaiveDatetime` | `z.iso.datetime(...)` where possible | `native_approximate` | `native.approx.iso_datetime` |
| Field alias | `Field(alias=...)` | metadata only + renderer policy | `native_approximate` | `native.approx.alias` |
| Field discriminator | `Field(discriminator=...)` | `z.discriminatedUnion(...)` when sound | `native_exact` | `native.exact.discriminator` |
| Left-to-right union | `Field(union_mode='left_to_right')` | no direct equivalent | `bridge_helper` | `bridge.union.left_to_right` |
| Temporal predicates | `PastDate`, `FutureDatetime`, etc. | refinement/helper | `bridge_helper` | `bridge.temporal.bound` |
| Custom validators | `field_validator`, `model_validator` | comment + hook only | `unsupported_stub` | `unsupported.custom_validator` |

### Zod to Pydantic

| Source Feature | Zod Source | Pydantic Target | Fidelity | Diagnostic |
| --- | --- | --- | --- | --- |
| Transform chain | `.trim()`, `.toLowerCase()` | `StringConstraints` or validators | mixed | `native.exact.transform.*` |
| Default | `.default()` | `Field(default=...)` | `native_exact` | `native.exact.default` |
| Prefault | `.prefault()` | no direct primitive | `bridge_helper` | `bridge.default.prefault` |
| Metadata | `.describe()`, `.meta()` | `Field(...)`, `json_schema_extra` | `native_exact` | `native.exact.metadata` |
| XOR semantics | refined unions | no native single API | `bridge_helper` | `bridge.object.xor` |

---

## Hard API Corrections

These are common mistakes that the renderer must avoid:

- Use `z.url()`, not `z.httpUrl()`
- Use instance `.prefault(value)`, not a top-level Zod helper
- Do not emit `z.xor()`
- Treat Pydantic `union_mode='left_to_right'` as field-level only
- Never emit always-failing runtime placeholders for unsupported validators

---

## Generation Policy

```mermaid
flowchart TD
    A["Schema feature detected"] --> B{"Native target construct?"}
    B -->|yes, exact| C["Emit native code"]
    B -->|yes, approximate| D["Emit native code + diagnostic"]
    B -->|no| E{"Helper can fill gap?"}
    E -->|yes| F["Emit bridge helper import + diagnostic"]
    E -->|no| G{"Strict mode?"}
    G -->|yes| H["Fail generation"]
    G -->|no| I["Emit diagnostic + comment"]

    style C fill:#e8f5e9,stroke:#059669
    style D fill:#fff3e0,stroke:#f59e0b
    style F fill:#fce4ec,stroke:#E92063
    style H fill:#fee2e2,stroke:#dc2626
    style I fill:#f3e5f5,stroke:#7c3aed
```

1. Prefer native target constructs first
2. Introduce a helper runtime only after a repeated parity gap is proven with tests
3. Emit explicit diagnostics for every non-exact mapping
4. Fail generation in strict mode instead of producing unsound runtime stubs
