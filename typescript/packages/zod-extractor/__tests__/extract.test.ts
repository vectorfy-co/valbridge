import { expect, test } from "vitest";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { z } from "zod";

import { extractSchema, extractFromModule, main } from "../src/index.js";
import { ensureSupportedZodVersion } from "../src/version.js";

test("extractSchema preserves zod-only semantics in x-valbridge", () => {
	const Cat = z
		.object({
			kind: z.literal("cat"),
			name: z.string().trim().toLowerCase(),
		})
		.strict();
	const Dog = z.object({
		kind: z.literal("dog"),
		bark: z.number(),
	});

	const schema = z
		.object({
			id: z.uuidv4(),
			preview: z.string().prefault("preview"),
			slug: z.string().default("guest"),
			label: z.coerce.string(),
			chooser: z.coerce.number(),
			enabled: z.coerce.boolean(),
			pet: z.discriminatedUnion("kind", [Cat, Dog]),
			name: z.string().trim().toLowerCase().describe("Display name"),
			computed: z.string().transform((value) => value.trim()),
			preprocessed: z.preprocess(
				(value) => (typeof value === "string" ? value.trim() : value),
				z.string().min(1),
			),
			decodedAt: z.codec(z.string(), z.date(), {
				decode: (value) => new Date(value),
				encode: (value) => value.toISOString(),
			}),
			checked: z.string().refine((value) => value.length > 1, "too short"),
		})
		.passthrough()
		.meta({ source: "zod" })
		.describe("Root");

	const result = extractSchema(schema);

	expect(result.diagnostics).toEqual([]);
	expect(result.schema["x-valbridge"]).toMatchObject({
		version: "1.0",
		extraMode: "allow",
	});
	expect(result.schema.properties.id["x-valbridge"].formatDetail).toEqual({
		kind: "uuid",
		version: "v4",
	});
	expect(result.schema.properties.preview["x-valbridge"].defaultBehavior).toEqual({
		kind: "prefault",
		value: "preview",
	});
	expect(result.schema.properties.slug["x-valbridge"].defaultBehavior).toEqual({
		kind: "default",
		value: "guest",
	});
	expect(result.schema.properties.label["x-valbridge"].coercionMode).toBe("coerce");
	expect(result.schema.properties.chooser["x-valbridge"].coercionMode).toBe("coerce");
	expect(result.schema.properties.enabled["x-valbridge"].coercionMode).toBe("coerce");
	expect(result.schema.properties.pet["x-valbridge"].discriminator).toBe("kind");
	expect(result.schema.properties.name["x-valbridge"].transforms).toEqual([
		{ kind: "trim" },
		{ kind: "toLowerCase" },
	]);
	expect(result.schema.properties.computed["x-valbridge"].codeStubs).toEqual([
		{ kind: "transform", name: "transform" },
	]);
	expect(result.schema.properties.preprocessed["x-valbridge"].codeStubs).toEqual([
		{ kind: "preprocess", name: "preprocess" },
	]);
	expect(result.schema.properties.decodedAt["x-valbridge"].codeStubs).toEqual([
		{
			kind: "codec",
			name: "codec",
			payload: { inputType: "string", outputType: "date" },
		},
	]);
	expect(result.schema.properties.decodedAt["x-valbridge"].registryMeta).toEqual({
		codecInputType: "string",
		codecOutputType: "date",
	});
	expect(result.schema.properties.checked["x-valbridge"].codeStubs).toEqual([
		{ kind: "validator", name: "custom" },
	]);
});

test("extractFromModule loads a local module export", async () => {
	const dir = mkdtempSync(join(process.cwd(), ".tmp-zod-extractor-"));
	try {
		const modulePath = join(dir, "sample-schema.mjs");
		writeFileSync(
			modulePath,
			[
				"import { z } from 'zod';",
				"export const sampleSchema = z.object({ name: z.string().trim() }).strict();",
			].join("\n"),
		);

		const result = await extractFromModule({
			modulePath,
			exportName: "sampleSchema",
		});

		expect(result.diagnostics).toEqual([]);
		expect(result.schema.type).toBe("object");
		expect(result.schema.properties.name["x-valbridge"].transforms).toEqual([
			{ kind: "trim" },
		]);
	} finally {
		rmSync(dir, { recursive: true, force: true });
	}
});

test("main prints JSON extraction output", async () => {
	const dir = mkdtempSync(join(process.cwd(), ".tmp-zod-extractor-cli-"));
	try {
		const modulePath = join(dir, "cli-schema.mjs");
		writeFileSync(
			modulePath,
			[
				"import { z } from 'zod';",
				"export const cliSchema = z.object({ count: z.coerce.number() });",
			].join("\n"),
		);

		const chunks: string[] = [];
		const exitCode = await main(
			["--module-path", modulePath, "--export-name", "cliSchema"],
			{
				write: (chunk) => chunks.push(chunk),
			},
		);

		const payload = JSON.parse(chunks.join(""));
		expect(exitCode).toBe(0);
		expect(payload.schema.properties.count["x-valbridge"].coercionMode).toBe(
			"coerce",
		);
	} finally {
		rmSync(dir, { recursive: true, force: true });
	}
});

test("ensureSupportedZodVersion rejects unsupported versions", () => {
	const diagnostics = ensureSupportedZodVersion({
		_zod: { version: { major: 5, minor: 0, patch: 0 } },
	} as never);

	expect(diagnostics).toEqual([
		expect.objectContaining({
			code: "zod_extractor.unsupported_version",
			severity: "error",
		}),
	]);
});

test("extractSchema fails fast on unsupported zod versions", () => {
	expect(() =>
		extractSchema({
			_zod: { version: { major: 5, minor: 0, patch: 0 } },
		} as never),
	).toThrow("Unsupported Zod version 5.0.0. Expected 4.3.x.");
});

test("main returns a non-zero exit code for unsupported zod versions", async () => {
	const dir = mkdtempSync(join(process.cwd(), ".tmp-zod-extractor-version-"));
	try {
		const modulePath = join(dir, "unsupported-schema.mjs");
		writeFileSync(
			modulePath,
			[
				"export const unsupportedSchema = {",
				"  _zod: { version: { major: 5, minor: 0, patch: 0 } }",
				"};",
			].join("\n"),
		);

		const chunks: string[] = [];
		const exitCode = await main(
			["--module-path", modulePath, "--export-name", "unsupportedSchema"],
			{
				write: (chunk) => chunks.push(chunk),
			},
		);

		const payload = JSON.parse(chunks.join(""));
		expect(exitCode).toBe(1);
		expect(payload.schema).toBeNull();
		expect(payload.diagnostics).toEqual([
			expect.objectContaining({
				code: "zod_extractor.unsupported_version",
				severity: "error",
			}),
		]);
	} finally {
		rmSync(dir, { recursive: true, force: true });
	}
});
