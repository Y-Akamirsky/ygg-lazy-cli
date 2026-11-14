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

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
)

// --- Constants & Vars ---

const (
	version    = "1.0.0"
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
	var langValue string
	installFlag := flag.Bool("ygginstall", false, "Install Yggdrasil automatically")
	installFlagShort := flag.Bool("i", false, "Install Yggdrasil automatically (shorthand)")
	versionFlag := flag.Bool("version", false, "Show version information")
	versionFlagShort := flag.Bool("v", false, "Show version information (shorthand)")
	helpFlag := flag.Bool("help", false, "Show help information")
	helpFlagShort := flag.Bool("h", false, "Show help information (shorthand)")
	flag.StringVar(&langValue, "lang", "auto", "Interface language: auto, en, ru")
	flag.StringVar(&langValue, "l", "auto", "Interface language: auto, en, ru (shorthand)")
	flag.StringVar(&customConfigPath, "config", "", "Path to yggdrasil.conf")
	flag.StringVar(&customConfigPath, "c", "", "Path to yggdrasil.conf (shorthand)")

	// Custom usage function
	flag.Usage = func() {
		fmt.Println(T("YggLazy-cli version %s - Lazy way to configure Yggdrasil Network!", version))
		fmt.Println()
		fmt.Println(T("USAGE:"))
		fmt.Println()
		fmt.Println("  ygglazy [OPTIONS]")
		fmt.Println()
		fmt.Println(T("OPTIONS:"))
		fmt.Println("  -h, --help           " + T("Show this help message"))
		fmt.Println("  -v, --version        " + T("Show version information"))
		fmt.Println("  -i, --ygginstall     " + T("Install Yggdrasil automatically"))
		fmt.Println("  -c, --config PATH    " + T("Use a custom yggdrasil.conf path"))
		fmt.Println("  -l, --lang LANG      " + T("Set interface language: auto, en, ru"))
		fmt.Println()
		fmt.Println(T("EXAMPLES:"))
		fmt.Println("  sudo ygglazy                        # " + T("Start interactive configurator"))
		fmt.Println("  sudo ygglazy --ygginstall           # " + T("Auto-install Yggdrasil"))
		fmt.Println("  sudo ygglazy -c /path/to/conf       # " + T("Use a custom config file"))
		fmt.Println("  ygglazy --version                   # " + T("Show version (no sudo needed)"))
		fmt.Println()
		fmt.Println(T("For more information, visit:"))
		fmt.Println("  https://github.com/Y-Akamirsky/ygg-lazy-cli")
	}

	flag.Parse()

	// Apply the interface language before any other output. An explicit flag
	// wins; --lang auto falls back to the OS locale.
	if err := setLanguage(langValue); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Handle Help Flag
	if *helpFlag || *helpFlagShort {
		flag.Usage()
		return
	}

	// Handle Version Flag (no admin required)
	if *versionFlag || *versionFlagShort {
		fmt.Println(T("YggLazy version %s", version))
		fmt.Println(T("Built with Go %s", runtime.Version()))
		fmt.Println(T("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
		return
	}

	// Check/Request Admin Privileges for other operations
	currentPlatform.EnsureAdmin()

	// Resolve the config path: explicit --config|-c wins over platform defaults
	if customConfigPath != "" {
		path, err := resolveConfigPath(customConfigPath)
		if err != nil {
			fmt.Println(red(T("Invalid config path: ")), err)
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

type choice struct {
	key   string
	label string
}

// askChoice shows localized labels and returns the stable internal key.
func askChoice(message string, choices []choice, defaultKey string, pageSize int) (string, error) {
	labels := make([]string, len(choices))
	defaultIndex := -1
	for i, c := range choices {
		labels[i] = c.label
		if c.key == defaultKey {
			defaultIndex = i
		}
	}

	indices, err := promptList(T(message), labels, defaultIndex, pageSize, false)
	if err != nil || len(indices) == 0 {
		return "", err
	}
	return choices[indices[0]].key, nil
}

// askKeysSelect builds choices from canonical keys translated via T.
func askKeysSelect(message string, keys []string, defaultKey string) (string, error) {
	choices := make([]choice, len(keys))
	for i, key := range keys {
		choices[i] = choice{key: key, label: T(key)}
	}
	return askChoice(message, choices, defaultKey, 0)
}

func languageMenu() {
	clearScreen()
	fmt.Println(cyan(T("Select language:")))
	fmt.Println()

	selected, err := askChoice("Select language:", []choice{
		{key: string(LangEN), label: "English"},
		{key: string(LangRU), label: "Русский"},
		{key: "back", label: T("Back")},
	}, string(currentLang), 0)
	if err == errPromptInterrupt || selected == "back" || selected == "" {
		return
	}

	currentLang = Lang(selected)
	fmt.Println(green(T("Language changed.")))
	waitContinue()
}

func serviceLabel(command string) string {
	switch strings.ToLower(command) {
	case "start":
		return T("Start")
	case "stop":
		return T("Stop")
	case "restart":
		return T("Restart")
	case "enable autostart":
		return T("Enable Autostart")
	case "disable autostart":
		return T("Disable Autostart")
	default:
		return command
	}
}

// --- Menus ---

func mainMenu() {
	for {
		clearScreen()
		printBanner()
		fmt.Println(T("Config loaded: %s", detectedConfigPath))

		peers := getConfigPeers()
		fmt.Println(T("Active peers in config: %d", len(peers)))
		fmt.Println()

		keys := []string{
			"Auto-select Peers (Best Latency)",
			"Manual Peer Selection",
			"View Configured Peers",
			"Check Active Peers Status",
			"Remove Dead Peers",
			"Remove Peers",
			"Add Custom Peer",
			"Node Status",
			"Service Control",
			"Language",
			"Exit",
		}
		choices := make([]choice, len(keys))
		for i, key := range keys {
			label := T(key)
			if key == "Language" {
				label = "Language / Язык"
			}
			choices[i] = choice{key: key, label: label}
		}

		mode, err := askChoice("Main Menu:", choices, "", 12)
		if err != nil {
			if err == errPromptInterrupt {
				fmt.Println(T("Bye!"))
				os.Exit(0)
			}
			fmt.Println(red(T("Error: ")), err)
			return
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
		case "Language":
			languageMenu()
		case "Exit":
			fmt.Println(T("Bye!"))
			os.Exit(0)
		}
	}
}

func serviceMenu() {
	for {
		clearScreen()
		actions := currentPlatform.GetServiceCommands()
		choices := make([]choice, 0, len(actions)+1)
		for _, action := range actions {
			choices = append(choices, choice{key: action, label: serviceLabel(action)})
		}
		choices = append(choices, choice{key: "Back", label: T("Back")})

		action, err := askChoice("Service Control (Esc to back):", choices, "", 0)
		if err == errPromptInterrupt || action == "Back" {
			return
		}
		if action == "" {
			return
		}

		if err := currentPlatform.ManageService(action); err != nil {
			fmt.Println(red(T("Service operation failed: ")), err)
		} else {
			fmt.Println(green(T("Done.")))
		}
		waitContinue()
	}
}

func viewCurrentPeers() {
	clearScreen()
	fmt.Println(cyan(T("=== Currently Configured Peers ===")))
	peers := getConfigPeers()

	if len(peers) == 0 {
		fmt.Println(yellow(T("No peers found in config.")))
	} else {
		for i, p := range peers {
			fmt.Printf("%d. %s\n", i+1, p)
		}
	}
	waitContinue()
}

// --- Logic: Installation ---

// ensureInstalled verifies that the yggdrasil binary is available in PATH
// instead of relying on the presence of a config file.
func ensureInstalled() {
	if yggdrasilInstalled() {
		return
	}

	fmt.Println(yellow(T("Yggdrasil is not installed: no yggdrasil binary found in PATH.")))
	fmt.Println(yellow(T("Automatic installation is experimental. Installing Yggdrasil separately is recommended.")))
	if !askConfirm("Install Yggdrasil now?", false) {
		fmt.Println(T("Exiting."))
		os.Exit(0)
	}

	installYggdrasil()
	if !yggdrasilInstalled() {
		fmt.Println(red(T("Yggdrasil installation did not complete. Exiting.")))
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
		fmt.Println(red(T("Config file not found: ")), detectedConfigPath)
		fmt.Println(yellow(T("Check that the path is correct and that the file exists.")))
		os.Exit(1)
	}

	printMissingConfigHelp()
	if !askConfirm("Generate yggdrasil.conf now?", true) {
		fmt.Println(T("Exiting."))
		os.Exit(0)
	}

	fmt.Println(T("Generating config..."))
	if err := generateConfig(detectedConfigPath); err != nil {
		fmt.Println(red(T("Failed to generate config: ")), err)
		fmt.Printf("%s %s\n", yellow(T("Generate it manually with:")), configGenCommand(detectedConfigPath))
		waitContinue()
		os.Exit(1)
	}
	fmt.Println(green(T("Config generated at %s", detectedConfigPath)))
}

func printMissingConfigHelp() {
	fmt.Println(red(T("Yggdrasil config not found in the default directory.")))
	fmt.Println(T("Expected path: %s", detectedConfigPath))
	fmt.Println()
	fmt.Println(yellow(T("Make sure that yggdrasil.conf is generated:")))
	fmt.Printf("  %s\n\n", configGenCommand(detectedConfigPath))
	fmt.Println(yellow(T("Or specify a different config path with the --config|-c flag, for example:")))
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
	fmt.Println(cyan(T("=== Yggdrasil Installer ===")))

	if err := currentPlatform.Install(); err != nil {
		fmt.Println(red(T("Installation failed: ")), err)
		waitContinue()
		return
	}

	// Re-detect the platform default after installation (a package may place
	// the config in a different location), but never override an explicit path.
	if !configPathExplicit {
		detectedConfigPath = currentPlatform.FindConfigPath()
	}

	if !fileExists(detectedConfigPath) {
		fmt.Println(T("Generating config..."))
		if err := generateConfig(detectedConfigPath); err != nil {
			fmt.Println(red(T("Failed to generate config automatically: ")), err)
			fmt.Println(yellow(T("Check if Yggdrasil is in PATH, or generate it manually with:")))
			fmt.Printf("  %s\n", configGenCommand(detectedConfigPath))
		} else {
			fmt.Println(green(T("Config generated at %s", detectedConfigPath)))
		}
	}
	waitContinue()
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
	return "", fmt.Errorf("%s", T("yggdrasil binary not found in PATH"))
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
		return fmt.Errorf("%s", T("yggdrasil -genconf failed: %v", err))
	}
	if len(out) == 0 {
		return fmt.Errorf("%s", T("yggdrasil -genconf returned empty output"))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// Helper function to keep code clean (used in Debian/Fedora blocks)

func runCommands(cmds [][]string) error {
	for _, c := range cmds {
		fmt.Println(T("Running: %s", strings.Join(c, " ")))
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %v", c[0], err)
		}
	}
	return nil
}

// --- Logic: GitHub & Peers ---

func fetchPeersStructure() (map[string][]string, error) {
	fmt.Println(T("Scanning repository..."))
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
		fmt.Println(red(T("Error: ")), err)
		return
	}

	// Build region list for selection
	var regions []string
	for region := range regionMap {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	// Ask user to select region
	regionChoices := []choice{{key: "All regions", label: T("All regions")}}
	for _, region := range regions {
		regionChoices = append(regionChoices, choice{key: region, label: region})
	}
	regionChoices = append(regionChoices, choice{key: "Cancel", label: T("Cancel")})

	selectedRegion, regionErr := askChoice("Select region to test peers from:", regionChoices, "All regions", 0)
	if regionErr != nil || selectedRegion == "Cancel" || selectedRegion == "" {
		return
	}

	// Collect URLs based on selection
	var allUrls []string
	if selectedRegion == "All regions" {
		for _, u := range regionMap {
			allUrls = append(allUrls, u...)
		}
		fmt.Println(T("Fetching from %d regions...", len(regionMap)))
	} else {
		allUrls = regionMap[selectedRegion]
		fmt.Println(T("Fetching peers from %s region...", cyan(selectedRegion)))
	}

	allPeers := fetchPeersFromURLs(allUrls)
	fmt.Println(T("Total peers found: %d. Testing all...", len(allPeers)))

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

	fmt.Println(T("Testing %d peers with 5 attempts each...", limit))
	fmt.Println(yellow(T("Note: Final peer verification happens after adding them to config.")))
	fmt.Println(yellow(T("Use 'Check Active Peers Status' to verify and 'Remove Dead Peers' to clean up.")))
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
					fmt.Print("\r", T("[%d/%d] ✓ Found %d peers (last: %s, jitter: %s)",
						tested, limit, len(ranked), peer.Latency, peer.Jitter))
				} else {
					fmt.Print("\r", T("[%d/%d] ✗ Testing... (%d peers found)", tested, limit, len(ranked)))
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
	fmt.Println()
	fmt.Println(cyan(T("=== Testing Summary ===")))
	fmt.Println(T("Total peers tested: %d", limit))
	fmt.Println(T("Peers found: %s", green(fmt.Sprintf("%d", len(ranked)))))

	if len(ranked) > 0 {
		fmt.Println(T("Best latency: %s", green(ranked[0].Latency.String())))
		fmt.Println(T("Best stability: %.2f%%", (1.0-ranked[0].Stability)*100))
	}
	fmt.Println()

	if len(ranked) == 0 {
		fmt.Println(yellow(T("No reachable peers found with latency < 5s.")))
		fmt.Println(yellow(T("This might mean:")))
		fmt.Println(yellow(T("  - Network connectivity issues")))
		fmt.Println(yellow(T("  - Firewall blocking connections")))
		fmt.Println(yellow(T("  - All peers are currently down")))
		waitContinue()
		return
	}

	// Show top 10 peers
	displayCount := 10
	if len(ranked) < displayCount {
		displayCount = len(ranked)
	}

	fmt.Println()
	fmt.Println(green(T("Top Peers by Latency & Stability:")))
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

		fmt.Print(T("%d. %s\n   Latency: %s (min: %s, max: %s, jitter: %s) - %s (%.0f%%)\n",
			i+1, ranked[i].URI, ranked[i].Latency, ranked[i].MinLatency,
			ranked[i].MaxLatency, ranked[i].Jitter, T(stability), stabilityPercent))
	}
	if len(ranked) > displayCount {
		fmt.Println()
		fmt.Println(T("(+%d more peers available)", len(ranked)-displayCount))
	}

	// Ask how many peers to add
	countToAdd := 0
	countChoice, err := askKeysSelect(
		"How many of the best peers would you like to add?",
		[]string{"3 peers (recommended)", "5 peers", "7 peers", "10 peers", "Custom number", "Cancel"},
		"3 peers (recommended)",
	)
	if err != nil {
		return
	}

	switch countChoice {
	case "3 peers (recommended)":
		countToAdd = 3
	case "5 peers":
		countToAdd = 5
	case "7 peers":
		countToAdd = 7
	case "10 peers":
		countToAdd = 10
	case "Custom number":
		customStr, err := promptInput(T("Enter number of peers to add (1-%d):", len(ranked)), "")
		if err != nil {
			return
		}
		fmt.Sscanf(customStr, "%d", &countToAdd)
		if countToAdd < 1 || countToAdd > len(ranked) {
			fmt.Println(red(T("Invalid number. Canceling.")))
			waitContinue()
			return
		}
	case "Cancel":
		return
	}

	if countToAdd > len(ranked) {
		countToAdd = len(ranked)
	}

	toAdd := []string{}
	fmt.Println(green(T("Peers to add:")))
	for i := 0; i < countToAdd; i++ {
		fmt.Println(T("%d. %s (%s avg, jitter: %s)", i+1, ranked[i].URI,
			ranked[i].Latency, ranked[i].Jitter))
		toAdd = append(toAdd, ranked[i].URI)
	}

	confirm := askConfirm("Confirm adding these peers?", true)
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

		regionChoices := make([]choice, 0, len(keys)+1)
		for _, key := range keys {
			regionChoices = append(regionChoices, choice{key: key, label: key})
		}
		regionChoices = append(regionChoices, choice{key: "Back", label: T("Back")})

		selReg, err := askChoice("Region:", regionChoices, "", 0)
		if err == errPromptInterrupt || selReg == "Back" || selReg == "" {
			return
		}

		peers := fetchPeersFromURLs(regionMap[selReg])
		if len(peers) == 0 {
			fmt.Println(T("No peers."))
			waitContinue()
			continue
		}
		indices, err := promptList(T("Select Peers:"), peers, -1, 0, true)
		if err == nil && len(indices) > 0 {
			selPeers := make([]string, len(indices))
			for i, index := range indices {
				selPeers[i] = peers[index]
			}
			addPeersToConfig(selPeers)
			restartServicePrompt()
		}
	}
}

func removePeersMenu() {
	current := getConfigPeers()
	if len(current) == 0 {
		fmt.Println(yellow(T("No peers in config.")))
		waitContinue()
		return
	}
	indices, err := promptList(T("Select to Remove:"), current, -1, 0, true)
	if err == nil && len(indices) > 0 {
		toRemove := make([]string, len(indices))
		for i, index := range indices {
			toRemove[i] = current[index]
		}
		if askConfirm("Remove selected peers from config?", false) {
			removePeersFromConfig(toRemove)
			restartServicePrompt()
		}
	}
}

func addCustomPeer() {
	input, err := promptInput(T("Enter URIs (space separated):"), "")
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
		fmt.Println(red(T("Failed to update config: ")), err)
		return
	}
	fmt.Println(green(T("Peers added and config formatted.")))
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
		fmt.Println(red(T("Failed to update config: ")), err)
		return
	}
	fmt.Println(green(T("Peers removed.")))
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
	waitContinue()
}

func restartServicePrompt() {
	if askConfirm("Restart Yggdrasil service now?", true) {
		if err := currentPlatform.ManageService("Restart"); err != nil {
			fmt.Println(red(T("Restart failed: ")), err)
		} else {
			fmt.Println(green(T("Service restarted.")))
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
	fmt.Println(cyan(T("=== Active Peers Status ===\n")))

	cmdName := linuxExe
	if isWindows {
		cmdName = windowsExe
	}

	out, err := exec.Command(cmdName, "getPeers").CombinedOutput()
	if err != nil {
		fmt.Println(red(T("Error: ")), err)
		fmt.Println(yellow(T("Make sure Yggdrasil service is running.")))
		waitContinue()
		return
	}

	fmt.Println(string(out))
	fmt.Println(yellow(T("\nTip: Use 'Remove Dead Peers' to clean up peers with 'Down' status.")))
	waitContinue()
}

// removeDeadPeers removes peers that are currently in "Down" state
func removeDeadPeers() {
	clearScreen()
	fmt.Println(cyan(T("=== Remove Dead Peers ===\n")))

	cmdName := linuxExe
	if isWindows {
		cmdName = windowsExe
	}

	// Get current peer status from yggdrasilctl
	fmt.Println(T("Fetching peer status from Yggdrasil..."))
	out, err := exec.Command(cmdName, "-json", "getPeers").CombinedOutput()
	if err != nil {
		fmt.Println(red(T("Error getting peer status: ")), err)
		fmt.Println(yellow(T("Make sure Yggdrasil service is running.")))
		waitContinue()
		return
	}

	// Parse the JSON response - it's a map with "peers" array
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		fmt.Println(red(T("Error parsing peers data: ")), err)
		waitContinue()
		return
	}

	// Get the peers array
	peersData, ok := result["peers"].([]interface{})
	if !ok {
		fmt.Println(yellow(T("No peer data found in response.")))
		fmt.Println(yellow(T("Make sure Yggdrasil service is running and has been started with some peers.")))
		waitContinue()
		return
	}

	if len(peersData) == 0 {
		fmt.Println(yellow(T("No active peer connections found.")))
		fmt.Println(yellow(T("Yggdrasil might still be starting up or no peers are configured.")))
		waitContinue()
		return
	}

	fmt.Print(T("\nFound %d peer connection(s) in Yggdrasil. Analyzing...\n\n", len(peersData)))

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
	fmt.Println(T("%s: %d", green(T("Up")), len(upPeers)))
	fmt.Println(T("%s: %d", red(T("Down")), len(downPeers)))
	fmt.Println()

	if len(downPeers) == 0 {
		fmt.Println(green(T("✓ All peers are up! No dead peers to remove.")))
		waitContinue()
		return
	}

	// Show dead peers with errors
	fmt.Println(red(T("Dead peers (status: Down):")))
	for i, peer := range downPeers {
		fmt.Printf("%d. %s\n", i+1, peer.URI)
		if peer.LastError != "" {
			// Truncate error if too long
			errMsg := peer.LastError
			if len(errMsg) > 80 {
				errMsg = errMsg[:77] + "..."
			}
			fmt.Print(T("   Error: %s\n", yellow(errMsg)))
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
		fmt.Println(yellow(T("Note: Some dead peers are not in config (possibly added temporarily):")))
		for _, uri := range deadPeersNotInConfig {
			fmt.Printf("  - %s\n", uri)
		}
		fmt.Println()
	}

	if len(deadPeersInConfig) == 0 {
		fmt.Println(yellow(T("No dead peers found in config to remove.")))
		fmt.Println(yellow(T("(All dead peers were added temporarily, not via config)")))
		waitContinue()
		return
	}

	// Ask for confirmation
	fmt.Println(T("Found %s dead peers in config:", red(fmt.Sprintf("%d", len(deadPeersInConfig)))))
	for i, uri := range deadPeersInConfig {
		fmt.Printf("%d. %s\n", i+1, uri)
	}
	fmt.Println()

	if askConfirm("Remove these dead peers from config?", false) {
		removePeersFromConfig(deadPeersInConfig)
		fmt.Println(green(T("\n✓ Dead peers removed from config.")))
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

// waitContinue blocks until the user presses any key or clicks with the mouse.
func waitContinue() {
	fmt.Println()
	fmt.Println(T("Press any key or click to continue..."))

	reader := newInputReader(true)
	defer reader.Close()

	for {
		ev, err := reader.ReadEvent()
		if err != nil {
			// Not an interactive terminal: fall back to a plain line read.
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			return
		}
		if ev.mouse == nil && ev.key == terminal.KeyInterrupt {
			fmt.Println(T("Bye!"))
			os.Exit(0)
		}
		if continueEvent(ev) {
			return
		}
	}
}

// continueEvent reports whether an input event dismisses the continue prompt.
func continueEvent(ev inputEvent) bool {
	if ev.mouse != nil {
		return ev.mouse.button == mouseLeft
	}
	return ev.key != terminal.IgnoreKey
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
	          ` + T("Configurator v") + version
	fmt.Println(cyan(banner))
}
