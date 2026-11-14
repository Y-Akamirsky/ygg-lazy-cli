//go:build freebsd
// +build freebsd

package main

import (
	"fmt"
	"os"
	"os/exec"
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
	fmt.Println(cyan(T("=== FreeBSD Yggdrasil Installation ===")))

	installMethod, _ := askChoice("Installation Method:", []choice{
		{key: "pkg (binary package)", label: T("pkg (binary package)")},
		{key: "ports (compile from source)", label: T("ports (compile from source)")},
		{key: "Cancel", label: T("Cancel")},
	}, "", 0)

	if installMethod == "Cancel" || installMethod == "" {
		return fmt.Errorf("%s", T("installation cancelled"))
	}

	if installMethod == "pkg (binary package)" {
		fmt.Println(T("Installing via pkg..."))
		cmd := exec.Command("pkg", "install", "-y", "yggdrasil")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %v", T("pkg install failed"), err)
		}
		fmt.Println(green(T("Installation complete.")))
		fmt.Println(yellow(T("Enable service: sysrc yggdrasil_enable=YES")))
		fmt.Println(yellow(T("Start service: service yggdrasil start")))
		return nil
	}

	// ports installation
	fmt.Println(T("Installing via ports..."))
	portsPath := "/usr/ports/net/yggdrasil"

	// Check if ports tree exists
	if !fileExists(portsPath) {
		fmt.Println(yellow(T("Ports tree not found. Please install it first:")))
		fmt.Println("  portsnap fetch extract")
		return fmt.Errorf("%s", T("ports tree not available"))
	}

	cmd := exec.Command("make", "-C", portsPath, "install", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", T("ports install failed"), err)
	}

	fmt.Println(green(T("Installation complete.")))
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
		return fmt.Errorf("%s: %s", T("unknown action"), act)
	}
}

func (p *FreeBSDPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}
