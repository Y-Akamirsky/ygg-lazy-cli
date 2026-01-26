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
	URI        string
	Latency    time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
	Jitter     time.Duration // Standard deviation of latency
	Stability  float64       // Lower is better (0-1 scale)
}

// --- Main ---

func main() {
	// 1. Check/Request Admin Privileges immediately
	ensureAdmin()

	installFlag := flag.Bool("ygginstall", false, "Install Yggdrasil automatically")
	flag.Parse()

	detectedConfigPath = findConfigPath()

	// Handle Install Flag
	if *installFlag {
		installYggdrasil()
		return
	}

	// Check if config actually exists on disk
	if !fileExists(detectedConfigPath) {
		color.Yellow("Config file not found (%s).", detectedConfigPath)
		confirm := false
		survey.AskOne(&survey.Confirm{Message: "Yggdrasil config missing. Install/Generate now?"}, &confirm)
		if confirm {
			installYggdrasil()
			detectedConfigPath = findConfigPath()
		} else {
			fmt.Println("Exiting.")
			os.Exit(0)
		}
	}

	mainMenu()
}

// --- Admin & Path Helpers ---

func ensureAdmin() {
	if isWindows {
		if !amAdminWindows() {
			fmt.Println(yellow("Administrator privileges required. Relaunching..."))
			runMeElevated()
		}
	} else {
		if os.Geteuid() != 0 {
			fmt.Println(red("Root required! Please run with sudo."))
			os.Exit(1)
		}
	}
}

func amAdminWindows() bool {
	// "net session" requires admin. If it fails (exit code != 0), we are not admin.
	_, err := exec.Command("net", "session").Output()
	return err == nil
}

func runMeElevated() {
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	// Reconstruct args
	args := strings.Join(os.Args[1:], " ")

	// Use PowerShell Start-Process -Verb RunAs to trigger UAC
	cmd := exec.Command("powershell", "Start-Process",
		"-FilePath", fmt.Sprintf("'%s'", exe),
		"-ArgumentList", fmt.Sprintf("'%s'", args),
		"-Verb", "RunAs",
		"-WorkingDirectory", fmt.Sprintf("'%s'", cwd))

	err := cmd.Start()
	if err != nil {
		fmt.Println(red("Failed to elevate: "), err)
		os.Exit(1)
	}
	os.Exit(0)
}

func findConfigPath() string {
	if isWindows {
		paths := []string{
			`C:\ProgramData\Yggdrasil\yggdrasil.conf`,
			`C:\Program Files\Yggdrasil\yggdrasil.conf`,
		}
		for _, p := range paths {
			if fileExists(p) {
				return p
			}
		}
		return `C:\ProgramData\Yggdrasil\yggdrasil.conf`
	}

	paths := []string{
		"/etc/yggdrasil.conf",
		"/etc/yggdrasil/yggdrasil.conf",
		"/etc/Yggdrasil/yggdrasil.conf",
	}
	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}
	return "/etc/yggdrasil.conf"
}

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
				"Remove Peers",
				"Add Custom Peer",
				"Node Status",
				"Service Control",
				"Exit",
			},
			PageSize: 10,
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
		prompt := &survey.Select{
			Message: "Service Control (Esc to back):",
			Options: []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart", "Back"},
		}
		err := survey.AskOne(prompt, &action)
		if err == terminal.InterruptErr || action == "Back" {
			return
		}
		manageService(action)
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

func installYggdrasil() {
	fmt.Println(cyan("=== Yggdrasil Installer ==="))

	if isWindows {
		if err := installWindows(); err != nil {
			fmt.Println(red("Installation failed: "), err)
			waitEnter()
			return
		}
	} else {
		if err := installLinux(); err != nil {
			fmt.Println(red("Installation failed: "), err)
			waitEnter()
			return
		}
	}

	detectedConfigPath = findConfigPath()
	if !fileExists(detectedConfigPath) {
		fmt.Println("Generating config...")
		cmdName := "yggdrasil"
		if isWindows {
			cmdName = `C:\Program Files\Yggdrasil\yggdrasil.exe`
		}

		// Try to generate
		out, err := exec.Command(cmdName, "-genconf").Output()
		if err == nil && len(out) > 0 {
			dir := filepath.Dir(detectedConfigPath)
			os.MkdirAll(dir, 0755)
			os.WriteFile(detectedConfigPath, out, 0644)
			fmt.Println(green("Config generated at " + detectedConfigPath))
		} else {
			fmt.Println(red("Failed to generate config automatically. Check if Yggdrasil is in PATH."))
		}
	}
	waitEnter()
}

func installWindows() error {
	fmt.Println("Fetching latest release...")

	// Determine Architecture for Windows
	arch := runtime.GOARCH
	var searchStr string
	if arch == "amd64" {
		searchStr = "x64"
	} else if arch == "386" {
		searchStr = "x86"
	} else {
		searchStr = arch
	}

	url, err := getLatestReleaseURL("yggdrasil-go", ".msi", searchStr)
	if err != nil {
		return err
	}

	filename := "yggdrasil_installer.msi"
	fmt.Printf("Downloading %s (Arch: %s)...\n", url, arch)
	if err := downloadFile(filename, url); err != nil {
		return err
	}

	fmt.Println("Running MSI installer (Admin rights required)...")
	// Using /norestart /qn for silent install, or /qb for basic UI
	cmd := exec.Command("msiexec", "/i", filename, "/qb", "/norestart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	os.Remove(filename)
	fmt.Println(green("Installation complete."))
	return nil
}

func installLinux() error {
	distroID, distroLike := getLinuxDistroInfo()
	fmt.Printf("Detected Distro: %s (Like: %s)\n", distroID, distroLike)

	// --- Detection Logic ---

	// Check if Debian/Ubuntu based
	isDebian := strings.Contains(distroID, "debian") || strings.Contains(distroID, "ubuntu") ||
		strings.Contains(distroID, "mint") || strings.Contains(distroID, "kali") ||
		strings.Contains(distroID, "pop") || strings.Contains(distroLike, "debian") ||
		strings.Contains(distroLike, "ubuntu")

	// Check if Arch based
	isArch := strings.Contains(distroID, "arch") || strings.Contains(distroID, "manjaro") ||
		strings.Contains(distroID, "cachyos") || strings.Contains(distroID, "endeavour") ||
		strings.Contains(distroLike, "arch")

	// Check if Fedora/RHEL based
	isFedora := strings.Contains(distroID, "fedora") || strings.Contains(distroID, "rhel") ||
		strings.Contains(distroID, "centos") || strings.Contains(distroID, "almalinux") ||
		strings.Contains(distroID, "rocky") || strings.Contains(distroLike, "fedora")

	// Check if Void Linux
	isVoid := strings.Contains(distroID, "void")

	// Check if Alpine Linux
	isAlpine := strings.Contains(distroID, "alpine")

	// --- Installation Logic ---

	if isDebian {
		choice := ""
		survey.AskOne(&survey.Select{
			Message: "Installation Method:",
			Options: []string{"Download .deb", "Use APT Repository"},
		}, &choice)

		if choice == "Download .deb" {
			arch := runtime.GOARCH
			// Adjust arch string if necessary for .deb naming conventions
			if arch == "arm64" {
				// usually matches, but good to keep in mind
			}

			url, err := getLatestReleaseURL("yggdrasil-go", ".deb", arch)
			if err != nil {
				return err
			}

			fmt.Printf("Downloading %s...\n", url)
			if err := downloadFile("ygg.deb", url); err != nil {
				return err
			}
			defer os.Remove("ygg.deb")

			cmd := exec.Command("apt", "install", "./ygg.deb", "-y")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		} else {
			// APT logic
			cmds := [][]string{
				{"sh", "-c", "curl -s https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/key/neilalexander.gpg | gpg --dearmor > /usr/local/share/keyrings/neilalexander.gpg"},
				{"sh", "-c", "echo 'deb [signed-by=/usr/local/share/keyrings/neilalexander.gpg] http://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' > /etc/apt/sources.list.d/yggdrasil.list"},
				{"apt", "update"},
				{"apt", "install", "yggdrasil", "-y"},
			}
			return runCommands(cmds)
		}

	} else if isFedora {
		// Fedora uses COPR for Yggdrasil
		fmt.Println("Detected Fedora-based system. Using DNF and COPR...")

		// Ensure dnf-plugins-core is installed to use 'copr' command
		// Note: On newer Fedora versions, this is often built-in or named 'dnf-command(copr)'
		cmds := [][]string{
			{"dnf", "install", "dnf-plugins-core", "-y"},
			{"dnf", "copr", "enable", "neilalexander/yggdrasil", "-y"},
			{"dnf", "install", "yggdrasil", "-y"},
		}
		return runCommands(cmds)

	} else if isVoid {
		// Void Linux logic (xbps)
		fmt.Println("Detected Void Linux. Using xbps-install...")

		// -S to sync repo index, -y for auto yes
		cmd := exec.Command("xbps-install", "-S", "yggdrasil", "-y")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("xbps-install failed: %v", err)
		}
		fmt.Println("Installed successfully via XBPS.")
		return nil

	} else if isAlpine {
		// Alpine Linux logic (apk)
		fmt.Println("Detected Alpine Linux. Using apk...")

		// Assuming yggdrasil is available in community/testing repos enabled by user
		cmd := exec.Command("apk", "add", "yggdrasil")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apk failed: %v", err)
		}
		fmt.Println("Installed successfully via APK.")
		return nil

	} else if isArch {
		// Arch / CachyOS logic
		fmt.Println("Detected Arch-based system. Attempting to install via pacman...")
		pkgs := []string{"yggdrasil-go", "yggdrasil"}
		for _, pkg := range pkgs {
			fmt.Printf("Trying package: %s\n", pkg)
			cmd := exec.Command("pacman", "-S", pkg, "--noconfirm")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err == nil {
				fmt.Println("Installed successfully.")
				return nil
			}
			fmt.Printf("Failed to install %s. Trying next...\n", pkg)
		}
		return fmt.Errorf("could not install yggdrasil via pacman. Try installing manually from AUR")
	}

	return fmt.Errorf("unsupported distribution: %s", distroID)
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
	var allUrls []string
	for _, u := range regionMap {
		allUrls = append(allUrls, u...)
	}

	fmt.Printf("Fetching from %d regions...\n", len(regionMap))
	allPeers := fetchPeersFromURLs(allUrls)
	fmt.Printf("Total peers found: %d. Pinging subset...\n", len(allPeers))

	// Shuffle
	for i := range allPeers {
		j := int(time.Now().UnixNano()) % len(allPeers)
		allPeers[i], allPeers[j] = allPeers[j], allPeers[i]
	}

	limit := 100
	if len(allPeers) < limit {
		limit = len(allPeers)
	}

	var ranked []Peer
	var mu sync.Mutex
	var wg sync.WaitGroup
	tested := 0

	// Use 20 concurrent workers for pinging
	workers := 20
	peerChan := make(chan string, limit)

	fmt.Printf("Testing %d peers with 5 attempts each (this may take a minute)...\n", limit)

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for uri := range peerChan {
				peer := pingPeerDetailed(uri)
				mu.Lock()
				tested++
				// Accept peers with reasonable latency (most important for mesh network)
				// Relaxed criteria: latency < 5s (was 3s), removed strict stability filter
				if peer.Latency < 5*time.Second {
					ranked = append(ranked, peer)
					fmt.Printf("\r[%d/%d] ✓ Found %d peers (last: %s, jitter: %s, stability: %.0f%%)",
						tested, limit, len(ranked), peer.Latency, peer.Jitter, (1.0-peer.Stability)*100)
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
	// Lower latency and better stability = better peer
	sort.Slice(ranked, func(i, j int) bool {
		// Calculate score: latency + (latency * stability)
		// This gives weight to both metrics
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

func getConfigPeers() []string {
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil {
		return []string{}
	}
	content := string(contentBytes)
	var peers []string

	startIdx := strings.Index(content, "Peers: [")
	if startIdx == -1 {
		return peers
	}
	// Look for closing bracket with proper indentation (not IPv6 bracket)
	// Search for "\n  ]" or "\n]" which marks the end of the Peers array
	endIdx := strings.Index(content[startIdx:], "\n  ]")
	if endIdx == -1 {
		// Try alternative formatting
		endIdx = strings.Index(content[startIdx:], "\n]")
	}
	if endIdx == -1 {
		return peers
	}
	endIdx += startIdx

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
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil {
		return
	}
	content := string(contentBytes)

	// Combine existing peers + new peers
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

	// Rebuild the "Peers: [...]" block entirely
	newBlock := "Peers: ["
	for _, p := range finalList {
		newBlock += fmt.Sprintf("\n  %s", p) // Indent 2 spaces
	}
	newBlock += "\n]"

	// Inject into file
	startIdx := strings.Index(content, "Peers: [")
	if startIdx != -1 {
		// Found existing block, replace it
		// Look for properly formatted closing bracket (not IPv6 bracket)
		closingPattern := "\n  ]"
		endIdx := strings.Index(content[startIdx:], closingPattern)
		if endIdx == -1 {
			closingPattern = "\n]"
			endIdx = strings.Index(content[startIdx:], closingPattern)
		}
		if endIdx != -1 {
			// Skip past the entire closing pattern including the bracket
			endIdx = startIdx + endIdx + len(closingPattern)
			newContent := content[:startIdx] + newBlock + content[endIdx:]
			os.WriteFile(detectedConfigPath, []byte(newContent), 0644)
			fmt.Println(green("Peers added and config formatted."))
			return
		}
	}

	// Block not found, append to end
	f, _ := os.OpenFile(detectedConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("\n" + newBlock + "\n")
	f.Close()
	fmt.Println(green("Peers block appended."))
}

func removePeersFromConfig(toRemove []string) {
	// Re-use logic: Get current -> Filter -> Re-add remaining
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

	// Reconstruct the block manually to overwrite file
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil {
		return
	}
	content := string(contentBytes)

	newBlock := "Peers: ["
	for _, p := range keep {
		newBlock += fmt.Sprintf("\n  %s", p)
	}
	newBlock += "\n  ]\n"

	startIdx := strings.Index(content, "Peers: [")
	if startIdx != -1 {
		// Look for properly formatted closing bracket (not IPv6 bracket)
		closingPattern := "\n  ]"
		endIdx := strings.Index(content[startIdx:], closingPattern)
		if endIdx == -1 {
			closingPattern = "\n]"
			endIdx = strings.Index(content[startIdx:], closingPattern)
		}
		if endIdx != -1 {
			// Skip past the entire closing pattern including the bracket
			endIdx = startIdx + endIdx + len(closingPattern)
			newContent := content[:startIdx] + newBlock + content[endIdx:]
			os.WriteFile(detectedConfigPath, []byte(newContent), 0644)
			fmt.Println(green("Peers removed."))
		}
	}
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

func manageService(act string) {
	if isWindows {
		cmd := ""
		switch act {
		case "Start":
			cmd = "Start-Service yggdrasil"
		case "Stop":
			cmd = "Stop-Service yggdrasil"
		case "Restart":
			cmd = "Restart-Service yggdrasil -Force"
		default:
			return
		}
		fmt.Printf("Executing: %s\n", cmd)
		exec.Command("powershell", "-Command", cmd).Run()
	} else {
		verb := strings.ToLower(act)
		if act == "Enable Autostart" {
			verb = "enable"
		}
		if act == "Disable Autostart" {
			verb = "disable"
		}
		exec.Command("systemctl", verb, "yggdrasil").Run()
	}
	fmt.Println(green("Done."))
}

func restartServicePrompt() {
	r := false
	survey.AskOne(&survey.Confirm{Message: "Restart service to apply changes?"}, &r)
	if r {
		manageService("Restart")
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
		URI:        uri,
		Latency:    avgLatency,
		MinLatency: minLatency,
		MaxLatency: maxLatency,
		Jitter:     jitter,
		Stability:  stability,
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
	fmt.Println(cyan(`
	__   __           _
	\ \ / /          | |
	 \ V / __ _  __ _| |     __ _ _____   _
	  \ / / _' |/ _' | |    / _' |_  / | | |
	  | || (_| | (_| | |___| (_| |/ /| |_| |
	  \_/ \__, |\__, |______\__,_/___|\__, |
	       __/ | __/ |                 __/ |
	      |___/ |___/                 |___/
	          Configurator v0.1.4a
	`))
}
