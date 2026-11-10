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
