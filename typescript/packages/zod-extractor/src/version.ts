import type { Diagnostic } from "@vectorfyco/valbridge-core";

const SUPPORTED_MAJOR = 4;
const SUPPORTED_MINOR = 3;

export function ensureSupportedZodVersion(schema: {
	_zod?: {
		version?: {
			major?: number;
			minor?: number;
			patch?: number;
		};
	};
}): Diagnostic[] {
	const version = schema._zod?.version;
	if (
		version?.major === SUPPORTED_MAJOR &&
		version?.minor === SUPPORTED_MINOR
	) {
		return [];
	}

	return [
		{
			severity: "error",
			code: "zod_extractor.unsupported_version",
			message: `Unsupported Zod version ${version?.major ?? "unknown"}.${version?.minor ?? "unknown"}.${version?.patch ?? "unknown"}. Expected 4.3.x.`,
			source: "zod",
			suggestion:
				"Pin zod to 4.3.x before using the direct extractor runtime.",
		},
	];
}
