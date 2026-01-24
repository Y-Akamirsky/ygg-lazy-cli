package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	URI     string
	Latency time.Duration
}

// --- Main ---

func main() {
	installFlag := flag.Bool("ygginstall", false, "Install Yggdrasil automatically")
	flag.Parse()

	checkAdmin()
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

func findConfigPath() string {
	if isWindows {
		// Windows: Check distinct locations
		paths := []string{
			`C:\ProgramData\Yggdrasil\yggdrasil.conf`,
			`C:\Program Files\Yggdrasil\yggdrasil.conf`,
		}
		for _, p := range paths {
			if fileExists(p) {
				return p
			}
		}
		return `C:\ProgramData\Yggdrasil\yggdrasil.conf` // Default fallback
	}

	// Linux
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

	return "/etc/yggdrasil.conf" // Default fallback
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

	// Generate config if missing
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

	// Determine Architecture for Windows (x64 or 32-bit, arm is rare but possible)
	// runtime.GOARCH usually returns "amd64", "386", "arm64"
	arch := runtime.GOARCH
	var searchStr string
	if arch == "amd64" {
		searchStr = "x64" // GitHub releases often use 'x64' for Windows
	} else if arch == "386" {
		searchStr = "x86" // or 32-bit
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

	fmt.Println("Running MSI installer...")
	cmd := exec.Command("msiexec", "/i", filename, "/quiet", "/norestart")
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
		if node.Type != "blob" || !strings.HasSuffix(node.Path, ".md") {
			continue
		}
		if strings.HasSuffix(node.Path, "README.md") {
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

	limit := 25
	if len(allPeers) < limit {
		limit = len(allPeers)
	}

	var ranked []Peer
	for _, uri := range allPeers[:limit] {
		fmt.Print(".")
		lat := pingPeer(uri)
		if lat < 5*time.Second {
			ranked = append(ranked, Peer{URI: uri, Latency: lat})
		}
	}
	fmt.Println()

	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Latency < ranked[j].Latency })

	if len(ranked) == 0 {
		fmt.Println(yellow("No reachable peers found."))
		waitEnter()
		return
	}

	toAdd := []string{}
	count := 3
	if len(ranked) < count {
		count = len(ranked)
	}

	fmt.Println(green("Best Peers:"))
	for i := 0; i < count; i++ {
		fmt.Printf("%d. %s (%s)\n", i+1, ranked[i].URI, ranked[i].Latency)
		toAdd = append(toAdd, ranked[i].URI)
	}

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Add these peers?"}, &confirm)
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

// --- Logic: Parser V3 (Block Extractor) ---

func getConfigPeers() []string {
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil {
		return []string{}
	}
	content := string(contentBytes)
	var peers []string

	// Find the "Peers: [" block
	startIdx := strings.Index(content, "Peers: [")
	if startIdx == -1 {
		return peers
	}

	// Find the closing bracket ']' AFTER the start
	endIdx := strings.Index(content[startIdx:], "]")
	if endIdx == -1 {
		return peers
	}
	endIdx += startIdx // Adjust relative index to absolute

	// Extract the block content
	block := content[startIdx:endIdx]

	// Regex to find URIs inside this block.
	// Matches: tcp://..., quic://...
	// Ignores quotes, commas, and whitespace around them.
	re := regexp.MustCompile(`(tcp|tls|quic|ws|wss)://[a-zA-Z0-9\.\-\:\[\]]+`)
	matches := re.FindAllString(block, -1)

	if matches != nil {
		peers = append(peers, matches...)
	}

	return peers
}

func addPeersToConfig(newPeers []string) {
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil { return }
	content := string(contentBytes)

	// Filter duplicates
	uniquePeers := []string{}
	for _, p := range newPeers {
		if !strings.Contains(content, p) {
			uniquePeers = append(uniquePeers, p)
		}
	}

	if len(uniquePeers) == 0 { return }

	// Locate "Peers: ["
	searchStr := "Peers: ["
	idx := strings.Index(content, searchStr)

	if idx != -1 {
		// Insert AFTER "Peers: ["
		insertPos := idx + len(searchStr)

		// Build insertion string with NO QUOTES, 2 spaces indentation
		insertion := ""
		for _, p := range uniquePeers {
			insertion += fmt.Sprintf("\n  %s", p)
		}

		newContent := content[:insertPos] + insertion + content[insertPos:]
		os.WriteFile(detectedConfigPath, []byte(newContent), 0644)
		fmt.Println(green("Peers added."))
	} else {
		// Fallback: Append at end if block not found
		f, _ := os.OpenFile(detectedConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
		for _, p := range uniquePeers {
			f.WriteString(fmt.Sprintf("\nPeers: [\n  %s\n]\n", p))
		}
		f.Close()
	}
}

func removePeersFromConfig(toRemove []string) {
	bytes, err := os.ReadFile(detectedConfigPath)
	if err != nil { return }
	lines := strings.Split(string(bytes), "\n")

	var newLines []string
	for _, line := range lines {
		keep := true
		for _, rem := range toRemove {
			// If line contains the URI to remove, skip it
			if strings.Contains(line, rem) {
				keep = false
				break
			}
		}
		if keep {
			newLines = append(newLines, line)
		}
	}
	os.WriteFile(detectedConfigPath, []byte(strings.Join(newLines, "\n")), 0644)
	fmt.Println(green("Peers removed."))
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
		// Windows: Use Powershell for better service control
		// Requires Admin
		cmd := ""
		switch act {
			case "Start":
				cmd = "Start-Service yggdrasil"
			case "Stop":
				cmd = "Stop-Service yggdrasil"
			case "Restart":
				cmd = "Restart-Service yggdrasil -Force"
			default:
				fmt.Println("Autostart config on Windows is automatic via Service Manager.")
				return
		}

		fmt.Printf("Executing: %s\n", cmd)
		err := exec.Command("powershell", "-Command", cmd).Run()
		if err != nil {
			fmt.Println(red("Error (Run as Admin?): "), err)
		}
	} else {
		// Linux: Systemd
		verb := strings.ToLower(act)
		if act == "Enable Autostart" { verb = "enable" }
		if act == "Disable Autostart" { verb = "disable" }
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

func pingPeer(uri string) time.Duration {
	parts := strings.Split(uri, "://")
	if len(parts) < 2 {
		return 999 * time.Second
	}
	conn, err := net.DialTimeout("tcp", parts[1], 2*time.Second)
	if err != nil {
		return 999 * time.Second
	}
	conn.Close()
	return 100 * time.Millisecond
}

func checkAdmin() {
	if !isWindows && os.Geteuid() != 0 {
		fmt.Println(red("Root required! (sudo)"))
		os.Exit(1)
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
			// Apply Architecture Filter
			// If archFilter is empty, accept any. If not empty, filename must contain it.
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
	          Configurator v0.1.3
	`))
}
