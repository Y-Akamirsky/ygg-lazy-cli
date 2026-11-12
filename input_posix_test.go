//go:build !windows

package main

import (
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
)

func TestParseEscapeArrowsAndDelete(t *testing.T) {
	cases := []struct {
		input string
		want  rune
	}{
		{"\x1b[A", terminal.KeyArrowUp},
		{"\x1b[B", terminal.KeyArrowDown},
		{"\x1b[C", terminal.KeyArrowRight},
		{"\x1b[D", terminal.KeyArrowLeft},
		{"\x1bOA", terminal.KeyArrowUp},
		{"\x1b[3~", terminal.SpecialKeyDelete},
		{"\x1b[H", terminal.SpecialKeyHome},
		{"\x1b[F", terminal.SpecialKeyEnd},
	}

	for _, tc := range cases {
		ev, size, complete := parseEscape([]byte(tc.input))
		if !complete {
			t.Fatalf("%q was not parsed as a complete sequence", tc.input)
		}
		if size != len(tc.input) {
			t.Fatalf("%q consumed %d bytes, want %d", tc.input, size, len(tc.input))
		}
		if ev.mouse != nil || ev.key != tc.want {
			t.Fatalf("%q = (%q, %+v), want key %q", tc.input, ev.key, ev.mouse, tc.want)
		}
	}
}

func TestParseEscapeMouse(t *testing.T) {
	cases := []struct {
		input  string
		button mouseButton
		x, y   int
	}{
		{"\x1b[<0;12;7M", mouseLeft, 12, 7},
		{"\x1b[<64;3;4M", mouseWheelUp, 3, 4},
		{"\x1b[<65;3;4M", mouseWheelDown, 3, 4},
	}

	for _, tc := range cases {
		ev, size, complete := parseEscape([]byte(tc.input))
		if !complete || size != len(tc.input) {
			t.Fatalf("%q: complete=%v size=%d, want complete with %d bytes", tc.input, complete, size, len(tc.input))
		}
		if ev.mouse == nil {
			t.Fatalf("%q did not produce a mouse event", tc.input)
		}
		if ev.mouse.button != tc.button || ev.mouse.x != tc.x || ev.mouse.y != tc.y {
			t.Fatalf("%q = %+v, want button %d at %d;%d", tc.input, *ev.mouse, tc.button, tc.x, tc.y)
		}
	}

	// Button release must not be reported as a click.
	ev, _, complete := parseEscape([]byte("\x1b[<0;12;7m"))
	if !complete || ev.mouse != nil || ev.key != terminal.IgnoreKey {
		t.Fatalf("release = (%q, %+v), want ignored key", ev.key, ev.mouse)
	}
}

func TestParseEscapeIncomplete(t *testing.T) {
	for _, input := range []string{"\x1b", "\x1b[", "\x1b[<0;12"} {
		if _, _, complete := parseEscape([]byte(input)); complete {
			t.Fatalf("%q must wait for more bytes", input)
		}
	}
}

func TestFindCursorReport(t *testing.T) {
	row, start, end, ok := findCursorReport([]byte("\x1b[12;40R"))
	if !ok || row != 12 || start != 0 || end != 8 {
		t.Fatalf("report = (%d, %d, %d, %v), want (12, 0, 8, true)", row, start, end, ok)
	}

	// Pending mouse events before the report must stay in the buffer.
	buf := []byte("\x1b[<0;3;4M\x1b[12;40R")
	row, start, end, ok = findCursorReport(buf)
	if !ok || row != 12 || start != 9 {
		t.Fatalf("report = (%d, %d, %d, %v), want row 12 at 9", row, start, end, ok)
	}
	ev, _, complete := parseEscape(buf[:start])
	if !complete || ev.mouse == nil || ev.mouse.x != 3 || ev.mouse.y != 4 {
		t.Fatalf("pending event was lost: %+v", ev)
	}
}
