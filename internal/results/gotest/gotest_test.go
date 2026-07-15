package gotest

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
)

func parse(t *testing.T, stream string) (*results.Report, *diagnostics.Collector) {
	t.Helper()
	diag := diagnostics.NewCollector()
	rep, err := Adapter{}.Parse(strings.NewReader(stream), results.Options{SourceName: "test", Diag: diag})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return rep, diag
}

func findCase(r *results.Report, pkg, name string) *results.TestCase {
	for si := range r.Suites {
		if r.Suites[si].Name != pkg {
			continue
		}
		for ci := range r.Suites[si].Cases {
			if r.Suites[si].Cases[ci].Name == name {
				return &r.Suites[si].Cases[ci]
			}
		}
	}
	return nil
}

const passFailSkip = `
{"Action":"run","Package":"example/pkg","Test":"TestPass"}
{"Action":"output","Package":"example/pkg","Test":"TestPass","Output":"=== RUN   TestPass\n"}
{"Action":"pass","Package":"example/pkg","Test":"TestPass","Elapsed":0.02}
{"Action":"run","Package":"example/pkg","Test":"TestFail"}
{"Action":"output","Package":"example/pkg","Test":"TestFail","Output":"    foo_test.go:10: Error: not equal\n"}
{"Action":"fail","Package":"example/pkg","Test":"TestFail","Elapsed":0.01}
{"Action":"run","Package":"example/pkg","Test":"TestSkip"}
{"Action":"output","Package":"example/pkg","Test":"TestSkip","Output":"    foo_test.go:20: skipping\n"}
{"Action":"skip","Package":"example/pkg","Test":"TestSkip","Elapsed":0}
{"Action":"pass","Package":"example/pkg","Elapsed":0.03}
`

func TestPassFailSkip(t *testing.T) {
	r, _ := parse(t, passFailSkip)
	if len(r.Suites) != 1 {
		t.Fatalf("want 1 suite, got %d", len(r.Suites))
	}
	tot := r.Totals()
	if tot.Tests != 3 || tot.Failures != 1 || tot.Skipped != 1 {
		t.Fatalf("unexpected totals: %+v", tot)
	}
	fc := findCase(r, "example/pkg", "TestFail")
	if fc == nil || fc.Status != results.StatusFailed {
		t.Fatalf("TestFail not failed: %+v", fc)
	}
	if !strings.Contains(fc.Failure.Message, "Error:") {
		t.Errorf("failure message not extracted: %q", fc.Failure.Message)
	}
}

const subtests = `
{"Action":"run","Package":"p","Test":"TestParent"}
{"Action":"run","Package":"p","Test":"TestParent/child_a"}
{"Action":"pass","Package":"p","Test":"TestParent/child_a","Elapsed":0.01}
{"Action":"run","Package":"p","Test":"TestParent/child_b"}
{"Action":"fail","Package":"p","Test":"TestParent/child_b","Elapsed":0.01}
{"Action":"fail","Package":"p","Test":"TestParent","Elapsed":0.02}
{"Action":"fail","Package":"p","Elapsed":0.03}
`

func TestSubtestPathsPreserved(t *testing.T) {
	r, _ := parse(t, subtests)
	if c := findCase(r, "p", "TestParent/child_a"); c == nil {
		t.Fatal("subtest child_a missing; subtest path not preserved")
	}
	if c := findCase(r, "p", "TestParent/child_b"); c == nil || c.Status != results.StatusFailed {
		t.Fatal("subtest child_b missing or not failed")
	}
}

const buildFailure = `
{"Action":"output","Package":"broken/pkg","Output":"# broken/pkg\n"}
{"Action":"output","Package":"broken/pkg","Output":"./x.go:3:1: syntax error\n"}
{"Action":"fail","Package":"broken/pkg","Elapsed":0}
`

func TestBuildFailureSynthesized(t *testing.T) {
	r, diag := parse(t, buildFailure)
	c := findCase(r, "broken/pkg", "build")
	if c == nil {
		t.Fatal("expected synthetic build testcase")
	}
	if c.Status != results.StatusError {
		t.Fatalf("build failure should be error, got %s", c.Status)
	}
	if !strings.Contains(c.Failure.Details, "syntax error") {
		t.Errorf("build error output discarded: %q", c.Failure.Details)
	}
	if diag.Len() == 0 {
		t.Error("expected a diagnostic for build failure")
	}
}

const crashed = `
{"Action":"run","Package":"c","Test":"TestHang"}
{"Action":"output","Package":"c","Test":"TestHang","Output":"panic: goroutine stuck\n"}
{"Action":"fail","Package":"c","Elapsed":120}
`

func TestUnfinishedTestBecomesError(t *testing.T) {
	r, _ := parse(t, crashed)
	c := findCase(r, "c", "TestHang")
	if c == nil || c.Status != results.StatusError {
		t.Fatalf("unfinished test should be error, got %+v", c)
	}
}

func TestMalformedLineFails(t *testing.T) {
	diag := diagnostics.NewCollector()
	_, err := Adapter{}.Parse(strings.NewReader("not json\n"), results.Options{Diag: diag})
	if err == nil {
		t.Fatal("expected error on malformed event line")
	}
}
