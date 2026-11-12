package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/term"
)

// errPromptInterrupt is returned when the user presses Ctrl+C.
var errPromptInterrupt = terminal.InterruptErr

// --- List state ---

// listModel keeps the state of a Select/MultiSelect prompt so navigation and
// filtering can be tested without a real terminal.
type listModel struct {
	options  []string
	visible  []int
	filter   string
	cursor   int
	selected map[int]bool
	multi    bool
	pageSize int
}

func newListModel(options []string, defaultIndex, pageSize int, multi bool) *listModel {
	m := &listModel{
		options:  options,
		selected: make(map[int]bool),
		multi:    multi,
		pageSize: pageSize,
	}
	m.refilter()
	if defaultIndex >= 0 && defaultIndex < len(options) {
		for i, idx := range m.visible {
			if idx == defaultIndex {
				m.cursor = i
				break
			}
		}
	}
	return m
}

func (m *listModel) refilter() {
	m.visible = m.visible[:0]
	needle := strings.ToLower(m.filter)
	for i, option := range m.options {
		if needle == "" || strings.Contains(strings.ToLower(option), needle) {
			m.visible = append(m.visible, i)
		}
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *listModel) move(delta int) {
	if len(m.visible) == 0 {
		return
	}
	m.cursor = (m.cursor + delta + len(m.visible)) % len(m.visible)
}

func (m *listModel) toggle() {
	if len(m.visible) == 0 {
		return
	}
	index := m.visible[m.cursor]
	m.selected[index] = !m.selected[index]
}

func (m *listModel) typeRune(r rune) {
	m.filter += string(r)
	m.cursor = 0
	m.refilter()
}

func (m *listModel) backspace() {
	runes := []rune(m.filter)
	if len(runes) == 0 {
		return
	}
	m.filter = string(runes[:len(runes)-1])
	m.cursor = 0
	m.refilter()
}

func (m *listModel) result() []int {
	if !m.multi {
		if len(m.visible) == 0 {
			return nil
		}
		return []int{m.visible[m.cursor]}
	}

	result := make([]int, 0, len(m.selected))
	for i := range m.options {
		if m.selected[i] {
			result = append(result, i)
		}
	}
	return result
}

func (m *listModel) answer(indices []int) string {
	if len(indices) == 0 {
		return "—"
	}
	if !m.multi {
		return m.options[indices[0]]
	}
	labels := make([]string, len(indices))
	for i, index := range indices {
		labels[i] = m.options[index]
	}
	return strings.Join(labels, ", ")
}

// page returns the visible window around the cursor.
func (m *listModel) page() (int, int) {
	count := len(m.visible)
	if count == 0 {
		return 0, 0
	}
	size := m.pageSize
	if size <= 0 || size > count {
		size = count
	}
	start := 0
	if m.cursor > size/2 {
		start = m.cursor - size/2
	}
	if start+size > count {
		start = count - size
	}
	if start < 0 {
		start = 0
	}
	return start, start + size
}

// visibleAtRow maps an absolute terminal row to a position in the visible
// list. lastRow is the row the cursor is on after drawing the prompt.
func (m *listModel) visibleAtRow(row, lastRow int) (int, bool) {
	start, end := m.page()
	if end == start {
		return 0, false
	}

	firstRow := lastRow - (end - start) + 1
	if row < firstRow || row > lastRow {
		return 0, false
	}

	pos := start + row - firstRow
	if pos < start || pos >= end {
		return 0, false
	}
	return pos, true
}

func (m *listModel) render(message string) string {
	hint := T("Use arrows or mouse to move, enter to confirm, type to filter")
	if m.multi {
		hint = T("Use arrows or mouse to move, space to mark, enter to confirm, type to filter")
	}

	var b strings.Builder
	b.WriteString(cyan("?"))
	b.WriteString(" ")
	b.WriteString(message)
	if m.filter != "" {
		b.WriteString(" ")
		b.WriteString(cyan(m.filter))
	}
	b.WriteString("  ")
	b.WriteString(yellow("[" + hint + "]"))

	start, end := m.page()
	for i := start; i < end; i++ {
		b.WriteString("\n")
		index := m.visible[i]
		focused := i == m.cursor
		if focused {
			b.WriteString(cyan("> "))
		} else {
			b.WriteString("  ")
		}
		if m.multi {
			if m.selected[index] {
				b.WriteString(green("[x] "))
			} else {
				b.WriteString("[ ] ")
			}
		}
		label := m.options[index]
		if focused {
			b.WriteString(cyan(label))
		} else {
			b.WriteString(label)
		}
	}
	return b.String()
}

// --- Terminal drawing ---

type promptScreen struct {
	out    io.Writer
	raw    terminal.FileWriter
	cursor *terminal.Cursor
	width  int
	rows   int
}

func newPromptScreen() *promptScreen {
	raw := os.Stdout
	return &promptScreen{
		out:    terminal.NewAnsiStdout(raw),
		raw:    raw,
		cursor: &terminal.Cursor{In: os.Stdin, Out: raw},
		width:  terminalWidth(),
	}
}

// terminalWidth returns the visible terminal width or 0 when it is unknown.
func terminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 0
	}
	return width
}

// physicalRows counts the terminal rows the text occupies, including line
// wrapping and ignoring ANSI color sequences.
func physicalRows(text string, width int) int {
	if width <= 0 {
		return strings.Count(text, "\n") + 1
	}

	rows, col := 1, 0
	for i := 0; i < len(text); {
		if text[i] == 0x1b && i+1 < len(text) && text[i+1] == '[' {
			i += 2
			for i < len(text) && (text[i] < 0x40 || text[i] > 0x7e) {
				i++
			}
			if i < len(text) {
				i++
			}
			continue
		}

		r, size := utf8.DecodeRuneInString(text[i:])
		i += size
		switch {
		case r == '\n':
			rows++
			col = 0
		case unicode.Is(unicode.Mn, r):
			// Combining marks do not advance the cursor.
		default:
			col++
			if col > width {
				rows++
				col = 1
			}
		}
	}
	return rows
}

// clear removes the previously drawn prompt block and leaves the cursor at its
// first line.
func (s *promptScreen) clear() {
	if s.rows == 0 {
		return
	}

	for i := 1; i < s.rows; i++ {
		_ = s.cursor.PreviousLine(1)
	}
	for i := 0; i < s.rows; i++ {
		_ = terminal.EraseLine(s.raw, terminal.ERASE_LINE_ALL)
		if i < s.rows-1 {
			_ = s.cursor.NextLine(1)
		}
	}
	for i := 1; i < s.rows; i++ {
		_ = s.cursor.PreviousLine(1)
	}
	s.rows = 0
}

func (s *promptScreen) draw(text string) {
	s.clear()
	s.width = terminalWidth()
	fmt.Fprint(s.out, text)
	s.rows = physicalRows(text, s.width)
}

func (s *promptScreen) finish(text string) {
	s.clear()
	fmt.Fprintln(s.out, text)
}

// --- Prompts ---

// promptList shows a single- or multi-select list and returns the chosen
// option indices. The list supports keyboard navigation, filtering and the
// mouse: a left click picks an item and the wheel moves the cursor.
func promptList(message string, options []string, defaultIndex, pageSize int, multi bool) ([]int, error) {
	if len(options) == 0 {
		return nil, nil
	}

	reader := newInputReader(true)
	defer reader.Close()

	screen := newPromptScreen()
	_ = screen.cursor.Hide()
	defer func() { _ = screen.cursor.Show() }()

	model := newListModel(options, defaultIndex, pageSize, multi)
	for {
		screen.draw(model.render(message))

		ev, err := reader.ReadEvent()
		if err != nil {
			screen.clear()
			fmt.Fprintln(screen.out)
			return nil, err
		}

		if ev.mouse != nil {
			switch ev.mouse.button {
			case mouseWheelUp:
				model.move(-1)
			case mouseWheelDown:
				model.move(1)
			case mouseLeft:
				lastRow, err := reader.LastRow()
				if err != nil {
					continue
				}
				pos, ok := model.visibleAtRow(ev.mouse.y, lastRow)
				if !ok {
					continue
				}
				model.cursor = pos
				if multi {
					model.toggle()
					continue
				}
				return finishList(screen, message, model), nil
			}
			continue
		}

		switch ev.key {
		case terminal.KeyInterrupt:
			screen.clear()
			fmt.Fprintln(screen.out)
			return nil, errPromptInterrupt
		case terminal.KeyEndTransmission:
			screen.clear()
			fmt.Fprintln(screen.out)
			return nil, io.EOF
		case '\r', '\n':
			if len(model.visible) == 0 {
				continue
			}
			return finishList(screen, message, model), nil
		case terminal.KeyArrowUp:
			model.move(-1)
		case terminal.KeyArrowDown:
			model.move(1)
		case terminal.KeySpace:
			if multi {
				model.toggle()
			}
		case terminal.KeyBackspace, terminal.KeyDelete:
			model.backspace()
		case terminal.KeyEscape:
			if model.filter != "" {
				model.filter = ""
				model.refilter()
			}
		default:
			if ev.key >= ' ' && ev.key != terminal.IgnoreKey {
				model.typeRune(ev.key)
			}
		}
	}
}

// finishList prints the answered prompt and returns the chosen indices.
func finishList(screen *promptScreen, message string, model *listModel) []int {
	result := model.result()
	screen.finish(cyan("? ") + message + "  " + green(model.answer(result)))
	return result
}

// readLine reads one line in raw mode, echoing typed characters.
func readLine(reader *inputReader, out io.Writer, initial string) (string, error) {
	line := []rune(initial)
	if len(line) > 0 {
		fmt.Fprint(out, initial)
	}

	for {
		ev, err := reader.ReadEvent()
		if err != nil {
			return "", err
		}
		if ev.mouse != nil {
			continue
		}

		switch ev.key {
		case '\r', '\n':
			fmt.Fprintln(out)
			return string(line), nil
		case terminal.KeyInterrupt:
			fmt.Fprintln(out)
			return "", errPromptInterrupt
		case terminal.KeyEndTransmission:
			fmt.Fprintln(out)
			return string(line), io.EOF
		case terminal.KeyBackspace, terminal.KeyDelete:
			if len(line) > 0 {
				line = line[:len(line)-1]
				fmt.Fprint(out, "\b \b")
			}
		case terminal.KeyEscape:
			// Ignore unknown escape sequences.
		default:
			if ev.key >= ' ' && ev.key != terminal.IgnoreKey {
				line = append(line, ev.key)
				fmt.Fprint(out, string(ev.key))
			}
		}
	}
}

func promptInput(message, defaultValue string) (string, error) {
	reader := newInputReader(false)
	defer reader.Close()

	out := terminal.NewAnsiStdout(os.Stdout)
	fmt.Fprint(out, cyan("? ")+message+" ")
	return readLine(reader, out, defaultValue)
}

// promptConfirm asks a yes/no question and accepts English and Russian answers.
func promptConfirm(message string, defaultValue bool) (bool, error) {
	reader := newInputReader(false)
	defer reader.Close()

	out := terminal.NewAnsiStdout(os.Stdout)
	for {
		options := T("(y/N)")
		if defaultValue {
			options = T("(Y/n)")
		}
		fmt.Fprint(out, cyan("? ")+message+" "+options+" ")

		answer, err := readLine(reader, out, "")
		if err != nil {
			return false, err
		}
		value, ok := parseConfirmAnswer(answer, defaultValue)
		if ok {
			return value, nil
		}
		fmt.Fprintln(out, red(T("%q is not a valid answer, please try again.", strings.TrimSpace(answer))))
	}
}
