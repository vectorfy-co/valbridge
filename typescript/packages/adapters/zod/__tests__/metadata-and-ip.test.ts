import { describe, expect, test } from "vitest";
import { z } from "zod";
import { convert } from "../src/index.js";

function getSchemaCode(jsonSchema: Record<string, unknown>) {
	const result = convert({
		namespace: "test",
		id: "metadata",
		varName: "test_metadata",
		schema: jsonSchema,
	});
	if (!result.schema) throw new Error("No schema generated");
	return result.schema;
}

function evalSchema(schemaCode: string) {
	const fn = new Function("z", `return ${schemaCode}`);
	return fn(z) as z.ZodTypeAny;
}

describe("IP format compatibility", () => {
	test("renders ipv4 without z.string().ip()", () => {
		const schemaCode = getSchemaCode({ type: "string", format: "ipv4" });
		expect(schemaCode.includes(".ip(")).toBe(false);

		const schema = evalSchema(schemaCode);
		expect(schema.safeParse("192.168.0.1").success).toBe(true);
		expect(schema.safeParse("999.168.0.1").success).toBe(false);
	});

	test("renders ipv6 without z.string().ip()", () => {
		const schemaCode = getSchemaCode({ type: "string", format: "ipv6" });
		expect(schemaCode.includes(".ip(")).toBe(false);

		const schema = evalSchema(schemaCode);
		expect(schema.safeParse("2001:db8::1").success).toBe(true);
		expect(schema.safeParse("2001:::1").success).toBe(false);
	});
});

describe("metadata emission", () => {
	test("emits description and zod 4 metadata when present", () => {
		const schemaCode = getSchemaCode({
			type: "string",
			title: "Email",
			description: "Primary email address",
			examples: ["a@example.com"],
			default: "a@example.com",
			deprecated: true,
			readOnly: true,
			writeOnly: false,
		});

		expect(schemaCode).toContain('.describe("Primary email address")');
		expect(schemaCode).toContain('Reflect.get(schema, "meta")');

		const schema = evalSchema(schemaCode);
		const jsonSchema = z.toJSONSchema(schema);

		expect(jsonSchema.title).toBe("Email");
		expect(jsonSchema.description).toBe("Primary email address");
		expect(jsonSchema.examples).toEqual(["a@example.com"]);
		expect(jsonSchema.default).toBe("a@example.com");
		expect(jsonSchema.deprecated).toBe(true);
		expect(jsonSchema.readOnly).toBe(true);
		expect(jsonSchema.writeOnly).toBe(false);
	});

	test("merges referenced and local metadata for $ref schemas", () => {
		const schemaCode = getSchemaCode({
			$defs: {
				Email: {
					type: "string",
					description: "Canonical email address",
					examples: ["a@example.com"],
				},
			},
			$ref: "#/$defs/Email",
			title: "User email",
		});

		const schema = evalSchema(schemaCode);
		const jsonSchema = z.toJSONSchema(schema);

		expect(jsonSchema.title).toBe("User email");
		expect(jsonSchema.description).toBe("Canonical email address");
		expect(jsonSchema.examples).toEqual(["a@example.com"]);
	});
});

describe("enriched rendering", () => {
	test("renders coercion-aware string, number, and boolean schemas", () => {
		const schemaCode = getSchemaCode({
			type: "object",
			properties: {
				name: {
					type: "string",
					"x-valbridge": { version: "1.0", coercionMode: "coerce" },
				},
				count: {
					type: "number",
					"x-valbridge": { version: "1.0", coercionMode: "coerce" },
				},
				enabled: {
					type: "boolean",
					"x-valbridge": { version: "1.0", coercionMode: "coerce" },
				},
			},
		});

		expect(schemaCode).toContain("z.coerce.string()");
		expect(schemaCode).toContain("z.coerce.number()");
		expect(schemaCode).toContain("z.preprocess((value) => {");

		const schema = evalSchema(schemaCode);
		const parsed = schema.safeParse({
			name: 42,
			count: "7.5",
			enabled: "true",
		});

		expect(parsed.success).toBe(true);
		if (parsed.success) {
			expect(parsed.data).toEqual({
				name: "42",
				count: 7.5,
				enabled: true,
			});
		}

		expect(
			schema.safeParse({ name: 42, count: "7.5", enabled: "false" }).data?.enabled,
		).toBe(false);
		expect(
			schema.safeParse({ name: 42, count: "7.5", enabled: "0" }).data?.enabled,
		).toBe(false);
		expect(
			schema.safeParse({ name: 42, count: "7.5", enabled: "yes" }).data?.enabled,
		).toBe(true);
		expect(
			schema.safeParse({ name: 42, count: "7.5", enabled: "" }).success,
		).toBe(false);
	});

	test("renders UUID format detail without losing coercion", () => {
		const schemaCode = getSchemaCode({
			type: "string",
			"x-valbridge": {
				version: "1.0",
				coercionMode: "coerce",
				formatDetail: { kind: "uuid", version: "v4" },
			},
		});

		expect(schemaCode).toContain("z.coerce.string().check(z.uuidv4())");

		const schema = evalSchema(schemaCode);
		expect(schema.safeParse(550).success).toBe(false);
		expect(schema.safeParse("550e8400-e29b-41d4-a716-446655440000").success).toBe(
			true,
		);
		expect(schema.safeParse("550e8400-e29b-11d4-a716-446655440000").success).toBe(
			false,
		);
	});

	test("uses explicit boolean parsing instead of JS truthiness for coercion", () => {
		const schemaCode = getSchemaCode({
			type: "boolean",
			"x-valbridge": { version: "1.0", coercionMode: "coerce" },
		});

		const schema = evalSchema(schemaCode);

		for (const value of ["t", "T", "true", "True", "1", "yes", "on", "y", 1, true]) {
			expect(schema.safeParse(value).data).toBe(true);
		}
		for (const value of ["f", "F", "false", "False", "0", "no", "off", "n", 0, false]) {
			expect(schema.safeParse(value).data).toBe(false);
		}
		for (const value of ["", "maybe", " false ", " 1 ", "\ttrue\n", 2, -1, null, [], {}]) {
			expect(schema.safeParse(value).success).toBe(false);
		}
	});

	test("renders formatDetail, transforms, and prefault from x-valbridge", () => {
		const schemaCode = getSchemaCode({
			type: "string",
			"x-valbridge": {
				version: "1.0",
				formatDetail: { kind: "uuid", version: "v4" },
				transforms: ["trim", { kind: "toLowerCase" }],
				defaultBehavior: { kind: "prefault", value: "guest" },
			},
		});

		expect(schemaCode).toContain("z.uuidv4()");
		expect(schemaCode).toContain(".trim()");
		expect(schemaCode).toContain(".toLowerCase()");
		expect(schemaCode).toContain('.prefault("guest")');

		const schema = evalSchema(schemaCode);
		expect(schema.safeParse("550e8400-e29b-41d4-a716-446655440000").success).toBe(
			true,
		);
	});

	test("prefers formatDetail uuid constructors over base uuid format chaining", () => {
		const schemaCode = getSchemaCode({
			type: "string",
			format: "uuid",
			"x-valbridge": {
				version: "1.0",
				formatDetail: { kind: "uuid", version: "v4" },
			},
		});

		expect(schemaCode).toContain("z.uuidv4()");
		expect(schemaCode).not.toContain(".uuid()");

		const schema = evalSchema(schemaCode);
		expect(schema.safeParse("550e8400-e29b-41d4-a716-446655440000").success).toBe(
			true,
		);
	});

	test("renders discriminated unions and object extra mode from x-valbridge", () => {
		const schemaCode = getSchemaCode({
			type: "object",
			properties: {
				pet: {
					anyOf: [
						{
							type: "object",
							properties: {
								kind: { const: "cat" },
								name: { type: "string" },
							},
							required: ["kind", "name"],
						},
						{
							type: "object",
							properties: {
								kind: { const: "dog" },
								bark: { type: "number" },
							},
							required: ["kind", "bark"],
						},
					],
					"x-valbridge": {
						version: "1.0",
						discriminator: "kind",
					},
				},
			},
			"x-valbridge": {
				version: "1.0",
				extraMode: "forbid",
			},
		});

		expect(schemaCode).toContain('z.discriminatedUnion("kind"');
		expect(schemaCode).toContain(".strict()");

		const schema = evalSchema(schemaCode);
		expect(schema.safeParse({ pet: { kind: "cat", name: "Milo" } }).success).toBe(
			true,
		);
		expect(
			schema.safeParse({ pet: { kind: "cat", name: "Milo" }, extra: true }).success,
		).toBe(false);
	});
});
