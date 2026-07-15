package xmlguard

import (
	"strings"
	"testing"
)

func TestCheckRejectsDeepNesting(t *testing.T) {
	var b strings.Builder
	depth := DefaultMaxDepth + 50
	for i := 0; i < depth; i++ {
		b.WriteString("<a>")
	}
	for i := 0; i < depth; i++ {
		b.WriteString("</a>")
	}
	if err := Check([]byte(b.String()), DefaultMaxDepth); err == nil {
		t.Fatalf("expected an error for %d-deep XML", depth)
	}
}

func TestCheckAllowsShallow(t *testing.T) {
	doc := `<root><a><b>x</b><c/></a></root>`
	if err := Check([]byte(doc), DefaultMaxDepth); err != nil {
		t.Fatalf("unexpected error for shallow XML: %v", err)
	}
}

// Self-closing elements must not leak depth (they open and close in one token).
func TestCheckSelfClosingBalanced(t *testing.T) {
	var b strings.Builder
	b.WriteString("<root>")
	for i := 0; i < DefaultMaxDepth*4; i++ {
		b.WriteString("<x/>")
	}
	b.WriteString("</root>")
	if err := Check([]byte(b.String()), DefaultMaxDepth); err != nil {
		t.Fatalf("sibling self-closing elements should not exceed depth: %v", err)
	}
}
