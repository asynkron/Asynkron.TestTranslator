package cli

import (
	"fmt"
	"io"
)

// maxStdinBytes bounds stdin so a stream cannot exhaust memory.
const maxStdinBytes = 1 << 30 // 1 GiB

// readAll reads r with an upper bound, erroring if the bound is exceeded.
func readAll(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("no input stream")
	}
	lr := &io.LimitedReader{R: r, N: maxStdinBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxStdinBytes {
		return nil, fmt.Errorf("stdin exceeds %d byte limit", maxStdinBytes)
	}
	return data, nil
}
