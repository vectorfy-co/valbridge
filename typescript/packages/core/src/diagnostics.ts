export const DIAGNOSTIC_SEVERITIES = ["error", "warning", "info"] as const;

export type DiagnosticSeverity = (typeof DIAGNOSTIC_SEVERITIES)[number];

export interface Diagnostic {
	severity: DiagnosticSeverity;
	code: string;
	message: string;
	path?: string;
	source?: string;
	target?: string;
	suggestion?: string;
}
