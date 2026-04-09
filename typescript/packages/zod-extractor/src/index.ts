import { z } from "zod";
import { register } from "tsx/esm/api";

import type { Diagnostic, JSONSchema } from "@vectorfyco/valbridge-core";

import { ensureSupportedZodVersion } from "./version.js";

declare const process: {
	argv: string[];
	exit(code?: number): void;
};

declare global {
	interface ImportMeta {
		main?: boolean;
	}
}

export interface ZodExtractionTarget {
	modulePath: string;
	exportName: string;
}

export interface ExtractedSchemaResult {
	schema: JSONSchema;
	diagnostics: Diagnostic[];
}

export interface OutputWriter {
	write(chunk: string): void;
}

export const ZOD_EXTRACTOR_SCOPE = "zod-4.3.x";

let scopeCounter = 0;

class ZodExtractorError extends Error {
	readonly diagnostics: Diagnostic[];

	constructor(message: string, diagnostics: Diagnostic[]) {
		super(message);
		this.name = "ZodExtractorError";
		this.diagnostics = diagnostics;
	}
}

export function extractSchema(schema: z.ZodType): ExtractedSchemaResult {
	const diagnostics = ensureSupportedZodVersion(schema);
	if (diagnostics.length > 0) {
		throw new ZodExtractorError(diagnostics[0]?.message ?? "Unsupported Zod version.", diagnostics);
	}
	const jsonSchema = z.toJSONSchema(schema, {
		unrepresentable: "any",
	}) as JSONSchema;
	ensureValbridgeExtension(jsonSchema).sourceProfile = "zod";

	enrichJsonSchema(jsonSchema, schema);

	return {
		schema: jsonSchema,
		diagnostics,
	};
}

export async function extractFromModule(
	target: ZodExtractionTarget,
): Promise<ExtractedSchemaResult> {
	const moduleUrl = toFileModuleUrl(target.modulePath);
	const scope = register({ namespace: `valbridge-zod-extractor-${nextScopeID()}` });
	let loaded: Record<string, unknown>;
	try {
		loaded = (await scope.import(moduleUrl, import.meta.url)) as Record<string, unknown>;
	} finally {
		await scope.unregister();
	}
	const schema = loaded[target.exportName];

	if (!schema || typeof schema !== "object" || !("_zod" in schema)) {
		throw new TypeError(
			`Export '${target.exportName}' from '${target.modulePath}' is not a Zod schema.`,
		);
	}

	return extractSchema(schema as z.ZodType);
}

function nextScopeID(): string {
	const maybeRandomUUID = globalThis.crypto?.randomUUID;
	if (typeof maybeRandomUUID === "function") {
		return maybeRandomUUID.call(globalThis.crypto);
	}

	scopeCounter += 1;
	return `fallback-${scopeCounter}`;
}

export async function main(
	argv: string[],
	output: OutputWriter = {
		write(chunk: string) {
			console.log(chunk);
		},
	},
): Promise<number> {
	try {
		const args = parseArgs(argv);
		const result = await extractFromModule(args);
		output.write(`${JSON.stringify(result, null, 2)}\n`);
		return 0;
	} catch (error) {
		const diagnostics = normalizeDiagnostics(error);
		output.write(`${JSON.stringify({ schema: null, diagnostics }, null, 2)}\n`);
		return 1;
	}
}

function parseArgs(argv: string[]): ZodExtractionTarget {
	let modulePath: string | undefined;
	let exportName: string | undefined;

	for (let index = 0; index < argv.length; index += 1) {
		const arg = argv[index];
		if (arg === "--module-path") {
			modulePath = argv[index + 1];
			index += 1;
			continue;
		}
		if (arg === "--export-name") {
			exportName = argv[index + 1];
			index += 1;
			continue;
		}
	}

	if (!modulePath || !exportName) {
		throw new Error(
			"Expected --module-path <path> and --export-name <name> for zod extraction.",
		);
	}

	return { modulePath, exportName };
}

function enrichJsonSchema(jsonSchema: JSONSchema, schema: z.ZodType): void {
	const def = getDef(schema);
	if (!def || typeof jsonSchema !== "object" || jsonSchema === null) {
		return;
	}

	switch (def.type) {
		case "default":
		case "prefault": {
			ensureValbridgeExtension(jsonSchema).defaultBehavior = {
				kind: def.type,
				value: def.defaultValue,
			};
			enrichJsonSchema(jsonSchema, def.innerType as z.ZodType);
			return;
		}
		case "pipe": {
			const inputSchema = def.in as z.ZodType;
			const outputSchema = def.out as z.ZodType;
			const inputDef = getDef(inputSchema);
			const outputDef = getDef(outputSchema);

			if (inputDef?.type === "transform") {
				if (isEmptySchema(jsonSchema)) {
					Object.assign(jsonSchema, stripSchemaKeyword(toInnerJsonSchema(outputSchema)));
				}
				enrichJsonSchema(jsonSchema, outputSchema);
				appendCodeStub(jsonSchema, {
					kind: "preprocess",
					name: "preprocess",
				});
				return;
			}

			if (outputDef?.type === "transform") {
				if (isEmptySchema(jsonSchema)) {
					Object.assign(jsonSchema, stripSchemaKeyword(toInnerJsonSchema(inputSchema)));
				}
				enrichJsonSchema(jsonSchema, inputSchema);
				appendCodeStub(jsonSchema, {
					kind: "transform",
					name: "transform",
				});
				return;
			}

			if (isEmptySchema(jsonSchema)) {
				Object.assign(jsonSchema, stripSchemaKeyword(toInnerJsonSchema(outputSchema)));
			}
			enrichJsonSchema(jsonSchema, outputSchema);
			appendCodeStub(jsonSchema, {
				kind: "codec",
				name: "codec",
				payload: {
					inputType: inputDef?.type,
					outputType: outputDef?.type,
				},
			});
			appendRegistryMeta(jsonSchema, {
				codecInputType: inputDef?.type,
				codecOutputType: outputDef?.type,
			});
			return;
		}
		case "object": {
			const extension = ensureValbridgeExtension(jsonSchema);
			extension.version = "1.0";
			extension.sourceProfile = "zod";
			extension.extraMode = mapExtraMode(def.catchall);

			const properties = jsonSchema.properties;
			if (properties && typeof properties === "object") {
				for (const [key, child] of Object.entries(def.shape ?? {})) {
					if (key in properties) {
						enrichJsonSchema(properties[key] as JSONSchema, child as z.ZodType);
					}
				}
			}
			return;
		}
		case "union": {
			if (typeof def.discriminator === "string") {
				ensureValbridgeExtension(jsonSchema).discriminator = def.discriminator;
			}
			const variants = Array.isArray(jsonSchema.oneOf)
				? jsonSchema.oneOf
				: Array.isArray(jsonSchema.anyOf)
					? jsonSchema.anyOf
					: [];
			for (const [index, option] of (def.options ?? []).entries()) {
				const variantSchema = variants[index];
				if (variantSchema && typeof variantSchema === "object") {
					enrichJsonSchema(variantSchema as JSONSchema, option as z.ZodType);
				}
			}
			return;
		}
		case "number": {
			if (def.coerce === true) {
				ensureValbridgeExtension(jsonSchema).coercionMode = "coerce";
			}
			return;
		}
		case "boolean": {
			if (def.coerce === true) {
				ensureValbridgeExtension(jsonSchema).coercionMode = "coerce";
			}
			return;
		}
		case "string": {
			const extension = ensureValbridgeExtension(jsonSchema);
			if (def.coerce === true) {
				extension.coercionMode = "coerce";
			}
			if (def.format === "uuid" && typeof def.version === "string") {
				extension.formatDetail = {
					kind: "uuid",
					version: def.version,
				};
			}

			for (const check of def.checks ?? []) {
				const checkDef = getDef(check as z.ZodType | { _zod?: { def?: unknown } });
				if (!checkDef) continue;

				if (checkDef.check === "overwrite") {
					const transform = inferOverwriteTransform(checkDef.tx);
					if (transform) {
						appendTransform(jsonSchema, transform);
					}
					continue;
				}

				if (checkDef.check === "custom" || checkDef.type === "custom") {
					appendCodeStub(jsonSchema, {
						kind: "validator",
						name: "custom",
					});
				}
			}
			return;
		}
	}
}

function ensureValbridgeExtension(schema: JSONSchema): Record<string, unknown> & {
	version?: string;
	extraMode?: string;
	coercionMode?: string;
	discriminator?: string;
	defaultBehavior?: { kind: string; value?: unknown };
	formatDetail?: { kind: string; version?: string };
	codeStubs?: Array<Record<string, unknown>>;
	transforms?: Array<Record<string, unknown>>;
	registryMeta?: Record<string, unknown>;
} {
	const existing = schema["x-valbridge"];
	if (existing && typeof existing === "object" && !Array.isArray(existing)) {
		return existing as ReturnType<typeof ensureValbridgeExtension>;
	}

	const created = {};
	schema["x-valbridge"] = created;
	return created as ReturnType<typeof ensureValbridgeExtension>;
}

function appendCodeStub(
	schema: JSONSchema,
	stub: Record<string, unknown>,
): void {
	const extension = ensureValbridgeExtension(schema);
	const stubs = Array.isArray(extension.codeStubs)
		? [...extension.codeStubs]
		: [];
	stubs.push(stub);
	extension.codeStubs = stubs;
}

function appendRegistryMeta(
	schema: JSONSchema,
	meta: Record<string, unknown>,
): void {
	const extension = ensureValbridgeExtension(schema);
	const existing = extension.registryMeta;
	extension.registryMeta =
		existing && typeof existing === "object" && !Array.isArray(existing)
			? { ...existing, ...meta }
			: { ...meta };
}

function appendTransform(
	schema: JSONSchema,
	transform: Record<string, unknown>,
): void {
	const extension = ensureValbridgeExtension(schema);
	const transforms = Array.isArray(extension.transforms)
		? [...extension.transforms]
		: [];
	transforms.push(transform);
	extension.transforms = transforms;
}

function inferOverwriteTransform(
	tx: ((value: string) => string) | undefined,
): Record<string, unknown> | undefined {
	if (!tx) return undefined;

	if (tx(" A ") === "A" && tx("AbC") === "AbC") {
		return { kind: "trim" };
	}
	if (tx(" A ") === " a " && tx("AbC") === "abc") {
		return { kind: "toLowerCase" };
	}
	if (tx(" A ") === " A " && tx("AbC") === "ABC") {
		return { kind: "toUpperCase" };
	}
	return undefined;
}

function mapExtraMode(catchall: unknown): "allow" | "ignore" | "forbid" {
	const catchallDef = getDef(catchall);
	if (catchallDef?.type === "unknown") return "allow";
	if (catchallDef?.type === "never") return "forbid";
	return "ignore";
}

function toInnerJsonSchema(schema: z.ZodType): JSONSchema {
	return z.toJSONSchema(schema, {
		unrepresentable: "any",
	}) as JSONSchema;
}

function stripSchemaKeyword(schema: JSONSchema): JSONSchema {
	const cloned = { ...schema };
	delete cloned.$schema;
	return cloned;
}

function getDef(
	value: { _zod?: { def?: unknown } } | unknown,
): Record<string, any> | undefined {
	if (!value || typeof value !== "object") return undefined;
	return (value as { _zod?: { def?: unknown } })._zod?.def as
		| Record<string, any>
		| undefined;
}

function isEmptySchema(schema: JSONSchema): boolean {
	return Object.keys(schema).length === 0;
}

function toFileModuleUrl(modulePath: string): string {
	return new URL(`file://${modulePath}`).href;
}

function normalizeDiagnostics(error: unknown): Diagnostic[] {
	if (error instanceof ZodExtractorError) {
		return error.diagnostics;
	}

	return [
		{
			severity: "error",
			code: "zod_extractor.extract_error",
			message: error instanceof Error ? error.message : String(error),
			source: "zod",
			suggestion:
				"Verify the module path, export name, and workspace runner before retrying extraction.",
		},
	];
}

if (import.meta.main) {
	const exitCode = await main(process.argv.slice(2));
	process.exit(exitCode);
}
