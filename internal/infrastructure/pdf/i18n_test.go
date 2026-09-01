package pdf

import "testing"

func TestL(t *testing.T) {
	cases := []struct {
		lang string
		en   string
		want string
	}{
		{"es", "NAME", "NOMBRE"},
		{"en", "NAME", "NAME"},
		{"", "NAME", "NAME"},
		{"es", "Sets", "Sets"}, // no translation entry -> passthrough
		{"es", "unknown string", "unknown string"},
	}
	for _, c := range cases {
		if got := L(c.lang, c.en); got != c.want {
			t.Errorf("L(%q, %q) = %q, want %q", c.lang, c.en, got, c.want)
		}
	}
}
