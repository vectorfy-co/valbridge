import type {
	AliasInfo,
	BooleanNode,
	CodeStub,
	DefaultBehavior,
	FormatDetail,
	NumberNode,
	ObjectNode,
	SchemaAnnotations,
	SchemaNode,
	StringNode,
	Transform,
	UnionNode,
} from "../ir/nodes.js";
import type { Diagnostic } from "../diagnostics.js";
import type { JSONSchema, ValbridgeExtension } from "../schema/json-schema.js";
import type { ParseContext } from "./context.js";

const KNOWN_VALBRIDGE_KEYS = new Set<keyof ValbridgeExtension>([
	"version",
	"coercionMode",
	"transforms",
	"formatDetail",
	"registryMeta",
	"codeStubs",
	"defaultBehavior",
	"aliasInfo",
	"extraMode",
	"discriminator",
	"resolution",
]);

function isRecord(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === "object" && !Array.isArray(value);
}

function extractValbridgeExtension(
	schema: JSONSchema | boolean,
): ValbridgeExtension | undefined {
	if (!schema || typeof schema !== "object") return undefined;
	const extension = schema["x-valbridge"];
	return extension && typeof extension === "object" ? extension : undefined;
}

function normalizeTransforms(
	value: ValbridgeExtension["transforms"],
): Transform[] | undefined {
	if (!Array.isArray(value) || value.length === 0) return undefined;
	return value.flatMap((transform) => {
		if (typeof transform === "string") {
			return [{ kind: transform }];
		}
		if (isRecord(transform) && typeof transform.kind === "string") {
			return [
				{
					kind: transform.kind,
					args: isRecord(transform.args) ? transform.args : undefined,
				},
			];
		}
		return [];
	});
}

function normalizeFormatDetail(
	value: ValbridgeExtension["formatDetail"],
): FormatDetail | undefined {
	if (!isRecord(value) || typeof value.kind !== "string") return undefined;
	const { kind, ...data } = value;
	return {
		kind,
		data: Object.keys(data).length > 0 ? data : undefined,
	};
}

function normalizeCodeStubs(
	value: ValbridgeExtension["codeStubs"],
): CodeStub[] | undefined {
	if (!Array.isArray(value) || value.length === 0) return undefined;
	return value.flatMap((stub) => {
		if (!isRecord(stub)) return [];
		if (typeof stub.kind !== "string" || typeof stub.name !== "string") return [];
		return [
			{
				kind: stub.kind,
				name: stub.name,
				payload: isRecord(stub.payload) ? stub.payload : undefined,
			},
		];
	});
}

function normalizeDefaultBehavior(
	value: ValbridgeExtension["defaultBehavior"],
): DefaultBehavior | undefined {
	if (!isRecord(value) || typeof value.kind !== "string") return undefined;
	if (
		value.kind !== "default" &&
		value.kind !== "prefault" &&
		value.kind !== "factory"
	) {
		return undefined;
	}
	return {
		kind: value.kind,
		value: value.value,
		factory: typeof value.factory === "string" ? value.factory : undefined,
	};
}

export function extractAliasInfo(
	schema: JSONSchema | boolean,
): AliasInfo | undefined {
	const extension = extractValbridgeExtension(schema);
	const aliasInfo = extension?.aliasInfo;
	if (!isRecord(aliasInfo)) return undefined;
	const normalized: AliasInfo = {};
	if (typeof aliasInfo.validationAlias === "string") {
		normalized.validationAlias = aliasInfo.validationAlias;
	}
	if (typeof aliasInfo.serializationAlias === "string") {
		normalized.serializationAlias = aliasInfo.serializationAlias;
	}
	if (Array.isArray(aliasInfo.aliasPath)) {
		normalized.aliasPath = aliasInfo.aliasPath.filter(
			(segment): segment is string => typeof segment === "string",
		);
	}
	return Object.keys(normalized).length > 0 ? normalized : undefined;
}

function mergeAnnotations(
	base?: SchemaAnnotations,
	override?: SchemaAnnotations,
): SchemaAnnotations | undefined {
	if (!base && !override) return undefined;
	return { ...(base ?? {}), ...(override ?? {}) };
}

function extractAnnotations(schema: JSONSchema): SchemaAnnotations | undefined {
	const annotations: SchemaAnnotations = {};

	if (schema.title !== undefined) annotations.title = schema.title;
	if (schema.description !== undefined)
		annotations.description = schema.description;
	if (schema.examples !== undefined) annotations.examples = schema.examples;
	if (schema.default !== undefined) annotations.default = schema.default;
	if (schema.deprecated !== undefined)
		annotations.deprecated = schema.deprecated;
	if (schema.readOnly !== undefined) annotations.readOnly = schema.readOnly;
	if (schema.writeOnly !== undefined)
		annotations.writeOnly = schema.writeOnly;

	const extension = extractValbridgeExtension(schema);
	if (extension?.registryMeta && isRecord(extension.registryMeta)) {
		annotations.registryMeta = extension.registryMeta;
	}
	const codeStubs = normalizeCodeStubs(extension?.codeStubs);
	if (codeStubs) annotations.codeStubs = codeStubs;
	const defaultBehavior = normalizeDefaultBehavior(extension?.defaultBehavior);
	if (defaultBehavior) annotations.defaultBehavior = defaultBehavior;

	return Object.keys(annotations).length > 0 ? annotations : undefined;
}

function createUnknownKeyDiagnostic(key: string): Diagnostic {
	return {
		severity: "warning",
		code: "valbridge.unknown_extension_key",
		message: `Unknown x-valbridge key '${key}' was preserved but has no runtime effect.`,
		path: `x-valbridge.${key}`,
		source: "valbridge",
		suggestion:
			"Remove the key or add runtime support before depending on it.",
	};
}

function collectUnknownExtensionDiagnostics(
	extension: ValbridgeExtension,
	ctx?: ParseContext,
): void {
	if (!ctx) return;
	for (const key of Object.keys(extension)) {
		if (!KNOWN_VALBRIDGE_KEYS.has(key as keyof ValbridgeExtension)) {
			ctx.diagnostics.push(createUnknownKeyDiagnostic(key));
		}
	}
}

export function withEnrichment<T extends SchemaNode>(
	node: T,
	schema: JSONSchema | boolean,
	ctx?: ParseContext,
): T {
	if (!schema || typeof schema !== "object") return node;

	const annotations = mergeAnnotations(node.annotations, extractAnnotations(schema));
	let result: SchemaNode =
		node.kind === "ref" && "resolved" in node && node.resolved
			? ((resolved) =>
					({
						...resolved,
					annotations: mergeAnnotations(
							resolved.annotations,
						annotations,
					),
					} as SchemaNode))(
					node.resolved as SchemaNode,
				)
			: annotations === undefined
				? node
				: ({ ...node, annotations } as T);

	const extension = extractValbridgeExtension(schema);
	if (!extension) {
		return result as T;
	}
	collectUnknownExtensionDiagnostics(extension, ctx);

	switch (result.kind) {
		case "string":
			result = {
				...(result as StringNode),
				coercionMode: extension.coercionMode ?? result.coercionMode,
				transforms:
					normalizeTransforms(extension.transforms) ?? result.transforms,
				formatDetail:
					normalizeFormatDetail(extension.formatDetail) ??
					result.formatDetail,
			};
			break;
		case "number":
			result = {
				...(result as NumberNode),
				coercionMode: extension.coercionMode ?? result.coercionMode,
			};
			break;
		case "boolean":
			result = {
				...(result as BooleanNode),
				coercionMode: extension.coercionMode ?? result.coercionMode,
			};
			break;
		case "object":
			result = {
				...(result as ObjectNode),
				extraMode: extension.extraMode ?? result.extraMode,
				discriminator:
					extension.discriminator ?? result.discriminator,
			};
			break;
		case "union":
			result = {
				...(result as UnionNode),
				resolution: extension.resolution ?? result.resolution,
				discriminator:
					extension.discriminator ?? result.discriminator,
			};
			break;
	}

	return result as T;
}
