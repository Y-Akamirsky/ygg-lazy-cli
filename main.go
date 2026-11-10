package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
)

// --- Constants & Vars ---

const (
	version    = "0.1.6"
	repoOwner  = "yggdrasil-network"
	repoPeers  = "public-peers"
	windowsExe = `C:\Program Files\Yggdrasil\yggdrasilctl.exe`
	linuxExe   = "yggdrasilctl"
)

var (
	isWindows = runtime.GOOS == "windows"
	green     = color.New(color.FgGreen).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
	cyan      = color.New(color.FgCyan).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()

	detectedConfigPath string
	configPathExplicit bool
)

// --- Structures ---

type GitTreeResponse struct {
	Tree []GitNode `json:"tree"`
}

type GitNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

type Peer struct {
	URI             string
	Latency         time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	Jitter          time.Duration // Standard deviation of latency
	Stability       float64       // Lower is better (0-1 scale)
	YggdrasilStatus bool          // true if confirmed to be a Yggdrasil node
}

// --- Main ---

func main() {
	// Parse flags first to allow --version without sudo
	var customConfigPath string
	installFlag := flag.Bool("ygginstall", false, "Install Yggdrasil automatically")
	installFlagShort := flag.Bool("i", false, "Install Yggdrasil automatically (shorthand)")
	versionFlag := flag.Bool("version", false, "Show version information")
	versionFlagShort := flag.Bool("v", false, "Show version information (shorthand)")
	helpFlag := flag.Bool("help", false, "Show help information")
	helpFlagShort := flag.Bool("h", false, "Show help information (shorthand)")
	flag.StringVar(&customConfigPath, "config", "", "Path to yggdrasil.conf")
	flag.StringVar(&customConfigPath, "c", "", "Path to yggdrasil.conf (shorthand)")

	// Custom usage function
	flag.Usage = func() {
		fmt.Printf("YggLazy-cli version %s - Lazy way to configure Yggdrasil Network!\n\n", version)
		fmt.Println("USAGE:")
		fmt.Printf("  %s [OPTIONS]\n\n", "ygglazy")
		fmt.Println("OPTIONS:")
		fmt.Println("  -h, --help           Show this help message")
		fmt.Println("  -v, --version        Show version information")
		fmt.Println("  -i, --ygginstall     Install Yggdrasil automatically")
		fmt.Println("  -c, --config PATH    Use a custom yggdrasil.conf path")
		fmt.Println("\nEXAMPLES:")
		fmt.Println("  sudo ygglazy                        # Start interactive configurator")
		fmt.Println("  sudo ygglazy --ygginstall           # Auto-install Yggdrasil")
		fmt.Println("  sudo ygglazy -c /path/to/conf       # Use a custom config file")
		fmt.Println("  ygglazy --version                   # Show version (no sudo needed)")
		fmt.Println("\nFor more information, visit:")
		fmt.Println("  https://github.com/Y-Akamirsky/ygg-lazy-cli")
	}

	flag.Parse()

	// Handle Help Flag
	if *helpFlag || *helpFlagShort {
		flag.Usage()
		return
	}

	// Handle Version Flag (no admin required)
	if *versionFlag || *versionFlagShort {
		fmt.Printf("YggLazy version %s\n", version)
		fmt.Printf("Built with Go %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	// Check/Request Admin Privileges for other operations
	currentPlatform.EnsureAdmin()

	// Resolve the config path: explicit --config|-c wins over platform defaults
	if customConfigPath != "" {
		path, err := resolveConfigPath(customConfigPath)
		if err != nil {
			fmt.Println(red("Invalid config path: "), err)
			os.Exit(1)
		}
		detectedConfigPath = path
		configPathExplicit = true
	} else {
		detectedConfigPath = currentPlatform.FindConfigPath()
	}

	// Handle Install Flag
	if *installFlag || *installFlagShort {
		installYggdrasil()
		return
	}

	// Check installation by the yggdrasil binary, then by config presence
	ensureInstalled()
	ensureConfig()

	mainMenu()
}

// --- Path & Menu Helpers ---

// --- Menus ---

func mainMenu() {
	for {
		clearScreen()
		printBanner()
		fmt.Printf("Config loaded: %s\n", detectedConfigPath)

		peers := getConfigPeers()
		fmt.Printf("Active peers in config: %d\n\n", len(peers))

		mode := ""
		prompt := &survey.Select{
			Message: "Main Menu:",
			Options: []string{
				"Auto-select Peers (Best Latency)",
				"Manual Peer Selection",
				"View Configured Peers",
				"Check Active Peers Status",
				"Remove Dead Peers",
				"Remove Peers",
				"Add Custom Peer",
				"Node Status",
				"Service Control",
				"Exit",
			},
			PageSize: 12,
		}

		err := survey.AskOne(prompt, &mode)
		if err == terminal.InterruptErr {
			fmt.Println("Bye!")
			os.Exit(0)
		}

		switch mode {
		case "Auto-select Peers (Best Latency)":
			autoAddPeers()
		case "Manual Peer Selection":
			manualAddPeers()
		case "View Configured Peers":
			viewCurrentPeers()
		case "Check Active Peers Status":
			checkActivePeersStatus()
		case "Remove Dead Peers":
			removeDeadPeers()
		case "Remove Peers":
			removePeersMenu()
		case "Add Custom Peer":
			addCustomPeer()
		case "Node Status":
			showStatus()
		case "Service Control":
			serviceMenu()
		case "Exit":
			fmt.Println("Bye!")
			os.Exit(0)
		}
	}
}

func serviceMenu() {
	for {
		clearScreen()
		action := ""
		serviceOptions := append(currentPlatform.GetServiceCommands(), "Back")
		prompt := &survey.Select{
			Message: "Service Control (Esc to back):",
			Options: serviceOptions,
		}
		err := survey.AskOne(prompt, &action)
		if err == terminal.InterruptErr || action == "Back" {
			return
		}
		if err := currentPlatform.ManageService(action); err != nil {
			fmt.Println(red("Service operation failed: "), err)
		} else {
			fmt.Println(green("Done."))
		}
		waitEnter()
	}
}

func viewCurrentPeers() {
	clearScreen()
	fmt.Println(cyan("=== Currently Configured Peers ==="))
	peers := getConfigPeers()

	if len(peers) == 0 {
		fmt.Println(yellow("No peers found in config."))
	} else {
		for i, p := range peers {
			fmt.Printf("%d. %s\n", i+1, p)
		}
	}
	waitEnter()
}

// --- Logic: Installation ---

// ensureInstalled verifies that the yggdrasil binary is available in PATH
// instead of relying on the presence of a config file.
func ensureInstalled() {
	if yggdrasilInstalled() {
		return
	}

	fmt.Println(yellow("Yggdrasil is not installed: no yggdrasil binary found in PATH."))
	fmt.Println(yellow("Automatic installation is experimental. Installing Yggdrasil separately is recommended."))
	if !askConfirm("Install Yggdrasil now?", false) {
		fmt.Println("Exiting.")
		os.Exit(0)
	}

	installYggdrasil()
	if !yggdrasilInstalled() {
		fmt.Println(red("Yggdrasil installation did not complete. Exiting."))
		os.Exit(1)
	}
}

// ensureConfig checks that the resolved config exists and offers guidance
// when it is missing.
func ensureConfig() {
	if fileExists(detectedConfigPath) {
		return
	}

	if configPathExplicit {
		fmt.Println(red("Config file not found: "), detectedConfigPath)
		fmt.Println(yellow("Check that the path is correct and that the file exists."))
		os.Exit(1)
	}

	printMissingConfigHelp()
	if !askConfirm("Generate yggdrasil.conf now?", true) {
		fmt.Println("Exiting.")
		os.Exit(0)
	}

	fmt.Println("Generating config...")
	if err := generateConfig(detectedConfigPath); err != nil {
		fmt.Println(red("Failed to generate config: "), err)
		fmt.Printf("%s %s\n", yellow("Generate it manually with:"), configGenCommand(detectedConfigPath))
		waitEnter()
		os.Exit(1)
	}
	fmt.Println(green("Config generated at " + detectedConfigPath))
}

func printMissingConfigHelp() {
	fmt.Println(red("Yggdrasil config not found in the default directory."))
	fmt.Printf("Expected path: %s\n\n", detectedConfigPath)
	fmt.Println(yellow("Make sure that yggdrasil.conf is generated:"))
	fmt.Printf("  %s\n\n", configGenCommand(detectedConfigPath))
	fmt.Println(yellow("Or specify a different config path with the --config|-c flag, for example:"))
	fmt.Printf("  %s\n\n", customConfigCommand("/path/to/yggdrasil.conf"))
}

func configGenCommand(path string) string {
	prefix := ""
	if !isWindows {
		prefix = "sudo "
	}
	return fmt.Sprintf("%syggdrasil -genconf > %s", prefix, path)
}

func customConfigCommand(path string) string {
	prefix := ""
	if !isWindows {
		prefix = "sudo "
	}
	return fmt.Sprintf("%sygglazy --config %s", prefix, path)
}

func installYggdrasil() {
	fmt.Println(cyan("=== Yggdrasil Installer ==="))

	if err := currentPlatform.Install(); err != nil {
		fmt.Println(red("Installation failed: "), err)
		waitEnter()
		return
	}

	// Re-detect the platform default after installation (a package may place
	// the config in a different location), but never override an explicit path.
	if !configPathExplicit {
		detectedConfigPath = currentPlatform.FindConfigPath()
	}

	if !fileExists(detectedConfigPath) {
		fmt.Println("Generating config...")
		if err := generateConfig(detectedConfigPath); err != nil {
			fmt.Println(red("Failed to generate config automatically: "), err)
			fmt.Println(yellow("Check if Yggdrasil is in PATH, or generate it manually with:"))
			fmt.Printf("  %s\n", configGenCommand(detectedConfigPath))
		} else {
			fmt.Println(green("Config generated at " + detectedConfigPath))
		}
	}
	waitEnter()
}

// yggdrasilBinary resolves the yggdrasil executable via PATH, with a Windows
// fallback for the default MSI installation directory.
func yggdrasilBinary() (string, error) {
	if path, err := exec.LookPath("yggdrasil"); err == nil {
		return path, nil
	}
	if isWindows {
		path := `C:\Program Files\Yggdrasil\yggdrasil.exe`
		if fileExists(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("yggdrasil binary not found in PATH")
}

func yggdrasilInstalled() bool {
	_, err := yggdrasilBinary()
	return err == nil
}

func generateConfig(path string) error {
	bin, err := yggdrasilBinary()
	if err != nil {
		return err
	}
	out, err := exec.Command(bin, "-genconf").Output()
	if err != nil {
		return fmt.Errorf("yggdrasil -genconf failed: %v", err)
	}
	if len(out) == 0 {
		return fmt.Errorf("yggdrasil -genconf returned empty output")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// Helper function to keep code clean (used in Debian/Fedora blocks)

func runCommands(cmds [][]string) error {
	for _, c := range cmds {
		fmt.Printf("Running: %s\n", strings.Join(c, " "))
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("step failed (%s): %v", c[0], err)
		}
	}
	return nil
}

// --- Logic: GitHub & Peers ---

func fetchPeersStructure() (map[string][]string, error) {
	fmt.Println("Scanning repository...")
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/master?recursive=1", repoOwner, repoPeers))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var treeResp GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return nil, err
	}

	regionMap := make(map[string][]string)
	for _, node := range treeResp.Tree {
		if node.Type != "blob" || !strings.HasSuffix(node.Path, ".md") || strings.HasSuffix(node.Path, "README.md") {
			continue
		}
		parts := strings.Split(node.Path, "/")
		if len(parts) < 2 {
			continue
		}
		region := parts[0]
		rawUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", repoOwner, repoPeers, node.Path)
		regionMap[region] = append(regionMap[region], rawUrl)
	}
	return regionMap, nil
}

func fetchPeersFromURLs(urls []string) []string {
	var allPeers []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, err := http.Get(url)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				re := regexp.MustCompile(`(tcp|tls|quic|ws|wss)://[a-zA-Z0-9\.\-\:\[\]]+`)
				matches := re.FindAllString(string(body), -1)
				mu.Lock()
				allPeers = append(allPeers, matches...)
				mu.Unlock()
			}
		}(u)
	}
	wg.Wait()
	return allPeers
}

func autoAddPeers() {
	regionMap, err := fetchPeersStructure()
	if err != nil {
		fmt.Println(red("Error: "), err)
		return
	}

	// Build region list for selection
	var regions []string
	for region := range regionMap {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	// Build region options
	regionOptions := []string{"All regions"}
	regionOptions = append(regionOptions, regions...)
	regionOptions = append(regionOptions, "Cancel")

	// Ask user to select region
	regionPrompt := &survey.Select{
		Message: "Select region to test peers from:",
		Options: regionOptions,
		Default: "All regions",
	}
	var selectedRegion string
	survey.AskOne(regionPrompt, &selectedRegion)

	if selectedRegion == "Cancel" {
		return
	}

	// Collect URLs based on selection
	var allUrls []string
	if selectedRegion == "All regions" {
		for _, u := range regionMap {
			allUrls = append(allUrls, u...)
		}
		fmt.Printf("Fetching from %d regions...\n", len(regionMap))
	} else {
		allUrls = regionMap[selectedRegion]
		fmt.Printf("Fetching peers from %s region...\n", cyan(selectedRegion))
	}

	allPeers := fetchPeersFromURLs(allUrls)
	fmt.Printf("Total peers found: %d. Testing all...\n", len(allPeers))

	// Shuffle
	for i := range allPeers {
		j := int(time.Now().UnixNano()) % len(allPeers)
		allPeers[i], allPeers[j] = allPeers[j], allPeers[i]
	}

	limit := len(allPeers)

	var ranked []Peer
	var mu sync.Mutex
	var wg sync.WaitGroup
	tested := 0

	// Use 20 concurrent workers for pinging
	workers := 20
	peerChan := make(chan string, limit)

	fmt.Printf("Testing %d peers with 5 attempts each...\n", limit)
	fmt.Println(yellow("Note: Final peer verification happens after adding them to config."))
	fmt.Println(yellow("Use 'Check Active Peers Status' to verify and 'Remove Dead Peers' to clean up."))
	fmt.Println()

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for uri := range peerChan {
				peer := pingPeerDetailed(uri)

				mu.Lock()
				tested++

				// Accept peers with reasonable latency
				if peer.Latency < 5*time.Second {
					ranked = append(ranked, peer)
					fmt.Printf("\r[%d/%d] ✓ Found %d peers (last: %s, jitter: %s)",
						tested, limit, len(ranked), peer.Latency, peer.Jitter)
				} else {
					fmt.Printf("\r[%d/%d] ✗ Testing... (%d peers found)", tested, limit, len(ranked))
				}
				mu.Unlock()
			}
		}()
	}

	// Send peers to workers
	for _, uri := range allPeers[:limit] {
		peerChan <- uri
	}
	close(peerChan)

	// Wait for all workers to finish
	wg.Wait()
	fmt.Println()

	// Sort by a combined score of latency and stability
	sort.Slice(ranked, func(i, j int) bool {
		// Calculate score: latency + (latency * stability)
		scoreI := float64(ranked[i].Latency) * (1.0 + ranked[i].Stability)
		scoreJ := float64(ranked[j].Latency) * (1.0 + ranked[j].Stability)
		return scoreI < scoreJ
	})

	// Print summary statistics
	fmt.Printf("\n%s\n", cyan("=== Testing Summary ==="))
	fmt.Printf("Total peers tested: %d\n", limit)
	fmt.Printf("Peers found: %s\n", green(fmt.Sprintf("%d", len(ranked))))

	if len(ranked) > 0 {
		fmt.Printf("Best latency: %s\n", green(ranked[0].Latency.String()))
		fmt.Printf("Best stability: %.2f%%\n", (1.0-ranked[0].Stability)*100)
	}
	fmt.Println()

	if len(ranked) == 0 {
		fmt.Println(yellow("No reachable peers found with latency < 5s."))
		fmt.Println(yellow("This might mean:"))
		fmt.Println(yellow("  - Network connectivity issues"))
		fmt.Println(yellow("  - Firewall blocking connections"))
		fmt.Println(yellow("  - All peers are currently down"))
		waitEnter()
		return
	}

	// Show top 10 peers
	displayCount := 10
	if len(ranked) < displayCount {
		displayCount = len(ranked)
	}

	fmt.Println(green("\nTop Peers by Latency & Stability:"))
	for i := 0; i < displayCount; i++ {
		stability := "excellent"
		stabilityPercent := (1.0 - ranked[i].Stability) * 100
		if ranked[i].Stability > 0.15 {
			stability = "good"
		}
		if ranked[i].Stability > 0.30 {
			stability = "fair"
		}
		if ranked[i].Stability > 0.50 {
			stability = "unstable"
		}

		fmt.Printf("%d. %s\n   Latency: %s (min: %s, max: %s, jitter: %s) - %s (%.0f%%)\n",
			i+1, ranked[i].URI, ranked[i].Latency, ranked[i].MinLatency,
			ranked[i].MaxLatency, ranked[i].Jitter, stability, stabilityPercent)
	}
	if len(ranked) > displayCount {
		fmt.Printf("\n(+%d more peers available)\n", len(ranked)-displayCount)
	}

	// Ask how many peers to add
	countToAdd := 0
	maxPeers := len(ranked)
	if maxPeers > 10 {
		maxPeers = 10
	}

	prompt := &survey.Select{
		Message: "How many of the best peers would you like to add?",
		Options: []string{"3 peers (recommended)", "5 peers", "7 peers", "10 peers", "Custom number", "Cancel"},
		Default: "3 peers (recommended)",
	}
	var choice string
	survey.AskOne(prompt, &choice)

	switch choice {
	case "3 peers (recommended)":
		countToAdd = 3
	case "5 peers":
		countToAdd = 5
	case "7 peers":
		countToAdd = 7
	case "10 peers":
		countToAdd = 10
	case "Custom number":
		customPrompt := &survey.Input{
			Message: fmt.Sprintf("Enter number of peers to add (1-%d):", len(ranked)),
		}
		var customStr string
		survey.AskOne(customPrompt, &customStr)
		fmt.Sscanf(customStr, "%d", &countToAdd)
		if countToAdd < 1 || countToAdd > len(ranked) {
			fmt.Println(red("Invalid number. Canceling."))
			waitEnter()
			return
		}
	case "Cancel":
		return
	}

	if countToAdd > len(ranked) {
		countToAdd = len(ranked)
	}

	toAdd := []string{}
	fmt.Println(green("\nPeers to add:"))
	for i := 0; i < countToAdd; i++ {
		fmt.Printf("%d. %s (%s avg, jitter: %s)\n", i+1, ranked[i].URI,
			ranked[i].Latency, ranked[i].Jitter)
		toAdd = append(toAdd, ranked[i].URI)
	}

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Confirm adding these peers?"}, &confirm)
	if confirm {
		addPeersToConfig(toAdd)
		restartServicePrompt()
	}
}

func manualAddPeers() {
	for {
		clearScreen()
		regionMap, err := fetchPeersStructure()
		if err != nil {
			return
		}
		keys := []string{}
		for k := range regionMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		keys = append(keys, "Back")

		selReg := ""
		err = survey.AskOne(&survey.Select{Message: "Region:", Options: keys}, &selReg)
		if err == terminal.InterruptErr || selReg == "Back" {
			return
		}

		peers := fetchPeersFromURLs(regionMap[selReg])
		if len(peers) == 0 {
			fmt.Println("No peers.")
			waitEnter()
			continue
		}
		selPeers := []string{}
		err = survey.AskOne(&survey.MultiSelect{Message: "Select Peers:", Options: peers}, &selPeers)
		if err == nil && len(selPeers) > 0 {
			addPeersToConfig(selPeers)
			restartServicePrompt()
		}
	}
}

func removePeersMenu() {
	current := getConfigPeers()
	if len(current) == 0 {
		fmt.Println(yellow("No peers in config."))
		waitEnter()
		return
	}
	toRemove := []string{}
	err := survey.AskOne(&survey.MultiSelect{Message: "Select to Remove:", Options: current}, &toRemove)
	if err == nil && len(toRemove) > 0 {
		removePeersFromConfig(toRemove)
		restartServicePrompt()
	}
}

func addCustomPeer() {
	input := ""
	err := survey.AskOne(&survey.Input{Message: "Enter URIs (space separated):"}, &input)
	if err == nil && input != "" {
		addPeersToConfig(strings.Fields(input))
		restartServicePrompt()
	}
}

// --- Logic: Config Writer V4 (Clean formatting) ---

// peersCloseRE matches the closing bracket of the Peers array: the first line
// that contains only "]" (optionally indented and with trailing spaces). This
// avoids matching the square brackets used by IPv6 addresses.
var peersCloseRE = regexp.MustCompile(`(?m)^[ \t]*\][ \t]*(?:\r?\n|$)`)

// peersBlockBounds returns the byte range of the "Peers: [...]" block,
// including the closing bracket and its line ending.
func peersBlockBounds(content string) (start, end int, ok bool) {
	start = strings.Index(content, "Peers: [")
	if start == -1 {
		return 0, 0, false
	}
	rest := content[start+len("Peers: ["):]
	loc := peersCloseRE.FindStringIndex(rest)
	if loc == nil {
		return 0, 0, false
	}
	return start, start + len("Peers: [") + loc[1], true
}

func getConfigPeers() []string {
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil {
		return []string{}
	}
	content := string(contentBytes)
	var peers []string

	startIdx, endIdx, ok := peersBlockBounds(content)
	if !ok {
		return peers
	}

	block := content[startIdx:endIdx]
	// Improved regex to properly handle IPv6 addresses in square brackets
	// Matches: protocol://hostname:port or protocol://[ipv6]:port
	re := regexp.MustCompile(`(tcp|tls|quic|ws|wss|udp)://(\[[0-9a-fA-F:]+\]|[a-zA-Z0-9\.\-]+)(:[0-9]+)?`)
	matches := re.FindAllString(block, -1)
	if matches != nil {
		peers = append(peers, matches...)
	}
	return peers
}

func addPeersToConfig(newPeers []string) {
	// Combine existing peers + new peers (skip duplicates)
	existing := getConfigPeers()
	finalList := existing
	for _, p := range newPeers {
		isDup := false
		for _, e := range existing {
			if e == p {
				isDup = true
				break
			}
		}
		if !isDup {
			finalList = append(finalList, p)
		}
	}

	if err := writeConfigPeers(finalList); err != nil {
		fmt.Println(red("Failed to update config: "), err)
		return
	}
	fmt.Println(green("Peers added and config formatted."))
}

func removePeersFromConfig(toRemove []string) {
	current := getConfigPeers()
	var keep []string

	for _, p := range current {
		shouldRemove := false
		for _, rem := range toRemove {
			if p == rem {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			keep = append(keep, p)
		}
	}

	if err := writeConfigPeers(keep); err != nil {
		fmt.Println(red("Failed to update config: "), err)
		return
	}
	fmt.Println(green("Peers removed."))
}

// writeConfigPeers rewrites the "Peers: [...]" block in the detected config.
func writeConfigPeers(peers []string) error {
	return writePeersToFile(detectedConfigPath, peers)
}

// writePeersToFile replaces the whole Peers block instead of patching it.
// Rebuilding the block also cleans up blank lines left by previous edits.
func writePeersToFile(path string, peers []string) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	block := formatPeersBlock(peers)

	startIdx, endIdx, ok := peersBlockBounds(content)
	if !ok {
		if content != "" {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "\n"
		}
		return os.WriteFile(path, []byte(content+block+"\n"), 0644)
	}

	// Keep the content after the block, collapsing extra blank lines that
	// older versions left behind on every add/remove operation.
	tail := strings.TrimLeft(content[endIdx:], "\r\n")
	newContent := content[:startIdx] + block
	if tail != "" {
		newContent += "\n\n" + tail
	} else {
		newContent += "\n"
	}

	return os.WriteFile(path, []byte(newContent), 0644)
}

func formatPeersBlock(peers []string) string {
	var b strings.Builder
	b.WriteString("Peers: [")
	for _, p := range peers {
		b.WriteString("\n  ")
		b.WriteString(p)
	}
	b.WriteString("\n]")
	return b.String()
}

// --- Helpers ---

func showStatus() {
	cmdName := linuxExe
	if isWindows {
		cmdName = windowsExe
	}
	out, _ := exec.Command(cmdName, "getself").CombinedOutput()
	fmt.Println(string(out))
	waitEnter()
}

func restartServicePrompt() {
	r := false
	survey.AskOne(&survey.Confirm{Message: "Restart Yggdrasil service now?"}, &r)
	if r {
		if err := currentPlatform.ManageService("Restart"); err != nil {
			fmt.Println(red("Restart failed: "), err)
		} else {
			fmt.Println(green("Service restarted."))
		}
	}
}

// pingPeerDetailed performs comprehensive latency testing with stability metrics.
// It performs multiple attempts (5 by default) and calculates:
// - Average latency
// - Minimum and maximum latency
// - Jitter (standard deviation)
// - Stability score (0-1, where lower is better)
//
// This is crucial for mesh networks like Yggdrasil where peer quality and
// stability matter significantly for routing performance.
//
// Parameters:
//   - uri: The peer URI in format "protocol://host:port"
//
// Returns:
//   - Peer struct with detailed metrics
//   - High latency (999s) and poor stability if URI is invalid or all attempts fail
func pingPeerDetailed(uri string) Peer {
	parts := strings.Split(uri, "://")
	if len(parts) < 2 {
		return Peer{
			URI:       uri,
			Latency:   999 * time.Second,
			Stability: 1.0,
		}
	}

	// Perform multiple attempts for statistical accuracy
	// More attempts = better data for stability analysis
	attempts := 5
	latencies := make([]time.Duration, 0, attempts)

	for i := 0; i < attempts; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", parts[1], 3*time.Second)
		if err != nil {
			// If connection fails, try next attempt
			continue
		}
		latency := time.Since(start)
		conn.Close()

		latencies = append(latencies, latency)

		// Longer delay between attempts for more realistic measurements
		// This helps detect connection instability
		if i < attempts-1 {
			time.Sleep(150 * time.Millisecond)
		}
	}

	// If all attempts failed, return poor metrics
	if len(latencies) == 0 {
		return Peer{
			URI:       uri,
			Latency:   999 * time.Second,
			Stability: 1.0,
		}
	}

	// Calculate statistics
	var totalLatency time.Duration
	minLatency := latencies[0]
	maxLatency := latencies[0]

	for _, lat := range latencies {
		totalLatency += lat
		if lat < minLatency {
			minLatency = lat
		}
		if lat > maxLatency {
			maxLatency = lat
		}
	}

	avgLatency := totalLatency / time.Duration(len(latencies))

	// Calculate standard deviation (jitter)
	var variance float64
	for _, lat := range latencies {
		diff := float64(lat - avgLatency)
		variance += diff * diff
	}
	variance /= float64(len(latencies))
	// Correctly calculate standard deviation (jitter) - need square root!
	stdDev := math.Sqrt(variance)
	jitter := time.Duration(stdDev)

	// Calculate stability score (0 = perfect, 1 = terrible)
	// Based on coefficient of variation (jitter relative to average latency)
	stability := 0.0
	if avgLatency > 0 {
		// Coefficient of variation as stability metric
		stability = stdDev / float64(avgLatency)
		// Cap at 1.0 for extremely unstable connections
		if stability > 1.0 {
			stability = 1.0
		}
	}

	return Peer{
		URI:             uri,
		Latency:         avgLatency,
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
		Jitter:          jitter,
		Stability:       stability,
		YggdrasilStatus: false,
	}
}

// checkActivePeersStatus displays the current status of all peers from yggdrasilctl
func checkActivePeersStatus() {
	clearScreen()
	fmt.Println(cyan("=== Active Peers Status ===\n"))

	cmdName := linuxExe
	if isWindows {
		cmdName = windowsExe
	}

	out, err := exec.Command(cmdName, "getPeers").CombinedOutput()
	if err != nil {
		fmt.Println(red("Error: "), err)
		fmt.Println(yellow("Make sure Yggdrasil service is running."))
		waitEnter()
		return
	}

	fmt.Println(string(out))
	fmt.Println(yellow("\nTip: Use 'Remove Dead Peers' to clean up peers with 'Down' status."))
	waitEnter()
}

// removeDeadPeers removes peers that are currently in "Down" state
func removeDeadPeers() {
	clearScreen()
	fmt.Println(cyan("=== Remove Dead Peers ===\n"))

	cmdName := linuxExe
	if isWindows {
		cmdName = windowsExe
	}

	// Get current peer status from yggdrasilctl
	fmt.Println("Fetching peer status from Yggdrasil...")
	out, err := exec.Command(cmdName, "-json", "getPeers").CombinedOutput()
	if err != nil {
		fmt.Println(red("Error getting peer status: "), err)
		fmt.Println(yellow("Make sure Yggdrasil service is running."))
		waitEnter()
		return
	}

	// Parse the JSON response - it's a map with "peers" array
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		fmt.Println(red("Error parsing peers data: "), err)
		waitEnter()
		return
	}

	// Get the peers array
	peersData, ok := result["peers"].([]interface{})
	if !ok {
		fmt.Println(yellow("No peer data found in response."))
		fmt.Println(yellow("Make sure Yggdrasil service is running and has been started with some peers."))
		waitEnter()
		return
	}

	if len(peersData) == 0 {
		fmt.Println(yellow("No active peer connections found."))
		fmt.Println(yellow("Yggdrasil might still be starting up or no peers are configured."))
		waitEnter()
		return
	}

	fmt.Printf("\nFound %d peer connection(s) in Yggdrasil. Analyzing...\n\n", len(peersData))

	// Parse peer statuses
	type PeerStatus struct {
		URI       string
		Up        bool
		LastError string
	}

	var upPeers []PeerStatus
	var downPeers []PeerStatus

	for _, peerInterface := range peersData {
		peer, ok := peerInterface.(map[string]interface{})
		if !ok {
			continue
		}

		status := PeerStatus{}

		// Get remote URI
		if remote, ok := peer["remote"].(string); ok {
			status.URI = remote
		} else {
			continue // Skip if no URI
		}

		// Get up status
		if up, ok := peer["up"].(bool); ok {
			status.Up = up
		}

		// Get last error if exists
		if lastError, ok := peer["last_error"].(string); ok {
			status.LastError = lastError
		}

		if status.Up {
			upPeers = append(upPeers, status)
		} else {
			downPeers = append(downPeers, status)
		}
	}

	// Display status
	fmt.Printf("%s: %d\n", green("Up"), len(upPeers))
	fmt.Printf("%s: %d\n", red("Down"), len(downPeers))
	fmt.Println()

	if len(downPeers) == 0 {
		fmt.Println(green("✓ All peers are up! No dead peers to remove."))
		waitEnter()
		return
	}

	// Show dead peers with errors
	fmt.Println(red("Dead peers (status: Down):"))
	for i, peer := range downPeers {
		fmt.Printf("%d. %s\n", i+1, peer.URI)
		if peer.LastError != "" {
			// Truncate error if too long
			errMsg := peer.LastError
			if len(errMsg) > 80 {
				errMsg = errMsg[:77] + "..."
			}
			fmt.Printf("   Error: %s\n", yellow(errMsg))
		}
	}
	fmt.Println()

	// Get configured peers to see which ones are in config
	configuredPeers := getConfigPeers()

	// Find which dead peers are actually in the config
	var deadPeersInConfig []string
	var deadPeersNotInConfig []string

	for _, deadPeer := range downPeers {
		found := false
		for _, configPeer := range configuredPeers {
			if configPeer == deadPeer.URI {
				found = true
				break
			}
		}
		if found {
			deadPeersInConfig = append(deadPeersInConfig, deadPeer.URI)
		} else {
			deadPeersNotInConfig = append(deadPeersNotInConfig, deadPeer.URI)
		}
	}

	if len(deadPeersNotInConfig) > 0 {
		fmt.Println(yellow("Note: Some dead peers are not in config (possibly added temporarily):"))
		for _, uri := range deadPeersNotInConfig {
			fmt.Printf("  - %s\n", uri)
		}
		fmt.Println()
	}

	if len(deadPeersInConfig) == 0 {
		fmt.Println(yellow("No dead peers found in config to remove."))
		fmt.Println(yellow("(All dead peers were added temporarily, not via config)"))
		waitEnter()
		return
	}

	// Ask for confirmation
	fmt.Printf("Found %s dead peers in config:\n", red(fmt.Sprintf("%d", len(deadPeersInConfig))))
	for i, uri := range deadPeersInConfig {
		fmt.Printf("%d. %s\n", i+1, uri)
	}
	fmt.Println()

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Remove these dead peers from config?"}, &confirm)
	if confirm {
		removePeersFromConfig(deadPeersInConfig)
		fmt.Println(green("\n✓ Dead peers removed from config."))
		restartServicePrompt()
	}
}

func getLinuxDistroInfo() (id string, like string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown", ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			like = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}
	return strings.ToLower(id), strings.ToLower(like)
}

func getLatestReleaseURL(repo, suffix, archFilter string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	assets, ok := res["assets"].([]interface{})
	if !ok {
		return "", fmt.Errorf("no assets found")
	}

	for _, a := range assets {
		m := a.(map[string]interface{})
		name, nameOk := m["name"].(string)
		url, urlOk := m["browser_download_url"].(string)

		if nameOk && urlOk && strings.HasSuffix(name, suffix) {
			if archFilter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(archFilter)) {
				continue
			}
			return url, nil
		}
	}
	return "", fmt.Errorf("release not found for arch: %s", archFilter)
}

func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// resolveConfigPath expands a leading "~" and returns an absolute path.
func resolveConfigPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return "", fmt.Errorf("%s is a directory", abs)
	}
	return abs, nil
}

// askConfirm asks a yes/no question and uses the safest logical answer as the
// default, so a stray Enter does not trigger an unexpected action.
func askConfirm(message string, defaultValue bool) bool {
	answer := defaultValue
	if err := survey.AskOne(&survey.Confirm{Message: message, Default: defaultValue}, &answer); err != nil {
		if err == terminal.InterruptErr {
			fmt.Println("Bye!")
			os.Exit(0)
		}
		return false
	}
	return answer
}

func waitEnter() {
	fmt.Println("\nPress Enter...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
	if isWindows {
		exec.Command("cmd", "/c", "cls").Run()
	}
}

func printBanner() {
	banner := `
	__   __           _
	\ \ / /          | |
	 \ V / __ _  __ _| |     __ _ _____   _
	  \ / / _' |/ _' | |    / _' |_  / | | |
	  | || (_| | (_| | |___| (_| |/ /| |_| |
	  \_/ \__, |\__, |______\__,_/___|\__, |
	       __/ | __/ |                 __/ |
	      |___/ |___/                 |___/
	          Configurator v` + version
	fmt.Println(cyan(banner))
}
