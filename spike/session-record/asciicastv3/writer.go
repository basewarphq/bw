// Package asciicastv3 implements a writer for the asciicast v3 file format.
//
// asciicast v3 is the recording format used by asciinema CLI 3.0+.
// It uses newline-delimited JSON with relative timestamps (intervals)
// between events, unlike v2 which used absolute timestamps.
//
// Spec: https://docs.asciinema.org/manual/asciicast/v3/
package asciicastv3

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// Event codes defined by the asciicast v3 spec.
const (
	CodeOutput = "o" // data written to terminal
	CodeInput  = "i" // data read from terminal
	CodeMarker = "m" // marker/breakpoint
	CodeResize = "r" // terminal resize
	CodeExit   = "x" // session exit status
)

// Header is the asciicast v3 header (first line of the file).
type Header struct {
	Version   int               `json:"version"`
	Term      TermInfo          `json:"term"`
	Timestamp int64             `json:"timestamp,omitempty"`
	Command   string            `json:"command,omitempty"`
	Title     string            `json:"title,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// TermInfo holds terminal metadata nested under "term" in the header.
type TermInfo struct {
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
	Type string `json:"type,omitempty"`
}

// Event is a single asciicast v3 event: [interval, code, data].
type Event struct {
	Interval float64
	Code     string
	Data     string
}

// MarshalJSON encodes an Event as a JSON array [interval, code, data].
func (e Event) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{e.Interval, e.Code, e.Data})
}

// Writer writes asciicast v3 format to an underlying io.Writer.
// It is safe for concurrent use.
type Writer struct {
	mu            sync.Mutex
	w             io.Writer
	lastEventTime time.Time
	headerWritten bool
}

// NewWriter creates a new asciicast v3 Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteHeader writes the asciicast v3 header. Must be called before any events.
func (w *Writer) WriteHeader(header Header) error {
	header.Version = 3
	if header.Timestamp == 0 {
		header.Timestamp = time.Now().Unix()
	}

	data, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("asciicastv3: marshal header: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.w.Write(data); err != nil {
		return err
	}
	if _, err := w.w.Write([]byte("\n")); err != nil {
		return err
	}

	w.lastEventTime = time.Now()
	w.headerWritten = true
	return nil
}

// WriteEvent writes a single event with the given code and data.
// The interval is computed automatically from the time since the last event.
func (w *Writer) WriteEvent(code, data string) error {
	now := time.Now()

	w.mu.Lock()
	defer w.mu.Unlock()

	interval := now.Sub(w.lastEventTime).Seconds()
	if interval < 0 {
		interval = 0
	}
	w.lastEventTime = now

	event := Event{Interval: interval, Code: code, Data: data}
	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("asciicastv3: marshal event: %w", err)
	}

	if _, err := w.w.Write(line); err != nil {
		return err
	}
	_, err = w.w.Write([]byte("\n"))
	return err
}

// WriteOutput writes an output event ("o").
func (w *Writer) WriteOutput(data []byte) error {
	return w.WriteEvent(CodeOutput, string(data))
}

// WriteInput writes an input event ("i").
func (w *Writer) WriteInput(data []byte) error {
	return w.WriteEvent(CodeInput, string(data))
}

// WriteResize writes a resize event ("r") with format "COLSxROWS".
func (w *Writer) WriteResize(cols, rows int) error {
	return w.WriteEvent(CodeResize, fmt.Sprintf("%dx%d", cols, rows))
}

// WriteExit writes an exit event ("x") with the process exit status.
func (w *Writer) WriteExit(status int) error {
	return w.WriteEvent(CodeExit, fmt.Sprintf("%d", status))
}

// OutputWriter returns an io.Writer that writes output events for each Write call.
func (w *Writer) OutputWriter() io.Writer {
	return &eventWriter{w: w, code: CodeOutput}
}

// InputWriter returns an io.Writer that writes input events for each Write call.
func (w *Writer) InputWriter() io.Writer {
	return &eventWriter{w: w, code: CodeInput}
}

// eventWriter adapts Writer to the io.Writer interface for a specific event code.
type eventWriter struct {
	w    *Writer
	code string
}

func (ew *eventWriter) Write(p []byte) (int, error) {
	if err := ew.w.WriteEvent(ew.code, string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}
