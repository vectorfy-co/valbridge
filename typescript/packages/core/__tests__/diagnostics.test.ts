import { expect, test } from "vitest";

import type { ConvertResult, Diagnostic } from "../src/index.js";
import { DIAGNOSTIC_SEVERITIES } from "../src/diagnostics.js";

test("diagnostics are serializable on convert results", () => {
	const diagnostics: Diagnostic[] = [
		{
			severity: "warning",
			code: "bridge.temporal.bound",
			path: "$.properties.createdAt",
			message: "Rendered with helper-backed temporal bound validation.",
			source: "pydantic",
			target: "zod",
			suggestion: "Install the local zod bridge helper when using this output.",
		},
	];

	const result: ConvertResult = {
		namespace: "example",
		id: "Event",
		varName: "eventSchema",
		imports: [],
		schema: "z.object({})",
		diagnostics,
	};

	expect(JSON.parse(JSON.stringify(result))).toEqual({
		namespace: "example",
		id: "Event",
		varName: "eventSchema",
		imports: [],
		schema: "z.object({})",
		diagnostics: [
			{
				severity: "warning",
				code: "bridge.temporal.bound",
				path: "$.properties.createdAt",
				message: "Rendered with helper-backed temporal bound validation.",
				source: "pydantic",
				target: "zod",
				suggestion:
					"Install the local zod bridge helper when using this output.",
			},
		],
	});
});

test("diagnostic severities stay stable", () => {
	expect(DIAGNOSTIC_SEVERITIES).toEqual(["error", "warning", "info"]);
});
