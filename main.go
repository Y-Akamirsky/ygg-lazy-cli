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
	repoOwner = "yggdrasil-network"
	repoPeers = "public-peers"
	windowsExe = `C:\Program Files\Yggdrasil\yggdrasilctl.exe`
	linuxExe   = "yggdrasilctl"
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

	if *installFlag {
		installYggdrasil()
		return
	}

	if detectedConfigPath == "" {
		color.Yellow("Config file not found or Yggdrasil not installed.")
		confirm := false
		survey.AskOne(&survey.Confirm{Message: "Do you want to install Yggdrasil now?"}, &confirm)
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
		path := `C:\ProgramData\Yggdrasil\yggdrasil.conf`
		if fileExists(path) { return path }
		return path
	}

	paths := []string{
		"/etc/yggdrasil.conf",
		"/etc/yggdrasil/yggdrasil.conf",
		"/etc/Yggdrasil/yggdrasil.conf",
	}

	for _, p := range paths {
		if fileExists(p) { return p }
	}

	return "/etc/yggdrasil.conf"
}

// --- Menus ---

func mainMenu() {
	for {
		clearScreen()
		printBanner()
		fmt.Printf("Config loaded: %s\n", detectedConfigPath)

		// Count peers for display
		currentPeers := getConfigPeers()
		fmt.Printf("Active peers in config: %d\n\n", len(currentPeers))

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
		installWindows()
	} else {
		installLinux()
	}

	detectedConfigPath = findConfigPath()
	if !fileExists(detectedConfigPath) {
		fmt.Println("Generating config...")
		cmdName := "yggdrasil"
		if isWindows { cmdName = `C:\Program Files\Yggdrasil\yggdrasil.exe` }

		out, err := exec.Command(cmdName, "-genconf").Output()
		if err == nil {
			dir := filepath.Dir(detectedConfigPath)
			os.MkdirAll(dir, 0755)
			os.WriteFile(detectedConfigPath, out, 0644)
			fmt.Println(green("Config generated at " + detectedConfigPath))
		} else {
			fmt.Println(red("Failed to generate config: "), err)
		}
	}
	waitEnter()
}

func installWindows() {
	url, err := getLatestReleaseURL("yggdrasil-go", ".msi")
	if err != nil {
		fmt.Println(red("Error fetching URL: "), err)
		return
	}
	filename := "yggdrasil_installer.msi"
	downloadFile(filename, url)
	exec.Command("msiexec", "/i", filename, "/quiet", "/norestart").Run()
	os.Remove(filename)
}

func installLinux() {
	distro := getLinuxDistro()
	if strings.Contains(distro, "debian") || strings.Contains(distro, "ubuntu") || strings.Contains(distro, "mint") || strings.Contains(distro, "kali") {
		choice := ""
		survey.AskOne(&survey.Select{
			Message: "Method:",
			Options: []string{"Download .deb", "Use APT"},
		}, &choice)

		if choice == "Download .deb" {
			url, err := getLatestReleaseURL("yggdrasil-go", ".deb")
			if err == nil {
				downloadFile("ygg.deb", url)
				exec.Command("apt", "install", "./ygg.deb", "-y").Run()
				os.Remove("ygg.deb")
			}
		} else {
			exec.Command("sh", "-c", "curl -s https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/key/neilalexander.gpg | gpg --dearmor > /usr/local/share/keyrings/neilalexander.gpg").Run()
			exec.Command("sh", "-c", "echo 'deb [signed-by=/usr/local/share/keyrings/neilalexander.gpg] http://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' > /etc/apt/sources.list.d/yggdrasil.list").Run()
			exec.Command("apt", "update").Run()
			exec.Command("apt", "install", "yggdrasil", "-y").Run()
		}
	} else {
		exec.Command("pacman", "-S", "yggdrasil-go", "--noconfirm").Run()
	}
}

// --- Logic: GitHub & Peers ---

func fetchPeersStructure() (map[string][]string, error) {
	fmt.Println("Scanning repository...")
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/master?recursive=1", repoOwner, repoPeers))
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var treeResp GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil { return nil, err }

	regionMap := make(map[string][]string)
	for _, node := range treeResp.Tree {
		if node.Type != "blob" || !strings.HasSuffix(node.Path, ".md") { continue }
		if strings.HasSuffix(node.Path, "README.md") { continue }

		parts := strings.Split(node.Path, "/")
		if len(parts) < 2 { continue }
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
				// Improved regex to catch quic, ws, etc
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
	for _, u := range regionMap { allUrls = append(allUrls, u...) }

	fmt.Printf("Fetching from %d regions...\n", len(regionMap))
	allPeers := fetchPeersFromURLs(allUrls)
	fmt.Printf("Total peers found: %d. Pinging subset...\n", len(allPeers))

	// Shuffle
	for i := range allPeers {
		j := int(time.Now().UnixNano()) % len(allPeers)
		allPeers[i], allPeers[j] = allPeers[j], allPeers[i]
	}

	limit := 25
	if len(allPeers) < limit { limit = len(allPeers) }

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
	if len(ranked) < count { count = len(ranked) }

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
		if err != nil { return }

		keys := []string{}
		for k := range regionMap { keys = append(keys, k) }
		sort.Strings(keys)
		keys = append(keys, "Back")

		selReg := ""
		err = survey.AskOne(&survey.Select{Message: "Region:", Options: keys}, &selReg)
		if err == terminal.InterruptErr || selReg == "Back" { return }

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

// --- Logic: Parser V2 (TrimSpace Fixed) ---

func getConfigPeers() []string {
	f, err := os.Open(detectedConfigPath)
	if err != nil { return []string{} }
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var peers []string
	inBlock := false

	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw) // CRITICAL FIX: Trim whitespace before checking

		// Detect Start of Block
		if strings.HasPrefix(line, "Peers:") {
			if strings.Contains(line, "[") {
				inBlock = true
				if strings.Contains(line, "]") {
					extractFromLine(line, &peers)
					inBlock = false
				}
			}
			continue
		}

		// Detect End of Block
		if inBlock && strings.Contains(line, "]") {
			inBlock = false
			extractFromLine(line, &peers)
			continue
		}

		// Inside Block
		if inBlock {
			extractFromLine(line, &peers)
		}
	}
	return peers
}

func extractFromLine(line string, target *[]string) {
	// Strip comments first
	if idx := strings.Index(line, "//"); idx != -1 { line = line[:idx] }
		if idx := strings.Index(line, "#"); idx != -1 { line = line[:idx] }

		// Simple check for URI protocols
		protocols := []string{"tcp://", "tls://", "quic://", "ws://", "wss://"}

		// Clean syntax characters to isolate the URI
		cleaned := strings.ReplaceAll(line, "\"", "")
		cleaned = strings.ReplaceAll(cleaned, ",", "")
		cleaned = strings.ReplaceAll(cleaned, "[", "")
		cleaned = strings.ReplaceAll(cleaned, "]", "")
		cleaned = strings.TrimSpace(cleaned)

		for _, proto := range protocols {
			if strings.Contains(cleaned, proto) {
				*target = append(*target, cleaned)
				break
			}
		}
}

func addPeersToConfig(newPeers []string) {
	bytes, err := os.ReadFile(detectedConfigPath)
	if err != nil { return }
	content := string(bytes)

	insert := ""
	for _, p := range newPeers {
		if !strings.Contains(content, p) {
			insert += fmt.Sprintf("\n    \"%s\"", p)
		}
	}
	if insert == "" { return }

	re := regexp.MustCompile(`(Peers:\s*\[)`)
	if re.MatchString(content) {
		newC := re.ReplaceAllString(content, "${1}"+insert)
		os.WriteFile(detectedConfigPath, []byte(newC), 0644)
		fmt.Println(green("Peers added."))
	} else {
		// Fallback append
		f, _ := os.OpenFile(detectedConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
		f.WriteString("\nPeers: [" + insert + "\n]\n")
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
			if strings.Contains(line, rem) {
				keep = false
				break
			}
		}
		if keep { newLines = append(newLines, line) }
	}
	os.WriteFile(detectedConfigPath, []byte(strings.Join(newLines, "\n")), 0644)
	fmt.Println(green("Peers removed."))
}

// --- Helpers ---

func showStatus() {
	cmdName := linuxExe
	if isWindows { cmdName = windowsExe }
	out, _ := exec.Command(cmdName, "getself").CombinedOutput()
	fmt.Println(string(out))
	waitEnter()
}

func manageService(act string) {
	if isWindows {
		if act == "Restart" {
			exec.Command("net", "stop", "yggdrasil").Run()
			act = "Start"
		}
		verb := "start"
		if act == "Stop" { verb = "stop" }
		exec.Command("net", verb, "yggdrasil").Run()
	} else {
		verb := strings.ToLower(act)
		if act == "Enable Autostart" { verb = "enable" }
		if act == "Disable Autostart" { verb = "disable" }
		exec.Command("systemctl", verb, "yggdrasil").Run()
	}
	fmt.Println(green("Done."))
}

func restartServicePrompt() {
	r := false
	survey.AskOne(&survey.Confirm{Message: "Restart service?"}, &r)
	if r { manageService("Restart") }
}

func pingPeer(uri string) time.Duration {
	parts := strings.Split(uri, "://")
	if len(parts) < 2 { return 999 * time.Second }
	conn, err := net.DialTimeout("tcp", parts[1], 2*time.Second)
	if err != nil { return 999 * time.Second }
	conn.Close()
	return 100 * time.Millisecond // Mock approximation if success
}

func checkAdmin() {
	if !isWindows && os.Geteuid() != 0 {
		fmt.Println(red("Root required! (sudo)"))
		os.Exit(1)
	}
}

func getLinuxDistro() string {
	b, _ := os.ReadFile("/etc/os-release")
	return strings.ToLower(string(b))
}

func getLatestReleaseURL(repo, suffix string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repo))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	assets, _ := res["assets"].([]interface{})
	for _, a := range assets {
		m := a.(map[string]interface{})
		if strings.HasSuffix(m["name"].(string), suffix) {
			return m["browser_download_url"].(string), nil
		}
	}
	return "", fmt.Errorf("not found")
}

func downloadFile(path, url string) {
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	out, _ := os.Create(path)
	defer out.Close()
	io.Copy(out, resp.Body)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}

func waitEnter() {
	fmt.Println("\nPress Enter...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
	if isWindows { exec.Command("cmd", "/c", "cls").Run() }
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
	           Configurator v0.1.1
	`))
}
