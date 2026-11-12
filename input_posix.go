//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlecAivazis/survey/v2/terminal"
)

const (
	// escTimeout is how long to wait for the rest of an escape sequence
	// before treating a lone ESC as the Escape key.
	escTimeout = 40 * time.Millisecond
	// reportTimeout bounds the cursor position report request.
	reportTimeout = 300 * time.Millisecond
)

// inputReader reads keyboard and mouse events from a raw POSIX terminal.
// Mouse events arrive as SGR sequences (ESC [ < b ; x ; y M) because
// enableMouse turns on normal tracking plus the SGR protocol.
type inputReader struct {
	fd           int
	term         *terminal.RuneReader
	buf          []byte
	mouseEnabled bool
}

func newInputReader(mouse bool) *inputReader {
	fd := int(os.Stdin.Fd())
	r := &inputReader{fd: fd}

	// Use survey's raw mode instead of x/term.MakeRaw: it keeps output
	// post-processing enabled, so "\n" still returns to column zero.
	tr := terminal.NewRuneReader(terminal.Stdio{In: os.Stdin, Out: os.Stdout, Err: os.Stderr})
	if err := tr.SetTermMode(); err == nil {
		r.term = tr
	}

	if r.term != nil && mouse {
		fmt.Fprint(os.Stdout, "\x1b[?1000h\x1b[?1006h")
		r.mouseEnabled = true
	}
	return r
}

func (r *inputReader) Close() {
	if r.mouseEnabled {
		fmt.Fprint(os.Stdout, "\x1b[?1006l\x1b[?1000l")
		r.mouseEnabled = false
	}
	if r.term != nil {
		_ = r.term.RestoreTermMode()
		r.term = nil
	}
}

// fill appends available input bytes to the buffer. A positive timeout makes
// the read return when nothing arrives in time.
func (r *inputReader) fill(timeout time.Duration) error {
	if timeout > 0 {
		_ = os.Stdin.SetReadDeadline(time.Now().Add(timeout))
	}

	var chunk [256]byte
	n, err := os.Stdin.Read(chunk[:])

	if timeout > 0 {
		_ = os.Stdin.SetReadDeadline(time.Time{})
	}
	if n > 0 {
		r.buf = append(r.buf, chunk[:n]...)
	}
	if err != nil && n == 0 {
		return err
	}
	return nil
}

func (r *inputReader) ReadEvent() (inputEvent, error) {
	for {
		if len(r.buf) == 0 {
			if err := r.fill(0); err != nil {
				return inputEvent{}, err
			}
			continue
		}

		if r.buf[0] == byte(terminal.KeyEscape) {
			if ev, size, complete := parseEscape(r.buf); complete {
				r.buf = r.buf[size:]
				return ev, nil
			}

			// Give the rest of the sequence a moment to arrive; if it does
			// not, this was a plain Escape key press.
			if err := r.fill(escTimeout); err != nil || len(r.buf) == 1 {
				r.buf = r.buf[1:]
				return inputEvent{key: terminal.KeyEscape}, nil
			}
			continue
		}

		b := r.buf[0]
		if b < utf8.RuneSelf {
			r.buf = r.buf[1:]
			return inputEvent{key: mapControlByte(b)}, nil
		}

		if !utf8.FullRune(r.buf) {
			if err := r.fill(0); err != nil {
				return inputEvent{}, err
			}
			continue
		}
		rn, size := utf8.DecodeRune(r.buf)
		r.buf = r.buf[size:]
		return inputEvent{key: rn}, nil
	}
}

// LastRow asks the terminal for the cursor position and returns its 1-based
// row. The reply is read from the same input stream, so any other pending
// bytes stay in the buffer for the next ReadEvent call.
func (r *inputReader) LastRow() (int, error) {
	if r.term == nil {
		return 0, errors.New("cursor position is unavailable")
	}

	fmt.Fprint(os.Stdout, "\x1b[6n")
	deadline := time.Now().Add(reportTimeout)
	for {
		if row, start, end, ok := findCursorReport(r.buf); ok {
			r.buf = append(r.buf[:start], r.buf[end:]...)
			return row, nil
		}
		if time.Now().After(deadline) {
			return 0, errors.New("no cursor position report")
		}
		_ = r.fill(50 * time.Millisecond)
	}
}

// findCursorReport looks for ESC [ row ; column R anywhere in buf and returns
// the row plus the start and end offsets of the report.
func findCursorReport(buf []byte) (int, int, int, bool) {
	for i := 0; i < len(buf); i++ {
		if buf[i] != byte(terminal.KeyEscape) || i+1 >= len(buf) || buf[i+1] != '[' {
			continue
		}
		j := i + 2
		first, ok := readNumber(buf, &j)
		if !ok || j >= len(buf) || buf[j] != ';' {
			continue
		}
		j++
		if _, ok := readNumber(buf, &j); !ok || j >= len(buf) || buf[j] != 'R' {
			continue
		}
		return first, i, j + 1, true
	}
	return 0, 0, 0, false
}

func readNumber(buf []byte, pos *int) (int, bool) {
	start := *pos
	value := 0
	for *pos < len(buf) && buf[*pos] >= '0' && buf[*pos] <= '9' {
		value = value*10 + int(buf[*pos]-'0')
		*pos++
	}
	if *pos == start {
		return 0, false
	}
	return value, true
}

// parseEscape decodes one escape sequence. It reports complete=false when the
// buffer ends in the middle of a sequence.
func parseEscape(buf []byte) (inputEvent, int, bool) {
	if len(buf) < 2 {
		return inputEvent{}, 0, false
	}

	switch buf[1] {
	case '[':
		for i := 2; i < len(buf); i++ {
			b := buf[i]
			if b >= 0x40 && b <= 0x7e {
				return decodeCSI(buf[2:i], b), i + 1, true
			}
			if b < 0x20 || b > 0x3f {
				return inputEvent{key: terminal.IgnoreKey}, i + 1, true
			}
		}
		return inputEvent{}, 0, false
	case 'O':
		if len(buf) < 3 {
			return inputEvent{}, 0, false
		}
		switch buf[2] {
		case 'A':
			return inputEvent{key: terminal.KeyArrowUp}, 3, true
		case 'B':
			return inputEvent{key: terminal.KeyArrowDown}, 3, true
		case 'C':
			return inputEvent{key: terminal.KeyArrowRight}, 3, true
		case 'D':
			return inputEvent{key: terminal.KeyArrowLeft}, 3, true
		case 'H':
			return inputEvent{key: terminal.SpecialKeyHome}, 3, true
		case 'F':
			return inputEvent{key: terminal.SpecialKeyEnd}, 3, true
		}
		return inputEvent{key: terminal.IgnoreKey}, 3, true
	default:
		// ESC followed by an unrelated key: return Escape and leave the byte.
		return inputEvent{key: terminal.KeyEscape}, 1, true
	}
}

func decodeCSI(params []byte, final byte) inputEvent {
	switch final {
	case 'A':
		return inputEvent{key: terminal.KeyArrowUp}
	case 'B':
		return inputEvent{key: terminal.KeyArrowDown}
	case 'C':
		return inputEvent{key: terminal.KeyArrowRight}
	case 'D':
		return inputEvent{key: terminal.KeyArrowLeft}
	case 'H':
		return inputEvent{key: terminal.SpecialKeyHome}
	case 'F':
		return inputEvent{key: terminal.SpecialKeyEnd}
	case 'M', 'm':
		if final == 'm' {
			// Button release; only presses are interesting.
			return inputEvent{key: terminal.IgnoreKey}
		}
		if ev, ok := parseSGRMouse(params); ok {
			return ev
		}
		return inputEvent{key: terminal.IgnoreKey}
	case '~':
		switch string(params) {
		case "3":
			return inputEvent{key: terminal.SpecialKeyDelete}
		}
		return inputEvent{key: terminal.IgnoreKey}
	}
	return inputEvent{key: terminal.IgnoreKey}
}

// parseSGRMouse decodes "<button;x;y" from an SGR mouse sequence.
func parseSGRMouse(params []byte) (inputEvent, bool) {
	if len(params) < 2 || params[0] != '<' {
		return inputEvent{}, false
	}
	fields := strings.Split(string(params[1:]), ";")
	if len(fields) != 3 {
		return inputEvent{}, false
	}

	button, err := strconv.Atoi(fields[0])
	if err != nil {
		return inputEvent{}, false
	}
	x, err := strconv.Atoi(fields[1])
	if err != nil {
		return inputEvent{}, false
	}
	y, err := strconv.Atoi(fields[2])
	if err != nil {
		return inputEvent{}, false
	}

	if button&32 != 0 {
		// Motion report; clicks and the wheel are handled below.
		return inputEvent{}, false
	}

	ev := mouseEvent{x: x, y: y}
	switch {
	case button&64 != 0:
		if button&1 == 0 {
			ev.button = mouseWheelUp
		} else {
			ev.button = mouseWheelDown
		}
	case button&3 == 0:
		ev.button = mouseLeft
	default:
		return inputEvent{}, false
	}
	return inputEvent{mouse: &ev}, true
}

func mapControlByte(b byte) rune {
	switch b {
	case '\r', '\n':
		return terminal.KeyEnter
	case 0x7f, '\b':
		return terminal.KeyBackspace
	case 0x03:
		return terminal.KeyInterrupt
	case 0x04:
		return terminal.KeyEndTransmission
	case 0x00:
		return terminal.IgnoreKey
	}
	return rune(b)
}
