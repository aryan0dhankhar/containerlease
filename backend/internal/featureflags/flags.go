package featureflags

import (
	"os"
	"strings"
)

// Enabled returns true if a flag is enabled via environment variable.
// Flags are read from env as FLAG_<NAME>=true/1/yes (case-insensitive)
func Enabled(name string) bool {
	v := os.Getenv("FLAG_" + strings.ToUpper(name))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
