// Package xunit parses xUnit.net v2 and v3 XML test results into the internal
// model. Both versions share the same shape: an optional <assemblies> wrapper
// containing one or more <assembly> elements, each holding <collection>
// elements of <test> results. A single <assembly> may also appear as the
// document root without the wrapper. Each collection becomes an internal suite
// named after its assembly (and, when present, its collection).
package xunit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
	"github.com/asynkron/Asynkron.TestTranslator/internal/xmlguard"
)

func init() {
	results.Register(
		"xunit",
		"xUnit.net v2 and v3 XML test results",
		Adapter{},
		"xunit2", "xunitv2", "xunit3", "xunitv3", "dotnet-xunit",
	)
}

// defaultMaxInput bounds decoding when opts.MaxInputBytes is not set.
const defaultMaxInput = 256 << 20

// Adapter implements results.Adapter for xUnit.net XML.
type Adapter struct{}

// xAssemblies is the outer <assemblies> wrapper used by xUnit.net.
type xAssemblies struct {
	XMLName    xml.Name    `xml:"assemblies"`
	Assemblies []xAssembly `xml:"assembly"`
}

// xAssembly represents one <assembly> element and its collections.
type xAssembly struct {
	Name        string        `xml:"name,attr"`
	Collections []xCollection `xml:"collection"`
	// Tests directly under <assembly> are tolerated even though standard
	// xUnit.net always nests tests inside a <collection>.
	Tests []xTest `xml:"test"`
}

// xCollection represents a <collection> grouping of tests.
type xCollection struct {
	Name  string  `xml:"name,attr"`
	Tests []xTest `xml:"test"`
}

// xTest represents a single <test> result.
type xTest struct {
	Name    string    `xml:"name,attr"`
	Type    string    `xml:"type,attr"`
	Method  string    `xml:"method,attr"`
	Time    string    `xml:"time,attr"`
	Result  string    `xml:"result,attr"`
	Failure *xFailure `xml:"failure"`
	Reason  string    `xml:"reason"`
	Output  string    `xml:"output"`
}

// xFailure represents a <failure> element on a failing test.
type xFailure struct {
	ExceptionType string `xml:"exception-type,attr"`
	Message       string `xml:"message"`
	StackTrace    string `xml:"stack-trace"`
}

// Parse reads an xUnit.net v2 or v3 document. It accepts either an
// <assemblies> wrapper or a bare <assembly> root and errors clearly when the
// root element is neither.
func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("xunit: read: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("xunit: input exceeds %d byte limit", limit)
	}
	if err := xmlguard.Check(data, xmlguard.DefaultMaxDepth); err != nil {
		return nil, fmt.Errorf("xunit: %w", err)
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	dec.Entity = map[string]string{}

	// Peek at the root element to decide how to decode, and to fail clearly on
	// an unexpected shape before consuming the whole document.
	root, err := firstElement(dec)
	if err != nil {
		return nil, err
	}

	var assemblies []xAssembly
	switch root.Name.Local {
	case "assemblies":
		var doc xAssemblies
		if err := dec.DecodeElement(&doc, root); err != nil {
			return nil, fmt.Errorf("xunit: malformed XML: %w", err)
		}
		assemblies = doc.Assemblies
	case "assembly":
		var asm xAssembly
		if err := dec.DecodeElement(&asm, root); err != nil {
			return nil, fmt.Errorf("xunit: malformed XML: %w", err)
		}
		assemblies = []xAssembly{asm}
	default:
		return nil, fmt.Errorf("xunit: root element is %q, want <assemblies> or <assembly>", root.Name.Local)
	}

	report := &results.Report{Name: "xunit"}
	for _, asm := range assemblies {
		if err := convertAssembly(asm, report, opts); err != nil {
			return nil, err
		}
	}
	if len(report.Suites) == 0 {
		return nil, fmt.Errorf("xunit: no test cases found")
	}
	return report, nil
}

// firstElement advances the decoder to the first start element (skipping the
// XML declaration, comments, and directives) and returns it.
func firstElement(dec *xml.Decoder) (*xml.StartElement, error) {
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, fmt.Errorf("xunit: empty document, expected <assemblies> or <assembly>")
		}
		if err != nil {
			return nil, fmt.Errorf("xunit: malformed XML: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return &se, nil
		}
	}
}

// convertAssembly emits one suite per collection (or a single suite for tests
// placed directly under the assembly).
func convertAssembly(asm xAssembly, report *results.Report, opts results.Options) error {
	// Tests placed directly under <assembly> without a collection.
	if len(asm.Tests) > 0 {
		if err := appendSuite(suiteName(asm.Name, ""), asm.Tests, report, opts); err != nil {
			return err
		}
	}
	for _, coll := range asm.Collections {
		if err := appendSuite(suiteName(asm.Name, coll.Name), coll.Tests, report, opts); err != nil {
			return err
		}
	}
	return nil
}

// suiteName joins the assembly and collection names into a stable suite name.
func suiteName(assembly, collection string) string {
	assembly = strings.TrimSpace(assembly)
	collection = strings.TrimSpace(collection)
	switch {
	case assembly == "" && collection == "":
		return "xunit"
	case collection == "":
		return assembly
	case assembly == "":
		return collection
	default:
		return assembly + " / " + collection
	}
}

// appendSuite converts a slice of tests into a suite and appends it to report.
func appendSuite(name string, tests []xTest, report *results.Report, opts results.Options) error {
	if len(tests) == 0 {
		return nil
	}
	suite := results.TestSuite{Name: name}
	for _, t := range tests {
		tc, err := convertTest(t, name, opts)
		if err != nil {
			return err
		}
		if err := suite.AddCase(tc); err != nil {
			return err
		}
	}
	report.Suites = append(report.Suites, suite)
	return nil
}

// convertTest maps a single xUnit.net <test> into a results.TestCase.
func convertTest(t xTest, suiteName string, opts results.Options) (results.TestCase, error) {
	name := strings.TrimSpace(t.Name)
	if name == "" {
		return results.TestCase{}, fmt.Errorf("xunit: suite %q has an unnamed test", suiteName)
	}
	tc := results.TestCase{
		Name:      name,
		Classname: strings.TrimSpace(t.Type),
		Duration:  parseSeconds(t.Time),
		SystemOut: strings.TrimSpace(t.Output),
	}

	switch strings.ToLower(strings.TrimSpace(t.Result)) {
	case "pass":
		tc.Status = results.StatusPassed
	case "skip":
		tc.Status = results.StatusSkipped
		tc.SkipMessage = strings.TrimSpace(t.Reason)
	case "fail":
		tc.Status = results.StatusFailed
		tc.Failure = failureOf(t)
	default:
		return results.TestCase{}, fmt.Errorf(
			"xunit: test %q has unknown result %q, want Pass, Fail, or Skip", name, t.Result)
	}

	// xUnit.net does not distinguish a per-method Classname from the declaring
	// type; if @type is missing we cannot represent one, so note the loss.
	if tc.Classname == "" && opts.Diag != nil {
		opts.Diag.Notef("xunit.no-type", opts.SourceName,
			"test %q has no type attribute; classname left empty", name)
	}
	return tc, nil
}

// failureOf builds a Failure from a <failure> element. A failing test without a
// <failure> element still yields a non-nil Failure so the invariant holds.
func failureOf(t xTest) *results.Failure {
	f := &results.Failure{}
	if t.Failure == nil {
		f.Message = "test failed"
		return f
	}
	msg := strings.TrimSpace(t.Failure.Message)
	stack := strings.TrimSpace(t.Failure.StackTrace)
	f.Type = strings.TrimSpace(t.Failure.ExceptionType)
	f.Message = firstLine(msg)
	if f.Message == "" {
		f.Message = "test failed"
	}
	switch {
	case msg != "" && stack != "":
		f.Details = msg + "\n" + stack
	case stack != "":
		f.Details = stack
	default:
		f.Details = msg
	}
	return f
}

// parseSeconds parses an xUnit.net time attribute (seconds, floating point).
func parseSeconds(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil && secs >= 0 {
		return time.Duration(secs * float64(time.Second))
	}
	return 0
}

// firstLine returns the first non-empty, trimmed line of s.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}
