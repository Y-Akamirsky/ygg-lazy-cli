//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type WindowsPlatform struct{}

func init() {
	currentPlatform = &WindowsPlatform{}
}

func (p *WindowsPlatform) EnsureAdmin() {
	if !amAdminWindows() {
		fmt.Println(yellow("Administrator privileges required. Relaunching..."))
		runMeElevated()
	}
}

func (p *WindowsPlatform) FindConfigPath() string {
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "Yggdrasil", "yggdrasil.conf")
}

func (p *WindowsPlatform) Install() error {
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

func (p *WindowsPlatform) ManageService(act string) error {
	cmd := ""
	switch act {
	case "Start":
		cmd = "Start-Service yggdrasil"
	case "Stop":
		cmd = "Stop-Service yggdrasil"
	case "Restart":
		cmd = "Restart-Service yggdrasil -Force"
	case "Enable Autostart":
		cmd = "Set-Service yggdrasil -StartupType Automatic"
	case "Disable Autostart":
		cmd = "Set-Service yggdrasil -StartupType Manual"
	default:
		return fmt.Errorf("unknown action: %s", act)
	}
	fmt.Printf("Executing: %s\n", cmd)
	return exec.Command("powershell", "-Command", cmd).Run()
}

func (p *WindowsPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}

// Windows-specific helper functions
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
