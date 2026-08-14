//go:build windows

package platform

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName    = "RegionLockpaw"
)

func isStartAtLoginEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	value, _, err := key.GetStringValue(runValueName)
	if err != nil {
		return false
	}
	executable, err := os.Executable()
	if err != nil {
		return true
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(executable))
}

func setStartAtLogin(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runRegistryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open startup registry key: %w", err)
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("remove startup entry: %w", err)
		}
		return nil
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	command := `"` + executable + `" --background`
	if err := key.SetStringValue(runValueName, command); err != nil {
		return fmt.Errorf("write startup entry: %w", err)
	}
	return nil
}
