package testtranslator

import "time"

// This file defines the public, JSON-tagged model types returned by
// [ParseResults] and [ParseCoverage]. They mirror the tool's internal models but
// are a deliberately separate, stable surface: the internal representation stays
// free to evolve, while these types form the contract a library consumer can
// depend on and persist. The JSON tags make them safe to serialize directly.

// --- Test-result model -------------------------------------------------------

// Status is the outcome of a single test case. The set is closed.
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusError   Status = "error"
	StatusSkipped Status = "skipped"
)

// TestReport is the parsed test-result model: ordered suites plus roll-up totals.
type TestReport struct {
	// Name is an optional top-level name for the set of suites.
	Name string `json:"name,omitempty"`
	// Suites are the ordered test suites.
	Suites []TestSuite `json:"suites"`
	// Totals is the roll-up across all suites.
	Totals Totals `json:"totals"`
}

// TestSuite groups related test cases.
type TestSuite struct {
	Name       string     `json:"name"`
	Cases      []TestCase `json:"cases"`
	Timestamp  time.Time  `json:"timestamp,omitempty"`
	SystemOut  string     `json:"system_out,omitempty"`
	SystemErr  string     `json:"system_err,omitempty"`
	Properties []Property `json:"properties,omitempty"`
}

// TestCase is a single executed or discovered test.
type TestCase struct {
	Name      string `json:"name"`
	Classname string `json:"classname,omitempty"`
	Status    Status `json:"status"`
	// DurationNanos is the wall-clock execution time in nanoseconds.
	DurationNanos int64        `json:"duration_nanos,omitempty"`
	Failure       *TestFailure `json:"failure,omitempty"`
	SkipMessage   string       `json:"skip_message,omitempty"`
	SystemOut     string       `json:"system_out,omitempty"`
	SystemErr     string       `json:"system_err,omitempty"`
	// File is the repo-relative source file, when the format provides it.
	File string `json:"file,omitempty"`
	// Line is the source line (0 when unknown).
	Line int `json:"line,omitempty"`
}

// TestFailure describes why a test failed or errored.
type TestFailure struct {
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
	Details string `json:"details,omitempty"`
}

// Property is a suite-level metadata key/value pair.
type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Totals summarizes a test report.
type Totals struct {
	Tests         int   `json:"tests"`
	Failures      int   `json:"failures"`
	Errors        int   `json:"errors"`
	Skipped       int   `json:"skipped"`
	DurationNanos int64 `json:"duration_nanos,omitempty"`
}

// --- Coverage model ----------------------------------------------------------

// Metric names a distinct coverage dimension. Keeping them separate prevents
// mislabeling (for example, Go statement counts are never reported as literal
// line counts).
type Metric string

const (
	MetricStatements   Metric = "statements"
	MetricLines        Metric = "lines"
	MetricBranches     Metric = "branches"
	MetricFunctions    Metric = "functions"
	MetricMethods      Metric = "methods"
	MetricInstructions Metric = "instructions"
	MetricRegions      Metric = "regions"
)

// CoverageReport is the parsed coverage model: per-file records keyed by
// repository-relative POSIX path.
type CoverageReport struct {
	// Producer names the coverage producer (e.g. "go", "vitest-v8").
	Producer string `json:"producer,omitempty"`
	// InterchangeFormat names the on-disk format (e.g. "istanbul-json").
	InterchangeFormat string `json:"interchange_format,omitempty"`
	// SourceRoots are roots preserved for consumers that need them.
	SourceRoots []string `json:"source_roots,omitempty"`
	// Files are per-file coverage records, ordered by path.
	Files []CoverageFile `json:"files"`
}

// CoverageFile is coverage for a single repository-relative source file. Path is
// the stable identity (and a natural join key against other per-file signals).
type CoverageFile struct {
	Path string `json:"path"`
	// Metrics holds native and derived counts keyed by metric. A metric is
	// present only when the source supplies (or the tool can honestly derive) it;
	// absence is never zero-filled. Check Count.Derived to distinguish.
	Metrics map[Metric]Count `json:"metrics,omitempty"`
	// Functions optionally records per-function coverage.
	Functions []CoverageFunction `json:"functions,omitempty"`
	// Lines optionally records per-line hit and branch data.
	Lines []LineHit `json:"lines,omitempty"`
}

// Count is a covered/total pair for one metric, optionally marked derived.
type Count struct {
	Covered int64 `json:"covered"`
	Total   int64 `json:"total"`
	// Derived is true when the metric was computed rather than supplied natively.
	Derived bool `json:"derived,omitempty"`
	// DerivationRule documents how a derived metric was computed.
	DerivationRule string `json:"derivation_rule,omitempty"`
}

// CoverageFunction is optional per-function coverage.
type CoverageFunction struct {
	Name string `json:"name"`
	Line int    `json:"line,omitempty"`
	Hits int64  `json:"hits"`
}

// LineHit records execution and branch data for a single source line.
type LineHit struct {
	Number          int   `json:"number"`
	Hits            int64 `json:"hits"`
	Branch          bool  `json:"branch,omitempty"`
	BranchesCovered int   `json:"branches_covered,omitempty"`
	BranchesTotal   int   `json:"branches_total,omitempty"`
}
