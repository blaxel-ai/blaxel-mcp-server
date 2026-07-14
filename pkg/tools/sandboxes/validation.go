package sandboxes

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func validateSandboxMemory(memory float64) error {
	if memory <= 0 || math.IsNaN(memory) || math.IsInf(memory, 0) {
		return fmt.Errorf("memory must be a positive finite number of MB")
	}
	if memory > math.MaxInt32 || math.Trunc(memory) != memory {
		return fmt.Errorf("memory must be a whole number of MB representable as a 32-bit integer")
	}
	return nil
}

func parseSandboxPorts(ports string) ([]int, error) {
	if ports == "" {
		return nil, nil
	}

	portStrings := strings.Split(ports, ",")
	parsed := make([]int, 0, len(portStrings))
	for _, value := range portStrings {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		port, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("port %q must be a number between 1 and 65535", value)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("port %q must be between 1 and 65535", value)
		}
		parsed = append(parsed, port)
	}
	return parsed, nil
}
