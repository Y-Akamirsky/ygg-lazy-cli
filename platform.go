package main

// Platform defines OS-specific operations
type Platform interface {
	// EnsureAdmin checks and requests admin/root privileges
	EnsureAdmin()
	
	// FindConfigPath returns the default Yggdrasil config path
	FindConfigPath() string
	
	// Install installs Yggdrasil on the system
	Install() error
	
	// ManageService starts/stops/restarts/enables/disables the service
	ManageService(action string) error
	
	// GetServiceCommands returns platform-specific service commands
	GetServiceCommands() []string
}

// currentPlatform holds the platform-specific implementation
var currentPlatform Platform
