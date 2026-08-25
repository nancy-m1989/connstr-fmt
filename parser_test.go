package connstr

import (
	"strings"
	"testing"
)

func TestParseSuccess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Pair
	}{
		{
			name:  "single pair",
			input: "Server=localhost",
			want: []Pair{
				{Key: "Server", Value: "localhost"},
			},
		},
		{
			name:  "multiple pairs",
			input: "Server=localhost;Database=app",
			want: []Pair{
				{Key: "Server", Value: "localhost"},
				{Key: "Database", Value: "app"},
			},
		},
		{
			name:  "whitespace around keys and values is trimmed",
			input: `Server = localhost ; Database=app ; Password = "p@ss;w0rd" `,
			want: []Pair{
				{Key: "Server", Value: "localhost"},
				{Key: "Database", Value: "app"},
				{Key: "Password", Value: "p@ss;w0rd"},
			},
		},
		{
			name:  "doubled quote is an escaped literal",
			input: `Password="a""b"`,
			want: []Pair{
				{Key: "Password", Value: `a"b`},
			},
		},
		{
			name:  "single-quoted value may contain a semicolon",
			input: `Password='a;b'`,
			want: []Pair{
				{Key: "Password", Value: "a;b"},
			},
		},
		{
			name:  "empty segments between semicolons are skipped",
			input: "Server=localhost;;Database=app",
			want: []Pair{
				{Key: "Server", Value: "localhost"},
				{Key: "Database", Value: "app"},
			},
		},
		{
			name:  "pairs may be split across lines",
			input: "Server=localhost\nDatabase=app",
			want: []Pair{
				{Key: "Server", Value: "localhost"},
				{Key: "Database", Value: "app"},
			},
		},
		{
			name:  "empty input yields no pairs",
			input: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.input, err)
			}
			if len(cs.Pairs) != len(tt.want) {
				t.Fatalf("Parse(%q) got %d pairs, want %d: %+v", tt.input, len(cs.Pairs), len(tt.want), cs.Pairs)
			}
			for i, p := range cs.Pairs {
				if p.Key != tt.want[i].Key || p.Value != tt.want[i].Value {
					t.Errorf("pair %d = {%q, %q}, want {%q, %q}", i, p.Key, p.Value, tt.want[i].Key, tt.want[i].Value)
				}
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
		wantPos Position
	}{
		{
			name:    "empty key",
			input:   "=foo",
			wantMsg: "empty key",
			wantPos: Position{Line: 1, Col: 1},
		},
		{
			name:    "key with no equals sign",
			input:   "Foo;Bar=1",
			wantMsg: `expected '=' after key "Foo", found ';'`,
			wantPos: Position{Line: 1, Col: 4},
		},
		{
			name:    "key with no equals sign at end of input",
			input:   "Foo",
			wantMsg: `expected '=' after key "Foo", reached end of input`,
			wantPos: Position{Line: 1, Col: 4},
		},
		{
			name:    "duplicate key",
			input:   "Foo=1;Foo=2",
			wantMsg: `duplicate key "Foo" (first set at line 1, column 1)`,
			wantPos: Position{Line: 1, Col: 7},
		},
		{
			name:    "duplicate key comparison is case-insensitive",
			input:   "Foo=1;foo=2",
			wantMsg: `duplicate key "foo" (first set at line 1, column 1)`,
			wantPos: Position{Line: 1, Col: 7},
		},
		{
			name:    "duplicate key comparison treats aliases as equal",
			input:   "Server=a;Data Source=b",
			wantMsg: `duplicate key "Data Source" (first set at line 1, column 1)`,
			wantPos: Position{Line: 1, Col: 10},
		},
		{
			name:    "unterminated quoted value",
			input:   `Password="oops`,
			wantMsg: "quoted value is never closed",
			wantPos: Position{Line: 1, Col: 10},
		},
		{
			name:    "stray character after closing quote",
			input:   `Key="a"b;`,
			wantMsg: `unexpected character 'b' after value`,
			wantPos: Position{Line: 1, Col: 8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", tt.input)
			}
			perr, ok := err.(*ParseError)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ParseError", tt.input, err)
			}
			if perr.Pos != tt.wantPos {
				t.Errorf("Parse(%q) error position = %+v, want %+v", tt.input, perr.Pos, tt.wantPos)
			}
			if !strings.Contains(perr.Error(), tt.wantMsg) {
				t.Errorf("Parse(%q) error = %q, want to contain %q", tt.input, perr.Error(), tt.wantMsg)
			}
		})
	}
}

func TestGet(t *testing.T) {
	cs, err := Parse("Server=localhost;Database=app")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if v, ok := cs.Get("server"); !ok || v != "localhost" {
		t.Errorf(`Get("server") = %q, %v, want "localhost", true`, v, ok)
	}
	if _, ok := cs.Get("Missing"); ok {
		t.Error(`Get("Missing") found a value, want not found`)
	}
}

func TestGetAliases(t *testing.T) {
	cs, err := Parse(`Data Source=localhost;User Id=me`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if v, ok := cs.Get("Server"); !ok || v != "localhost" {
		t.Errorf(`Get("Server") = %q, %v, want "localhost", true`, v, ok)
	}
	if v, ok := cs.Get("uid"); !ok || v != "me" {
		t.Errorf(`Get("uid") = %q, %v, want "me", true`, v, ok)
	}

	cs, err = Parse("Server=localhost;Uid=me")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if v, ok := cs.Get("Data Source"); !ok || v != "localhost" {
		t.Errorf(`Get("Data Source") = %q, %v, want "localhost", true`, v, ok)
	}
	if v, ok := cs.Get("User Id"); !ok || v != "me" {
		t.Errorf(`Get("User Id") = %q, %v, want "me", true`, v, ok)
	}
}
