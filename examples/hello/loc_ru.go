// Code generated by go-l10n; DO NOT EDIT.

package l10n

import "strings"

type ru_Localizer struct{}

func (ru_l ru_Localizer) Hello(name string) string {
	var b0 strings.Builder

	b0.WriteString("Привет, ")
	b0.WriteString(name)
	b0.WriteString("!")

	return b0.String()
}