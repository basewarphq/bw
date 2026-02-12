package asciicastv3

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Reader reads asciicast v3 files event by event.
type Reader struct {
	scanner       *bufio.Scanner
	header        Header
	headerDecoded bool
}

// NewReader creates a new asciicast v3 Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{scanner: bufio.NewScanner(r)}
}

// ReadHeader reads and returns the header (first non-comment line).
// Must be called before ReadEvent.
func (r *Reader) ReadHeader() (Header, error) {
	if r.headerDecoded {
		return r.header, nil
	}

	for r.scanner.Scan() {
		line := r.scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &r.header); err != nil {
			return Header{}, fmt.Errorf("asciicastv3: parse header: %w", err)
		}
		if r.header.Version != 3 {
			return Header{}, fmt.Errorf("asciicastv3: expected version 3, got %d", r.header.Version)
		}
		r.headerDecoded = true
		return r.header, nil
	}
	if err := r.scanner.Err(); err != nil {
		return Header{}, fmt.Errorf("asciicastv3: read header: %w", err)
	}
	return Header{}, fmt.Errorf("asciicastv3: empty file, no header found")
}

// ReadEvent reads the next event. Returns io.EOF when no more events.
func (r *Reader) ReadEvent() (Event, error) {
	if !r.headerDecoded {
		if _, err := r.ReadHeader(); err != nil {
			return Event{}, err
		}
	}

	for r.scanner.Scan() {
		line := r.scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		var raw [3]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return Event{}, fmt.Errorf("asciicastv3: parse event: %w", err)
		}

		var ev Event
		if err := json.Unmarshal(raw[0], &ev.Interval); err != nil {
			return Event{}, fmt.Errorf("asciicastv3: parse interval: %w", err)
		}
		if err := json.Unmarshal(raw[1], &ev.Code); err != nil {
			return Event{}, fmt.Errorf("asciicastv3: parse code: %w", err)
		}
		if err := json.Unmarshal(raw[2], &ev.Data); err != nil {
			return Event{}, fmt.Errorf("asciicastv3: parse data: %w", err)
		}
		return ev, nil
	}

	if err := r.scanner.Err(); err != nil {
		return Event{}, fmt.Errorf("asciicastv3: read event: %w", err)
	}
	return Event{}, io.EOF
}
