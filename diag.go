package connstr

import (
	"fmt"
	"strings"
)

// srcErrors renders *ParseError values against the original source lines.
// The key=value parser and the URI parser share it so error output looks
// the same regardless of which grammar rejected the input.
type srcErrors struct {
	lines []string
}

func newSrcErrors(input string) srcErrors {
	return srcErrors{lines: splitLines(input)}
}

func splitLines(input string) []string {
	lines := strings.Split(input, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSuffix(l, "\r")
	}
	return lines
}

func (s srcErrors) errorf(pos Position, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Pos:     pos,
		Msg:     fmt.Sprintf(format, args...),
		snippet: s.snippet(pos),
	}
}

// snippet renders the source line pos occurred on, with a caret under the
// exact column.
func (s srcErrors) snippet(pos Position) string {
	idx := pos.Line - 1
	if idx < 0 || idx >= len(s.lines) {
		return ""
	}
	line := s.lines[idx]
	col := pos.Col - 1
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	return line + "\n" + strings.Repeat(" ", col) + "^"
}
