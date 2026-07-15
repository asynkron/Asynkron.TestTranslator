package cobertura

import (
	"bytes"
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/coverage"
)

func sample(t *testing.T) *coverage.Report {
	t.Helper()
	f1 := coverage.File{Path: "pkg/foo.go"}
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(f1.AddLine(coverage.LineHit{Number: 1, Hits: 3}))
	must(f1.AddLine(coverage.LineHit{Number: 2, Hits: 0}))
	must(f1.AddLine(coverage.LineHit{Number: 3, Hits: 1, Branch: true, BranchesCovered: 1, BranchesTotal: 2}))
	f2 := coverage.File{Path: "pkg/sub/bar.go"}
	must(f2.AddLine(coverage.LineHit{Number: 10, Hits: 5}))
	r := &coverage.Report{Files: []coverage.File{f1, f2}, SourceRoots: []string{"."}}
	if err := r.Normalize(); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMarshalAndValidate(t *testing.T) {
	r := sample(t)
	out, err := Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "<!DOCTYPE coverage") {
		t.Error("missing Cobertura DOCTYPE")
	}
	for _, want := range []string{
		`lines-valid="4"`, `lines-covered="3"`,
		`branches-covered="1"`, `branches-valid="2"`,
		`filename="pkg/foo.go"`, `condition-coverage="50% (1/2)"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q\n%s", want, s)
		}
	}
	if err := Validate(bytes.NewReader(out)); err != nil {
		t.Fatalf("Validate rejected our own output: %v", err)
	}
}

func TestMarshalDeterministic(t *testing.T) {
	a, _ := Marshal(sample(t))
	b, _ := Marshal(sample(t))
	if !bytes.Equal(a, b) {
		t.Fatal("Cobertura output is not deterministic")
	}
}

func TestValidateRejectsBadRoot(t *testing.T) {
	if err := Validate(strings.NewReader(`<testsuites/>`)); err == nil {
		t.Fatal("expected error for wrong root")
	}
}

func TestValidateRejectsBadRate(t *testing.T) {
	bad := `<?xml version="1.0"?><coverage line-rate="9.9"><packages/></coverage>`
	if err := Validate(strings.NewReader(bad)); err == nil {
		t.Fatal("expected error for out-of-range line-rate")
	}
}
