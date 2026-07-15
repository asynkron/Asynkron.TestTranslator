// Package nunit parses NUnit 2 (<test-results>) and NUnit 3 (<test-run>) XML
// into the internal model. Both nest <test-suite> elements recursively; NUnit 2
// wraps children in a <results> element while NUnit 3 nests them directly. Each
// leaf fixture becomes a JUnit suite.
package nunit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/asynkron/testtranslator/internal/results"
	"github.com/asynkron/testtranslator/internal/xmlguard"
)

func init() {
	results.Register("nunit", "NUnit 2 and NUnit 3 XML", Adapter{}, "nunit2", "nunit3")
}

const defaultMaxInput = 256 << 20

// Adapter implements results.Adapter for NUnit XML.
type Adapter struct{}

type nRoot struct {
	XMLName xml.Name
	Suites  []nSuite `xml:"test-suite"`
}

type nSuite struct {
	Name    string    `xml:"name,attr"`
	Full    string    `xml:"fullname,attr"`
	Suites  []nSuite  `xml:"test-suite"`
	Cases   []nCase   `xml:"test-case"`
	Results *nResults `xml:"results"`
}

type nResults struct {
	Suites []nSuite `xml:"test-suite"`
	Cases  []nCase  `xml:"test-case"`
}

type nCase struct {
	Name      string    `xml:"name,attr"`
	Full      string    `xml:"fullname,attr"`
	ClassName string    `xml:"classname,attr"`
	Result    string    `xml:"result,attr"`
	Success   string    `xml:"success,attr"`
	Executed  string    `xml:"executed,attr"`
	Duration  string    `xml:"duration,attr"`
	Time      string    `xml:"time,attr"`
	Failure   *nFailure `xml:"failure"`
	Reason    *nReason  `xml:"reason"`
	Output    string    `xml:"output"`
}

type nFailure struct {
	Message    string `xml:"message"`
	StackTrace string `xml:"stack-trace"`
}

type nReason struct {
	Message string `xml:"message"`
}

// Parse reads an NUnit 2 or 3 document.
func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("nunit: read: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("nunit: input exceeds %d byte limit", limit)
	}
	if err := xmlguard.Check(data, xmlguard.DefaultMaxDepth); err != nil {
		return nil, fmt.Errorf("nunit: %w", err)
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	dec.Entity = map[string]string{}

	var root nRoot
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("nunit: malformed XML: %w", err)
	}
	switch root.XMLName.Local {
	case "test-run", "test-results":
	default:
		return nil, fmt.Errorf("nunit: root element is %q, want <test-run> or <test-results>", root.XMLName.Local)
	}

	report := &results.Report{Name: "nunit"}
	for i := range root.Suites {
		if err := walk("", &root.Suites[i], report); err != nil {
			return nil, err
		}
	}
	if len(report.Suites) == 0 {
		return nil, fmt.Errorf("nunit: no test cases found")
	}
	return report, nil
}

// walk emits a suite for each node that directly contains test cases, then
// recurses into child suites.
func walk(prefix string, s *nSuite, report *results.Report) error {
	name := s.Name
	if s.Full != "" {
		name = s.Full
	}
	full := name
	if prefix != "" && s.Full == "" {
		full = prefix + "." + name
	}

	directCases := s.Cases
	var childSuites []nSuite
	childSuites = append(childSuites, s.Suites...)
	if s.Results != nil {
		directCases = append(directCases, s.Results.Cases...)
		childSuites = append(childSuites, s.Results.Suites...)
	}

	if len(directCases) > 0 {
		suite := results.TestSuite{Name: full}
		for _, c := range directCases {
			tc, err := convertCase(c, full)
			if err != nil {
				return err
			}
			if err := suite.AddCase(tc); err != nil {
				return err
			}
		}
		report.Suites = append(report.Suites, suite)
	}
	for i := range childSuites {
		if err := walk(full, &childSuites[i], report); err != nil {
			return err
		}
	}
	return nil
}

func convertCase(c nCase, suiteName string) (results.TestCase, error) {
	name := c.Name
	if name == "" {
		name = c.Full
	}
	if name == "" {
		return results.TestCase{}, fmt.Errorf("nunit: suite %q has an unnamed test case", suiteName)
	}
	classname := c.ClassName
	if classname == "" {
		classname = suiteName
	}
	tc := results.TestCase{
		Name:      name,
		Classname: classname,
		Duration:  parseSeconds(c.Duration, c.Time),
		SystemOut: strings.TrimSpace(c.Output),
	}
	switch classifyOutcome(c) {
	case "passed":
		tc.Status = results.StatusPassed
	case "skipped":
		tc.Status = results.StatusSkipped
		if c.Reason != nil {
			tc.SkipMessage = strings.TrimSpace(c.Reason.Message)
		}
	case "error":
		tc.Status = results.StatusError
		tc.Failure = failureOf(c, "error")
	default:
		tc.Status = results.StatusFailed
		tc.Failure = failureOf(c, "failure")
	}
	return tc, nil
}

// classifyOutcome interprets NUnit 2 and 3 result semantics.
func classifyOutcome(c nCase) string {
	switch strings.ToLower(strings.TrimSpace(c.Result)) {
	case "passed", "success":
		return "passed"
	case "failed", "failure":
		return "failed"
	case "error":
		return "error"
	case "skipped", "ignored", "inconclusive", "notrunnable":
		return "skipped"
	}
	// NUnit 2 fallback via success/executed when result is absent.
	if strings.EqualFold(c.Executed, "False") {
		return "skipped"
	}
	if strings.EqualFold(c.Success, "True") {
		return "passed"
	}
	if c.Success != "" {
		return "failed"
	}
	return "passed"
}

func failureOf(c nCase, typ string) *results.Failure {
	f := &results.Failure{Type: typ, Message: typ}
	if c.Failure != nil {
		if m := strings.TrimSpace(c.Failure.Message); m != "" {
			f.Message = firstLine(m)
		}
		f.Details = strings.TrimSpace(c.Failure.Message + "\n" + c.Failure.StackTrace)
	} else if c.Reason != nil {
		f.Message = firstLine(strings.TrimSpace(c.Reason.Message))
		f.Details = strings.TrimSpace(c.Reason.Message)
	}
	return f
}

func parseSeconds(vals ...string) time.Duration {
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if secs, err := strconv.ParseFloat(v, 64); err == nil && secs >= 0 {
			return time.Duration(secs * float64(time.Second))
		}
	}
	return 0
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}
