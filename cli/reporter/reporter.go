package reporter

import (
	"fmt"

	"github.com/vectorfy-co/valbridge/adapter"
	"github.com/vectorfy-co/valbridge/ui"
)

type SchemaDiagnostics struct {
	Key         string
	Diagnostics []adapter.Diagnostic
}

type Options struct {
	Quiet bool
}

type Summary struct {
	Errors   int
	Warnings int
	Info     int
}

func Report(groups []SchemaDiagnostics, opts Options) Summary {
	summary := Summary{}

	for _, group := range groups {
		filtered := make([]adapter.Diagnostic, 0, len(group.Diagnostics))
		for _, diagnostic := range group.Diagnostics {
			switch diagnostic.Severity {
			case "error":
				summary.Errors++
			case "warning":
				summary.Warnings++
			case "info":
				summary.Info++
				if opts.Quiet {
					continue
				}
			default:
				if opts.Quiet {
					continue
				}
			}
			filtered = append(filtered, diagnostic)
		}

		if len(filtered) == 0 {
			continue
		}

		ui.Println()
		ui.Printf("%s %s\n", ui.Bold.Render("Diagnostics for"), ui.Primary.Render(group.Key))
		for _, diagnostic := range filtered {
			line := fmt.Sprintf("  [%s] %s: %s", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
			if diagnostic.Suggestion != "" {
				line += fmt.Sprintf(" (%s)", diagnostic.Suggestion)
			}
			ui.Println(line)
		}
	}

	return summary
}
