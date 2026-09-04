package connstr

import (
	"strings"
	"testing"
)

func TestLooksLikeURI(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"postgres://localhost/db", true},
		{"mysql://root@localhost:3306/app", true},
		{"a+b-c.d://x", true},
		{"Server=localhost;Database=app", false},
		{"://localhost", false},
		{"not a uri", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := LooksLikeURI(tt.input); got != tt.want {
			t.Errorf("LooksLikeURI(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseURISuccess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Pair
	}{
		{
			name:  "scheme and host only",
			input: "postgres://localhost",
			want: []Pair{
				{Key: "scheme", Value: "postgres"},
				{Key: "host", Value: "localhost"},
			},
		},
		{
			name:  "host, port, and database",
			input: "postgres://localhost:5432/mydb",
			want: []Pair{
				{Key: "scheme", Value: "postgres"},
				{Key: "host", Value: "localhost"},
				{Key: "port", Value: "5432"},
				{Key: "database", Value: "mydb"},
			},
		},
		{
			name:  "user with no password",
			input: "postgres://alice@localhost/mydb",
			want: []Pair{
				{Key: "scheme", Value: "postgres"},
				{Key: "user", Value: "alice"},
				{Key: "host", Value: "localhost"},
				{Key: "database", Value: "mydb"},
			},
		},
		{
			name:  "user, password, host, port, database, and query",
			input: "postgres://alice:s3cret@localhost:5432/mydb?sslmode=disable&timeout=5",
			want: []Pair{
				{Key: "scheme", Value: "postgres"},
				{Key: "user", Value: "alice"},
				{Key: "password", Value: "s3cret"},
				{Key: "host", Value: "localhost"},
				{Key: "port", Value: "5432"},
				{Key: "database", Value: "mydb"},
				{Key: "sslmode", Value: "disable"},
				{Key: "timeout", Value: "5"},
			},
		},
		{
			name:  "percent-encoded password and query value",
			input: "postgres://alice:p%40ss@localhost/db?name=a%20b",
			want: []Pair{
				{Key: "scheme", Value: "postgres"},
				{Key: "user", Value: "alice"},
				{Key: "password", Value: "p@ss"},
				{Key: "host", Value: "localhost"},
				{Key: "database", Value: "db"},
				{Key: "name", Value: "a b"},
			},
		},
		{
			name:  "mysql scheme with numeric host and port",
			input: "mysql://root@127.0.0.1:3306/app",
			want: []Pair{
				{Key: "scheme", Value: "mysql"},
				{Key: "user", Value: "root"},
				{Key: "host", Value: "127.0.0.1"},
				{Key: "port", Value: "3306"},
				{Key: "database", Value: "app"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := ParseURI(tt.input)
			if err != nil {
				t.Fatalf("ParseURI(%q) returned error: %v", tt.input, err)
			}
			if len(cs.Pairs) != len(tt.want) {
				t.Fatalf("ParseURI(%q) got %d pairs, want %d: %+v", tt.input, len(cs.Pairs), len(tt.want), cs.Pairs)
			}
			for i, p := range cs.Pairs {
				if p.Key != tt.want[i].Key || p.Value != tt.want[i].Value {
					t.Errorf("pair %d = {%q, %q}, want {%q, %q}", i, p.Key, p.Value, tt.want[i].Key, tt.want[i].Value)
				}
			}
		})
	}
}

func TestParseURIErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
		wantPos Position
	}{
		{
			name:    "missing scheme",
			input:   "://localhost",
			wantMsg: "missing scheme",
			wantPos: Position{Line: 1, Col: 1},
		},
		{
			name:    "invalid character in scheme",
			input:   "post gres://x",
			wantMsg: `invalid character ' ' in scheme`,
			wantPos: Position{Line: 1, Col: 5},
		},
		{
			name:    "missing :// after scheme",
			input:   "postgres:localhost",
			wantMsg: `missing '://' after scheme "postgres"`,
			wantPos: Position{Line: 1, Col: 11},
		},
		{
			name:    "missing host",
			input:   "postgres:///db",
			wantMsg: "missing host after scheme",
			wantPos: Position{Line: 1, Col: 12},
		},
		{
			name:    "duplicate key between host and query",
			input:   "postgres://localhost?host=other",
			wantMsg: `duplicate key "host" (first set at line 1, column 12)`,
			wantPos: Position{Line: 1, Col: 22},
		},
		{
			name:    "incomplete percent-encoding",
			input:   "postgres://localhost/db?x=%2",
			wantMsg: "incomplete percent-encoding",
			wantPos: Position{Line: 1, Col: 27},
		},
		{
			name:    "invalid percent-encoding",
			input:   "postgres://localhost/db?x=%zz",
			wantMsg: `invalid percent-encoding "%zz"`,
			wantPos: Position{Line: 1, Col: 27},
		},
		{
			name:    "unexpected character in query string",
			input:   "postgres://localhost?a=1;b=2",
			wantMsg: `unexpected character ';' in query string`,
			wantPos: Position{Line: 1, Col: 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseURI(tt.input)
			if err == nil {
				t.Fatalf("ParseURI(%q) succeeded, want error", tt.input)
			}
			perr, ok := err.(*ParseError)
			if !ok {
				t.Fatalf("ParseURI(%q) returned %T, want *ParseError", tt.input, err)
			}
			if perr.Pos != tt.wantPos {
				t.Errorf("ParseURI(%q) error position = %+v, want %+v", tt.input, perr.Pos, tt.wantPos)
			}
			if !strings.Contains(perr.Error(), tt.wantMsg) {
				t.Errorf("ParseURI(%q) error = %q, want to contain %q", tt.input, perr.Error(), tt.wantMsg)
			}
		})
	}
}

func TestParseURIThenFormat(t *testing.T) {
	cs, err := ParseURI("postgres://alice:s3cret@localhost:5432/mydb?sslmode=disable")
	if err != nil {
		t.Fatalf("ParseURI: %v", err)
	}
	want := `scheme=postgres; user=alice; password=s3cret; host=localhost; port=5432; database=mydb; sslmode=disable`
	if got := cs.Format(); got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
	if v, ok := cs.Get("host"); !ok || v != "localhost" {
		t.Errorf(`Get("host") = %q, %v, want "localhost", true`, v, ok)
	}
}
