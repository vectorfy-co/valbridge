// Adapter protocol types (existing)
export type { ConvertInput, ConvertResult } from "./types.js";
export { createAdapterCLI } from "./cli.js";
export {
  DIAGNOSTIC_SEVERITIES,
  type Diagnostic,
  type DiagnosticSeverity,
} from "./diagnostics.js";

// JSON Schema types
export type { JSONSchema } from "./schema/index.js";

// Intermediate Representation (IR)
export type {
  // Main types
  SchemaAnnotations,
  Transform,
  AliasInfo,
  CodeStub,
  DefaultBehavior,
  FormatDetail,
  CoercionMode,
  ExtraMode,
  UnionResolution,
  SchemaNode,
  NodeKind,
  NodeOfKind,

  // Node types
  StringNode,
  NumberNode,
  BooleanNode,
  NullNode,
  ObjectNode,
  ArrayNode,
  TupleNode,
  UnionNode,
  IntersectionNode,
  OneOfNode,
  NotNode,
  LiteralNode,
  EnumNode,
  AnyNode,
  NeverNode,
  RefNode,

  ConditionalNode,
  TypeGuardedNode,
  NullableNode,
  TypeGuard,
  TypeCheck,

  // Constraints
  StringConstraints,
  NumberConstraints,
  ArrayConstraints,
  ContainsConstraint,
  PropertyDef,
  PatternPropertyDef,
  Dependency,
  PropertyDependency,
  SchemaDependency,
} from "./ir/index.js";

// Parser
export {
  parse,
  parseEnriched,
  parseSchema,
  type ParseContext,
  type ParseEnrichedResult,
} from "./parser/index.js";

// Utilities
export {
  // Primitives
  isPrimitive,
  escapeString,
  isEmptyObject,
  getOwnProperty,
  PROTOTYPE_PROPERTY_NAMES,
  hasPrototypeProperties,

  // JSON Pointer
  getRefName,

  // Code builder
  chain,
  buildUnion,
  buildIntersection,
  buildSuperRefine,
  buildRefine,
  buildLiteral,
  buildSafeParseCheck,
  buildPropertyCheck,
  sortedStringify,
  DEEP_SORTED_STRINGIFY_RUNTIME,
} from "./utils/index.js";
