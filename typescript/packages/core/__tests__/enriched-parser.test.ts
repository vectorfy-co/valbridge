import { expect, test } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { parse, parseEnriched } from "../src/index.js";

test("parse reads x-valbridge string enrichment", () => {
	const result = parse({
		type: "string",
		"x-valbridge": {
			version: "1.0",
			coercionMode: "strict",
			transforms: ["trim", { kind: "toLowerCase" }],
			formatDetail: { kind: "uuid", version: "v4" },
			registryMeta: { source: "zod" },
			codeStubs: [{ kind: "validator", name: "slugify" }],
			defaultBehavior: { kind: "prefault", value: "guest" },
		},
	});

	expect(result).toMatchObject({
		kind: "string",
		coercionMode: "strict",
		transforms: [{ kind: "trim" }, { kind: "toLowerCase" }],
		formatDetail: {
			kind: "uuid",
			data: { version: "v4" },
		},
		annotations: {
			registryMeta: { source: "zod" },
			codeStubs: [{ kind: "validator", name: "slugify" }],
			defaultBehavior: { kind: "prefault", value: "guest" },
		},
	});
});

test("parse stores property alias info on object properties", () => {
	const result = parse({
		type: "object",
		properties: {
			displayName: {
				type: "string",
				"x-valbridge": {
					version: "1.0",
					aliasInfo: {
						validationAlias: "display_name",
						serializationAlias: "displayName",
					},
				},
			},
		},
	});

	expect(result.kind).toBe("object");
	if (result.kind !== "object") {
		throw new Error("expected object node");
	}

	expect(result.properties.get("displayName")).toMatchObject({
		required: false,
		aliasInfo: {
			validationAlias: "display_name",
			serializationAlias: "displayName",
		},
	});
});

test("parseEnriched merges ref-site enrichment and reports unknown keys", () => {
	const fixturePath = resolve(
		import.meta.dirname,
		"../test-fixtures/enriched/string-ref-merge.json",
	);
	const schema = JSON.parse(readFileSync(fixturePath, "utf8"));

	const result = parseEnriched(schema);

	expect(result.node).toMatchObject({
		kind: "string",
		coercionMode: "strict",
		transforms: [{ kind: "trim" }],
	});
	expect(result.diagnostics).toEqual([
		expect.objectContaining({
			severity: "warning",
			code: "valbridge.unknown_extension_key",
			path: "x-valbridge.unexpectedKey",
		}),
	]);
});
