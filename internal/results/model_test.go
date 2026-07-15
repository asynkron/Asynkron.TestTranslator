package results

import "testing"

func TestAddCaseRejectsContradictions(t *testing.T) {
	ts := &TestSuite{Name: "s"}
	cases := []struct {
		name string
		tc   TestCase
	}{
		{"empty name", TestCase{Status: StatusPassed}},
		{"invalid status", TestCase{Name: "a", Status: Status("bogus")}},
		{"failed without failure", TestCase{Name: "a", Status: StatusFailed}},
		{"passed with failure", TestCase{Name: "a", Status: StatusPassed, Failure: &Failure{Message: "x"}}},
		{"negative duration", TestCase{Name: "a", Status: StatusPassed, Duration: -1}},
	}
	for _, c := range cases {
		if err := ts.AddCase(c.tc); err == nil {
			t.Errorf("%s: expected AddCase to reject", c.name)
		}
	}
}

func TestAddCaseAcceptsValid(t *testing.T) {
	ts := &TestSuite{Name: "s"}
	if err := ts.AddCase(TestCase{Name: "a", Status: StatusPassed}); err != nil {
		t.Fatalf("valid passed case rejected: %v", err)
	}
	if err := ts.AddCase(TestCase{Name: "b", Status: StatusFailed, Failure: &Failure{Message: "x"}}); err != nil {
		t.Fatalf("valid failed case rejected: %v", err)
	}
}

func TestSortDeterministic(t *testing.T) {
	ts := TestSuite{Name: "z"}
	_ = ts.AddCase(TestCase{Name: "B", Classname: "z", Status: StatusPassed})
	_ = ts.AddCase(TestCase{Name: "A", Classname: "z", Status: StatusPassed})
	r := &Report{Suites: []TestSuite{ts, {Name: "a"}}}
	r.Sort()
	if r.Suites[0].Name != "a" {
		t.Errorf("suites not sorted: %q first", r.Suites[0].Name)
	}
	if r.Suites[1].Cases[0].Name != "A" {
		t.Errorf("cases not sorted: %q first", r.Suites[1].Cases[0].Name)
	}
}

func TestTotals(t *testing.T) {
	ts := TestSuite{Name: "s"}
	_ = ts.AddCase(TestCase{Name: "p", Status: StatusPassed})
	_ = ts.AddCase(TestCase{Name: "f", Status: StatusFailed, Failure: &Failure{Message: "x"}})
	_ = ts.AddCase(TestCase{Name: "e", Status: StatusError, Failure: &Failure{Message: "x"}})
	_ = ts.AddCase(TestCase{Name: "s", Status: StatusSkipped})
	r := &Report{Suites: []TestSuite{ts}}
	tot := r.Totals()
	if tot.Tests != 4 || tot.Failures != 1 || tot.Errors != 1 || tot.Skipped != 1 {
		t.Fatalf("wrong totals: %+v", tot)
	}
}
