// Package connstr parses and formats ADO.NET/ODBC-style connection strings:
// semicolon-separated key=value pairs, with values that are either bare or
// wrapped in a matching pair of single or double quotes. Parse errors report
// the exact line and column of the problem, with a snippet of the offending
// line.
package connstr

import (
	"fmt"
	"strings"
)

// Position identifies a location in the original input by 1-based line and
// column.
type Position struct {
	Line int
	Col  int
}

func (p Position) String() string {
	return fmt.Sprintf("line %d, column %d", p.Line, p.Col)
}

// ParseError describes exactly where and why parsing failed.
type ParseError struct {
	Pos     Position
	Msg     string
	snippet string
}

func (e *ParseError) Error() string {
	if e.snippet == "" {
		return fmt.Sprintf("%s: %s", e.Pos, e.Msg)
	}
	return fmt.Sprintf("%s: %s\n%s", e.Pos, e.Msg, e.snippet)
}

// Pair is a single key/value entry from a connection string, in the order it
// appeared in the input.
type Pair struct {
	Key    string
	Value  string
	KeyPos Position
	ValPos Position
}

// ConnectionString is the parsed, validated form of a connection string.
type ConnectionString struct {
	Pairs []Pair
}

// Get looks up a value by key, case-insensitively and treating known
// aliases (such as "Server" and "Data Source") as equivalent, returning the
// first match.
func (cs *ConnectionString) Get(key string) (string, bool) {
	target := canonicalKey(key)
	for _, p := range cs.Pairs {
		if canonicalKey(p.Key) == target {
			return p.Value, true
		}
	}
	return "", false
}

type parser struct {
	sc   *scanner
	seen map[string]Position
	srcErrors

	// collectErrors puts the parser in Validate mode: instead of stopping at
	// the first error, it records the error and resynchronizes at the next
	// pair boundary so it can keep looking for more problems.
	collectErrors bool
	errs          []*ParseError
}

// Parse validates and parses a connection string, returning a *ParseError on
// the first problem it finds.
func Parse(input string) (*ConnectionString, error) {
	p := &parser{
		sc:        newScanner(input),
		seen:      make(map[string]Position),
		srcErrors: newSrcErrors(input),
	}
	return p.run()
}

// Validate checks a connection string the same way Parse does, but keeps
// going after a recoverable error instead of stopping at the first one, so
// it can report every problem in the input in a single pass. It returns nil
// if the input is valid.
func Validate(input string) []*ParseError {
	p := &parser{
		sc:            newScanner(input),
		seen:          make(map[string]Position),
		srcErrors:     newSrcErrors(input),
		collectErrors: true,
	}
	p.run()
	return p.errs
}

func isInsignificantWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

// skipWhitespace skips whitespace between pairs, including newlines, so a
// connection string may be spread across multiple lines.
func (p *parser) skipWhitespace() {
	for {
		r, ok := p.sc.peek()
		if !ok || !isInsignificantWhitespace(r) {
			return
		}
		p.sc.advance()
	}
}

// skipInlineWhitespace skips spaces and tabs only; a bare value ends at the
// end of its line, so a newline here is significant.
func (p *parser) skipInlineWhitespace() {
	for {
		r, ok := p.sc.peek()
		if !ok || (r != ' ' && r != '\t') {
			return
		}
		p.sc.advance()
	}
}

func (p *parser) run() (*ConnectionString, error) {
	cs := &ConnectionString{}

	for {
		p.skipWhitespace()
		if _, ok := p.sc.peek(); !ok {
			break
		}

		pair, err := p.parsePair()
		if err != nil {
			if p.recordParseError(err) {
				return nil, err
			}
			continue
		}
		if pair == nil {
			// a stray ';' with nothing before it: an empty segment, skipped.
			continue
		}

		canon := canonicalKey(pair.Key)
		if first, dup := p.seen[canon]; dup {
			dupErr := p.errorf(pair.KeyPos, "duplicate key %q (first set at %s)", pair.Key, first)
			if p.recordSemanticError(dupErr) {
				return nil, dupErr
			}
			continue
		}
		p.seen[canon] = pair.KeyPos

		cs.Pairs = append(cs.Pairs, *pair)
	}

	return cs, nil
}

// recordParseError handles an error raised while parsing a single pair. In
// Parse mode it reports that the caller should return the error as-is; in
// Validate mode it records the error and skips ahead to the next pair
// boundary so parsing can continue.
func (p *parser) recordParseError(err *ParseError) (stop bool) {
	if !p.collectErrors {
		return true
	}
	p.errs = append(p.errs, err)
	p.resync()
	return false
}

// recordSemanticError handles an error found after a pair was parsed
// successfully, such as a duplicate key. The scanner is already positioned
// at the next pair boundary, so unlike recordParseError this does not
// resynchronize.
func (p *parser) recordSemanticError(err *ParseError) (stop bool) {
	if !p.collectErrors {
		return true
	}
	p.errs = append(p.errs, err)
	return false
}

// resync skips ahead to the start of what looks like the next pair, so a
// single malformed segment doesn't prevent Validate from reporting problems
// in the rest of the input. It is a heuristic, not a full parse: it just
// looks for the next ';', ignoring quoting.
func (p *parser) resync() {
	if p.sc.pos > 0 && p.sc.runes[p.sc.pos-1] == ';' {
		return // already sitting at the start of the next pair
	}
	for {
		r, ok := p.sc.advance()
		if !ok || r == ';' {
			return
		}
	}
}

func (p *parser) parsePair() (*Pair, error) {
	keyPos := p.sc.position()
	var keyRunes []rune

	for {
		r, ok := p.sc.peek()
		if !ok || r == '=' || r == ';' {
			break
		}
		p.sc.advance()
		keyRunes = append(keyRunes, r)
	}

	key := strings.TrimSpace(string(keyRunes))

	sep, ok := p.sc.peek()
	if !ok {
		if key == "" {
			return nil, nil
		}
		return nil, p.errorf(p.sc.position(), "expected '=' after key %q, reached end of input", key)
	}

	if sep == ';' {
		sepPos := p.sc.position()
		p.sc.advance()
		if key == "" {
			return nil, nil
		}
		return nil, p.errorf(sepPos, "expected '=' after key %q, found ';'", key)
	}

	// sep == '='
	p.sc.advance()
	if key == "" {
		return nil, p.errorf(keyPos, "empty key")
	}

	value, valPos, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if r, ok := p.sc.peek(); ok {
		switch r {
		case ';':
			p.sc.advance()
		case '\n':
			// left for skipWhitespace on the next loop iteration.
		default:
			return nil, p.errorf(p.sc.position(), "unexpected character %q after value", r)
		}
	}

	return &Pair{Key: key, Value: value, KeyPos: keyPos, ValPos: valPos}, nil
}

func (p *parser) parseValue() (string, Position, error) {
	p.skipInlineWhitespace()
	valPos := p.sc.position()

	if r, ok := p.sc.peek(); ok && (r == '\'' || r == '"') {
		val, err := p.parseQuotedValue(r, valPos)
		return val, valPos, err
	}

	var runes []rune
	for {
		r, ok := p.sc.peek()
		if !ok || r == ';' || r == '\n' {
			break
		}
		p.sc.advance()
		runes = append(runes, r)
	}

	return strings.TrimRight(string(runes), " \t\r"), valPos, nil
}

func (p *parser) parseQuotedValue(quote rune, openPos Position) (string, error) {
	p.sc.advance() // consume the opening quote

	var runes []rune
	for {
		r, ok := p.sc.advance()
		if !ok {
			return "", p.errorf(openPos, "quoted value is never closed")
		}
		if r == quote {
			if next, ok := p.sc.peek(); ok && next == quote {
				// a doubled quote character is an escaped literal.
				p.sc.advance()
				runes = append(runes, quote)
				continue
			}
			break
		}
		runes = append(runes, r)
	}

	p.skipInlineWhitespace()
	return string(runes), nil
}
