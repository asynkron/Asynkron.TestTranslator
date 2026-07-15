// Package surefire parses JUnit-family XML (Maven Surefire, Gradle, sbt,
// pytest, PHPUnit, cargo-nextest, Erlang Common Test) into the internal result
// model. These tools all emit the de-facto JUnit XML schema with a
// <testsuites> or single <testsuite> root. Normalizing JUnit XML input lets the
// tool re-emit canonical, validated JUnit XML.
package surefire

import (
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
	results.Register("junit-xml", "JUnit-family XML (Surefire, Gradle, sbt, pytest, PHPUnit, nextest, Erlang CT)", Adapter{},
		"surefire", "gradle-junit", "sbt-junit", "pytest-junit", "phpunit", "cargo-nextest", "erlang-ct", "junit4")
}

const defaultMaxInput = 256 << 20

// Adapter implements results.Adapter for JUnit-family XML.
type Adapter struct{}

type xSuites struct {
	XMLName xml.Name `xml:"testsuites"`
	Name    string   `xml:"name,attr"`
	Suites  []xSuite `xml:"testsuite"`
}

type xSuite struct {
	XMLName    xml.Name `xml:"testsuite"`
	Name       string   `xml:"name,attr"`
	Timestamp  string   `xml:"timestamp,attr"`
	Properties []xProp  `xml:"properties>property"`
	Cases      []xCase  `xml:"testcase"`
	SystemOut  string   `xml:"system-out"`
	SystemErr  string   `xml:"system-err"`
	Suites     []xSuite `xml:"testsuite"` // nested suites (some producers nest)
}

type xProp struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type xCase struct {
	Name      string    `xml:"name,attr"`
	Classname string    `xml:"classname,attr"`
	Time      string    `xml:"time,attr"`
	File      string    `xml:"file,attr"`
	Line      string    `xml:"line,attr"`
	Failure   *xDetail  `xml:"failure"`
	Error     *xDetail  `xml:"error"`
	Skipped   *xSkipped `xml:"skipped"`
	SystemOut string    `xml:"system-out"`
	SystemErr string    `xml:"system-err"`
}

type xDetail struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type xSkipped struct {
	Message string `xml:"message,attr"`
}

// Parse reads JUnit-family XML from either a <testsuites> or <testsuite> root.
func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("surefire: read: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("surefire: input exceeds %d byte limit", limit)
	}
	if err := xmlguard.Check(data, xmlguard.DefaultMaxDepth); err != nil {
		return nil, fmt.Errorf("surefire: %w", err)
	}

	root, err := detectRoot(data)
	if err != nil {
		return nil, err
	}

	report := &results.Report{}
	var walk func(prefix string, s *xSuite) error
	walk = func(prefix string, s *xSuite) error {
		name := s.Name
		if name == "" {
			name = "testsuite"
		}
		if prefix != "" {
			name = prefix + "." + name
		}
		suite := results.TestSuite{
			Name:      name,
			SystemOut: strings.TrimSpace(s.SystemOut),
			SystemErr: strings.TrimSpace(s.SystemErr),
		}
		if s.Timestamp != "" {
			if t, err := parseTimestamp(s.Timestamp); err == nil {
				suite.Timestamp = t
			}
		}
		for _, p := range s.Properties {
			suite.Properties = append(suite.Properties, results.Property{Name: p.Name, Value: p.Value})
		}
		for _, c := range s.Cases {
			tc, err := convertCase(c, name, opts)
			if err != nil {
				return err
			}
			if err := suite.AddCase(tc); err != nil {
				return err
			}
		}
		report.Suites = append(report.Suites, suite)
		for i := range s.Suites {
			if err := walk(name, &s.Suites[i]); err != nil {
				return err
			}
		}
		return nil
	}

	for i := range root {
		if err := walk("", &root[i]); err != nil {
			return nil, err
		}
	}
	if len(report.Suites) == 0 {
		return nil, fmt.Errorf("surefire: no <testsuite> elements found")
	}
	return report, nil
}

// detectRoot handles both <testsuites> and a bare <testsuite> root.
func detectRoot(data []byte) ([]xSuite, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.Contains(firstTag(trimmed), "testsuites") {
		var s xSuites
		if err := unmarshalStrict(data, &s); err != nil {
			return nil, fmt.Errorf("surefire: malformed <testsuites>: %w", err)
		}
		return s.Suites, nil
	}
	var one xSuite
	if err := unmarshalStrict(data, &one); err != nil {
		return nil, fmt.Errorf("surefire: malformed <testsuite>: %w", err)
	}
	if one.XMLName.Local != "testsuite" {
		return nil, fmt.Errorf("surefire: root element is %q, want <testsuites> or <testsuite>", one.XMLName.Local)
	}
	return []xSuite{one}, nil
}

// firstTag returns the first start tag's text for root detection.
func firstTag(s string) string {
	i := strings.IndexByte(s, '<')
	for i >= 0 && i+1 < len(s) && (s[i+1] == '?' || s[i+1] == '!') {
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			return ""
		}
		s = s[i+end+1:]
		i = strings.IndexByte(s, '<')
	}
	if i < 0 {
		return ""
	}
	end := strings.IndexByte(s[i:], '>')
	if end < 0 {
		return s[i:]
	}
	return s[i : i+end+1]
}

// unmarshalStrict decodes XML with external entities disabled.
func unmarshalStrict(data []byte, v any) error {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = true
	dec.Entity = map[string]string{}
	return dec.Decode(v)
}

// convertCase maps one JUnit-family testcase to the internal model.
func convertCase(c xCase, suiteName string, opts results.Options) (results.TestCase, error) {
	tc := results.TestCase{
		Name:      c.Name,
		Classname: c.Classname,
		SystemOut: strings.TrimSpace(c.SystemOut),
		SystemErr: strings.TrimSpace(c.SystemErr),
		File:      c.File,
	}
	if tc.Name == "" {
		return tc, fmt.Errorf("surefire: suite %q has a testcase with no name", suiteName)
	}
	if c.Time != "" {
		if secs, err := strconv.ParseFloat(strings.TrimSpace(c.Time), 64); err == nil && secs >= 0 {
			tc.Duration = time.Duration(secs * float64(time.Second))
		}
	}
	if c.Line != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(c.Line)); err == nil && n > 0 {
			tc.Line = n
		}
	}
	switch {
	case c.Error != nil:
		tc.Status = results.StatusError
		tc.Failure = detailToFailure(c.Error, "error")
	case c.Failure != nil:
		tc.Status = results.StatusFailed
		tc.Failure = detailToFailure(c.Failure, "failure")
	case c.Skipped != nil:
		tc.Status = results.StatusSkipped
		tc.SkipMessage = c.Skipped.Message
	default:
		tc.Status = results.StatusPassed
	}
	return tc, nil
}

func detailToFailure(d *xDetail, defType string) *results.Failure {
	typ := d.Type
	if typ == "" {
		typ = defType
	}
	msg := d.Message
	if msg == "" {
		msg = firstLine(d.Body)
	}
	if msg == "" {
		msg = defType
	}
	return &results.Failure{Message: msg, Type: typ, Details: strings.TrimSpace(d.Body)}
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// parseTimestamp accepts the common JUnit timestamp formats.
func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", s)
}
