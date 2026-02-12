// Package snapshot extracts timestamped screen snapshots from asciicast v3
// recordings by feeding events through a VT100 terminal emulator.
package snapshot

import (
	"fmt"
	"io"
	"strings"

	"github.com/basewarphq/bw/spike/session-record/asciicastv3"
	"github.com/hinshun/vt10x"
)

// Snapshot holds the visible terminal text captured at a point in time.
type Snapshot struct {
	Time float64
	Text string
}

// UserInput holds text typed by the user between Enter presses.
type UserInput struct {
	Time float64
	Text string
}

// Result holds the complete extraction output.
type Result struct {
	Snapshots []Snapshot
	Inputs    []UserInput
}

// DumpScreen extracts visible text from the virtual terminal, trimming
// trailing whitespace from each line and trailing empty lines.
func DumpScreen(term vt10x.Terminal) string {
	term.Lock()
	defer term.Unlock()

	cols, rows := term.Size()
	lines := make([]string, 0, rows)
	for y := 0; y < rows; y++ {
		var line strings.Builder
		for x := 0; x < cols; x++ {
			g := term.Cell(x, y)
			if g.Char == 0 {
				line.WriteRune(' ')
			} else {
				line.WriteRune(g.Char)
			}
		}
		lines = append(lines, strings.TrimRight(line.String(), " "))
	}

	// Trim trailing empty lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// Extract reads an asciicast v3 stream from r, feeds output events through a
// VT100 terminal emulator, and returns timestamped screen snapshots along with
// reconstructed user input segments.
//
// Snapshots are taken on input events containing a newline, periodically at
// snapshotInterval seconds of recording time, and at the end of the stream.
// Consecutive identical snapshots are deduplicated.
//
// User inputs are reconstructed by collecting printable characters from input
// events between Enter presses, filtering out escape sequences and control
// characters (terminal queries, cursor reports, etc.).
func Extract(r io.Reader, snapshotInterval float64) (Result, error) {
	reader := asciicastv3.NewReader(r)
	header, err := reader.ReadHeader()
	if err != nil {
		return Result{}, fmt.Errorf("snapshot: reading header: %w", err)
	}

	cols := header.Term.Cols
	rows := header.Term.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	// Create virtual terminal emulator.
	term := vt10x.New(vt10x.WithSize(cols, rows))

	var (
		elapsed          float64 // total elapsed recording time
		lastSnapshotTime float64
		lastSnapshot     string
		snapshots        []Snapshot
		inputs           []UserInput
		inputBuf         strings.Builder // accumulates typed text between Enters
		inputStartTime   float64         // time of first keystroke in current segment
	)

	for {
		ev, err := reader.ReadEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Result{}, fmt.Errorf("snapshot: reading event: %w", err)
		}

		elapsed += ev.Interval

		switch ev.Code {
		case asciicastv3.CodeOutput:
			term.Write([]byte(ev.Data))

		case asciicastv3.CodeResize:
			var newCols, newRows int
			if _, err := fmt.Sscanf(ev.Data, "%dx%d", &newCols, &newRows); err == nil {
				term.Resize(newCols, newRows)
			}

		case asciicastv3.CodeInput:
			// Collect user-typed text from input events.
			for _, r := range ev.Data {
				if r == '\r' || r == '\n' {
					// Enter pressed — flush accumulated input.
					typed := strings.TrimSpace(inputBuf.String())
					if typed != "" {
						inputs = append(inputs, UserInput{Time: inputStartTime, Text: typed})
					}
					inputBuf.Reset()
					inputStartTime = 0

					// Also take a screen snapshot on Enter.
					snap := DumpScreen(term)
					if snap != lastSnapshot && strings.TrimSpace(snap) != "" {
						snapshots = append(snapshots, Snapshot{Time: elapsed, Text: snap})
						lastSnapshot = snap
						lastSnapshotTime = elapsed
					}
				} else if r == 0x7f || r == '\b' {
					// Backspace — remove last character from buffer.
					s := inputBuf.String()
					if len(s) > 0 {
						s = s[:len(s)-1]
						inputBuf.Reset()
						inputBuf.WriteString(s)
					}
				} else if r == 0x1b {
					// Start of escape sequence — skip this event's remaining
					// characters as they're likely a terminal control sequence.
					break
				} else if r >= 0x20 {
					// Printable character.
					if inputBuf.Len() == 0 {
						inputStartTime = elapsed
					}
					inputBuf.WriteRune(r)
				}
			}
		}

		// Periodic snapshot.
		if elapsed-lastSnapshotTime >= snapshotInterval {
			snap := DumpScreen(term)
			if snap != lastSnapshot && strings.TrimSpace(snap) != "" {
				snapshots = append(snapshots, Snapshot{Time: elapsed, Text: snap})
				lastSnapshot = snap
			}
			lastSnapshotTime = elapsed
		}
	}

	// Flush any remaining input.
	typed := strings.TrimSpace(inputBuf.String())
	if typed != "" {
		inputs = append(inputs, UserInput{Time: elapsed, Text: typed})
	}

	// Final snapshot.
	finalSnap := DumpScreen(term)
	if finalSnap != lastSnapshot && strings.TrimSpace(finalSnap) != "" {
		snapshots = append(snapshots, Snapshot{Time: elapsed, Text: finalSnap})
	}

	return Result{Snapshots: snapshots, Inputs: inputs}, nil
}
