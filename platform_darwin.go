//go:build darwin
// +build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/AlecAivazis/survey/v2"
)

type DarwinPlatform struct{}

func init() {
	currentPlatform = &DarwinPlatform{}
}

func (p *DarwinPlatform) EnsureAdmin() {
	if os.Geteuid() != 0 {
		fmt.Println(red("Root required! Please run with sudo."))
		os.Exit(1)
	}
}

func (p *DarwinPlatform) FindConfigPath() string {
	paths := []string{
		"/etc/yggdrasil.conf",
		"/usr/local/etc/yggdrasil.conf",
		"/opt/homebrew/etc/yggdrasil.conf",
	}
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	return "/etc/yggdrasil.conf"
}

func (p *DarwinPlatform) Install() error {
	fmt.Println(cyan("=== macOS Yggdrasil Installation ==="))
	
	// Check if Homebrew is installed
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("Homebrew detected.")
		choice := ""
		survey.AskOne(&survey.Select{
			Message: "Installation Method:",
			Options: []string{"Homebrew (recommended)", "Download .pkg", "Cancel"},
		}, &choice)

		if choice == "Cancel" {
			return fmt.Errorf("installation cancelled")
		}

		if choice == "Homebrew (recommended)" {
			fmt.Println("Installing via Homebrew...")
			cmd := exec.Command("brew", "install", "yggdrasil-go")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("brew install failed: %v", err)
			}
			fmt.Println(green("Installation complete."))
			fmt.Println(yellow("Note: To start service, use: brew services start yggdrasil-go"))
			return nil
		}
	}

	// Download .pkg installer
	fmt.Println("Downloading .pkg installer...")
	arch := runtime.GOARCH
	var searchStr string
	if arch == "arm64" {
		searchStr = "arm64"
	} else {
		searchStr = "amd64"
	}

	url, err := getLatestReleaseURL("yggdrasil-go", ".pkg", searchStr)
	if err != nil {
		return err
	}

	filename := "yggdrasil_installer.pkg"
	fmt.Printf("Downloading %s (Arch: %s)...\n", url, arch)
	if err := downloadFile(filename, url); err != nil {
		return err
	}

	fmt.Println("Installing package...")
	cmd := exec.Command("installer", "-pkg", filename, "-target", "/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	os.Remove(filename)
	fmt.Println(green("Installation complete."))
	return nil
}

func (p *DarwinPlatform) ManageService(act string) error {
	// macOS uses launchctl for service management
	serviceName := "com.github.yggdrasil-network.yggdrasil"
	plistPath := "/Library/LaunchDaemons/" + serviceName + ".plist"

	switch act {
	case "Start", "start":
		return exec.Command("launchctl", "load", plistPath).Run()
	case "Stop", "stop":
		return exec.Command("launchctl", "unload", plistPath).Run()
	case "Restart", "restart":
		exec.Command("launchctl", "unload", plistPath).Run()
		return exec.Command("launchctl", "load", plistPath).Run()
	case "Enable Autostart":
		// On macOS, loaded LaunchDaemons auto-start by default
		fmt.Println(yellow("Service will auto-start after load."))
		return exec.Command("launchctl", "load", plistPath).Run()
	case "Disable Autostart":
		return exec.Command("launchctl", "unload", plistPath).Run()
	default:
		return fmt.Errorf("unknown action: %s", act)
	}
}

func (p *DarwinPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}
