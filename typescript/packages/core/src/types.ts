import type { Diagnostic } from "./diagnostics.js";

// Core types for valbridge adapters
export interface ConvertInput {
  namespace: string;
  id: string;
  varName: string;
  schema: object;
  sourceProfile?: "json-schema" | "pydantic" | "zod";
}

export interface ConvertResult {
  namespace: string;
  id: string;
  varName: string;
  imports: string[];
  schema?: string;
  type?: string;
  validate?: string;
  validationImports?: string[];
  diagnostics?: Diagnostic[];
}
