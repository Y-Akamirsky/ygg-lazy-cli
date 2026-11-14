package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "yggdrasil.conf")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func readTestConfig(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	return string(data)
}

func TestWritePeersToFileDoesNotAccumulateBlankLines(t *testing.T) {
	path := writeTestConfig(t, `# Yggdrasil config
Peers: [
  tls://old:443
]

InterfacePeers: {}
`)

	want := `# Yggdrasil config
Peers: [
  tls://new:443
]

InterfacePeers: {}
`

	// Simulate several add/remove cycles. The old implementation appended a
	// newline on every write and left a growing block of empty lines behind.
	for i := 0; i < 5; i++ {
		if err := writePeersToFile(path, []string{"tls://new:443"}); err != nil {
			t.Fatalf("write peers: %v", err)
		}
	}

	if got := readTestConfig(t, path); got != want {
		t.Fatalf("unexpected config after repeated writes:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWritePeersToFileCollapsesExistingBlankLines(t *testing.T) {
	path := writeTestConfig(t, `Peers: [
  tls://old:443
]




InterfacePeers: {}
`)

	if err := writePeersToFile(path, []string{"tls://old:443", "tls://[200:1234::1]:443"}); err != nil {
		t.Fatalf("write peers: %v", err)
	}

	want := `Peers: [
  tls://old:443
  tls://[200:1234::1]:443
]

InterfacePeers: {}
`
	if got := readTestConfig(t, path); got != want {
		t.Fatalf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWritePeersToFileEmptyPeers(t *testing.T) {
	path := writeTestConfig(t, `Peers: [
  tls://old:443
]
`)

	if err := writePeersToFile(path, nil); err != nil {
		t.Fatalf("write peers: %v", err)
	}

	want := `Peers: [
]
`
	if got := readTestConfig(t, path); got != want {
		t.Fatalf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWritePeersToFileAppendsMissingSection(t *testing.T) {
	path := writeTestConfig(t, "InterfacePeers: {}\n")

	if err := writePeersToFile(path, []string{"tls://new:443"}); err != nil {
		t.Fatalf("write peers: %v", err)
	}

	want := `InterfacePeers: {}

Peers: [
  tls://new:443
]
`
	if got := readTestConfig(t, path); got != want {
		t.Fatalf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWritePeersToFileInlineEmptyPeers(t *testing.T) {
	path := writeTestConfig(t, "Peers: []\n\nInterfacePeers: {}\n")

	if err := writePeersToFile(path, []string{"tls://new:443"}); err != nil {
		t.Fatalf("write peers: %v", err)
	}

	want := `Peers: [
  tls://new:443
]

InterfacePeers: {}
`
	if got := readTestConfig(t, path); got != want {
		t.Fatalf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGetConfigPeersHandlesIPv6(t *testing.T) {
	detectedConfigPath = writeTestConfig(t, `Peers: [
  tls://[200:1234::1]:443
  tcp://example.org:1234
]
`)
	defer func() { detectedConfigPath = "" }()

	got := getConfigPeers()
	want := []string{"tls://[200:1234::1]:443", "tcp://example.org:1234"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("peers = %v, want %v", got, want)
	}
}

func TestAddRemovePeersKeepsConfigClean(t *testing.T) {
	path := writeTestConfig(t, `Peers: [
  tls://old:443
]

InterfacePeers: {}
`)

	oldPath := detectedConfigPath
	detectedConfigPath = path
	defer func() { detectedConfigPath = oldPath }()

	addPeersToConfig([]string{"tls://a:443", "tls://b:443"})
	addPeersToConfig([]string{"tls://b:443"}) // duplicate must be skipped
	removePeersFromConfig([]string{"tls://old:443", "tls://a:443"})

	want := `Peers: [
  tls://b:443
]

InterfacePeers: {}
`
	if got := readTestConfig(t, path); got != want {
		t.Fatalf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestResolveConfigPathRejectsDirectory(t *testing.T) {
	if _, err := resolveConfigPath(t.TempDir()); err == nil {
		t.Fatal("expected an error for a directory path")
	}
}

func TestListModelSelection(t *testing.T) {
	m := newListModel([]string{"alpha", "beta", "gamma"}, -1, 0, false)

	if got := m.result(); len(got) != 1 || got[0] != 0 {
		t.Fatalf("initial selection = %v, want [0]", got)
	}

	m.move(1)
	if got := m.result(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("selection after down = %v, want [1]", got)
	}

	m.move(-1)
	m.move(-1)
	if got := m.result(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("selection must wrap around = %v, want [2]", got)
	}
}

func TestListModelFilterKeepsSelection(t *testing.T) {
	m := newListModel([]string{"tls://a:443", "quic://b:443", "tls://c:443"}, -1, 0, false)

	for _, r := range "quic" {
		m.typeRune(r)
	}
	if len(m.visible) != 1 || m.options[m.visible[0]] != "quic://b:443" {
		t.Fatalf("visible = %v, want the quic peer", m.visible)
	}
	if got := m.result(); len(got) != 1 || m.options[got[0]] != "quic://b:443" {
		t.Fatalf("filtered selection = %v", got)
	}

	for range "quic" {
		m.backspace()
	}
	if len(m.visible) != 3 {
		t.Fatalf("visible after backspace = %v, want all options", m.visible)
	}
}

func TestListModelMultiSelect(t *testing.T) {
	m := newListModel([]string{"one", "two", "three"}, -1, 0, true)

	m.toggle()
	m.move(1)
	m.toggle()
	m.move(1)
	m.toggle() // toggle "three" on
	m.toggle() // and off again

	if got := m.result(); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("multi selection = %v, want [0 1]", got)
	}
}

func TestListModelDefaultsAndPaging(t *testing.T) {
	m := newListModel([]string{"a", "b", "c", "d", "e"}, 3, 2, false)
	if m.cursor != 3 {
		t.Fatalf("cursor = %d, want 3", m.cursor)
	}

	start, end := m.page()
	if start < 0 || end > len(m.visible) || end-start > 2 {
		t.Fatalf("page = [%d %d] for %d options", start, end, len(m.visible))
	}
	if m.cursor < start || m.cursor >= end {
		t.Fatalf("cursor %d is outside page [%d %d)", m.cursor, start, end)
	}
}

func TestListModelVisibleAtRow(t *testing.T) {
	m := newListModel([]string{"a", "b", "c", "d", "e"}, -1, 3, false)

	// Header is on row 8, options on rows 9-11.
	for row, want := range map[int]int{9: 0, 10: 1, 11: 2} {
		pos, ok := m.visibleAtRow(row, 11)
		if !ok || pos != want {
			t.Fatalf("row %d = (%d, %v), want (%d, true)", row, pos, ok, want)
		}
	}
	for _, row := range []int{7, 8, 12} {
		if _, ok := m.visibleAtRow(row, 11); ok {
			t.Fatalf("row %d must not map to an option", row)
		}
	}

	// Scroll the window to the end and check the mapping follows it.
	m.cursor = 4
	if _, ok := m.visibleAtRow(10, 10); !ok {
		t.Fatal("last visible row must map to an option after scrolling")
	}
	if pos, ok := m.visibleAtRow(10, 10); !ok || pos != 4 {
		t.Fatalf("last visible row = (%d, %v), want (4, true)", pos, ok)
	}
}

func TestPhysicalRowsCountsWrapping(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  int
	}{
		{"one\ntwo", 0, 2},
		{"1234567890", 10, 1},
		{"12345678901", 10, 2},
		{"12345678901234567890", 10, 2},
		{"one\ntwo", 10, 2},
		{"\x1b[36m12345\x1b[0m", 5, 1},
		{"\x1b[36m123456\x1b[0m", 5, 2},
		{"e\u0301", 1, 1},
	}

	for _, tc := range cases {
		if got := physicalRows(tc.text, tc.width); got != tc.want {
			t.Fatalf("physicalRows(%q, %d) = %d, want %d", tc.text, tc.width, got, tc.want)
		}
	}
}
