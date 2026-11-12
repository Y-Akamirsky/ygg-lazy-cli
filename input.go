package main

// mouseButton identifies which mouse control produced an event.
type mouseButton int

const (
	mouseLeft mouseButton = iota
	mouseWheelUp
	mouseWheelDown
)

// mouseEvent is a click or wheel action in terminal coordinates (1-based).
type mouseEvent struct {
	x, y   int
	button mouseButton
}

// inputEvent is either a key press (key) or a mouse action (mouse).
type inputEvent struct {
	key   rune
	mouse *mouseEvent
}
