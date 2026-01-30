//go:build freebsd
// +build freebsd

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
)

type FreeBSDPlatform struct{}

func init() {
	currentPlatform = &FreeBSDPlatform{}
}

func (p *FreeBSDPlatform) EnsureAdmin() {
	if os.Geteuid() != 0 {
		fmt.Println(red("Root required! Please run with sudo or as root."))
		os.Exit(1)
	}
}

func (p *FreeBSDPlatform) FindConfigPath() string {
	paths := []string{
		"/usr/local/etc/yggdrasil.conf",
		"/etc/yggdrasil.conf",
	}
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	return "/usr/local/etc/yggdrasil.conf"
}

func (p *FreeBSDPlatform) Install() error {
	fmt.Println(cyan("=== FreeBSD Yggdrasil Installation ==="))
	
	choice := ""
	survey.AskOne(&survey.Select{
		Message: "Installation Method:",
		Options: []string{"pkg (binary package)", "ports (compile from source)", "Cancel"},
	}, &choice)

	if choice == "Cancel" {
		return fmt.Errorf("installation cancelled")
	}

	if choice == "pkg (binary package)" {
		fmt.Println("Installing via pkg...")
		cmd := exec.Command("pkg", "install", "-y", "yggdrasil")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pkg install failed: %v", err)
		}
		fmt.Println(green("Installation complete."))
		fmt.Println(yellow("Enable service: sysrc yggdrasil_enable=YES"))
		fmt.Println(yellow("Start service: service yggdrasil start"))
		return nil
	}

	// ports installation
	fmt.Println("Installing via ports...")
	portsPath := "/usr/ports/net/yggdrasil"
	
	// Check if ports tree exists
	if !fileExists(portsPath) {
		fmt.Println(yellow("Ports tree not found. Please install it first:"))
		fmt.Println("  portsnap fetch extract")
		return fmt.Errorf("ports tree not available")
	}

	cmd := exec.Command("make", "-C", portsPath, "install", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ports install failed: %v", err)
	}

	fmt.Println(green("Installation complete."))
	return nil
}

func (p *FreeBSDPlatform) ManageService(act string) error {
	// FreeBSD uses rc.d service management
	switch act {
	case "Start", "start":
		return exec.Command("service", "yggdrasil", "start").Run()
	case "Stop", "stop":
		return exec.Command("service", "yggdrasil", "stop").Run()
	case "Restart", "restart":
		return exec.Command("service", "yggdrasil", "restart").Run()
	case "Enable Autostart":
		// Add yggdrasil_enable="YES" to /etc/rc.conf
		return exec.Command("sysrc", "yggdrasil_enable=YES").Run()
	case "Disable Autostart":
		return exec.Command("sysrc", "yggdrasil_enable=NO").Run()
	default:
		return fmt.Errorf("unknown action: %s", act)
	}
}

func (p *FreeBSDPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}
