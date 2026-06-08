//go:build windows

package urlproto

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func getDefaultLauncher(proto string) (string, error) {
	// Look up the deep-link command path in the Windows Registry
	key, err := registry.OpenKey(registry.CLASSES_ROOT, proto+`\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return "", nil // Safe fallback: protocol isn't registered
		}
		return "", err
	}
	defer key.Close()

	val, _, err := key.GetStringValue("")
	if err != nil {
		return "", err
	}

	// Isolate the executable path from arguments/parameters
	var execPath string
	if strings.HasPrefix(val, `"`) {
		parts := strings.Split(val, `"`)
		if len(parts) > 1 {
			execPath = parts[1]
		}
	} else {
		execPath = strings.Fields(val)[0]
	}

	return filepath.Clean(execPath), nil
}
