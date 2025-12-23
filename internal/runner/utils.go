package runner

import (
	"strconv"
	"strings"
)

// GetJavaVersionForMC
// 1.20.5+ -> Java 21
// 1.18+   -> Java 17
// < 1.18  -> Java 8
func GetJavaVersionForMC(mcVersion string) int {
	parts := strings.Split(mcVersion, ".")

	first, _ := strconv.Atoi(parts[0])

	if first == 1 {
		minor, _ := strconv.Atoi(parts[1])

		if minor >= 20 {
			if len(parts) > 2 {
				patch, _ := strconv.Atoi(parts[2])
				if minor == 20 && patch >= 5 {
					return 21
				}
			}
			if minor >= 21 {
				return 21
			}
		}

		if minor >= 18 {
			return 17
		}

		return 8
	}

	if first >= 26 {
		return 21
	}

	return 21
}
