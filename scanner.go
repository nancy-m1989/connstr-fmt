package connstr

// scanner walks the input rune by rune, tracking a 1-based line and column
// so parse errors can point at an exact spot in the original text.
type scanner struct {
	runes []rune
	pos   int
	line  int
	col   int
}

func newScanner(input string) *scanner {
	return &scanner{runes: []rune(input), line: 1, col: 1}
}

func (s *scanner) peek() (rune, bool) {
	if s.pos >= len(s.runes) {
		return 0, false
	}
	return s.runes[s.pos], true
}

func (s *scanner) advance() (rune, bool) {
	r, ok := s.peek()
	if !ok {
		return 0, false
	}
	s.pos++
	if r == '\n' {
		s.line++
		s.col = 1
	} else {
		s.col++
	}
	return r, true
}

func (s *scanner) position() Position {
	return Position{Line: s.line, Col: s.col}
}
