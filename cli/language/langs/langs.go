package langs

import (
	"github.com/vectorfy-co/valbridge/language"
	_ "github.com/vectorfy-co/valbridge/language/langs/python"
	"github.com/vectorfy-co/valbridge/language/langs/typescript"
)

func RegisterBuiltins() error {
	if err := language.Register(typescript.Language()); err != nil {
		return err
	}
	return nil
}
