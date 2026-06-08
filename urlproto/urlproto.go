package urlproto

import "path/filepath"

func GetDefaultHandler(proto string) (string, error) {
	return getDefaultLauncher(proto)
}

func IsDefaultHandler(proto, executable string) (bool, error) {
	handler, err := GetDefaultHandler(proto)

	if err != nil {
		return false, err
	}

	executable, err = filepath.EvalSymlinks(executable)

	if err != nil {
		return false, err
	}

	return filepath.Clean(handler) == filepath.Clean(executable), nil
}
