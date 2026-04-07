/**
 * valbridge example for generated TypeScript validators.
 *
 * Run `pnpm run generate` to regenerate schemas, then `pnpm run start`.
 */

import { schemas } from "./.valbridge/valbridge.gen";
import { createValbridgeClient, ValbridgeType } from "@vectorfyco/valbridge";

// Create the valbridge client with generated schemas
// defaultNamespace allows shorthand lookups: valbridge("Calendar") instead of valbridge("user:Calendar")
export const valbridge = createValbridgeClient({
  schemas,
  defaultNamespace: "user",
});

// ============================================
// Type extraction using ValbridgeType helper
// ============================================

// "namespace:id"
export type CalendarType = ValbridgeType<"user:Calendar">;
export type TSConfigType = ValbridgeType<"another:TSConfig">;
export type UserType = ValbridgeType<"user:User">;

// Native ts converter
export type TSConfigNativeTS = ValbridgeType<"another:TSConfigNative">;

// ============================================
// Zod schema
// ============================================

// Use full "namespace:id" to get schemas from any namespace
const tsConfigSchema = valbridge("another:TSConfig");
const calendarSchema = valbridge("user:Calendar");

// When defaultNamespace is set, you can omit it for that namespace
const calendar = valbridge("Calendar"); // Same as valbridge("user:Calendar")
const validCalendar = calendar.parse({
  dtstart: "2024-01-01",
  summary: "New Year's Day",
});
const result = calendar.safeParse({ invalid: "data" });
type InferredCalendar = typeof validCalendar;

// ============================================
// Additional Zod schema
// ============================================

const user = valbridge("User");
const validUser = user.parse({
  id: "123",
  email: "alice@example.com",
  name: "Alice",
  age: 30,
});
