package connstr

import "strings"

// Format renders the connection string in a normalized form: pairs in their
// original order, "; " between entries, and values double-quoted only when
// left unquoted they would be ambiguous or lossy (they contain ';' or '"',
// or have leading/trailing whitespace).
func (cs *ConnectionString) Format() string {
	return cs.render(func(p Pair) string {
		return formatValue(p.Value)
	})
}

// render is shared by Format and Mask: both print every pair as
// "Key=Value" joined by "; ", differing only in how a pair's value is
// turned into text.
func (cs *ConnectionString) render(value func(Pair) string) string {
	var b strings.Builder
	for i, p := range cs.Pairs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(p.Key)
		b.WriteByte('=')
		b.WriteString(value(p))
	}
	return b.String()
}

func formatValue(v string) string {
	if !needsQuoting(v) {
		return v
	}
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func needsQuoting(v string) bool {
	if v == "" {
		return false
	}
	if strings.ContainsAny(v, ";\"") {
		return true
	}
	return strings.TrimSpace(v) != v
}
