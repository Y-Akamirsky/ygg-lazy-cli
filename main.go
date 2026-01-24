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
	"github.com/fatih/color"
)

// --- Constants & Vars ---

const (
	repoOwner    = "yggdrasil-network"
	repoPeers    = "public-peers"
	windowsExe   = `C:\Program Files\Yggdrasil\yggdrasilctl.exe`
	linuxExe     = "yggdrasilctl"
)

var (
	isWindows = runtime.GOOS == "windows"
	green     = color.New(color.FgGreen).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
	cyan      = color.New(color.FgCyan).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()

	// Global variable to store the detected config path
	detectedConfigPath string
)

// --- Structures for GitHub API ---

type GitTreeResponse struct {
	Tree []GitNode `json:"tree"`
}

type GitNode struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Url  string `json:"url"`
}

type Peer struct {
	URI     string
	Region  string
	Latency time.Duration
}

// --- Main ---

func main() {
	installFlag := flag.Bool("ygginstall", false, "Install Yggdrasil automatically")
	flag.Parse()

	checkAdmin()

	// Locate config file immediately
	detectedConfigPath = findConfigPath()

	if *installFlag {
		installYggdrasil()
		return
	}

	// Check if installed/config exists
	if detectedConfigPath == "" {
		color.Yellow("Config file not found or Yggdrasil not installed.")
		confirm := false
		survey.AskOne(&survey.Confirm{Message: "Do you want to install Yggdrasil now?"}, &confirm)
		if confirm {
			installYggdrasil()
			detectedConfigPath = findConfigPath() // Re-detect after install
		} else {
			fmt.Println("Exiting.")
			os.Exit(0)
		}
	}

	mainMenu()
}

// --- Config Path Logic ---

func findConfigPath() string {
	if isWindows {
		path := `C:\ProgramData\Yggdrasil\yggdrasil.conf`
		if fileExists(path) { return path }
		return path // Return default even if not exists (for creation)
	}

	// Priority list for Linux
	paths := []string{
		"/etc/yggdrasil.conf",
		"/etc/yggdrasil/yggdrasil.conf",
		"/etc/Yggdrasil/yggdrasil.conf", // Case sensitive check just in case
	}

	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}

	// Default fallback if nothing found
	return "/etc/yggdrasil.conf"
}

// --- Menus ---

func mainMenu() {
	for {
		clearScreen()
		printBanner()
		fmt.Printf("Config path: %s\n\n", detectedConfigPath)

		mode := ""
		prompt := &survey.Select{
			Message: "Main Menu:",
			Options: []string{
				"Auto-select Peers (Best Latency)",
				"Manual Peer Selection (By Region)",
				"Remove Peers",
				"Add Custom Peer",
				"Node Status (getself)",
				"Service Control (Start/Stop/Restart)",
				"Exit",
			},
		}
		survey.AskOne(prompt, &mode)

		switch mode {
			case "Auto-select Peers (Best Latency)":
				autoAddPeers()
			case "Manual Peer Selection (By Region)":
				manualAddPeers()
			case "Remove Peers":
				removePeersMenu()
			case "Add Custom Peer":
				addCustomPeer()
			case "Node Status (getself)":
				showStatus()
			case "Service Control (Start/Stop/Restart)":
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
			Message: "Service Control:",
			Options: []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart", "Back"},
		}
		survey.AskOne(prompt, &action)

		if action == "Back" {
			return
		}

		manageService(action)
		fmt.Println("\nPress Enter to continue...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}

// --- Logic: Installation ---

func installYggdrasil() {
	fmt.Println(cyan("=== Yggdrasil Installer ==="))

	if isWindows {
		installWindows()
	} else {
		installLinux()
	}

	// Post install: Generate config if needed
	detectedConfigPath = findConfigPath() // Refresh path
	if !fileExists(detectedConfigPath) {
		fmt.Println("Generating config...")
		cmdName := "yggdrasil"
		if isWindows { cmdName = `C:\Program Files\Yggdrasil\yggdrasil.exe` }

		out, err := exec.Command(cmdName, "-genconf").Output()
		if err == nil {
			// Ensure directory exists
			dir := filepath.Dir(detectedConfigPath)
			os.MkdirAll(dir, 0755)

			os.WriteFile(detectedConfigPath, out, 0644)
			fmt.Println(green("Config generated at " + detectedConfigPath))
		} else {
			fmt.Println(red("Failed to generate config: "), err)
		}
	}

	fmt.Println(green("Installation complete!"))
	fmt.Println("Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func installWindows() {
	fmt.Println("Downloading latest MSI...")
	url, err := getLatestReleaseURL("yggdrasil-go", ".msi")
	if err != nil {
		fmt.Println(red("Error fetching URL: "), err)
		return
	}

	filename := "yggdrasil_installer.msi"
	if err := downloadFile(filename, url); err != nil {
		fmt.Println(red("Download error: "), err)
		return
	}

	fmt.Println("Running installer (quiet mode)...")
	cmd := exec.Command("msiexec", "/i", filename, "/quiet", "/norestart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(red("Install error: "), err)
	}
	os.Remove(filename)
}

func installLinux() {
	distro := getLinuxDistro()
	fmt.Printf("Detected Distro: %s\n", distro)

	if strings.Contains(distro, "debian") || strings.Contains(distro, "ubuntu") || strings.Contains(distro, "mint") || strings.Contains(distro, "kali") {
		choice := ""
		prompt := &survey.Select{
			Message: "Installation Method:",
			Options: []string{"Download .deb from GitHub", "Use APT (Recommended)"},
		}
		survey.AskOne(prompt, &choice)

		if choice == "Download .deb from GitHub" {
			url, err := getLatestReleaseURL("yggdrasil-go", ".deb")
			if err != nil {
				fmt.Println(red("GitHub API Error:"), err)
				return
			}
			file := "ygg.deb"
			downloadFile(file, url)
			exec.Command("apt", "install", "./"+file, "-y").Run()
			os.Remove(file)
		} else {
			// APT Logic
			cmds := [][]string{
				{"mkdir", "-p", "/usr/local/share/keyrings"},
				{"sh", "-c", "curl -s https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/key/neilalexander.gpg | gpg --dearmor > /usr/local/share/keyrings/neilalexander.gpg"},
				{"sh", "-c", "echo 'deb [signed-by=/usr/local/share/keyrings/neilalexander.gpg] http://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' > /etc/apt/sources.list.d/yggdrasil.list"},
				{"apt", "update"},
				{"apt", "install", "yggdrasil", "-y"},
			}
			for _, c := range cmds {
				fmt.Printf("Running: %v\n", c)
				exec.Command(c[0], c[1:]...).Run()
			}
		}
	} else if strings.Contains(distro, "arch") || strings.Contains(distro, "manjaro") {
		fmt.Println("Using pacman...")
		exec.Command("pacman", "-S", "yggdrasil-go", "--noconfirm").Run()
	} else {
		fmt.Println(yellow("Unknown distro. Please install Yggdrasil manually via your package manager."))
	}
}

// --- Logic: GitHub Recursive Parsing ---

func fetchPeersStructure() (map[string][]string, error) {
	fmt.Println("Scanning public-peers repository structure...")

	// Get Recursive Tree
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/master?recursive=1", repoOwner, repoPeers)
	resp, err := http.Get(url)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var treeResp GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return nil, err
	}

	// Map: Region -> List of File URLs (raw)
	regionMap := make(map[string][]string)

	for _, node := range treeResp.Tree {
		if node.Type != "blob" { continue } // Skip directories
		if strings.HasSuffix(node.Path, "README.md") || strings.HasSuffix(node.Path, "LICENSE") { continue }
		if !strings.HasSuffix(node.Path, ".md") { continue }

		// Path example: "Europe/Germany.md" or "Asia/Japan.md"
		parts := strings.Split(node.Path, "/")
		if len(parts) < 2 { continue } // Skip root files if any

		region := parts[0] // "Europe"
		// file := parts[1]   // "Germany.md"

		rawUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", repoOwner, repoPeers, node.Path)
		regionMap[region] = append(regionMap[region], rawUrl)
	}

	return regionMap, nil
}

// Fetch content of specific MD files and extract URIs
func fetchPeersFromURLs(urls []string) []string {
	var allPeers []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, 10) // Limit concurrency to 10

	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{} // Acquire
			defer func() { <-sem }() // Release

			resp, err := http.Get(url)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				peers := extractPeersFromText(string(body))

				mu.Lock()
				allPeers = append(allPeers, peers...)
				mu.Unlock()
			}
		}(u)
	}
	wg.Wait()
	return allPeers
}

func extractPeersFromText(text string) []string {
	var peers []string
	// Matches tcp://, tls://, quic://, ws:// inside code blocks or plain text
	// We use a simple regex for URI detection
	re := regexp.MustCompile(`(tcp|tls|quic|ws)://[a-zA-Z0-9\.\-\:\[\]]+`)
	matches := re.FindAllString(text, -1)
	peers = append(peers, matches...)
	return peers
}

// --- Logic: Peer Menus ---

func autoAddPeers() {
	regionMap, err := fetchPeersStructure()
	if err != nil {
		fmt.Println(red("Failed to fetch repository tree: "), err)
		return
	}

	// Flatten all URLs to scan everything (or ask user?)
	// For "Auto", users usually want best global ping.
	var allUrls []string
	for _, urls := range regionMap {
		allUrls = append(allUrls, urls...)
	}

	fmt.Printf("Found %d regions. Fetching peer lists from %d files...\n", len(regionMap), len(allUrls))
	allPeers := fetchPeersFromURLs(allUrls)
	fmt.Printf("Total distinct peers found: %d\n", len(allPeers))

	if len(allPeers) == 0 {
		fmt.Println(red("No peers found in repository."))
		return
	}

	// Limit pinging to random 30 to save time? Or ping all?
	// Let's ping up to 20 random ones to be fast for "Lazy" CLI
	// Ideally we ping more, but let's keep it responsive.
	fmt.Println("Pinging peers (this may take a moment)...")

	// Shuffle
	// (Skipping math/rand seed for brevity, defaults to deterministic in old Go, random in new Go)
	// simple shuffle
	for i := range allPeers {
		j := int(time.Now().UnixNano()) % len(allPeers)
		allPeers[i], allPeers[j] = allPeers[j], allPeers[i]
	}

	limitCheck := 25
	if len(allPeers) < limitCheck { limitCheck = len(allPeers) }
	checkList := allPeers[:limitCheck]

	var rankedPeers []Peer
	for _, uri := range checkList {
		fmt.Print(".")
		latency := pingPeer(uri)
		if latency < 5*time.Second {
			rankedPeers = append(rankedPeers, Peer{URI: uri, Latency: latency})
		}
	}
	fmt.Println()

	sort.Slice(rankedPeers, func(i, j int) bool {
		return rankedPeers[i].Latency < rankedPeers[j].Latency
	})

	toAdd := []string{}
	count := 3
	if len(rankedPeers) < count { count = len(rankedPeers) }

	fmt.Println(green("Best Peers:"))
	for i := 0; i < count; i++ {
		p := rankedPeers[i]
		fmt.Printf("%d. %s (%s)\n", i+1, p.URI, p.Latency)
		toAdd = append(toAdd, p.URI)
	}

	if len(toAdd) == 0 {
		fmt.Println(yellow("Could not connect to any peers."))
		waitEnter()
		return
	}

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Add these peers to config?"}, &confirm)
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
			fmt.Println(red("Error:"), err)
			return
		}

		regionKeys := []string{}
		for k := range regionMap {
			regionKeys = append(regionKeys, k)
		}
		sort.Strings(regionKeys)
		regionKeys = append(regionKeys, "Back")

		selectedRegion := ""
		survey.AskOne(&survey.Select{
			Message: "Choose Region:",
			Options: regionKeys,
		}, &selectedRegion)

		if selectedRegion == "Back" { return }

		// Fetch files in this region
		fmt.Printf("Fetching peers for %s...\n", selectedRegion)
		urls := regionMap[selectedRegion]
		peersList := fetchPeersFromURLs(urls)

		if len(peersList) == 0 {
			fmt.Println("No peers found in this region.")
			waitEnter()
			continue
		}

		selectedPeers := []string{}
		survey.AskOne(&survey.MultiSelect{
			Message: "Select Peers (Space to toggle, Enter to confirm):",
			      Options: peersList,
		}, &selectedPeers)

		if len(selectedPeers) > 0 {
			addPeersToConfig(selectedPeers)
			restartServicePrompt()
		}
	}
}

func removePeersMenu() {
	currentPeers := getConfigPeers()
	if len(currentPeers) == 0 {
		fmt.Println(yellow("No peers found in config file."))
		waitEnter()
		return
	}

	toRemove := []string{}
	survey.AskOne(&survey.MultiSelect{
		Message: "Select peers to REMOVE:",
		Options: currentPeers,
	}, &toRemove)

	if len(toRemove) > 0 {
		removePeersFromConfig(toRemove)
		restartServicePrompt()
	}
}

func addCustomPeer() {
	peersStr := ""
	survey.AskOne(&survey.Input{
		Message: "Enter peers (tcp://... tls://...) separated by space:",
	}, &peersStr)

	list := strings.Fields(peersStr)
	if len(list) > 0 {
		addPeersToConfig(list)
		restartServicePrompt()
	}
}

// --- Logic: Robust Config Editor ---

// Reads config and extracts peers using a State Machine approach
// strictly looking for "Peers: [" ... "]"
func getConfigPeers() []string {
	f, err := os.Open(detectedConfigPath)
	if err != nil { return []string{} }
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var peers []string
	inPeersBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 1. Detect Start
		if strings.HasPrefix(line, "Peers:") {
			if strings.Contains(line, "[") {
				inPeersBlock = true
				// Handle case: Peers: [ "tcp://..." ] on one line
				if strings.Contains(line, "]") {
					// One-liner
					extractFromLine(line, &peers)
					inPeersBlock = false
				}
			}
			continue
		}

		// 2. Detect End
		if inPeersBlock && strings.Contains(line, "]") {
			inPeersBlock = false
			// Handle case: "tcp://..." ]
			extractFromLine(line, &peers)
			continue
		}

		// 3. Extract inside block
		if inPeersBlock {
			extractFromLine(line, &peers)
		}
	}
	return peers
}

func extractFromLine(line string, target *[]string) {
	// Remove comments
	if idx := strings.Index(line, "//"); idx != -1 { line = line[:idx] }
		if idx := strings.Index(line, "#"); idx != -1 { line = line[:idx] }

		// Remove syntax
		line = strings.ReplaceAll(line, "Peers:", "")
		line = strings.ReplaceAll(line, "[", "")
		line = strings.ReplaceAll(line, "]", "")
		line = strings.ReplaceAll(line, "\"", "")
		line = strings.ReplaceAll(line, ",", "")
		line = strings.TrimSpace(line)

		if strings.Contains(line, "://") {
			*target = append(*target, line)
		}
}

func addPeersToConfig(newPeers []string) {
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil {
		fmt.Println(red("Cannot read config: "), err)
		return
	}
	content := string(contentBytes)

	// Create block to insert
	insertion := ""
	for _, p := range newPeers {
		if !strings.Contains(content, p) {
			insertion += fmt.Sprintf("\n    \"%s\"", p)
		}
	}

	if insertion == "" {
		fmt.Println("Selected peers are already in config.")
		return
	}

	// Replace logic: Find "Peers: [" and append
	// We use regex here just to find the opening tag effectively
	re := regexp.MustCompile(`(Peers:\s*\[)`)
	if re.MatchString(content) {
		newContent := re.ReplaceAllString(content, "${1}"+insertion)
		err = os.WriteFile(detectedConfigPath, []byte(newContent), 0644)
		if err != nil {
			fmt.Println(red("Write error: "), err)
		} else {
			fmt.Println(green("Peers added successfully."))
		}
	} else {
		// If Peers: [] block is missing (rare), append it to end
		fmt.Println(yellow("'Peers: []' block not found. Appending to end of file."))
		block := "\nPeers: [" + insertion + "\n]\n"
		f, err := os.OpenFile(detectedConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(block)
			f.Close()
		}
	}
}

func removePeersFromConfig(toRemove []string) {
	contentBytes, err := os.ReadFile(detectedConfigPath)
	if err != nil { return }

	lines := strings.Split(string(contentBytes), "\n")
	var newLines []string

	for _, line := range lines {
		keep := true
		for _, rem := range toRemove {
			// Check if line contains the peer URI
			if strings.Contains(line, rem) {
				keep = false
				break
			}
		}
		if keep {
			newLines = append(newLines, line)
		}
	}

	output := strings.Join(newLines, "\n")
	os.WriteFile(detectedConfigPath, []byte(output), 0644)
	fmt.Println(green("Peers removed."))
}

// --- Utils ---

func showStatus() {
	cmdName := linuxExe
	if isWindows { cmdName = windowsExe }

	cmd := exec.Command(cmdName, "getself")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(red("Status error (is service running?):"), err)
	} else {
		fmt.Println(string(out))
	}
	waitEnter()
}

func manageService(action string) {
	var cmd *exec.Cmd
	if isWindows {
		verb := ""
		switch action {
			case "Start": verb = "start"
			case "Stop": verb = "stop"
			case "Restart":
				exec.Command("net", "stop", "yggdrasil").Run()
				verb = "start"
			case "Enable Autostart":
				cmd = exec.Command("sc", "config", "yggdrasil", "start=", "auto")
			case "Disable Autostart":
				cmd = exec.Command("sc", "config", "yggdrasil", "start=", "demand")
		}
		if verb != "" {
			cmd = exec.Command("net", verb, "yggdrasil")
		}
	} else {
		verb := strings.ToLower(action)
		if action == "Enable Autostart" { verb = "enable" }
		if action == "Disable Autostart" { verb = "disable" }
		cmd = exec.Command("systemctl", verb, "yggdrasil")
	}

	if cmd != nil {
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(red("Error: "), err, string(out))
		} else {
			fmt.Println(green("Success."))
		}
	}
}

func restartServicePrompt() {
	restart := false
	survey.AskOne(&survey.Confirm{Message: "Restart Yggdrasil to apply changes?"}, &restart)
	if restart {
		manageService("Restart")
	}
	waitEnter()
}

func pingPeer(uri string) time.Duration {
	parts := strings.Split(uri, "://")
	if len(parts) < 2 { return 999 * time.Second }
	address := parts[1]

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return 999 * time.Second
	}
	conn.Close()
	return time.Since(start)
}

func checkAdmin() {
	var isAdmin bool
	if isWindows {
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		isAdmin = err == nil
	} else {
		isAdmin = os.Geteuid() == 0
	}
	if !isAdmin {
		fmt.Println(red("ERROR: Administrator/Root privileges required!"))
		if !isWindows { fmt.Println("Try: sudo ./ygg-lazy-cli") }
		waitEnter()
		os.Exit(1)
	}
}

func getLinuxDistro() string {
	out, _ := os.ReadFile("/etc/os-release")
	return strings.ToLower(string(out))
}

func getLatestReleaseURL(repoName string, suffix string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName))
	if err != nil { return "", err }
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assets, ok := result["assets"].([]interface{})
	if !ok { return "", fmt.Errorf("no assets found") }

	for _, a := range assets {
		asset := a.(map[string]interface{})
		name := asset["name"].(string)
		if strings.HasSuffix(name, suffix) {
			return asset["browser_download_url"].(string), nil
		}
	}
	return "", fmt.Errorf("file %s not found in release", suffix)
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil { return err }
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil { return err }
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) { return false }
	return !info.IsDir()
}

func waitEnter() {
	fmt.Println("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func clearScreen() {
	fmt.Print("\033[H\033[2J") // ANSI Clear works on modern Win10+ Terminals too usually
	if isWindows {
		// Fallback for older cmd
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
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
	           Configurator v1.1
	`))
}
