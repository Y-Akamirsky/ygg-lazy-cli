//go:build netbsd
// +build netbsd

package main

import (
	"fmt"
	"os"
	"os/exec"
)

type NetBSDPlatform struct{}

func init() {
	currentPlatform = &NetBSDPlatform{}
}

func (p *NetBSDPlatform) EnsureAdmin() {
	if os.Geteuid() != 0 {
		fmt.Println(red("Root required! Please run with su or as root."))
		os.Exit(1)
	}
}

func (p *NetBSDPlatform) FindConfigPath() string {
	paths := []string{
		"/etc/yggdrasil.conf",
		"/usr/pkg/etc/yggdrasil.conf",
	}
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	return "/etc/yggdrasil.conf"
}

func (p *NetBSDPlatform) Install() error {
	fmt.Println(cyan(T("=== NetBSD Yggdrasil Installation ===")))

	installMethod, _ := askChoice("Installation Method:", []choice{
		{key: "pkgin (binary package)", label: T("pkgin (binary package)")},
		{key: "pkgsrc (compile from source)", label: T("pkgsrc (compile from source)")},
		{key: "Cancel", label: T("Cancel")},
	}, "", 0)

	if installMethod == "Cancel" || installMethod == "" {
		return fmt.Errorf("%s", T("installation cancelled"))
	}

	if installMethod == "pkgin (binary package)" {
		fmt.Println(T("Installing via pkgin..."))
		cmd := exec.Command("pkgin", "-y", "install", "yggdrasil")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %v", T("pkgin install failed"), err)
		}
		fmt.Println(green(T("Installation complete.")))
		return nil
	}

	// pkgsrc installation
	fmt.Println(T("Installing via pkgsrc..."))
	pkgsrcPath := "/usr/pkgsrc/net/yggdrasil"

	if !fileExists(pkgsrcPath) {
		fmt.Println(yellow(T("pkgsrc tree not found. Please install it first.")))
		return fmt.Errorf("%s", T("pkgsrc tree not available"))
	}

	cmd := exec.Command("make", "-C", pkgsrcPath, "install", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", T("pkgsrc install failed"), err)
	}

	fmt.Println(green(T("Installation complete.")))
	return nil
}

func (p *NetBSDPlatform) ManageService(act string) error {
	// NetBSD uses rc.d service management
	switch act {
	case "Start", "start":
		return exec.Command("/etc/rc.d/yggdrasil", "start").Run()
	case "Stop", "stop":
		return exec.Command("/etc/rc.d/yggdrasil", "stop").Run()
	case "Restart", "restart":
		return exec.Command("/etc/rc.d/yggdrasil", "restart").Run()
	case "Enable Autostart":
		// Add yggdrasil=YES to /etc/rc.conf
		fmt.Println(yellow(T("Please add 'yggdrasil=YES' to /etc/rc.conf manually")))
		return nil
	case "Disable Autostart":
		fmt.Println(yellow(T("Please remove 'yggdrasil=YES' from /etc/rc.conf manually")))
		return nil
	default:
		return fmt.Errorf("%s: %s", T("unknown action"), act)
	}
}

func (p *NetBSDPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}
