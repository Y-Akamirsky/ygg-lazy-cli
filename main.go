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
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

// --- Constants & Vars ---

const (
	repoOwner    = "yggdrasil-network"
	repoName     = "yggdrasil-go"
	peersURL     = "https://raw.githubusercontent.com/yggdrasil-network/public-peers/master/README.md"
	windowsConf  = `C:\ProgramData\Yggdrasil\yggdrasil.conf`
	linuxConf    = "/etc/yggdrasil.conf"
	windowsExe   = `C:\Program Files\Yggdrasil\yggdrasilctl.exe`
	linuxExe     = "yggdrasilctl"
)

var (
	isWindows = runtime.GOOS == "windows"
	green     = color.New(color.FgGreen).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
	cyan      = color.New(color.FgCyan).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()
)

// --- Main ---

func main() {
	installFlag := flag.Bool("ygginstall", false, "Install Yggdrasil automatically")
	flag.Parse()

	checkAdmin()

	if *installFlag {
		installYggdrasil()
		return
	}

	// Check if installed before showing menu
	if !checkInstalled() {
		color.Yellow("Yggdrasil не найден! Запускаю установку...")
		installYggdrasil()
	}

	mainMenu()
}

// --- Menus ---

func mainMenu() {
	for {
		clearScreen()
		printBanner()

		mode := ""
		prompt := &survey.Select{
			Message: "Главное меню:",
			Options: []string{
				"Авто-подбор пиров (по пингу)",
				"Ручной выбор пиров (из списка)",
				"Удалить пиры",
				"Добавить кастомный пир",
				"Статус узла (getself)",
				"Управление службой (Start/Stop/Restart)",
				"Выход",
			},
		}
		survey.AskOne(prompt, &mode)

		switch mode {
			case "Авто-подбор пиров (по пингу)":
				autoAddPeers()
			case "Ручной выбор пиров (из списка)":
				manualAddPeers()
			case "Удалить пиры":
				removePeersMenu()
			case "Добавить кастомный пир":
				addCustomPeer()
			case "Статус узла (getself)":
				showStatus()
			case "Управление службой (Start/Stop/Restart)":
				serviceMenu()
			case "Выход":
				fmt.Println("Пока!")
				os.Exit(0)
		}

		fmt.Println("\nНажмите Enter, чтобы вернуться в меню...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}

func serviceMenu() {
	action := ""
	prompt := &survey.Select{
		Message: "Управление службой:",
		Options: []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart", "Назад"},
	}
	survey.AskOne(prompt, &action)

	if action == "Назад" {
		return
	}

	manageService(action)
}

// --- Logic: Installation ---

func installYggdrasil() {
	fmt.Println(cyan("=== Установка Yggdrasil ==="))

	if isWindows {
		installWindows()
	} else {
		installLinux()
	}

	// Post install: Generate config if needed
	if !fileExists(getConfigPath()) {
		fmt.Println("Генерация конфига...")
		// В Windows установщик обычно сам создает конфиг, но проверим
		// В Linux post-install скрипт пакета делает это
		// Если нет - пробуем генерировать
		cmdName := "yggdrasil"
		if isWindows { cmdName = `C:\Program Files\Yggdrasil\yggdrasil.exe` }

		out, err := exec.Command(cmdName, "-genconf").Output()
		if err == nil {
			os.WriteFile(getConfigPath(), out, 0644)
			fmt.Println(green("Конфиг создан."))
		}
	}

	fmt.Println(green("Установка завершена!"))
}

func installWindows() {
	fmt.Println("Скачивание последнего MSI...")
	url, err := getLatestReleaseURL(".msi")
	if err != nil {
		fmt.Println(red("Ошибка получения ссылки: "), err)
		return
	}

	filename := "yggdrasil_installer.msi"
	if err := downloadFile(filename, url); err != nil {
		fmt.Println(red("Ошибка загрузки: "), err)
		return
	}

	fmt.Println("Запуск установщика (тихий режим)...")
	cmd := exec.Command("msiexec", "/i", filename, "/quiet", "/norestart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(red("Ошибка установки: "), err)
	}
	os.Remove(filename)
}

func installLinux() {
	distro := getLinuxDistro()
	fmt.Printf("Обнаружен дистрибутив: %s\n", distro)

	// Debian/Ubuntu/Mint
	if strings.Contains(distro, "debian") || strings.Contains(distro, "ubuntu") || strings.Contains(distro, "mint") || strings.Contains(distro, "kali") {
		choice := ""
		prompt := &survey.Select{
			Message: "Как установить Yggdrasil?",
			Options: []string{"Скачать .deb с GitHub (быстро)", "Использовать APT (рекомендуется)"},
		}
		survey.AskOne(prompt, &choice)

		if choice == "Скачать .deb с GitHub (быстро)" {
			url, err := getLatestReleaseURL(".deb")
			if err != nil {
				fmt.Println(red("Ошибка API GitHub:"), err)
				return
			}
			file := "ygg.deb"
			downloadFile(file, url)
			exec.Command("apt", "install", "./"+file, "-y").Run()
			os.Remove(file)
		} else {
			// APT Logic (simplified instructions from off site)
			cmds := [][]string{
				{"mkdir", "-p", "/usr/local/share/keyrings"},
				{"sh", "-c", "curl -s https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/key/neilalexander.gpg | gpg --dearmor > /usr/local/share/keyrings/neilalexander.gpg"},
				{"sh", "-c", "echo 'deb [signed-by=/usr/local/share/keyrings/neilalexander.gpg] http://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' > /etc/apt/sources.list.d/yggdrasil.list"},
				{"apt", "update"},
				{"apt", "install", "yggdrasil", "-y"},
			}
			for _, c := range cmds {
				fmt.Printf("Выполняю: %v\n", c)
				exec.Command(c[0], c[1:]...).Run()
			}
		}
	} else if strings.Contains(distro, "arch") || strings.Contains(distro, "manjaro") {
		fmt.Println("Использую pacman...")
		exec.Command("pacman", "-S", "yggdrasil-go", "--noconfirm").Run()
	} else {
		fmt.Println(yellow("Ваш дистрибутив не распознан автоматически. Попробуйте установить через системный пакетный менеджер вручную."))
	}
}

// --- Logic: Peers ---

type Peer struct {
	URI     string
	Region  string
	Latency time.Duration
}

func fetchPeersFromGithub() (map[string][]string, error) {
	resp, err := http.Get(peersURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	content := string(bodyBytes)

	// Simple parser: Look for Headers (## Region) and lists
	regions := make(map[string][]string)
	lines := strings.Split(content, "\n")

	currentRegion := "Other"
	var peerRegex = regexp.MustCompile(`(tcp|tls)://[a-zA-Z0-9\.\-\:\[\]]+`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "##") || strings.HasPrefix(line, "# ") {
			// Clean header to get region name
			clean := strings.TrimLeft(line, "# ")
			if len(clean) > 0 {
				currentRegion = clean
			}
			continue
		}

		matches := peerRegex.FindAllString(line, -1)
		for _, uri := range matches {
			regions[currentRegion] = append(regions[currentRegion], uri)
		}
	}
	return regions, nil
}

func autoAddPeers() {
	fmt.Println("Загрузка списка пиров...")
	regions, err := fetchPeersFromGithub()
	if err != nil {
		fmt.Println(red("Ошибка загрузки пиров"), err)
		return
	}

	var allPeers []string
	for _, p := range regions {
		allPeers = append(allPeers, p...)
	}

	fmt.Printf("Найдено %d пиров. Начинаю пинг (это займет время)...\n", len(allPeers))

	// Ограничим количество проверяемых пиров для скорости (например, берем 20 случайных или первых)
	if len(allPeers) > 20 {
		allPeers = allPeers[:20]
	}

	var rankedPeers []Peer
	for _, uri := range allPeers {
		fmt.Printf(".")
		latency := pingPeer(uri)
		if latency < 999*time.Second {
			rankedPeers = append(rankedPeers, Peer{URI: uri, Latency: latency})
		}
	}
	fmt.Println()

	sort.Slice(rankedPeers, func(i, j int) bool {
		return rankedPeers[i].Latency < rankedPeers[j].Latency
	})

	toAdd := []string{}
	limit := 3
	if len(rankedPeers) < 3 { limit = len(rankedPeers) }

	fmt.Println(green("Топ пиров:"))
	for i := 0; i < limit; i++ {
		p := rankedPeers[i]
		fmt.Printf("%d. %s (%s)\n", i+1, p.URI, p.Latency)
		toAdd = append(toAdd, p.URI)
	}

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Добавить эти пиры в конфиг?"}, &confirm)
	if confirm {
		addPeersToConfig(toAdd)
		restartServicePrompt()
	}
}

func manualAddPeers() {
	regions, err := fetchPeersFromGithub()
	if err != nil {
		fmt.Println(red("Ошибка:"), err)
		return
	}

	regionKeys := []string{}
	for k := range regions {
		regionKeys = append(regionKeys, k)
	}
	sort.Strings(regionKeys)

	selectedRegion := ""
	survey.AskOne(&survey.Select{
		Message: "Выберите регион:",
		Options: regionKeys,
	}, &selectedRegion)

	peersList := regions[selectedRegion]
	selectedPeers := []string{}

	survey.AskOne(&survey.MultiSelect{
		Message: "Выберите пиры (Space - выбрать, Enter - подтвердить):",
		      Options: peersList,
	}, &selectedPeers)

	if len(selectedPeers) > 0 {
		addPeersToConfig(selectedPeers)
		restartServicePrompt()
	}
}

func removePeersMenu() {
	currentPeers := getConfigPeers()
	if len(currentPeers) == 0 {
		fmt.Println(yellow("В конфиге нет пиров."))
		return
	}

	toRemove := []string{}
	survey.AskOne(&survey.MultiSelect{
		Message: "Отметьте пиры для удаления:",
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
		Message: "Введите адреса пиров через пробел (tcp://... tls://...):",
	}, &peersStr)

	list := strings.Fields(peersStr)
	if len(list) > 0 {
		addPeersToConfig(list)
		restartServicePrompt()
	}
}

// --- Logic: Config Editor ---
// Yggdrasil использует HJSON. Чтобы не тянуть сложные зависимости и не ломать комменты,
// используем простой поиск блока Peers: []

func getConfigPath() string {
	if isWindows {
		return windowsConf
	}
	return linuxConf
}

func getConfigPeers() []string {
	path := getConfigPath()
	content, err := os.ReadFile(path)
	if err != nil {
		return []string{}
	}
	s := string(content)

	// Regex hacky parsing for Peers: [...]
	// Находим контент внутри Peers: [ ... ]
	re := regexp.MustCompile(`(?s)Peers:\s*\[(.*?)\]`)
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return []string{}
	}

	rawPeers := match[1]
	// Clean up quotes, commas, comments
	lines := strings.Split(rawPeers, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove comments // or #
		if idx := strings.Index(line, "//"); idx != -1 { line = line[:idx] }
			if idx := strings.Index(line, "#"); idx != -1 { line = line[:idx] }

			// Extract URI inside quotes or just string
			line = strings.Trim(line, `", `)
			if strings.Contains(line, "tcp://") || strings.Contains(line, "tls://") || strings.Contains(line, "quic://") {
				result = append(result, line)
			}
	}
	return result
}

func addPeersToConfig(newPeers []string) {
	path := getConfigPath()
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(red("Не могу прочитать конфиг!"))
		return
	}
	content := string(contentBytes)

	// Формируем блок для вставки
	insertion := ""
	for _, p := range newPeers {
		// Простая проверка на дубликаты
		if !strings.Contains(content, p) {
			insertion += fmt.Sprintf("\n    \"%s\"", p)
		}
	}

	if insertion == "" {
		fmt.Println("Пиры уже есть в конфиге.")
		return
	}

	// Вставляем перед закрывающей скобкой Peers
	re := regexp.MustCompile(`(Peers:\s*\[[\s\S]*?)(\])`)
	if re.MatchString(content) {
		newContent := re.ReplaceAllString(content, "${1}"+insertion+"\n  ${2}")
		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			fmt.Println(red("Ошибка записи конфига:"), err)
		} else {
			fmt.Println(green("Пиры записаны в конфиг."))
		}
	} else {
		fmt.Println(red("Не нашел блок Peers: [] в конфиге. Структура файла нарушена?"))
	}
}

func removePeersFromConfig(toRemove []string) {
	path := getConfigPath()
	contentBytes, err := os.ReadFile(path)
	if err != nil { return }
	content := string(contentBytes)

	for _, p := range toRemove {
		// Очень грубое удаление строки, содержащей пир.
		// Надежнее было бы распарсить, но для MVP сойдет.
		lines := strings.Split(content, "\n")
		var newLines []string
		for _, line := range lines {
			if strings.Contains(line, p) && (strings.Contains(line, "tcp://") || strings.Contains(line, "tls://")) {
				continue // Skip this line
			}
			newLines = append(newLines, line)
		}
		content = strings.Join(newLines, "\n")
	}

	os.WriteFile(path, []byte(content), 0644)
	fmt.Println(green("Пиры удалены."))
}

// --- Utils ---

func showStatus() {
	cmdName := linuxExe
	if isWindows { cmdName = windowsExe }

	cmd := exec.Command(cmdName, "getself")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(red("Ошибка получения статуса (запущен ли сервис?):"), err)
	} else {
		fmt.Println(string(out))
	}
}

func manageService(action string) {
	var cmd *exec.Cmd
	if isWindows {
		// Windows service control
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
		// Systemd
		verb := strings.ToLower(action)
		if action == "Enable Autostart" { verb = "enable" }
		if action == "Disable Autostart" { verb = "disable" }
		cmd = exec.Command("systemctl", verb, "yggdrasil")
	}

	if cmd != nil {
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(red("Ошибка: "), err, string(out))
		} else {
			fmt.Println(green("Успешно."))
		}
	}
}

func restartServicePrompt() {
	restart := false
	survey.AskOne(&survey.Confirm{Message: "Перезапустить Yggdrasil для применения настроек?"}, &restart)
	if restart {
		manageService("Restart")
	}
}

func pingPeer(uri string) time.Duration {
	// Extract host:port from tcp://host:port
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
		fmt.Println(red("ВНИМАНИЕ: Программа требует прав Администратора/Root!"))
		fmt.Println("Пожалуйста, перезапустите программу с повышенными правами.")
		if !isWindows { fmt.Println("Используйте: sudo ./ygg-lazy-cli") }
		fmt.Print("Нажмите Enter для выхода...")
		fmt.Scanln()
		os.Exit(1)
	}
}

func checkInstalled() bool {
	p := getConfigPath()
	return fileExists(p)
}

func getLinuxDistro() string {
	out, _ := os.ReadFile("/etc/os-release")
	return strings.ToLower(string(out))
}

func getLatestReleaseURL(suffix string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName))
	if err != nil { return "", err }
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assets := result["assets"].([]interface{})
	for _, a := range assets {
		asset := a.(map[string]interface{})
		name := asset["name"].(string)
		if strings.HasSuffix(name, suffix) {
			return asset["browser_download_url"].(string), nil
		}
	}
	return "", fmt.Errorf("файл %s не найден в релизе", suffix)
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

func clearScreen() {
	if isWindows {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
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
	            Configurator v1.0
	`))
}
