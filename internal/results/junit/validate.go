package junit

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// vSuites mirrors the JUnit structure for validation. It is intentionally
// independent of the writer's serialization structs so validation exercises the
// bytes, not the in-memory model.
type vSuites struct {
	XMLName  xml.Name `xml:"testsuites"`
	Tests    *int     `xml:"tests,attr"`
	Failures *int     `xml:"failures,attr"`
	Errors   *int     `xml:"errors,attr"`
	Skipped  *int     `xml:"skipped,attr"`
	Suites   []vSuite `xml:"testsuite"`
}

type vSuite struct {
	Name     string  `xml:"name,attr"`
	Tests    *int    `xml:"tests,attr"`
	Failures *int    `xml:"failures,attr"`
	Errors   *int    `xml:"errors,attr"`
	Skipped  *int    `xml:"skipped,attr"`
	Cases    []vCase `xml:"testcase"`
}

type vCase struct {
	Name    string   `xml:"name,attr"`
	Time    string   `xml:"time,attr"`
	Failure *vDetail `xml:"failure"`
	Error   *vDetail `xml:"error"`
	Skipped *vSkipEl `xml:"skipped"`
}

type vDetail struct {
	Message string `xml:"message,attr"`
}

type vSkipEl struct {
	Message string `xml:"message,attr"`
}

// Validate parses JUnit XML and checks structural and semantic invariants: it
// must be a well-formed <testsuites> document whose per-suite and top-level
// counts agree with the actual test cases. It rejects unsafe XML features by
// disabling external entity resolution.
func Validate(r io.Reader) error {
	dec := xml.NewDecoder(r)
	// Disable DTD/external entity expansion: leaving Entity nil and providing
	// no CharsetReader means custom entities are rejected as errors.
	dec.Strict = true
	dec.Entity = map[string]string{}

	var doc vSuites
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("junit: malformed XML: %w", err)
	}
	if doc.XMLName.Local != "testsuites" {
		return fmt.Errorf("junit: root element is %q, want <testsuites>", doc.XMLName.Local)
	}

	var gotTests, gotFailures, gotErrors, gotSkipped int
	for si := range doc.Suites {
		s := &doc.Suites[si]
		var sTests, sFail, sErr, sSkip int
		for ci := range s.Cases {
			c := &s.Cases[ci]
			if c.Name == "" {
				return fmt.Errorf("junit: suite %q has a testcase with no name", s.Name)
			}
			if c.Time != "" {
				if _, err := strconv.ParseFloat(c.Time, 64); err != nil {
					return fmt.Errorf("junit: testcase %q has non-numeric time %q", c.Name, c.Time)
				}
			}
			outcomes := 0
			if c.Failure != nil {
				outcomes++
				sFail++
			}
			if c.Error != nil {
				outcomes++
				sErr++
			}
			if c.Skipped != nil {
				outcomes++
				sSkip++
			}
			if outcomes > 1 {
				return fmt.Errorf("junit: testcase %q has multiple outcome elements", c.Name)
			}
			sTests++
		}
		if err := checkCount("suite "+s.Name+" tests", s.Tests, sTests); err != nil {
			return err
		}
		if err := checkCount("suite "+s.Name+" failures", s.Failures, sFail); err != nil {
			return err
		}
		if err := checkCount("suite "+s.Name+" errors", s.Errors, sErr); err != nil {
			return err
		}
		if err := checkCount("suite "+s.Name+" skipped", s.Skipped, sSkip); err != nil {
			return err
		}
		gotTests += sTests
		gotFailures += sFail
		gotErrors += sErr
		gotSkipped += sSkip
	}

	if err := checkCount("testsuites tests", doc.Tests, gotTests); err != nil {
		return err
	}
	if err := checkCount("testsuites failures", doc.Failures, gotFailures); err != nil {
		return err
	}
	if err := checkCount("testsuites errors", doc.Errors, gotErrors); err != nil {
		return err
	}
	if err := checkCount("testsuites skipped", doc.Skipped, gotSkipped); err != nil {
		return err
	}
	return nil
}

// checkCount verifies a declared attribute matches the counted value. A missing
// attribute (nil) is permitted for inputs that omit it, but the writer always
// emits them.
func checkCount(label string, declared *int, actual int) error {
	if declared == nil {
		return nil
	}
	if *declared != actual {
		return fmt.Errorf("junit: %s declared %d but found %d", label, *declared, actual)
	}
	return nil
}
