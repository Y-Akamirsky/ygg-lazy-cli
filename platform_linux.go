//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

type LinuxPlatform struct{}

func init() {
	currentPlatform = &LinuxPlatform{}
}

func (p *LinuxPlatform) EnsureAdmin() {
	if os.Geteuid() != 0 {
		fmt.Println(red("Root required! Please run with sudo."))
		os.Exit(1)
	}
}

func (p *LinuxPlatform) FindConfigPath() string {
	paths := []string{
		"/etc/yggdrasil.conf",
		"/etc/yggdrasil/yggdrasil.conf",
	}
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	return "/etc/yggdrasil.conf"
}

func (p *LinuxPlatform) Install() error {
	distroID, distroLike := getLinuxDistroInfo()
	fmt.Printf("Detected Distro: %s (Like: %s)\n", distroID, distroLike)

	// Detection Logic
	isDebian := strings.Contains(distroID, "debian") || strings.Contains(distroID, "ubuntu") ||
		strings.Contains(distroID, "mint") || strings.Contains(distroID, "kali") ||
		strings.Contains(distroID, "pop") || strings.Contains(distroLike, "debian") ||
		strings.Contains(distroLike, "ubuntu")

	isArch := strings.Contains(distroID, "arch") || strings.Contains(distroID, "manjaro") ||
		strings.Contains(distroID, "cachyos") || strings.Contains(distroID, "endeavour") ||
		strings.Contains(distroLike, "arch")

	isFedora := strings.Contains(distroID, "fedora") || strings.Contains(distroID, "rhel") ||
		strings.Contains(distroID, "centos") || strings.Contains(distroID, "almalinux") ||
		strings.Contains(distroID, "rocky") || strings.Contains(distroLike, "fedora")

	isVoid := strings.Contains(distroID, "void")
	isAlpine := strings.Contains(distroID, "alpine")

	// Installation Logic
	if isDebian {
		choice := ""
		survey.AskOne(&survey.Select{
			Message: "Installation Method:",
			Options: []string{"Download .deb", "Use APT Repository"},
		}, &choice)

		if choice == "Download .deb" {
			arch := runtime.GOARCH
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
			cmds := [][]string{
				{"sh", "-c", "curl -s https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/key/neilalexander.gpg | gpg --dearmor > /usr/local/share/keyrings/neilalexander.gpg"},
				{"sh", "-c", "echo 'deb [signed-by=/usr/local/share/keyrings/neilalexander.gpg] http://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' > /etc/apt/sources.list.d/yggdrasil.list"},
				{"apt", "update"},
				{"apt", "install", "yggdrasil", "-y"},
			}
			return runCommands(cmds)
		}

	} else if isFedora {
		fmt.Println("Detected Fedora-based system. Using DNF and COPR...")
		cmds := [][]string{
			{"dnf", "install", "dnf-plugins-core", "-y"},
			{"dnf", "copr", "enable", "neilalexander/yggdrasil", "-y"},
			{"dnf", "install", "yggdrasil", "-y"},
		}
		return runCommands(cmds)

	} else if isVoid {
		fmt.Println("Detected Void Linux. Using xbps-install...")
		cmd := exec.Command("xbps-install", "-S", "yggdrasil", "-y")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("xbps-install failed: %v", err)
		}
		fmt.Println("Installed successfully via XBPS.")
		return nil

	} else if isAlpine {
		fmt.Println("Detected Alpine Linux. Using apk...")
		cmd := exec.Command("apk", "add", "yggdrasil")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apk failed: %v", err)
		}
		fmt.Println("Installed successfully via APK.")
		return nil

	} else if isArch {
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

func (p *LinuxPlatform) ManageService(act string) error {
	verb := strings.ToLower(act)
	if act == "Enable Autostart" {
		verb = "enable"
	}
	if act == "Disable Autostart" {
		verb = "disable"
	}
	return exec.Command("systemctl", verb, "yggdrasil").Run()
}

func (p *LinuxPlatform) GetServiceCommands() []string {
	return []string{"start", "stop", "restart", "Enable Autostart", "Disable Autostart"}
}
