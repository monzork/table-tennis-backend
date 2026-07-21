package i18n

import (
	"testing"
)

func TestT(t *testing.T) {
	tests := []struct {
		name     string
		locale   string
		key      string
		expected string
	}{
		{
			name:     "English translation",
			locale:   "en",
			key:      "nav.singles",
			expected: "Singles",
		},
		{
			name:     "Spanish translation",
			locale:   "es",
			key:      "nav.singles",
			expected: "Individuales",
		},
		{
			name:     "Empty locale defaults to English",
			locale:   "",
			key:      "nav.singles",
			expected: "Singles",
		},
		{
			name:     "Non-existent key returns key itself",
			locale:   "en",
			key:      "unknown.key.foo",
			expected: "unknown.key.foo",
		},
		{
			name:     "Non-existent locale falls back to English",
			locale:   "fr",
			key:      "nav.singles",
			expected: "Singles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := T(tt.locale, tt.key)
			if got != tt.expected {
				t.Errorf("T(%q, %q) = %q; want %q", tt.locale, tt.key, got, tt.expected)
			}
		})
	}
}

func TestT_FallbackToEnglish(t *testing.T) {
	Translations["en"]["test.only_english_key"] = "Only English Value"
	defer delete(Translations["en"], "test.only_english_key")

	got := T("es", "test.only_english_key")
	if got != "Only English Value" {
		t.Errorf("expected fallback to English 'Only English Value', got %q", got)
	}
}

func TestPrecomputedMaps(t *testing.T) {
	if PrecomputedMaps == nil {
		t.Fatal("PrecomputedMaps is nil")
	}

	for _, lang := range []string{"en", "es"} {
		m, ok := PrecomputedMaps[lang]
		if !ok {
			t.Errorf("PrecomputedMaps missing locale %q", lang)
			continue
		}

		if len(m) == 0 {
			t.Errorf("PrecomputedMaps[%q] is empty", lang)
		}

		if val, exists := m["nav.singles"]; !exists || val == "" {
			t.Errorf("PrecomputedMaps[%q] missing 'nav.singles'", lang)
		}
	}
}
