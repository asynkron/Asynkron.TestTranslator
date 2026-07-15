// Package tap converts Test Anything Protocol (TAP) streams, versions 12 and
// 13, into the internal result model. TAP is a line-oriented plain-text format
// (not XML): an optional "TAP version 13" header, a plan line "1..N", and a
// sequence of "ok"/"not ok" result lines optionally carrying "# SKIP" or
// "# TODO" directives, comment/diagnostic lines beginning with '#', an
// indented YAML block under a result (TAP 13 only), and a terminal
// "Bail out!" line.
//
// This is a clean-room parser of the public TAP specification
// (https://testanything.org/tap-version-13-specification.html). The whole
// stream maps to a single suite named "tap" (or the name given by a leading
// "# Suite:" comment). Directive handling:
//
//   - "ok ... # SKIP"      -> Skipped (the test was deliberately not run).
//   - "not ok ... # SKIP"  -> Skipped (skips are skips regardless of ok/not ok).
//   - "ok ... # TODO"      -> Passed  (an unexpected success of a known bug).
//   - "not ok ... # TODO"  -> Skipped (an expected failure; not a real failure).
//   - "ok"                 -> Passed.
//   - "not ok"             -> Failed.
//
// A "not ok" result captures any following YAML block or diagnostic comment
// lines as the Failure.Details.
package tap

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/asynkron/testtranslator/internal/results"
)

func init() {
	results.Register("tap", "Test Anything Protocol (TAP) v12/v13 stream", Adapter{}, "tap12", "tap13")
}

// defaultMaxInput bounds total input bytes to avoid unbounded memory use.
const defaultMaxInput = 256 << 20 // 256 MiB

// maxTokenBytes bounds a single line so a pathological stream cannot exhaust
// memory.
const maxTokenBytes = 8 << 20 // 8 MiB per line

// Adapter implements results.Adapter for TAP v12/v13 streams.
type Adapter struct{}

// pendingCase accumulates a single result line and any diagnostics or YAML
// block that follow it before the next result line.
type pendingCase struct {
	tc       results.TestCase
	details  strings.Builder // collected YAML block and trailing diagnostics
	isNotOk  bool            // whether the source line was "not ok"
	frame    *frame          // the nesting level this result belongs to
	bareName string          // description only (no number), used to name a buffered subtest
}

// frame tracks one TAP nesting level: the root stream plus any subtests. TAP 13
// subtests are indentation-delimited blocks that carry their own plan and result
// lines and are summarized by a single result line at the parent level.
type frame struct {
	parent      *frame
	name        string // subtest name (empty for the root or an unannounced block)
	indent      int    // indentation width of this frame's plan/result lines
	brace       bool   // opened by a node-tap "{" delimiter (closed by "}")
	havePlan    bool
	planCount   int
	resultLines int
}

// prefix returns the "a / b / " name prefix a frame's ancestry contributes to the
// names of results nested inside it. The root and anonymous frames contribute
// nothing of their own.
func (f *frame) prefix() string {
	if f == nil || f.parent == nil {
		return ""
	}
	if f.name == "" {
		return f.parent.prefix()
	}
	return f.parent.prefix() + f.name + " / "
}

// Parse reads a TAP stream and produces one suite named "tap". TAP 13 subtests
// (indentation-delimited blocks with their own plan and results, summarized by a
// single result line at the parent level) are flattened into the suite: each
// child result is named with its "parent / child" path, and the parent's
// summary line is absorbed rather than emitted as a duplicate.
func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	sc := bufio.NewScanner(lr)
	sc.Buffer(make([]byte, 0, 64*1024), maxTokenBytes)

	suiteName := "tap"

	var cases []*pendingCase
	var cur *pendingCase // the result line currently collecting trailing details

	root := &frame{}
	stack := []*frame{root}
	allFrames := []*frame{root}
	var justClosed *frame // a subtest frame just closed by a dedent
	pendingSubName := ""  // name from a "# Subtest: <name>" announcement

	var (
		inYAML       bool
		totalResults int
		bailed       bool
		bailReason   string
		braceDepth   int
	)

	top := func() *frame { return stack[len(stack)-1] }

	line := 0
	for sc.Scan() {
		line++
		if lr.N <= 0 {
			return nil, fmt.Errorf("tap: input exceeds %d byte limit", limit)
		}
		raw := sc.Text()

		// A TAP 13 YAML block is delimited by an indented "---" / "...". While
		// inside it, capture every line verbatim as the current case's details.
		if inYAML {
			if strings.TrimSpace(raw) == "..." {
				inYAML = false
				continue
			}
			if cur != nil {
				cur.details.WriteString(strings.TrimSpace(raw))
				cur.details.WriteByte('\n')
			}
			continue
		}

		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		// Start of a YAML block belongs to the immediately preceding result.
		if trimmed == "---" && cur != nil {
			inYAML = true
			continue
		}

		// Bail out! signals a hard abort of the whole run.
		if strings.HasPrefix(trimmed, "Bail out!") {
			bailed = true
			bailReason = strings.TrimSpace(strings.TrimPrefix(trimmed, "Bail out!"))
			cur = nil
			break
		}

		// TAP version header, e.g. "TAP version 13".
		if strings.HasPrefix(trimmed, "TAP version") {
			continue
		}

		// Comment / diagnostic lines begin with '#'.
		if strings.HasPrefix(trimmed, "#") {
			body := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if name, ok := suiteNameFromComment(body); ok {
				suiteName = name
				continue
			}
			if name, ok := subtestNameFromComment(body); ok {
				pendingSubName = name
				continue
			}
			// Attach diagnostics to the current case (they typically explain a
			// preceding "not ok").
			if cur != nil {
				cur.details.WriteString(body)
				cur.details.WriteByte('\n')
			}
			continue
		}

		// node-tap "buffered" subtests delimit the child block with braces. Treat
		// "{" as opening a subtest bound to the preceding result and "}" as closing
		// it, so braces are not misread as unknown lines and the body nests.
		if trimmed == "{" {
			name := ""
			if cur != nil {
				name = cur.bareName
			}
			f := &frame{parent: top(), name: name, indent: top().indent, brace: true}
			stack = append(stack, f)
			allFrames = append(allFrames, f)
			braceDepth++
			justClosed = nil
			cur = nil
			continue
		}
		if trimmed == "}" && braceDepth > 0 {
			justClosed = top()
			stack = stack[:len(stack)-1]
			braceDepth--
			cur = nil
			continue
		}

		// Remaining lines are structural: a plan "1..N" or a result "ok"/"not ok".
		planN, isPlan := parsePlan(trimmed)
		isResult := isResultLine(trimmed)
		if !isPlan && !isResult {
			// TAP treats unknown lines as ignorable noise, but we surface it so
			// nothing is silently lost.
			opts.Diag.Warnf("tap.unknownline", opts.SourceName, "line %d: ignoring unrecognized TAP line %q", line, trimmed)
			continue
		}

		// Reconcile the frame stack with this line's indentation before handling
		// it: deeper opens a subtest, shallower closes one (or more). Inside a
		// brace-delimited subtest, the braces (not indentation) define nesting.
		if braceDepth == 0 {
			indent := indentWidth(raw)
			if indent > top().indent {
				f := &frame{parent: top(), name: pendingSubName, indent: indent}
				pendingSubName = ""
				stack = append(stack, f)
				allFrames = append(allFrames, f)
				justClosed = nil
				cur = nil
			} else {
				for len(stack) > 1 && indent < top().indent {
					justClosed = top()
					stack = stack[:len(stack)-1]
					cur = nil
				}
			}
		}

		if isPlan {
			if top().havePlan {
				return nil, fmt.Errorf("tap: line %d: duplicate plan line", line)
			}
			top().havePlan = true
			top().planCount = planN
			cur = nil
			justClosed = nil
			continue
		}

		// Result line.
		pc, _ := parseResult(trimmed, opts)
		pc.bareName = resultDescription(trimmed)

		// A result immediately following a closed subtest is that subtest's
		// summary. It counts toward the parent plan but must not be duplicated as
		// a leaf. A failing summary is kept (named for the subtest) so a subtest
		// failure is never silently dropped; a passing one is absorbed.
		if justClosed != nil {
			desc := resultDescription(trimmed)
			if justClosed.name == "" || justClosed.name == desc {
				if justClosed.name == "" {
					justClosed.name = desc
				}
				top().resultLines++
				totalResults++
				sub := justClosed
				justClosed = nil
				if !pc.isNotOk {
					cur = nil
					continue
				}
				pc.frame = top()
				pc.tc.Name = sub.name
				cases = append(cases, pc)
				cur = pc
				continue
			}
			justClosed = nil
		}

		top().resultLines++
		totalResults++
		pc.frame = top()
		cases = append(cases, pc)
		cur = pc
	}
	if err := sc.Err(); err != nil {
		if err == bufio.ErrTooLong {
			return nil, fmt.Errorf("tap: a line exceeds the %d byte limit", maxTokenBytes)
		}
		return nil, fmt.Errorf("tap: read error: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("tap: input exceeds %d byte limit", limit)
	}

	// A stream with neither a plan nor any result lines is not valid TAP.
	if !anyPlan(allFrames) && totalResults == 0 && !bailed {
		return nil, fmt.Errorf("tap: input has no plan line and no test result lines; not valid TAP")
	}

	// Validate each level's plan against its observed result count (soft check).
	for _, f := range allFrames {
		if f.havePlan && f.planCount != f.resultLines {
			where := ""
			if f.parent != nil {
				where = fmt.Sprintf("subtest %q: ", strings.TrimSpace(f.prefix()+f.name))
			}
			opts.Diag.Warnf("tap.planmismatch", opts.SourceName,
				"%splan declared %d tests but %d result lines were parsed", where, f.planCount, f.resultLines)
		}
	}

	suite := results.TestSuite{Name: suiteName}
	for _, pc := range cases {
		details := strings.TrimRight(pc.details.String(), "\n")
		if pc.tc.Failure != nil {
			pc.tc.Failure.Details = details
		}
		pc.tc.Name = pc.frame.prefix() + pc.tc.Name
		if err := suite.AddCase(pc.tc); err != nil {
			return nil, fmt.Errorf("tap: %w", err)
		}
	}

	// A Bail out! aborts the run: record it as a synthetic error testcase and a
	// diagnostic so the abort is never silently dropped.
	if bailed {
		msg := "TAP run bailed out"
		if bailReason != "" {
			msg = "Bail out! " + bailReason
		}
		tc := results.TestCase{
			Name:    "Bail out!",
			Status:  results.StatusError,
			Failure: &results.Failure{Message: msg, Type: "bailout", Details: bailReason},
		}
		if err := suite.AddCase(tc); err != nil {
			return nil, fmt.Errorf("tap: %w", err)
		}
		opts.Diag.Warnf("tap.bailout", opts.SourceName, "run bailed out: %s", msg)
	}

	return &results.Report{Name: "tap", Suites: []results.TestSuite{suite}}, nil
}

// suiteNameFromComment recognizes a leading "Suite: <name>" diagnostic and
// returns the suite name it declares.
func suiteNameFromComment(body string) (string, bool) {
	const prefix = "Suite:"
	if strings.HasPrefix(body, prefix) {
		name := strings.TrimSpace(strings.TrimPrefix(body, prefix))
		if name != "" {
			return name, true
		}
	}
	return "", false
}

// subtestNameFromComment recognizes a "Subtest: <name>" announcement (emitted by
// node-tap and others before an indented subtest block) and returns its name.
func subtestNameFromComment(body string) (string, bool) {
	const prefix = "Subtest:"
	if strings.HasPrefix(body, prefix) {
		name := strings.TrimSpace(strings.TrimPrefix(body, prefix))
		if name != "" {
			return name, true
		}
	}
	return "", false
}

// indentWidth returns the visual indentation of a line, counting a tab as eight
// columns. Only relative ordering matters, so the exact tab width is immaterial
// as long as it is consistent.
func indentWidth(raw string) int {
	w := 0
	for _, r := range raw {
		switch r {
		case ' ':
			w++
		case '\t':
			w += 8
		default:
			return w
		}
	}
	return w
}

// resultDescription returns just the human description of a result line, with the
// leading token, test number, and any trailing directive removed. It is used to
// match a subtest's summary line against the subtest's announced name.
func resultDescription(trimmed string) string {
	rest, _ := matchResultPrefix(trimmed)
	if h := strings.IndexByte(rest, '#'); h >= 0 {
		rest = rest[:h]
	}
	_, desc := splitNumber(strings.TrimSpace(rest))
	return desc
}

// anyPlan reports whether any frame declared a plan line.
func anyPlan(frames []*frame) bool {
	for _, f := range frames {
		if f.havePlan {
			return true
		}
	}
	return false
}

// parsePlan parses a TAP plan line "1..N" (N >= 0) and reports whether trimmed
// is a plan line.
func parsePlan(trimmed string) (int, bool) {
	idx := strings.Index(trimmed, "..")
	if idx <= 0 {
		return 0, false
	}
	lo := trimmed[:idx]
	rest := trimmed[idx+2:]
	// A plan may carry a trailing "# SKIP" directive; ignore it for counting.
	if h := strings.IndexByte(rest, '#'); h >= 0 {
		rest = strings.TrimSpace(rest[:h])
	}
	if lo != "1" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// parseResult parses a single "ok"/"not ok" result line into a pending case and
// reports whether trimmed is a result line. It applies SKIP/TODO directive
// mapping.
func parseResult(trimmed string, opts results.Options) (*pendingCase, bool) {
	if !isResultLine(trimmed) {
		return nil, false
	}
	rest, notOk := matchResultPrefix(trimmed)

	// Split off an optional trailing directive introduced by '#'.
	description := rest
	directive := ""
	if h := strings.IndexByte(rest, '#'); h >= 0 {
		description = rest[:h]
		directive = strings.TrimSpace(rest[h+1:])
	}
	description = strings.TrimSpace(description)

	// Strip a leading test number from the description.
	number, desc := splitNumber(description)
	name := desc
	if number != "" {
		if desc != "" {
			name = number + " " + desc
		} else {
			name = number
		}
	}
	if name == "" {
		name = "(unnamed)"
	}

	pc := &pendingCase{isNotOk: notOk}
	pc.tc.Name = name

	switch dir := directiveKind(directive); dir {
	case dirSkip:
		pc.tc.Status = results.StatusSkipped
		pc.tc.SkipMessage = directiveReason(directive)
	case dirTodo:
		// A passing TODO is an unexpected success -> Passed. A failing TODO is
		// the expected failure of a known bug -> Skipped (not a real failure).
		if notOk {
			pc.tc.Status = results.StatusSkipped
			pc.tc.SkipMessage = directiveReason(directive)
		} else {
			pc.tc.Status = results.StatusPassed
		}
	default:
		if notOk {
			pc.tc.Status = results.StatusFailed
			pc.tc.Failure = &results.Failure{Message: firstNonEmpty(desc, "test failed"), Type: "failure"}
		} else {
			pc.tc.Status = results.StatusPassed
		}
	}
	return pc, true
}

// matchResultPrefix strips a leading "ok" or "not ok" token and returns the
// remainder plus whether the line was "not ok". The boolean here only reflects
// the "not ok" branch; callers must also consult isResultLine.
func matchResultPrefix(trimmed string) (rest string, notOk bool) {
	if r, ok := afterToken(trimmed, "not ok"); ok {
		return r, true
	}
	if r, ok := afterToken(trimmed, "ok"); ok {
		return r, false
	}
	return "", false
}

// isResultLine reports whether trimmed begins with a well-formed "ok" or
// "not ok" token (followed by end-of-line or whitespace), rejecting words such
// as "okapi".
func isResultLine(trimmed string) bool {
	if _, ok := afterToken(trimmed, "not ok"); ok {
		return true
	}
	if _, ok := afterToken(trimmed, "ok"); ok {
		return true
	}
	return false
}

// afterToken returns the remainder of s after a leading token, requiring the
// token to be followed by whitespace or end-of-line so partial words do not
// match.
func afterToken(s, token string) (string, bool) {
	if !strings.HasPrefix(s, token) {
		return "", false
	}
	rest := s[len(token):]
	if rest == "" {
		return "", true
	}
	if rest[0] == ' ' || rest[0] == '\t' {
		return strings.TrimLeft(rest, " \t"), true
	}
	return "", false
}

// splitNumber separates a leading integer test number from the rest of a
// description. It returns the number (as text, empty if absent) and the
// remaining description.
func splitNumber(desc string) (number, rest string) {
	i := 0
	for i < len(desc) && desc[i] >= '0' && desc[i] <= '9' {
		i++
	}
	if i == 0 {
		return "", desc
	}
	// Require the number to be followed by whitespace or end so we do not chop
	// digits off a description like "3rd case".
	if i < len(desc) && desc[i] != ' ' && desc[i] != '\t' && desc[i] != '-' {
		return "", desc
	}
	num := desc[:i]
	r := strings.TrimSpace(desc[i:])
	// A conventional "- description" separator may follow the number.
	r = strings.TrimPrefix(r, "- ")
	return num, strings.TrimSpace(r)
}

// directiveKind classifies a directive body as SKIP, TODO, or none. The
// comparison is case-insensitive per the TAP spec.
type directive int

const (
	dirNone directive = iota
	dirSkip
	dirTodo
)

func directiveKind(d string) directive {
	up := strings.ToUpper(strings.TrimSpace(d))
	switch {
	case strings.HasPrefix(up, "SKIP"):
		return dirSkip
	case strings.HasPrefix(up, "TODO"):
		return dirTodo
	default:
		return dirNone
	}
}

// directiveReason strips the leading SKIP/TODO keyword and returns the trailing
// human-readable reason.
func directiveReason(d string) string {
	d = strings.TrimSpace(d)
	fields := strings.SplitN(d, " ", 2)
	if len(fields) == 2 {
		return strings.TrimSpace(fields[1])
	}
	return ""
}

// firstNonEmpty returns a if it is non-empty, otherwise b.
func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
