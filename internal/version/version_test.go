package version

import "testing"

func TestNormalize(t *testing.T) {
	for input, want := range map[string]string{
		"v0.1.0": "0.1.0",
		"1.2.3":  "1.2.3",
		" v2.0 ": "2.0",
	} {
		if got := normalize(input); got != want {
			t.Errorf("normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCurrentUsesLinkerVersion(t *testing.T) {
	previous := buildVersion
	buildVersion = "v9.8.7"
	t.Cleanup(func() { buildVersion = previous })

	if got := Current(); got != "9.8.7" {
		t.Fatalf("Current() = %q, want 9.8.7", got)
	}
}
