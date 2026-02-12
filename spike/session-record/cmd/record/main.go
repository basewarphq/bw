// Command session-record starts a command in a PTY, records all terminal
// I/O to an asciicast v3 file, and lets you interact with the command normally.
//
// Usage:
//
//	session-record [-o output.cast] [--] <command> [args...]
//
// If no command is given, it defaults to "amp".
// If no output file is given, a timestamped filename is generated.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/basewarphq/bw/spike/session-record/asciicastv3"
	"github.com/creack/pty"
	"golang.org/x/term"
)

func main() {
	os.Exit(run())
}

func run() int {
	outputFile := flag.String("o", "", "output .cast file (default: timestamped)")
	recordInput := flag.Bool("i", false, "record input (keystrokes) in addition to output")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"amp"}
	}

	// Resolve output filename.
	castPath := *outputFile
	if castPath == "" {
		castPath = fmt.Sprintf("session-%s.cast", time.Now().Format("20060102-150405"))
	}

	// Ensure stdin is a terminal.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "session-record: stdin is not a terminal")
		return 1
	}

	// Get current terminal size.
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "session-record: get terminal size: %v\n", err)
		return 1
	}

	// Open output file.
	castFile, err := os.Create(castPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "session-record: create %s: %v\n", castPath, err)
		return 1
	}
	defer castFile.Close()

	// Create asciicast writer and write header.
	cast := asciicastv3.NewWriter(castFile)
	header := asciicastv3.Header{
		Term: asciicastv3.TermInfo{
			Cols: cols,
			Rows: rows,
			Type: os.Getenv("TERM"),
		},
		Command: args[0],
		Env: map[string]string{
			"SHELL": os.Getenv("SHELL"),
		},
	}
	if err := cast.WriteHeader(header); err != nil {
		fmt.Fprintf(os.Stderr, "session-record: write header: %v\n", err)
		return 1
	}

	// Start the command in a PTY.
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "session-record: start command: %v\n", err)
		return 1
	}
	defer ptmx.Close()

	// Put stdin into raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "session-record: raw mode: %v\n", err)
		return 1
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Handle SIGWINCH: resize PTY and record resize event.
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			newCols, newRows, err := term.GetSize(int(os.Stdin.Fd()))
			if err != nil {
				continue
			}
			_ = pty.Setsize(ptmx, &pty.Winsize{
				Cols: uint16(newCols),
				Rows: uint16(newRows),
			})
			_ = cast.WriteResize(newCols, newRows)
		}
	}()
	// Trigger an initial resize in case terminal changed between GetSize and pty start.
	sigwinch <- syscall.SIGWINCH

	// Copy stdin → PTY (optionally recording input).
	go func() {
		var w io.Writer = ptmx
		if *recordInput {
			w = io.MultiWriter(ptmx, cast.InputWriter())
		}
		_, _ = io.Copy(w, os.Stdin)
	}()

	// Copy PTY → stdout, recording output.
	_, _ = io.Copy(io.MultiWriter(os.Stdout, cast.OutputWriter()), ptmx)

	// Wait for command to exit.
	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	_ = cast.WriteExit(exitCode)

	// Restore terminal before printing message.
	term.Restore(int(os.Stdin.Fd()), oldState)
	fmt.Fprintf(os.Stderr, "\nsession-record: saved to %s\n", castPath)

	return exitCode
}
