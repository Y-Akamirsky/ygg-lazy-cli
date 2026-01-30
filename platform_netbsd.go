//go:build netbsd
// +build netbsd

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
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
	fmt.Println(cyan("=== NetBSD Yggdrasil Installation ==="))
	
	choice := ""
	survey.AskOne(&survey.Select{
		Message: "Installation Method:",
		Options: []string{"pkgin (binary package)", "pkgsrc (compile from source)", "Cancel"},
	}, &choice)

	if choice == "Cancel" {
		return fmt.Errorf("installation cancelled")
	}

	if choice == "pkgin (binary package)" {
		fmt.Println("Installing via pkgin...")
		cmd := exec.Command("pkgin", "-y", "install", "yggdrasil")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pkgin install failed: %v", err)
		}
		fmt.Println(green("Installation complete."))
		return nil
	}

	// pkgsrc installation
	fmt.Println("Installing via pkgsrc...")
	pkgsrcPath := "/usr/pkgsrc/net/yggdrasil"
	
	if !fileExists(pkgsrcPath) {
		fmt.Println(yellow("pkgsrc tree not found. Please install it first."))
		return fmt.Errorf("pkgsrc tree not available")
	}

	cmd := exec.Command("make", "-C", pkgsrcPath, "install", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pkgsrc install failed: %v", err)
	}

	fmt.Println(green("Installation complete."))
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
		fmt.Println(yellow("Please add 'yggdrasil=YES' to /etc/rc.conf manually"))
		return nil
	case "Disable Autostart":
		fmt.Println(yellow("Please remove 'yggdrasil=YES' from /etc/rc.conf manually"))
		return nil
	default:
		return fmt.Errorf("unknown action: %s", act)
	}
}

func (p *NetBSDPlatform) GetServiceCommands() []string {
	return []string{"Start", "Stop", "Restart", "Enable Autostart", "Disable Autostart"}
}
