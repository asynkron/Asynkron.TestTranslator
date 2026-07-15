package tap

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

// parse is a helper that runs the adapter over s with a fresh collector.
func parse(t *testing.T, s string) (*results.Report, *diagnostics.Collector, error) {
	t.Helper()
	diag := diagnostics.NewCollector()
	rep, err := Adapter{}.Parse(strings.NewReader(s), results.Options{SourceName: "test.tap", Diag: diag})
	return rep, diag, err
}

const validTAP13 = `TAP version 13
# Suite: acceptance
1..5
ok 1 - first passes
not ok 2 - second fails
  ---
  message: expected 1 got 2
  severity: fail
  ...
ok 3 - third skipped # SKIP not applicable
not ok 4 - known bug # TODO fix later
ok 5 - unexpected success # TODO should stay broken
`

func TestParseValid(t *testing.T) {
	rep, diag, err := parse(t, validTAP13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Suites) != 1 {
		t.Fatalf("want 1 suite, got %d", len(rep.Suites))
	}
	suite := rep.Suites[0]
	if suite.Name != "acceptance" {
		t.Errorf("suite name = %q, want %q", suite.Name, "acceptance")
	}
	if len(suite.Cases) != 5 {
		t.Fatalf("want 5 cases, got %d", len(suite.Cases))
	}

	c := suite.Cases
	if c[0].Status != results.StatusPassed {
		t.Errorf("case 1 status = %v, want Passed", c[0].Status)
	}
	if c[0].Name != "1 first passes" {
		t.Errorf("case 1 name = %q", c[0].Name)
	}

	// "not ok" with a YAML block -> Failed with details captured.
	if c[1].Status != results.StatusFailed {
		t.Errorf("case 2 status = %v, want Failed", c[1].Status)
	}
	if c[1].Failure == nil {
		t.Fatalf("case 2 must have a Failure")
	}
	if !strings.Contains(c[1].Failure.Details, "expected 1 got 2") {
		t.Errorf("case 2 details missing YAML content: %q", c[1].Failure.Details)
	}

	// SKIP directive -> Skipped.
	if c[2].Status != results.StatusSkipped {
		t.Errorf("case 3 status = %v, want Skipped", c[2].Status)
	}
	if c[2].SkipMessage != "not applicable" {
		t.Errorf("case 3 skip message = %q", c[2].SkipMessage)
	}

	// "not ok ... # TODO" -> Skipped (expected failure).
	if c[3].Status != results.StatusSkipped {
		t.Errorf("case 4 status = %v, want Skipped (TODO)", c[3].Status)
	}

	// "ok ... # TODO" -> Passed (unexpected success).
	if c[4].Status != results.StatusPassed {
		t.Errorf("case 5 status = %v, want Passed (TODO)", c[4].Status)
	}

	if diag.Len() != 0 {
		t.Errorf("did not expect diagnostic errors")
	}
}

func TestParsePlanMismatchWarns(t *testing.T) {
	const src = `1..3
ok 1 a
ok 2 b
`
	_, diag, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diag.Len() == 0 {
		t.Errorf("expected a plan-mismatch diagnostic")
	}
}

func TestParseBailOut(t *testing.T) {
	const src = `1..2
ok 1 a
Bail out! database unavailable
`
	rep, _, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	suite := rep.Suites[0]
	if len(suite.Cases) != 2 {
		t.Fatalf("want 2 cases (1 result + bailout), got %d", len(suite.Cases))
	}
	last := suite.Cases[len(suite.Cases)-1]
	if last.Status != results.StatusError {
		t.Errorf("bail out case status = %v, want Error", last.Status)
	}
	if last.Failure == nil || !strings.Contains(last.Failure.Message, "database unavailable") {
		t.Errorf("bail out message missing reason: %+v", last.Failure)
	}
}

func TestParseNoPlanNoResults(t *testing.T) {
	const src = `# just a comment
# nothing testable here
`
	_, _, err := parse(t, src)
	if err == nil {
		t.Fatalf("expected error for input with no plan and no results")
	}
	if !strings.Contains(err.Error(), "not valid TAP") {
		t.Errorf("error = %q, want it to mention invalid TAP", err.Error())
	}
}

// A TAP 13 subtest with its own inner plan must not be mistaken for a duplicate
// top-level plan (regression: this used to fail the whole parse), and its
// children must be flattened under the parent's name without double-counting the
// parent summary line.
func TestParseNestedSubtest(t *testing.T) {
	const src = `TAP version 13
1..1
# Subtest: parent
    ok 1 - childA
    ok 2 - childB
    1..2
ok 1 - parent
`
	rep, diag, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	suite := rep.Suites[0]
	if len(suite.Cases) != 2 {
		t.Fatalf("want 2 flattened child cases, got %d: %+v", len(suite.Cases), suite.Cases)
	}
	if suite.Cases[0].Name != "parent / 1 childA" {
		t.Errorf("child 1 name = %q, want %q", suite.Cases[0].Name, "parent / 1 childA")
	}
	if suite.Cases[1].Name != "parent / 2 childB" {
		t.Errorf("child 2 name = %q, want %q", suite.Cases[1].Name, "parent / 2 childB")
	}
	if diag.Len() != 0 {
		t.Errorf("did not expect diagnostics (plans match per level), got %d", diag.Len())
	}
}

// A failing subtest must remain visible: the "not ok" summary is kept, named for
// the subtest, so the failure is never silently dropped.
func TestParseNestedSubtestFailurePreserved(t *testing.T) {
	const src = `1..1
# Subtest: outer
    not ok 1 - inner fails
    1..1
not ok 1 - outer
`
	rep, _, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	suite := rep.Suites[0]
	var names []string
	failed := 0
	for _, c := range suite.Cases {
		names = append(names, c.Name)
		if c.Status == results.StatusFailed {
			failed++
		}
	}
	if failed == 0 {
		t.Fatalf("expected the failing subtest to be represented; cases=%v", names)
	}
	found := false
	for _, n := range names {
		if strings.Contains(n, "outer") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a case named for the failing subtest, got %v", names)
	}
}

// node-tap "buffered" subtests wrap the child block in braces. The braces must
// not be reported as unrecognized lines, and the inner assertions must nest
// under the parent result's name (regression: braces produced tap.unknownline
// warnings and a " / 1" name, and the inner plan risked a duplicate-plan error).
func TestParseBufferedBraceSubtest(t *testing.T) {
	const src = `not ok 1 - child
  ---
  some: diagnostics
  ...
{
    ok 1
    1..1
}
1..1
`
	rep, diag, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diag.Len() != 0 {
		t.Errorf("expected no diagnostics for buffered braces, got %d", diag.Len())
	}
	suite := rep.Suites[0]
	names := make(map[string]results.Status)
	for _, c := range suite.Cases {
		names[c.Name] = c.Status
	}
	if st, ok := names["1 child"]; !ok || st != results.StatusFailed {
		t.Errorf("want a failed \"1 child\" parent case, got %v", names)
	}
	if _, ok := names["child / 1"]; !ok {
		t.Errorf("want the inner assertion nested as \"child / 1\", got %v", names)
	}
}

// A "1..0 # SKIP <reason>" plan skips the whole run with no result lines. The
// reason must not be dropped: it becomes a synthetic skipped testcase plus a
// diagnostic note.
func TestParsePlanSkipAll(t *testing.T) {
	const src = "1..0 # SKIP no fixtures on this platform\n"
	rep, diag, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	suite := rep.Suites[0]
	if len(suite.Cases) != 1 {
		t.Fatalf("want 1 synthetic skipped case, got %d", len(suite.Cases))
	}
	c := suite.Cases[0]
	if c.Status != results.StatusSkipped {
		t.Errorf("status = %v, want Skipped", c.Status)
	}
	if !strings.Contains(c.SkipMessage, "no fixtures on this platform") {
		t.Errorf("skip reason lost: %q", c.SkipMessage)
	}
	if diag.Len() == 0 {
		t.Errorf("expected a tap.skipall diagnostic note")
	}
}

// A plain "1..0" (no SKIP directive) is a legitimately empty run and stays empty.
func TestParsePlanEmptyNoSkip(t *testing.T) {
	rep, _, err := parse(t, "1..0\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := len(rep.Suites[0].Cases); n != 0 {
		t.Errorf("want 0 cases for an empty plan, got %d", n)
	}
}

func TestParsePlanLast(t *testing.T) {
	const src = `ok 1 first
not ok 2 second
1..2
`
	rep, diag, err := parse(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Suites[0].Cases) != 2 {
		t.Fatalf("want 2 cases, got %d", len(rep.Suites[0].Cases))
	}
	if diag.Len() != 0 {
		t.Errorf("unexpected diagnostic errors")
	}
}
