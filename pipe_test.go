package main

import (
	"bytes"
	"io"
	"testing"
)

type BatchCacheReadLine struct {
	data []string
}

func NewBatchCacheReadLine(data []string) *BatchCacheReadLine {
	return &BatchCacheReadLine{data}
}

func (b *BatchCacheReadLine) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, io.EOF
	}

	thisBatch := []byte(b.data[0])
	batchLength := len(thisBatch)
	for i := 0; i < batchLength; i++ {
		p[i] = thisBatch[i]
	}

	b.data = b.data[1:]

	return batchLength, nil
}

func Test_inPipe(t *testing.T) {
	testcases := []struct {
		name               string
		commands           []string
		wantOut            string
		wantRows, wantCols int
	}{
		{
			name: "Simple 1",
			commands: []string{
				"this is simple command",
			},
			wantOut:  "this is simple command",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Simple 2",
			commands: []string{
				"this is \nsimple command\n\n",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\nwith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Simple 3",
			commands: []string{
				"this is \nsimple command\n\n",
				"with $ a lot many more",
				"with a l%ot many more",
				"w",
			},
			wantOut:  "this is \nsimple command\n\nwith $ a lot many morewith a l%ot many morew",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event entire 1",
			commands: []string{
				"\x1B[8;120;32t",
			},
			wantOut:  "",
			wantRows: 120, wantCols: 32,
		},
		{
			name: "Event entire 2",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;120;32t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\nwith \rmultiple lines\n\n",
			wantRows: 120, wantCols: 32,
		},
		{
			name: "Event entire 3",
			commands: []string{
				"this is \n",
				"simple command\n\n\x1B[8;120;32t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\nwith \rmultiple lines\n\n",
			wantRows: 120, wantCols: 32,
		},
		{
			name: "Event entire 4",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;120;32twith \rmu",
				"ltiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\nwith \rmultiple lines\n\n",
			wantRows: 120, wantCols: 32,
		},
		{
			name: "Event entire 5",
			commands: []string{
				"this is \nsim",
				"ple command\n\n\x1B[8;120;32twith \rmu",
				"ltiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\nwith \rmultiple lines\n\n",
			wantRows: 120, wantCols: 32,
		},
		{
			name: "Event split 1",
			commands: []string{
				"\x1B[8;120",
				";32t",
			},
			wantOut:  "\u001B[8;120;32t",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event split 2",
			commands: []string{
				"\x1B[8;",
				"120",
				";32t",
			},
			wantOut:  "\u001B[8;120;32t",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event split 3",
			commands: []string{
				"\x1B[8;120",
				";32",
				"t",
			},
			wantOut:  "\u001B[8;120;32t",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event split 4",
			commands: []string{
				"\x1B[8;",
				"120",
				";32",
				"t",
			},
			wantOut:  "\u001B[8;120;32t",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event split 5",
			commands: []string{
				"\x1B[8;",
				"120",
				";32",
				"t\x1B[8;",
				"121",
				";33t\x1B",
				"[8;120",
				";32",
			},
			wantOut:  "\u001B[8;120;32t\u001B[8;121;33t\u001B[8;120;32",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event wrong 1",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;120;32;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;120;32;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event wrong 2",
			commands: []string{
				"this is \nsimple ",
				"command\n\n\x1B[8;120;32;0twith \rmul",
				"tiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;120;32;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event wrong 3",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event wrong 4",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;a;b;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;a;b;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event wrong 5",
			commands: []string{
				"\x1B[8;123;321t",
				"this is \nsimple command\n\n",
				"\x1B[8;120;32;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;120;32;0twith \rmultiple lines\n\n",
			wantRows: 123, wantCols: 321,
		},
		{
			name: "Event wrong 6",
			commands: []string{
				"\x1B[8;121\x1B[9;\x1B[8;122;33\x1B[8;12435t\x1B[8;126;37",
			},
			wantOut:  "\u001B[8;121\u001B[9;\u001B[8;122;33\u001B[8;12435t\u001B[8;126;37",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event irrelevant 1",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[6;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[6;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event irrelevant 2",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;\x1B]8;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;\u001B]8;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event irrelevant 3",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;\x1D]8;0t",
				"with \rmultiple lines\n\n",
			},
			wantOut:  "this is \nsimple command\n\n\u001B[8;\u001D]8;0twith \rmultiple lines\n\n",
			wantRows: 0, wantCols: 0,
		},
		{
			name: "Event dup 1",
			commands: []string{
				"\x1B[8;120;32t",
				"\x1B[8;32;120t",
			},
			wantOut:  "",
			wantRows: 32, wantCols: 120,
		},
		{
			name: "Event dup 2",
			commands: []string{
				"\x1B[8;120;32t\x1B[8;32;120t",
			},
			wantOut:  "",
			wantRows: 32, wantCols: 120,
		},
		{
			name: "Event dup 3",
			commands: []string{
				"this is \nsimple command\n\n",
				"\x1B[8;120;32t",
				"with \rmultiple lines\n\n",
				"\x1B[8;32;120t",
			},
			wantOut:  "this is \nsimple command\n\nwith \rmultiple lines\n\n",
			wantRows: 32, wantCols: 120,
		},
		{
			name: "Event dup 4",
			commands: []string{
				"this is \nsimple command\n\n\x1B[8;120;32twith \rsingle line\n\n\x1B[8;32;120t",
			},
			wantOut:  "this is \nsimple command\n\nwith \rsingle line\n\n",
			wantRows: 32, wantCols: 120,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			rows, cols := 0, 0

			windowResize := func(h, w int) error {
				rows, cols = h, w
				return nil
			}

			r := NewBatchCacheReadLine(testcase.commands)

			resBuf := bytes.NewBuffer(nil)

			// Exec and collect error
			if err := inPipe(r, resBuf, windowResize); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Compare out
			resStr := resBuf.String()
			if resStr != testcase.wantOut {
				t.Errorf("Unexpected output: expected %q, got %q", testcase.wantOut, resStr)
			}

			// Compare window size
			if rows != testcase.wantRows || cols != testcase.wantCols {
				t.Errorf("Unexpected window size: expected %dx%d, got %dx%d", testcase.wantRows, testcase.wantCols, rows, cols)
			}

		})
	}

}
