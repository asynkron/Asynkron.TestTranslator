// Package junit writes and validates JUnit XML, the public test-result output
// contract. The writer produces deterministic output with stable attribute
// ordering and sanitizes text so generated documents are always well-formed
// XML.
package junit

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/asynkron/testtranslator/internal/results"
)

// xmlHeader is prepended to every document.
const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

// xmlSuites is the serialization root.
type xmlSuites struct {
	XMLName  xml.Name   `xml:"testsuites"`
	Name     string     `xml:"name,attr,omitempty"`
	Tests    int        `xml:"tests,attr"`
	Failures int        `xml:"failures,attr"`
	Errors   int        `xml:"errors,attr"`
	Skipped  int        `xml:"skipped,attr"`
	Time     string     `xml:"time,attr"`
	Suites   []xmlSuite `xml:"testsuite"`
}

type xmlSuite struct {
	Name       string    `xml:"name,attr"`
	Tests      int       `xml:"tests,attr"`
	Failures   int       `xml:"failures,attr"`
	Errors     int       `xml:"errors,attr"`
	Skipped    int       `xml:"skipped,attr"`
	Time       string    `xml:"time,attr"`
	Timestamp  string    `xml:"timestamp,attr,omitempty"`
	Properties *xmlProps `xml:"properties,omitempty"`
	Cases      []xmlCase `xml:"testcase"`
	SystemOut  *string   `xml:"system-out,omitempty"`
	SystemErr  *string   `xml:"system-err,omitempty"`
}

type xmlProps struct {
	Props []xmlProp `xml:"property"`
}

type xmlProp struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type xmlCase struct {
	Name      string      `xml:"name,attr"`
	Classname string      `xml:"classname,attr,omitempty"`
	Time      string      `xml:"time,attr"`
	File      string      `xml:"file,attr,omitempty"`
	Line      int         `xml:"line,attr,omitempty"`
	Failure   *xmlFailure `xml:"failure,omitempty"`
	Error     *xmlFailure `xml:"error,omitempty"`
	Skipped   *xmlSkipped `xml:"skipped,omitempty"`
	SystemOut *string     `xml:"system-out,omitempty"`
	SystemErr *string     `xml:"system-err,omitempty"`
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Details string `xml:",chardata"`
}

type xmlSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// fmtSeconds renders a duration as seconds with three decimals, deterministically.
func fmtSeconds(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}

// sanitize removes characters that are not legal in XML 1.0 text, replacing
// them with the Unicode replacement character so output is always well-formed.
func sanitize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0x9 || r == 0xA || r == 0xD:
			b.WriteRune(r)
		case r >= 0x20 && r <= 0xD7FF:
			b.WriteRune(r)
		case r >= 0xE000 && r <= 0xFFFD:
			b.WriteRune(r)
		case r >= 0x10000 && r <= 0x10FFFF:
			b.WriteRune(r)
		default:
			b.WriteRune('�')
		}
	}
	return b.String()
}

// ptr returns a pointer to a sanitized non-empty string, or nil when empty, so
// optional elements are omitted rather than emitted empty.
func optText(s string) *string {
	if s == "" {
		return nil
	}
	v := sanitize(s)
	return &v
}

// Marshal renders a Report as JUnit XML bytes. The report is not mutated; call
// Report.Sort beforehand for deterministic ordering.
func Marshal(r *results.Report) ([]byte, error) {
	total := r.Totals()
	root := xmlSuites{
		Name:     sanitize(r.Name),
		Tests:    total.Tests,
		Failures: total.Failures,
		Errors:   total.Errors,
		Skipped:  total.Skipped,
		Time:     fmtSeconds(total.Duration),
	}
	for i := range r.Suites {
		s := &r.Suites[i]
		st := s.SuiteTotals()
		xs := xmlSuite{
			Name:      sanitize(s.Name),
			Tests:     st.Tests,
			Failures:  st.Failures,
			Errors:    st.Errors,
			Skipped:   st.Skipped,
			Time:      fmtSeconds(st.Duration),
			SystemOut: optText(s.SystemOut),
			SystemErr: optText(s.SystemErr),
		}
		if !s.Timestamp.IsZero() {
			xs.Timestamp = s.Timestamp.UTC().Format("2006-01-02T15:04:05")
		}
		if len(s.Properties) > 0 {
			props := &xmlProps{}
			for _, p := range s.Properties {
				props.Props = append(props.Props, xmlProp{Name: sanitize(p.Name), Value: sanitize(p.Value)})
			}
			xs.Properties = props
		}
		for _, c := range s.Cases {
			xc := xmlCase{
				Name:      sanitize(c.Name),
				Classname: sanitize(c.Classname),
				Time:      fmtSeconds(c.Duration),
				File:      sanitize(c.File),
				Line:      c.Line,
				SystemOut: optText(c.SystemOut),
				SystemErr: optText(c.SystemErr),
			}
			switch c.Status {
			case results.StatusFailed:
				xc.Failure = &xmlFailure{
					Message: sanitize(c.Failure.Message),
					Type:    sanitize(c.Failure.Type),
					Details: sanitize(c.Failure.Details),
				}
			case results.StatusError:
				xc.Error = &xmlFailure{
					Message: sanitize(c.Failure.Message),
					Type:    sanitize(c.Failure.Type),
					Details: sanitize(c.Failure.Details),
				}
			case results.StatusSkipped:
				xc.Skipped = &xmlSkipped{Message: sanitize(c.SkipMessage)}
			}
			xs.Cases = append(xs.Cases, xc)
		}
		root.Suites = append(root.Suites, xs)
	}

	body, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal junit: %w", err)
	}
	out := make([]byte, 0, len(xmlHeader)+len(body)+1)
	out = append(out, xmlHeader...)
	out = append(out, body...)
	out = append(out, '\n')
	return out, nil
}
