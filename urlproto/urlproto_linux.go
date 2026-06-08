package urlproto

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getDefaultLauncher(proto string) (string, error) {
	// Query XDG for the registered .desktop handler file name
	cmd := exec.Command("xdg-settings", "get", "default-url-scheme-handler", proto)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	desktopFile := strings.TrimSpace(string(out))
	if desktopFile == "" {
		return "", nil // No handler registered
	}

	// Resolve common paths for desktop entries
	home, _ := os.UserHomeDir()
	searchPaths := []string{
		filepath.Join(home, ".local/share/applications", desktopFile),
		filepath.Join("/usr/share/applications", desktopFile),
		filepath.Join("/usr/local/share/applications", desktopFile),
	}

	var execLine string
	for _, path := range searchPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "Exec=") {
					execLine = strings.TrimPrefix(line, "Exec=")
					break
				}
			}
			break
		}
	}

	if execLine == "" {
		return "", nil
	}

	// Clean up trailing placeholders like %u or outer quotes
	rawExec := strings.Fields(execLine)[0]
	rawExec = strings.Trim(rawExec, `"'`)

	// Evaluate symlinks to return a true physical absolute path
	return filepath.EvalSymlinks(rawExec)
}
