package connstr

import "strings"

// LooksLikeURI reports whether input has the scheme:// prefix that marks a
// URI-style connection string (postgres://, mysql://, and similar), as
// opposed to the key=value form Parse expects. Callers that need to accept
// either form can use it to pick which parser to call.
func LooksLikeURI(input string) bool {
	i := strings.Index(input, "://")
	if i <= 0 {
		return false
	}
	for _, r := range input[:i] {
		if !isSchemeRune(r) {
			return false
		}
	}
	return true
}

// ParseAny parses input as either form of connection string, using
// LooksLikeURI to decide whether to call Parse or ParseURI. It's the entry
// point for callers that accept both forms and don't want to duplicate that
// check themselves.
func ParseAny(input string) (*ConnectionString, error) {
	if LooksLikeURI(input) {
		return ParseURI(input)
	}
	return Parse(input)
}

// ParseURI parses a URI-style connection string:
//
//	scheme://[user[:password]@]host[:port][/database][?key=value&...]
//
// as used by postgres:// and mysql:// connection strings, among others.
// The result uses the same Pair/ConnectionString shape as Parse, under the
// keys "scheme", "user", "password", "host", "port", and "database", plus
// one pair per query parameter, so Get and Format work the same way
// regardless of which form the input used. The userinfo, path, and query
// components are percent-decoded.
func ParseURI(input string) (*ConnectionString, error) {
	p := &uriParser{
		sc:        newScanner(input),
		seen:      make(map[string]Position),
		srcErrors: newSrcErrors(input),
	}
	return p.run()
}

func isSchemeRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '+' || r == '-' || r == '.':
		return true
	}
	return false
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

type uriParser struct {
	sc   *scanner
	seen map[string]Position
	srcErrors
}

func (p *uriParser) run() (*ConnectionString, error) {
	cs := &ConnectionString{}
	start := p.sc.position()

	scheme, err := p.parseScheme()
	if err != nil {
		return nil, err
	}
	if err := p.addPair(cs, "scheme", scheme, start, start); err != nil {
		return nil, err
	}

	userinfo, userinfoPos, hostport, hostportPos, err := p.parseAuthority()
	if err != nil {
		return nil, err
	}
	if err := p.addUserinfo(cs, userinfo, userinfoPos); err != nil {
		return nil, err
	}
	if err := p.addHostPort(cs, hostport, hostportPos); err != nil {
		return nil, err
	}
	if err := p.parsePathAndQuery(cs); err != nil {
		return nil, err
	}

	return cs, nil
}

// addPair records a pair under key, rejecting it as a duplicate if a pair
// with the same canonical key (following the same alias rules Parse uses)
// was already added.
func (p *uriParser) addPair(cs *ConnectionString, key, value string, keyPos, valPos Position) error {
	canon := canonicalKey(key)
	if first, dup := p.seen[canon]; dup {
		return p.errorf(keyPos, "duplicate key %q (first set at %s)", key, first)
	}
	p.seen[canon] = keyPos
	cs.Pairs = append(cs.Pairs, Pair{Key: key, Value: value, KeyPos: keyPos, ValPos: valPos})
	return nil
}

// parseScheme reads the scheme up to and including the "://" that
// introduces the authority.
func (p *uriParser) parseScheme() (string, error) {
	start := p.sc.position()
	var runes []rune
	for {
		r, ok := p.sc.peek()
		if !ok || r == ':' {
			break
		}
		if !isSchemeRune(r) {
			return "", p.errorf(p.sc.position(), "invalid character %q in scheme", r)
		}
		p.sc.advance()
		runes = append(runes, r)
	}
	if len(runes) == 0 {
		return "", p.errorf(start, "missing scheme")
	}
	for _, want := range []rune{':', '/', '/'} {
		r, ok := p.sc.advance()
		if !ok || r != want {
			return "", p.errorf(p.sc.position(), "missing '://' after scheme %q", string(runes))
		}
	}
	return string(runes), nil
}

// parseAuthority reads everything between the "://" and the next '/' or
// '?' (or the end of input), and splits it into a userinfo part and a
// host[:port] part on the last '@'.
func (p *uriParser) parseAuthority() (userinfo string, userinfoPos Position, hostport string, hostportPos Position, err error) {
	pos := p.sc.position()
	var runes []rune
	for {
		r, ok := p.sc.peek()
		if !ok || r == '/' || r == '?' {
			break
		}
		p.sc.advance()
		runes = append(runes, r)
	}
	raw := string(runes)
	if raw == "" {
		return "", Position{}, "", Position{}, p.errorf(pos, "missing host after scheme")
	}
	if at := strings.LastIndexByte(raw, '@'); at >= 0 {
		userRunes := []rune(raw[:at])
		return raw[:at], pos, raw[at+1:], addCol(pos, len(userRunes)+1), nil
	}
	return "", Position{}, raw, pos, nil
}

// addUserinfo splits raw (the part of the authority before '@', if any) on
// the first ':' into a user and an optional password, percent-decodes
// each, and records them.
func (p *uriParser) addUserinfo(cs *ConnectionString, raw string, pos Position) error {
	if raw == "" {
		return nil
	}

	userRaw, passRaw, hasPass := raw, "", false
	if c := strings.IndexByte(raw, ':'); c >= 0 {
		userRaw, passRaw, hasPass = raw[:c], raw[c+1:], true
	}

	user, err := p.decodePercent(userRaw, pos)
	if err != nil {
		return err
	}
	if err := p.addPair(cs, "user", user, pos, pos); err != nil {
		return err
	}

	if hasPass {
		passPos := addCol(pos, len([]rune(userRaw))+1)
		pass, err := p.decodePercent(passRaw, passPos)
		if err != nil {
			return err
		}
		if err := p.addPair(cs, "password", pass, passPos, passPos); err != nil {
			return err
		}
	}
	return nil
}

// addHostPort splits raw (the part of the authority after '@', or the
// whole authority if there was no userinfo) into a host and an optional
// numeric port, taken from after the last ':'.
func (p *uriParser) addHostPort(cs *ConnectionString, raw string, pos Position) error {
	if raw == "" {
		return p.errorf(pos, "missing host")
	}

	host, port := raw, ""
	if c := strings.LastIndexByte(raw, ':'); c >= 0 {
		candidate := raw[c+1:]
		if candidate != "" && isAllDigits(candidate) {
			host, port = raw[:c], candidate
		}
	}
	if host == "" {
		return p.errorf(pos, "missing host")
	}

	if err := p.addPair(cs, "host", host, pos, pos); err != nil {
		return err
	}
	if port != "" {
		portPos := addCol(pos, len([]rune(host))+1)
		if err := p.addPair(cs, "port", port, portPos, portPos); err != nil {
			return err
		}
	}
	return nil
}

// parsePathAndQuery consumes an optional "/database" and an optional
// "?key=value&..." query string, in that order, which is everything that
// can remain after the authority.
func (p *uriParser) parsePathAndQuery(cs *ConnectionString) error {
	if r, ok := p.sc.peek(); ok && r == '/' {
		p.sc.advance()
		pos := p.sc.position()
		var runes []rune
		for {
			r, ok := p.sc.peek()
			if !ok || r == '?' {
				break
			}
			p.sc.advance()
			runes = append(runes, r)
		}
		if len(runes) > 0 {
			db, err := p.decodePercent(string(runes), pos)
			if err != nil {
				return err
			}
			if err := p.addPair(cs, "database", db, pos, pos); err != nil {
				return err
			}
		}
	}

	if r, ok := p.sc.peek(); ok && r == '?' {
		p.sc.advance()
		return p.parseQuery(cs)
	}
	return nil
}

func (p *uriParser) parseQuery(cs *ConnectionString) error {
	for {
		if _, ok := p.sc.peek(); !ok {
			return nil
		}

		keyPos := p.sc.position()
		var keyRunes []rune
		for {
			r, ok := p.sc.peek()
			if !ok || r == '=' || r == '&' {
				break
			}
			p.sc.advance()
			keyRunes = append(keyRunes, r)
		}
		key, err := p.decodePercent(string(keyRunes), keyPos)
		if err != nil {
			return err
		}

		var value string
		valPos := p.sc.position()
		if r, ok := p.sc.peek(); ok && r == '=' {
			p.sc.advance()
			valPos = p.sc.position()
			var valRunes []rune
			for {
				r, ok := p.sc.peek()
				if !ok || r == '&' {
					break
				}
				p.sc.advance()
				valRunes = append(valRunes, r)
			}
			value, err = p.decodePercent(string(valRunes), valPos)
			if err != nil {
				return err
			}
		}

		if key == "" {
			return p.errorf(keyPos, "empty query key")
		}
		if err := p.addPair(cs, key, value, keyPos, valPos); err != nil {
			return err
		}

		r, ok := p.sc.peek()
		if !ok {
			return nil
		}
		if r == '&' {
			p.sc.advance()
			continue
		}
		return p.errorf(p.sc.position(), "unexpected character %q in query string", r)
	}
}

// decodePercent percent-decodes raw, which must come from a single line of
// input starting at startPos, so that a malformed escape can be reported at
// its exact column.
func (p *uriParser) decodePercent(raw string, startPos Position) (string, error) {
	runes := []rune(raw)
	var out strings.Builder
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r != '%' {
			out.WriteRune(r)
			continue
		}
		pos := addCol(startPos, i)
		if i+2 >= len(runes) {
			return "", p.errorf(pos, "incomplete percent-encoding")
		}
		hi, ok1 := hexVal(runes[i+1])
		lo, ok2 := hexVal(runes[i+2])
		if !ok1 || !ok2 {
			return "", p.errorf(pos, "invalid percent-encoding %q", string(runes[i:i+3]))
		}
		out.WriteByte(byte(hi<<4 | lo))
		i += 2
	}
	return out.String(), nil
}

func hexVal(r rune) (int, bool) {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0'), true
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10, true
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10, true
	}
	return 0, false
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if !isASCIIDigit(r) {
			return false
		}
	}
	return true
}

// addCol offsets pos by n columns. It assumes pos and the offset both fall
// on the same source line, which holds for every caller here: URI
// components never contain a literal, unencoded newline.
func addCol(pos Position, n int) Position {
	return Position{Line: pos.Line, Col: pos.Col + n}
}
