// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fileconsumer // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"

import (
	"bufio"
	"errors"
	"io"

	stanzaerrors "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/errors"
)

const defaultBufSize = 16 * 1024

// PositionalScanner is a scanner that maintains position
type PositionalScanner struct {
	pos int64
	*bufio.Scanner
}

// NewPositionalScanner creates a new positional scanner
func NewPositionalScanner(r io.Reader, maxLogSize int, startOffset int64, splitFunc bufio.SplitFunc) *PositionalScanner {
	ps := &PositionalScanner{
		pos:     startOffset,
		Scanner: bufio.NewScanner(r),
	}

	buf := make([]byte, 0, defaultBufSize)
	ps.Scanner.Buffer(buf, maxLogSize*2)

	scanFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = splitFunc(data, atEOF)
		if (advance == 0 && token == nil && err == nil) && len(data) >= 2*maxLogSize {
			// reference: https://pkg.go.dev/bufio#SplitFunc
			// splitFunc returns (0, nil, nil) to signal the Scanner to read more data but the buffer is full.
			// Truncate the log entry.
			advance, token, err = maxLogSize, data[:maxLogSize], nil
		} else if len(token) > maxLogSize {
			advance, token = maxLogSize, token[:maxLogSize]
		}
		ps.pos += int64(advance)
		return
	}
	ps.Scanner.Split(scanFunc)
	return ps
}

// Pos returns the current position of the scanner
func (ps *PositionalScanner) Pos() int64 {
	return ps.pos
}

func (ps *PositionalScanner) getError() error {
	err := ps.Err()
	if errors.Is(err, bufio.ErrTooLong) {
		return stanzaerrors.NewError("log entry too large", "increase max_log_size or ensure that multiline regex patterns terminate")
	} else if err != nil {
		return stanzaerrors.Wrap(err, "scanner error")
	}
	return nil
}
