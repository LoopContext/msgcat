package plural

import "testing"

func TestForm(t *testing.T) {
	tests := []struct {
		lang  string
		count int
		want  string
	}{
		{"en", 0, "other"},
		{"en", 1, "one"},
		{"en", 2, "other"},
		{"en-US", 1, "one"},
		{"es", 1, "one"},
		{"es", 5, "other"},
		{"ar", 0, "zero"},
		{"ar", 1, "one"},
		{"ar", 2, "two"},
		{"ar", 5, "few"},
		{"ar", 11, "many"},
		{"ar", 100, "other"},
		{"ru", 1, "one"},
		{"ru", 2, "few"},
		{"ru", 5, "many"},
		{"ru", 21, "one"},
		{"ru", 11, "many"},
		{"pl", 1, "one"},
		{"pl", 2, "few"},
		{"pl", 5, "many"},
		{"pl", 12, "many"},
		{"cy", 0, "zero"},
		{"cy", 1, "one"},
		{"cy", 2, "two"},
		{"cy", 3, "few"},
		{"cy", 6, "many"},
		{"unknown", 1, "other"},
		{"unknown", 99, "other"},
	}
	for _, tt := range tests {
		got := Form(tt.lang, tt.count)
		if got != tt.want {
			t.Errorf("Form(%q, %d) = %q, want %q", tt.lang, tt.count, got, tt.want)
		}
	}
}
