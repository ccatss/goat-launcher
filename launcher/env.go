package launcher

import (
	"fmt"
	"os"
	"strings"
)

type EnvMap map[string]string

// NewEnv initializes an EnvMap with the current system environment variables
func NewEnv() EnvMap {
	em := make(EnvMap)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) == 2 {
			em[parts[0]] = parts[1]
		}
	}
	
	return em
}

// Set adds or overrides an environment variable
func (em EnvMap) Set(key, value string) {
	em[key] = value
}

// Slice converts the map back into the []string format required by exec.Cmd
func (em EnvMap) Slice() []string {
	slice := make([]string, 0, len(em))

	for k, v := range em {
		slice = append(slice, fmt.Sprintf("%s=%s", k, v))
	}

	return slice
}
