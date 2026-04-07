/**
 * SchemaNode - Intermediate Representation for JSON Schema
 *
 * A discriminated union representing all possible schema shapes.
 * This IR is designed to be:
 * 1. Exhaustively matchable via the `kind` discriminator
 * 2. Language-agnostic within the TypeScript ecosystem
 * 3. Renderer-friendly (contains all info needed for code generation)
 */

import type {
	StringConstraints,
	NumberConstraints,
	ArrayConstraints,
	ContainsConstraint,
	PropertyDef,
	PatternPropertyDef,
	Dependency,
} from "./constraints.js";

// Re-export with proper typing (resolves forward references)
export type {
	StringConstraints,
	NumberConstraints,
	ArrayConstraints,
	ContainsConstraint,
	PropertyDef,
	PatternPropertyDef,
	Dependency,
};

export type CoercionMode = "strict" | "coerce" | "passthrough";
export type ExtraMode = "allow" | "ignore" | "forbid";
export type UnionResolution = "smart" | "leftToRight" | "allErrors";

export interface Transform {
	kind: string;
	args?: Record<string, unknown>;
}

export interface AliasInfo {
	validationAlias?: string;
	serializationAlias?: string;
	aliasPath?: string[];
}

export interface CodeStub {
	kind: string;
	name: string;
	payload?: Record<string, unknown>;
}

export interface DefaultBehavior {
	kind: "default" | "prefault" | "factory";
	value?: unknown;
	factory?: string;
}

export interface FormatDetail {
	kind: string;
	data?: Record<string, unknown>;
}

export interface SchemaAnnotations {
	title?: string;
	description?: string;
	examples?: unknown[];
	default?: unknown;
	deprecated?: boolean;
	readOnly?: boolean;
	writeOnly?: boolean;
	registryMeta?: Record<string, unknown>;
	codeStubs?: CodeStub[];
	defaultBehavior?: DefaultBehavior;
}

/** String schema node */
export interface StringNode {
	kind: "string";
	constraints: StringConstraints;
	format?: string;
	coercionMode?: CoercionMode;
	transforms?: Transform[];
	formatDetail?: FormatDetail;
	annotations?: SchemaAnnotations;
}

/** Number schema node (includes integer) */
export interface NumberNode {
	kind: "number";
	constraints: NumberConstraints;
	integer: boolean;
	coercionMode?: CoercionMode;
	annotations?: SchemaAnnotations;
}

/** Boolean schema node */
export interface BooleanNode {
	kind: "boolean";
	coercionMode?: CoercionMode;
	annotations?: SchemaAnnotations;
}

/** Null schema node */
export interface NullNode {
	kind: "null";
	annotations?: SchemaAnnotations;
}

/** Object schema node */
export interface ObjectNode {
	kind: "object";
	properties: Map<string, PropertyDef>;
	/**
	 * additionalProperties - undefined means not specified in schema.
	 * Explicitly true means additionalProperties: true was in schema (evaluates extra props).
	 * false means additionalProperties: false.
	 * SchemaNode means additionalProperties was a schema (evaluates and validates extra props).
	 */
	additionalProperties?: SchemaNode | boolean;
	patternProperties: PatternPropertyDef[];
	propertyNames?: SchemaNode;
	minProperties?: number;
	maxProperties?: number;
	dependencies: Map<string, Dependency>;
	extraMode?: ExtraMode;
	discriminator?: string;
	/**
	 * unevaluatedProperties - only valid for standalone use (no applicators).
	 * Go CLI blocks schemas with unevaluated+applicators, so adapters only see simple cases.
	 */
	unevaluatedProperties?: SchemaNode | false;
	annotations?: SchemaAnnotations;
}

/** Array schema node (homogeneous items) */
export interface ArrayNode {
	kind: "array";
	items: SchemaNode;
	constraints: ArrayConstraints;
	/**
	 * unevaluatedItems - only valid for standalone use (no applicators).
	 * Go CLI blocks schemas with unevaluated+applicators, so adapters only see simple cases.
	 */
	unevaluatedItems?: SchemaNode | false;
	annotations?: SchemaAnnotations;
}

/** Tuple schema node (positional items) */
export interface TupleNode {
	kind: "tuple";
	prefixItems: SchemaNode[];
	restItems: SchemaNode | false;
	constraints: ArrayConstraints;
	/**
	 * unevaluatedItems - only valid for standalone use (no applicators).
	 * Go CLI blocks schemas with unevaluated+applicators, so adapters only see simple cases.
	 */
	unevaluatedItems?: SchemaNode | false;
	annotations?: SchemaAnnotations;
}

/** Union schema node (anyOf - at least one must match) */
export interface UnionNode {
	kind: "union";
	variants: SchemaNode[];
	resolution?: UnionResolution;
	discriminator?: string;
	annotations?: SchemaAnnotations;
}

/** Intersection schema node (allOf - all must match) */
export interface IntersectionNode {
	kind: "intersection";
	schemas: SchemaNode[];
	annotations?: SchemaAnnotations;
}

/** OneOf schema node (exactly one must match) */
export interface OneOfNode {
	kind: "oneOf";
	schemas: SchemaNode[];
	annotations?: SchemaAnnotations;
}

/** Not schema node (must not match) */
export interface NotNode {
	kind: "not";
	schema: SchemaNode;
	annotations?: SchemaAnnotations;
}

/** Literal schema node (const) */
export interface LiteralNode {
	kind: "literal";
	value: unknown;
	annotations?: SchemaAnnotations;
}

/** Enum schema node */
export interface EnumNode {
	kind: "enum";
	values: unknown[];
	annotations?: SchemaAnnotations;
}

/** Any schema node (accepts everything) */
export interface AnyNode {
	kind: "any";
	annotations?: SchemaAnnotations;
}

/** Never schema node (accepts nothing) */
export interface NeverNode {
	kind: "never";
	annotations?: SchemaAnnotations;
}

/** Reference schema node — recursive back-edge kept by the processor */
export interface RefNode {
	kind: "ref";
	ref: string;
	annotations?: SchemaAnnotations;
}

/** Conditional schema node (if/then/else) */
export interface ConditionalNode {
	kind: "conditional";
	if: SchemaNode;
	then?: SchemaNode;
	else?: SchemaNode;
	annotations?: SchemaAnnotations;
}

/**
 * Type-guarded schema node
 * Used when JSON Schema has type-specific keywords but no explicit type.
 * The schema only applies when the value matches the type check.
 */
export interface TypeGuardedNode {
	kind: "typeGuarded";
	guards: TypeGuard[];
	annotations?: SchemaAnnotations;
}

export interface TypeGuard {
	check: TypeCheck;
	schema: SchemaNode;
}

export type TypeCheck = "object" | "array" | "string" | "number";

/**
 * Nullable wrapper node
 * Represents OpenAPI 3.0 nullable or union with null
 */
export interface NullableNode {
	kind: "nullable";
	inner: SchemaNode;
	annotations?: SchemaAnnotations;
}

/** The discriminated union of all schema nodes */
export type SchemaNode =
	| StringNode
	| NumberNode
	| BooleanNode
	| NullNode
	| ObjectNode
	| ArrayNode
	| TupleNode
	| UnionNode
	| IntersectionNode
	| OneOfNode
	| NotNode
	| LiteralNode
	| EnumNode
	| AnyNode
	| NeverNode
	| RefNode
	| ConditionalNode
	| TypeGuardedNode
	| NullableNode;

/** Helper type to extract a specific node kind */
export type NodeOfKind<K extends SchemaNode["kind"]> = Extract<
	SchemaNode,
	{ kind: K }
>;

/** All possible node kinds */
export type NodeKind = SchemaNode["kind"];
