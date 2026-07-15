// Package trx parses Visual Studio Test Platform and Microsoft.Testing.Platform
// TRX result files into the internal model. TRX stores outcomes in <Results> and
// class/method names in <TestDefinitions>, keyed by test id.
package trx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
	"github.com/asynkron/Asynkron.TestTranslator/internal/xmlguard"
)

func init() {
	results.Register("vstest-trx", "Visual Studio Test Platform / Microsoft.Testing.Platform TRX", Adapter{}, "trx", "mtp-trx")
}

const defaultMaxInput = 256 << 20

// Adapter implements results.Adapter for TRX.
type Adapter struct{}

type testRun struct {
	XMLName         xml.Name         `xml:"TestRun"`
	Name            string           `xml:"name,attr"`
	Results         []unitTestResult `xml:"Results>UnitTestResult"`
	TestDefinitions []unitTest       `xml:"TestDefinitions>UnitTest"`
}

type unitTestResult struct {
	TestName string     `xml:"testName,attr"`
	TestID   string     `xml:"testId,attr"`
	Outcome  string     `xml:"outcome,attr"`
	Duration string     `xml:"duration,attr"`
	Output   *trxOutput `xml:"Output"`
}

type trxOutput struct {
	StdOut    string     `xml:"StdOut"`
	StdErr    string     `xml:"StdErr"`
	ErrorInfo *errorInfo `xml:"ErrorInfo"`
}

type errorInfo struct {
	Message    string `xml:"Message"`
	StackTrace string `xml:"StackTrace"`
}

type unitTest struct {
	ID         string      `xml:"id,attr"`
	Name       string      `xml:"name,attr"`
	TestMethod *testMethod `xml:"TestMethod"`
}

type testMethod struct {
	ClassName string `xml:"className,attr"`
	Name      string `xml:"name,attr"`
}

// Parse reads a TRX document and produces a single suite named by the run.
func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("trx: read: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("trx: input exceeds %d byte limit", limit)
	}
	if err := xmlguard.Check(data, xmlguard.DefaultMaxDepth); err != nil {
		return nil, fmt.Errorf("trx: %w", err)
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	dec.Entity = map[string]string{}

	var run testRun
	if err := dec.Decode(&run); err != nil {
		return nil, fmt.Errorf("trx: malformed TRX: %w", err)
	}
	if run.XMLName.Local != "TestRun" {
		return nil, fmt.Errorf("trx: root element is %q, want <TestRun>", run.XMLName.Local)
	}

	classByID := map[string]string{}
	for _, d := range run.TestDefinitions {
		if d.TestMethod != nil {
			classByID[d.ID] = d.TestMethod.ClassName
		}
	}

	suiteName := run.Name
	if suiteName == "" {
		suiteName = "TestRun"
	}
	suite := results.TestSuite{Name: suiteName}
	for _, res := range run.Results {
		if res.TestName == "" {
			return nil, fmt.Errorf("trx: a UnitTestResult has no testName")
		}
		tc := results.TestCase{
			Name:      res.TestName,
			Classname: classByID[res.TestID],
			Duration:  parseTRXDuration(res.Duration),
		}
		if res.Output != nil {
			tc.SystemOut = strings.TrimSpace(res.Output.StdOut)
			tc.SystemErr = strings.TrimSpace(res.Output.StdErr)
		}
		switch normalizeOutcome(res.Outcome) {
		case "passed":
			tc.Status = results.StatusPassed
		case "skipped":
			tc.Status = results.StatusSkipped
			tc.SkipMessage = res.Outcome
		case "error":
			tc.Status = results.StatusError
			tc.Failure = trxFailure(res.Output, res.Outcome)
		default: // failed
			tc.Status = results.StatusFailed
			tc.Failure = trxFailure(res.Output, res.Outcome)
		}
		if err := suite.AddCase(tc); err != nil {
			return nil, err
		}
	}

	if len(suite.Cases) == 0 {
		opts.Diag.Notef("trx.empty", opts.SourceName, "TRX run %q contained no results", suiteName)
	}
	return &results.Report{Name: "trx", Suites: []results.TestSuite{suite}}, nil
}

// normalizeOutcome maps TRX outcome strings to internal buckets.
func normalizeOutcome(o string) string {
	switch strings.ToLower(strings.TrimSpace(o)) {
	case "passed", "passedbutrunaborted":
		return "passed"
	case "notexecuted", "inconclusive", "pending", "disconnected", "warning":
		return "skipped"
	case "timeout", "aborted", "error":
		return "error"
	default: // Failed and anything unrecognized fails loudly as a failure
		return "failed"
	}
}

func trxFailure(out *trxOutput, outcome string) *results.Failure {
	f := &results.Failure{Type: outcome, Message: outcome}
	if out != nil && out.ErrorInfo != nil {
		if m := strings.TrimSpace(out.ErrorInfo.Message); m != "" {
			f.Message = firstLine(m)
		}
		f.Details = strings.TrimSpace(out.ErrorInfo.Message + "\n" + out.ErrorInfo.StackTrace)
	}
	return f
}

// parseTRXDuration parses "HH:MM:SS.fffffff".
func parseTRXDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}
	var h, m int
	var sec float64
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)
	fmt.Sscanf(parts[2], "%f", &sec)
	total := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec*float64(time.Second))
	if total < 0 {
		return 0
	}
	return total
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}
