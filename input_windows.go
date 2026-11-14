//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/AlecAivazis/survey/v2/terminal"
)

const (
	enableMouseInput uint32 = 0x0010
	enableQuickEdit  uint32 = 0x0040
	enableExtended   uint32 = 0x0080

	eventKey   = 0x0001
	eventMouse = 0x0002

	mouseMoved  = 0x0001
	mouseWheel  = 0x0004
	mouseHWheel = 0x0008

	fromLeftFirstButton = 0x0001

	vkDelete = 0x2E
	vkEnd    = 0x23
	vkHome   = 0x24
	vkLeft   = 0x25
	vkUp     = 0x26
	vkRight  = 0x27
	vkDown   = 0x28
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procReadConsoleInputW          = kernel32.NewProc("ReadConsoleInputW")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type coord struct {
	x int16
	y int16
}

type smallRect struct {
	left   int16
	top    int16
	right  int16
	bottom int16
}

type screenBufferInfo struct {
	size              coord
	cursorPosition    coord
	attributes        uint16
	window            smallRect
	maximumWindowSize coord
}

type inputRecord struct {
	eventType uint16
	padding   uint16
	event     [16]byte
}

type keyEventRecord struct {
	keyDown         int32
	repeatCount     uint16
	virtualKeyCode  uint16
	virtualScanCode uint16
	unicodeChar     uint16
	controlKeyState uint32
}

type mouseEventRecord struct {
	position        coord
	buttonState     uint32
	controlKeyState uint32
	eventFlags      uint32
}

// inputReader reads keyboard and mouse events from the Windows console.
type inputReader struct {
	fd          int
	term        *terminal.RuneReader
	mouse       bool
	prevLeftBtn bool
}

func newInputReader(mouse bool) *inputReader {
	fd := int(os.Stdin.Fd())
	r := &inputReader{fd: fd}

	tr := terminal.NewRuneReader(terminal.Stdio{In: os.Stdin, Out: os.Stdout, Err: os.Stderr})
	if err := tr.SetTermMode(); err != nil {
		return r
	}
	r.term = tr

	if mouse {
		var mode uint32
		if ok, _, _ := procGetConsoleMode.Call(uintptr(fd), uintptr(unsafe.Pointer(&mode))); ok != 0 {
			// Quick edit would swallow clicks before the program sees them.
			mode |= enableMouseInput | enableExtended
			mode &^= enableQuickEdit
			if ok, _, _ := procSetConsoleMode.Call(uintptr(fd), uintptr(mode)); ok != 0 {
				r.mouse = true
			}
		}
	}
	return r
}

func (r *inputReader) Close() {
	if r.term != nil {
		_ = r.term.RestoreTermMode()
		r.term = nil
	}
}

func (r *inputReader) ReadEvent() (inputEvent, error) {
	rec := inputRecord{}
	var read uint32

	for {
		ok, _, err := procReadConsoleInputW.Call(
			uintptr(r.fd),
			uintptr(unsafe.Pointer(&rec)),
			1,
			uintptr(unsafe.Pointer(&read)),
		)
		if ok == 0 {
			return inputEvent{}, err
		}

		switch rec.eventType {
		case eventKey:
			key := (*keyEventRecord)(unsafe.Pointer(&rec.event[0]))
			if key.keyDown == 0 {
				continue
			}
			if key.unicodeChar != 0 {
				return inputEvent{key: rune(key.unicodeChar)}, nil
			}
			switch key.virtualKeyCode {
			case vkUp:
				return inputEvent{key: terminal.KeyArrowUp}, nil
			case vkDown:
				return inputEvent{key: terminal.KeyArrowDown}, nil
			case vkLeft:
				return inputEvent{key: terminal.KeyArrowLeft}, nil
			case vkRight:
				return inputEvent{key: terminal.KeyArrowRight}, nil
			case vkHome:
				return inputEvent{key: terminal.SpecialKeyHome}, nil
			case vkEnd:
				return inputEvent{key: terminal.SpecialKeyEnd}, nil
			case vkDelete:
				return inputEvent{key: terminal.SpecialKeyDelete}, nil
			}
		case eventMouse:
			if ev, ok := r.mouseEvent((*mouseEventRecord)(unsafe.Pointer(&rec.event[0]))); ok {
				return ev, nil
			}
		}
	}
}

func (r *inputReader) mouseEvent(m *mouseEventRecord) (inputEvent, bool) {
	left := m.buttonState&fromLeftFirstButton != 0
	wasLeft := r.prevLeftBtn
	r.prevLeftBtn = left

	if !r.mouse || m.eventFlags&mouseMoved != 0 {
		return inputEvent{}, false
	}

	if m.eventFlags&(mouseWheel|mouseHWheel) != 0 {
		delta := int16(m.buttonState >> 16)
		ev := mouseEvent{x: int(m.position.x) + 1, y: int(m.position.y) + 1}
		switch {
		case delta > 0:
			ev.button = mouseWheelUp
		case delta < 0:
			ev.button = mouseWheelDown
		default:
			return inputEvent{}, false
		}
		return inputEvent{mouse: &ev}, true
	}

	// Report the press edge only, so a click does not fire twice.
	if left && !wasLeft {
		ev := mouseEvent{x: int(m.position.x) + 1, y: int(m.position.y) + 1, button: mouseLeft}
		return inputEvent{mouse: &ev}, true
	}
	return inputEvent{}, false
}

// LastRow returns the 1-based row the cursor is on. Mouse events use the same
// screen buffer coordinates, so both values can be compared directly.
func (r *inputReader) LastRow() (int, error) {
	var info screenBufferInfo
	ok, _, err := procGetConsoleScreenBufferInfo.Call(
		uintptr(os.Stdout.Fd()),
		uintptr(unsafe.Pointer(&info)),
	)
	if ok == 0 {
		return 0, err
	}
	return int(info.cursorPosition.y) + 1, nil
}
