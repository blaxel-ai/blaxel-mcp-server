package tools

import (
	"encoding/json"
	"strings"
)

// FilterAndMarshal applies optional filtering to a list response and returns JSON.
// This generic function handles the common pattern across all list handlers.
//
// Parameters:
//   - items: The slice of items from the SDK response (e.g., *[]sdk.Model)
//   - filter: Optional filter string to match against names
//   - getName: Function to extract the name from an item for filtering
//
// Returns:
//   - JSON marshaled result with indentation
func FilterAndMarshal[T any](items *[]T, filter string, getName func(T) string) ([]byte, error) {
	if items == nil {
		return json.MarshalIndent([]T{}, "", "  ")
	}

	// If no filter, return all items
	if filter == "" {
		return json.MarshalIndent(items, "", "  ")
	}

	// Apply filter
	var filtered []T
	filterLower := strings.ToLower(filter)

	for _, item := range *items {
		name := getName(item)
		if name != "" && strings.Contains(strings.ToLower(name), filterLower) {
			filtered = append(filtered, item)
		}
	}

	return json.MarshalIndent(filtered, "", "  ")
}

// ContainsString checks if a string contains a substring (case-insensitive)
func ContainsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func SetRuntimeEnv(env string) *[]interface{} {
	if env == "" {
		return nil
	}
	envMap := make([]interface{}, 0)
	for _, e := range strings.Split(env, ",") {
		parts := strings.SplitN(e, "=", 2)
		envMap = append(envMap, map[string]interface{}{
			"name":  parts[0],
			"value": parts[1],
		})
	}
	return &envMap
}
