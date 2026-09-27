package cli

import (
	"fmt"
	"strings"
)

// Newtonsoft.Json-compatible output for the v1 console's -listParams json and
// -getTemplateInfo json: PascalCase keys in declaration order, nulls written
// out, two-space indent, `"Key": value`, and Newtonsoft's default escaping.
// Scripts written against the C# console parse this byte-for-byte the same.

type netField struct {
	key   string
	value *string // nil → null
}

func netObject(fields []netField, indent, nl string) string {
	if len(fields) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{")
	for i, f := range fields {
		b.WriteString(nl + indent + "  " + netString(f.key) + ": ")
		if f.value == nil {
			b.WriteString("null")
		} else {
			b.WriteString(netString(*f.value))
		}
		if i < len(fields)-1 {
			b.WriteString(",")
		}
	}
	b.WriteString(nl + indent + "}")
	return b.String()
}

func netArray(objects [][]netField, nl string) string {
	if len(objects) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteString("[")
	for i, o := range objects {
		b.WriteString(nl + "  " + netObject(o, "  ", nl))
		if i < len(objects)-1 {
			b.WriteString(",")
		}
	}
	b.WriteString(nl + "]")
	return b.String()
}

// netString quotes s the way Newtonsoft's default StringEscapeHandling does:
// quote, backslash and control characters escaped, plus U+0085, U+2028 and
// U+2029; everything else (non-ASCII included) written as-is.
func netString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\u0085', ' ', ' ':
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
