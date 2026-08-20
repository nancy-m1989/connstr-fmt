package connstr

import "strings"

// Format renders the connection string in a normalized form: pairs in their
// original order, "; " between entries, and values double-quoted only when
// left unquoted they would be ambiguous or lossy (they contain ';' or '"',
// or have leading/trailing whitespace).
func (cs *ConnectionString) Format() string {
	var b strings.Builder
	for i, p := range cs.Pairs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(p.Key)
		b.WriteByte('=')
		b.WriteString(formatValue(p.Value))
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
