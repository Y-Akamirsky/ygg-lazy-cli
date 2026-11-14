//go:build openbsd
// +build openbsd

package main

import (
	"fmt"
	"os"
	"os/exec"
)

type OpenBSDPlatform struct{}

func init() {
	currentPlatform = &OpenBSDPlatform{}
}

func (p *OpenBSDPlatform) EnsureAdmin() {
	if os.Geteuid() != 0 {
		fmt.Println(red("Root required! Please run with doas or as root."))
		os.Exit(1)
	}
}

func (p *OpenBSDPlatform) FindConfigPath() string {
	paths := []string{
		"/etc/yggdrasil.conf",
		"/usr/local/etc/yggdrasil.conf",
	}
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	return "/etc/yggdrasil.conf"
}

func (p *OpenBSDPlatform) Install() error {
	fmt.Println(cyan(T("=== OpenBSD Yggdrasil Installation ===")))
	fmt.Println(T("Installing via pkg_add..."))

	cmd := exec.Command("pkg_add", "yggdrasil")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", T("pkg_add failed"), err)
	}

	fmt.Println(green(T("Installation complete.")))
	fmt.Println(yellow(T("Enable service: rcctl enable yggdrasil")))
	fmt.Println(yellow(T("Start service: rcctl start yggdrasil")))
	return nil
}

func (p *OpenBSDPlatform) ManageService(act string) error {
	// OpenBSD uses rcctl for service management
	switch act {
	case "Start", "start":
		return exec.Command("rcctl", "start", "yggdrasil").Run()
	case "Stop", "stop":
		return exec.Command("rcctl", "stop", "yggdrasil").Run()
	case "Restart", "restart":
		return exec.Command("rcctl", "restart", "yggdrasil").Run()
	case "Enable Autostart":
		return exec.Command("rcctl", "enable", "yggdrasil").Run()
	case "Disable Autostart":
		return exec.Command("rcctl", "disable", "yggdrasil").Run()
	default:
		return fmt.Errorf("%s: %s", T("unknown action"), act)
	}
}

func (p *OpenBSDPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}
