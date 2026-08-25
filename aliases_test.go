package connstr

import "testing"

func TestCanonicalKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"Server", "server"},
		{"Data Source", "server"},
		{"data  source", "server"},
		{"DATA SOURCE", "server"},
		{"Uid", "uid"},
		{"User Id", "uid"},
		{"user   id", "uid"},
		{"Database", "database"},
		{"Password", "password"},
	}

	for _, tt := range tests {
		if got := canonicalKey(tt.key); got != tt.want {
			t.Errorf("canonicalKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
