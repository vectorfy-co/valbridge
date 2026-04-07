package typescript

import "testing"

func TestMergeImports(t *testing.T) {
	tests := []struct {
		name     string
		imports  []string
		expected string
	}{
		{
			name:     "empty",
			imports:  []string{},
			expected: "",
		},
		{
			name: "dedupe same import",
			imports: []string{
				`import { z } from "zod"`,
				`import { z } from "zod"`,
			},
			expected: `import { z } from "zod"`,
		},
		{
			name: "merge named imports from same source",
			imports: []string{
				`import { z } from "zod"`,
				`import { ZodError } from "zod"`,
			},
			expected: `import { ZodError, z } from "zod"`,
		},
		{
			name: "multiple sources",
			imports: []string{
				`import { z } from "zod"`,
				`import { foo } from "bar"`,
			},
			expected: "import { foo } from \"bar\"\nimport { z } from \"zod\"",
		},
		{
			name: "default import",
			imports: []string{
				`import React from "react"`,
			},
			expected: `import React from "react"`,
		},
		{
			name: "namespace import",
			imports: []string{
				`import * as bridge from "@vectorfyco/valbridge"`,
			},
			expected: `import * as bridge from "@vectorfyco/valbridge"`,
		},
		{
			name: "namespace import with named from same source",
			imports: []string{
				`import * as bridge from "@vectorfyco/valbridge"`,
				`import { createValbridgeClient } from "@vectorfyco/valbridge"`,
			},
			expected: "import { createValbridgeClient } from \"@vectorfyco/valbridge\"\nimport * as bridge from \"@vectorfyco/valbridge\"",
		},
		{
			name: "type-only named import",
			imports: []string{
				`import type { Static } from "@sinclair/typebox"`,
			},
			expected: `import { type Static } from "@sinclair/typebox"`,
		},
		{
			name: "mixed regular and type named imports",
			imports: []string{
				`import { Type, type Static } from "@sinclair/typebox"`,
			},
			expected: `import { Type, type Static } from "@sinclair/typebox"`,
		},
		{
			name: "merge type-only with regular from same source",
			imports: []string{
				`import { Type } from "@sinclair/typebox"`,
				`import type { Static } from "@sinclair/typebox"`,
			},
			expected: `import { Type, type Static } from "@sinclair/typebox"`,
		},
		{
			name: "dedupe type imports",
			imports: []string{
				`import type { Static } from "@sinclair/typebox"`,
				`import type { Static } from "@sinclair/typebox"`,
			},
			expected: `import { type Static } from "@sinclair/typebox"`,
		},
		{
			name: "side-effect import",
			imports: []string{
				`import "reflect-metadata"`,
			},
			expected: `import "reflect-metadata"`,
		},
		{
			name: "side-effect with named",
			imports: []string{
				`import "reflect-metadata"`,
				`import { z } from "zod"`,
			},
			expected: "import \"reflect-metadata\"\nimport { z } from \"zod\"",
		},
		{
			name: "real-world valbridge package case",
			imports: []string{
				`import * as bridge from "@vectorfyco/valbridge"`,
				`import { createValbridgeClient } from "@vectorfyco/valbridge"`,
				`import * as bridge from "@vectorfyco/valbridge"`,
			},
			expected: "import { createValbridgeClient } from \"@vectorfyco/valbridge\"\nimport * as bridge from \"@vectorfyco/valbridge\"",
		},
		{
			name: "real-world typebox case",
			imports: []string{
				`import { Type, type Static } from "@sinclair/typebox"`,
				`import { Value } from "@sinclair/typebox/value"`,
			},
			expected: "import { Type, type Static } from \"@sinclair/typebox\"\nimport { Value } from \"@sinclair/typebox/value\"",
		},
		{
			name: "type named export",
			imports: []string{
				`import { type Diagnostic } from "@vectorfyco/valbridge-core"`,
			},
			expected: `import { type Diagnostic } from "@vectorfyco/valbridge-core"`,
		},
		{
			name: "aliased named import",
			imports: []string{
				`import { convert as convertSchema } from "@vectorfyco/valbridge-zod"`,
			},
			expected: `import { convert as convertSchema } from "@vectorfyco/valbridge-zod"`,
		},
		{
			name: "multiple namespace imports from different sources",
			imports: []string{
				`import * as bridge from "@vectorfyco/valbridge"`,
				`import * as z from "zod"`,
			},
			expected: "import * as bridge from \"@vectorfyco/valbridge\"\nimport * as z from \"zod\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeImports(tt.imports)
			if got != tt.expected {
				t.Errorf("MergeImports() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}
