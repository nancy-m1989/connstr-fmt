package connstr

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare values pass through unquoted",
			input: "Server=localhost;Database=app",
			want:  "Server=localhost; Database=app",
		},
		{
			name:  "value with a semicolon is quoted",
			input: `Password="p@ss;w0rd"`,
			want:  `Password="p@ss;w0rd"`,
		},
		{
			name:  "embedded quote is doubled on output",
			input: `Password="a""b"`,
			want:  `Password="a""b"`,
		},
		{
			name:  "value with leading or trailing whitespace is quoted",
			input: `Value=' padded '`,
			want:  `Value=" padded "`,
		},
		{
			name:  "empty value is left bare",
			input: "Value=",
			want:  "Value=",
		},
		{
			name:  "spacing is normalized without reordering pairs",
			input: `Server = localhost ; Database=app ; Password = "p@ss;w0rd" `,
			want:  `Server=localhost; Database=app; Password="p@ss;w0rd"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.input, err)
			}
			if got := cs.Format(); got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"plain", false},
		{"no whitespace issue", false},
		{"has;semicolon", true},
		{`has"quote`, true},
		{" leading", true},
		{"trailing ", true},
	}

	for _, tt := range tests {
		if got := needsQuoting(tt.value); got != tt.want {
			t.Errorf("needsQuoting(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}
