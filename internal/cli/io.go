package cli

import (
	"fmt"
	"io"
)

// maxInputBytes bounds any single input (stdin or a file) so it cannot exhaust
// memory before an adapter's own decoding limit applies.
const maxInputBytes = 1 << 30 // 1 GiB

// readAll reads r with an upper bound, erroring if the bound is exceeded.
func readAll(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("no input stream")
	}
	lr := &io.LimitedReader{R: r, N: maxInputBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxInputBytes {
		return nil, fmt.Errorf("input exceeds %d byte limit", maxInputBytes)
	}
	return data, nil
}
