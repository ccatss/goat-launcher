package urlproto

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func getDefaultLauncher(proto string) (string, error) {
	// 1. Query the macOS Launch Services registry for URL handlers.
	// This pulls the secure handlers plist as text.
	cmd := exec.Command("defaults", "read", "com.apple.LaunchServices/com.apple.launchservices.secure", "LSHandlers")
	var out bytes.Buffer
	cmd.Stdout = &out

	// If defaults returns an error, it usually means no custom handlers are set yet
	if err := cmd.Run(); err != nil {
		return "", nil
	}

	// 2. Parse the output text block to locate the target protocol scheme block.
	// Plist format will cluster LSHandlerURLScheme = "jagex" and LSHandlerRoleAll = "bundle.id".
	lines := strings.Split(out.String(), "\n")
	var targetBundleID string

	for i, line := range lines {
		if strings.Contains(line, fmt.Sprintf(`LSHandlerURLScheme = "%s";`, proto)) {
			// Scan backwards or forwards in the matching block to find the paired Bundle ID
			for j := i - 3; j <= i+3; j++ {
				if j >= 0 && j < len(lines) && strings.Contains(lines[j], "LSHandlerRoleAll") {
					parts := strings.Split(lines[j], `"`)
					if len(parts) > 1 {
						targetBundleID = parts[1]
						break
					}
				}
			}
		}
		if targetBundleID != "" {
			break
		}
	}

	// If no custom bundle overrides exist, fall back to empty string
	if targetBundleID == "" {
		return "", nil
	}

	// 3. Use AppleScript (osascript) to locate the absolute path of the app bundle via its ID
	// Example script: POSIX path of (path to application id "com.jagex.launcher")
	appleScript := fmt.Sprintf(`POSIX path of (path to application id "%s")`, targetBundleID)
	pathCmd := exec.Command("osascript", "-e", appleScript)

	pathOut, err := pathCmd.Output()
	if err != nil {
		// Bundle ID might be registered but deleted from the system
		return "", fmt.Errorf("failed to locate application bundle path for %s: %w", targetBundleID, err)
	}

	// Returns an absolute path like: /Applications/Jagex Launcher.app/
	finalPath := strings.TrimSpace(string(pathOut))

	return finalPath, nil
}
