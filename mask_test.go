package connstr

import "testing"

func TestMask(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "password is masked",
			input: "Server=localhost; Password=hunter2",
			want:  "Server=localhost; Password=****",
		},
		{
			name:  "pwd alias is masked the same as password",
			input: "Server=localhost; Pwd=hunter2",
			want:  "Server=localhost; Pwd=****",
		},
		{
			name:  "matching is case and spacing insensitive",
			input: "Server=localhost; CLIENT  SECRET=abc",
			want:  "Server=localhost; CLIENT  SECRET=****",
		},
		{
			name:  "non-secret values are formatted normally",
			input: "Server=localhost; Database=app",
			want:  "Server=localhost; Database=app",
		},
		{
			name:  "masking hides length differences",
			input: "Password=x; Database=app",
			want:  "Password=****; Database=app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.input, err)
			}
			if got := cs.Mask(); got != tt.want {
				t.Errorf("Mask() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaskURIPassword(t *testing.T) {
	cs, err := ParseURI("postgres://alice:hunter2@localhost:5432/mydb")
	if err != nil {
		t.Fatalf("ParseURI returned error: %v", err)
	}
	want := "scheme=postgres; user=alice; password=****; host=localhost; port=5432; database=mydb"
	if got := cs.Mask(); got != want {
		t.Errorf("Mask() = %q, want %q", got, want)
	}
}
