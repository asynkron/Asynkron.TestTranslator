// Package xmlguard bounds the element-nesting depth of an XML document before it
// is unmarshalled. Go's encoding/xml decoder recurses through nested elements
// (including via Skip on unknown ones), so a pathologically deep document could
// otherwise drive unbounded recursion and exhaust the stack. A single cheap
// token pass rejects such inputs up front.
package xmlguard

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// DefaultMaxDepth is a generous ceiling: far above any legitimate test-result or
// coverage report, but low enough to stop a stack-exhaustion attempt.
const DefaultMaxDepth = 256

// Check streams data once and returns an error if element nesting exceeds max.
// Malformed XML is not this function's concern; it defers to the caller's own
// decode, which reports parse errors.
func Check(data []byte, max int) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	depth := 0
	for {
		tok, err := dec.RawToken()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			// Leave malformed-input reporting to the real decode.
			return nil
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
			if depth > max {
				return fmt.Errorf("xml nesting depth exceeds %d", max)
			}
		case xml.EndElement:
			depth--
		}
	}
}
