// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

func isSuperset(superset map[string]any, candidate map[string]any) bool {
	if candidate == nil {
		return true
	}

	for key, value := range candidate {
		if supValue, ok := superset[key]; ok && supValue == value {
			continue
		}
		return false
	}
	return true
}

func filterResults(results []map[string]any, filterData map[string]any) []map[string]any {
	filtered := []map[string]any{}

	for _, element := range results {
		if isSuperset(element, filterData) {
			filtered = append(filtered, element)
		}
	}

	return filtered
}
