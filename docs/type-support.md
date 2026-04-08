# Type Support Reference

Comprehensive mapping of every type and construct valbridge handles when converting between Zod v4+ and Pydantic v2.

---

## Compliance Summary

Tested against the official [JSON Schema Test Suite](https://github.com/json-schema-org/JSON-Schema-Test-Suite):

| Draft | Zod Adapter | Pydantic Adapter | Excluded |
| --- | --- | --- | --- |
| draft2020-12 | 1048/1048 (100%) | 1046/1048 (99.8%) | 223 (dynamic scope) |
| draft2019-09 | 1034/1034 (100%) | 1032/1034 (99.8%) | 200 (dynamic scope) |
| draft7 | 909/909 (100%) | 907/909 (99.8%) | 7 (metaschema) |
| draft6 | 825/825 (100%) | 823/825 (99.8%) | 7 (metaschema) |
| draft4 | 606/606 (100%) | 605/606 (99.8%) | 7 (metaschema) |
| draft3 | 429/429 (100%) | 428/429 (99.8%) | 5 (metaschema) |

**Excluded tests** are `$recursiveAnchor`/`$recursiveRef` dynamic scope tracking (cannot be compiled to static validator code) and metaschema self-validation `$ref` tests. Regular recursive `$ref` works in both adapters.

The 2 Pydantic failures per draft are `$ref` cases where adjacent validation keywords require scope-aware evaluation that naive inlining cannot achieve.

---

## Primitive Types

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `"type": "string"` | `z.string()` | `str` |
| `"type": "integer"` | `z.int()` | `StrictInt` (rejects `True`/`False`) |
| `"type": "number"` | `z.number()` | `StrictFloat` (accepts int, rejects bool) |
| `"type": "boolean"` | `z.boolean()` | `StrictBool` |
| `"type": "null"` | `z.null()` | `None` |
| `"type": "object"` | `z.object({...})` | `BaseModel` subclass |
| `"type": "array"` | `z.array(...)` | `list[T]` |

---

## String Constraints

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `minLength` / `maxLength` | `.min(n)` / `.max(n)` (grapheme-aware via `Intl.Segmenter`) | `StringConstraints(min_length=n, max_length=n)` |
| `pattern` | `.regex(...)` | `StringConstraints(pattern=...)` |
| `format: "email"` | `.email()` | `EmailStr` or format validator |
| `format: "uri"` | `.url()` | `AnyUrl` |
| `format: "uuid"` | `.uuid()` | `UUID` |
| `format: "date-time"` | `z.iso.datetime()` | `datetime` with format validation |
| `format: "date"` | `z.iso.date()` | `date` with format validation |
| `format: "time"` | `z.iso.time()` | `time` with format validation |
| `format: "ipv4"` | `.refine(...)` with full regex | Format validator |
| `format: "ipv6"` | `.refine(...)` with full regex | Format validator |
| `format: "hostname"` | `.regex(...)` RFC 1123 | Format validator |
| `x-valbridge.transforms: ["trim"]` | `.trim()` | `StringConstraints(strip_whitespace=True)` |
| `x-valbridge.transforms: ["toLowerCase"]` | `.toLowerCase()` | Custom validator |
| `x-valbridge.transforms: ["toUpperCase"]` | `.toUpperCase()` | Custom validator |

---

## Numeric Constraints

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `minimum` / `maximum` | `.min(n)` / `.max(n)` | `Field(ge=n)` / `Field(le=n)` |
| `exclusiveMinimum` / `exclusiveMaximum` | `.min(n + 1)` or `.refine(...)` | `Field(gt=n)` / `Field(lt=n)` |
| `multipleOf` | `.multipleOf(n)` | `Field(multiple_of=n)` or `BeforeValidator` for non-integer multipleOf on int types |

---

## Object Types

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `properties` + `required` | `z.object({ key: schema })` | `class Model(BaseModel): key: T` |
| `additionalProperties: false` | `.strict()` | `ConfigDict(extra='forbid')` |
| `additionalProperties: schema` | `.passthrough()` + refinement | `ConfigDict(extra='allow')` + `model_validator` |
| `patternProperties` | `.passthrough().superRefine(...)` with regex tests | `model_validator` with `re.search()` per pattern |
| `propertyNames` | `.superRefine(...)` validating each key | `model_validator` checking key validity |
| `minProperties` / `maxProperties` | `.refine(...)` counting `Object.keys()` | `model_validator` counting fields |
| `unevaluatedProperties` | `.superRefine(...)` checking undeclared keys | `model_validator` or `ConfigDict(extra='forbid')` |
| `dependentRequired` | `.refine(...)` checking co-presence | `model_validator` checking co-presence |
| `dependentSchemas` | `.superRefine(...)` conditional schema application | `model_validator` running TypeAdapter |
| Non-identifier property names | Standard `z.object()` with quoted keys | `Field(alias="original-name")` with sanitized Python names |
| Prototype properties (`__proto__`, `constructor`) | `z.any().superRefine(...)` with manual validation | Standard handling (Python has no prototype chain) |

---

## Array / Tuple Types

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `items: schema` | `z.array(schema)` | `list[T]` |
| `minItems` / `maxItems` | `.min(n)` / `.max(n)` | `Field(min_length=n, max_length=n)` |
| `uniqueItems` | `.refine(...)` with JSON stringify dedup | `AfterValidator` with `_json_equals()` |
| `contains` | `.refine(...)` counting matches | `AfterValidator` counting TypeAdapter matches |
| `minContains` / `maxContains` | `.refine(...)` counting matches with bounds | `AfterValidator` with bounded count |
| `prefixItems` (tuple) | `z.array(z.any()).superRefine(...)` per-position | `BeforeValidator` with per-position TypeAdapter |
| `prefixItems` + `items: false` (closed tuple) | Above + `.max(n)` length check | Above + length validation |
| `prefixItems` + `items: schema` (tuple with rest) | Above + rest item validation | Above + rest item TypeAdapter |
| `unevaluatedItems` | `z.array(z.never()).max(0)` or schema validation | `AfterValidator` checking length or validating |

---

## Composition / Complex Types

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `anyOf` (union) | `z.union([...])` | `T1 \| T2 \| T3` |
| `anyOf` with discriminator | `z.discriminatedUnion(key, [...])` | `Annotated[Union[...], Field(discriminator=key)]` |
| `oneOf` (exactly one) | `z.unknown().superRefine(...)` counting safeParse successes | `BeforeValidator` counting TypeAdapter matches, requiring exactly 1 |
| `allOf` (all objects) | `z.intersection(a, b)` | Static field merge into single BaseModel |
| `allOf` (mixed types) | `buildIntersection()` helper | `BeforeValidator` running each TypeAdapter sequentially |
| `not` | `z.unknown().refine(val => !schema.safeParse(val).success)` | `BeforeValidator` accepting only if TypeAdapter raises |
| `if` / `then` / `else` | `z.unknown().superRefine(...)` with branch dispatch | `BeforeValidator` with if-check and branch dispatch |
| Type guards | `z.unknown().superRefine(...)` with `typeof`/`Array.isArray` | `BeforeValidator` with `isinstance` dispatch |
| `nullable` | `.nullable()` | `T \| None` |

---

## References

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `$ref` (local) | Inlined or `z.lazy(() => ref)` | Inlined or forward reference string |
| `$ref` (external URL) | Fetched, bundled, then inlined | Fetched, bundled, then inlined |
| `$ref` (recursive/circular) | `z.lazy(() => selfRef)` | Forward reference `'ClassName'` + `model_rebuild()` |
| `$anchor` | Resolved at bundle time | Resolved at bundle time |
| `$defs` | Flattened into root `$defs` during bundling | Flattened into root `$defs` during bundling |

**Not supported:** `$dynamicRef` / `$dynamicAnchor` / `$recursiveRef` / `$recursiveAnchor` -- these require runtime scope tracking that cannot be compiled into static validators. Regular recursive `$ref` works fine.

---

## Constants and Enumerations

| JSON Schema | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `const` (primitive) | `z.literal(value)` | `Literal[value]` |
| `const` (boolean / 0 / 1) | `z.literal(true)` | `_make_const_validator` with `_json_equals` (avoids Python `Literal[False]` accepting `0`) |
| `const` (object) | `z.object({}).passthrough().refine(...)` with deep equality | `_make_const_validator` with deep equality |
| `const` (array) | `z.array(z.any()).refine(...)` with deep equality | `_make_const_validator` with deep equality |
| `enum` (simple primitives) | `z.union([z.literal(v1), z.literal(v2)])` | `Literal[v1, v2, ...]` |
| `enum` (mixed / complex) | `z.any().refine(...)` with deep-sorted JSON comparison | `_make_enum_validator` with `_json_equals` |

---

## Metadata Transport

valbridge preserves all standard JSON Schema annotations across languages:

| Annotation | Zod v4 Output | Pydantic v2 Output |
| --- | --- | --- |
| `description` | `.describe("...")` | `Field(description="...")` |
| `title` | `.meta({ title: "..." })` | `Field(title="...")` |
| `examples` | `.meta({ examples: [...] })` | `Field(examples=[...])` |
| `default` | `.default(value)` | `Field(default=value)` |
| `deprecated` | `.meta({ deprecated: true })` | `Field(deprecated=True)` |
| `readOnly` | `.meta({ readOnly: true })` | `Field(json_schema_extra={"readOnly": True})` |
| `writeOnly` | `.meta({ writeOnly: true })` | `Field(json_schema_extra={"writeOnly": True})` |

### `x-valbridge` enrichment

The `x-valbridge` extension carries cross-language metadata that JSON Schema alone cannot express:

| Field | Purpose | Example |
| --- | --- | --- |
| `coercionMode` | Whether the target should coerce or reject | `"strict"` or `"coerce"` |
| `transforms` | Ordered transform chain | `["trim", "toLowerCase"]` |
| `registryMeta.source` | Which language the schema was extracted from | `"pydantic"` or `"zod"` |
| `defaultBehavior.kind` | Default timing semantics | `"default"` (post-validation) or `"prefault"` (pre-validation, Zod v4 only) |
| `formatDetail` | Specific format variant | `{ "format": "uuid", "version": "v4" }` |
| `discriminator` | Explicit discriminator key for unions | `"type"` |
| `extraMode` | Object extra properties policy | `"allow"`, `"forbid"`, or `"ignore"` |

---

## Zod v4-Specific Features

These are Zod v4 APIs that valbridge natively supports -- they do not exist in Zod v3:

| Feature | Zod v4 API | How valbridge uses it |
| --- | --- | --- |
| Rich metadata | `.meta({ title, examples, deprecated })` | Transports annotations beyond `description` |
| Pre-validation defaults | `.prefault(value)` | Default applied before validation, not after |
| Versioned UUIDs | `z.uuidv4()`, `z.uuidv6()`, `z.uuidv7()` | Format-specific UUID validators |
| ISO datetime | `z.iso.datetime()`, `z.iso.date()`, `z.iso.time()` | String-based temporal validators |
| Pipe transforms | `z.pipe(input, ...transforms)` | Detected and annotated with code stubs |
| Internal introspection | `schema._zod.def.type` | Used by extractor for high-fidelity extraction |
| Schema-to-JSON-Schema | `z.toJSONSchema(schema)` | Official v4 export API used by extractor |

---

## IR Node Types

The valbridge intermediate representation uses 19 node types to model the full JSON Schema surface:

`string` | `number` | `boolean` | `null` | `object` | `array` | `tuple` | `union` | `intersection` | `oneOf` | `not` | `literal` | `enum` | `any` | `never` | `ref` | `conditional` | `typeGuarded` | `nullable`
