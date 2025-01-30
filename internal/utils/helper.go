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

// nolint:unused
func filterMap(input map[string]any, fieldNames ...string) map[string]any {
	output := map[string]any{}

	for _, fieldName := range fieldNames {
		if value, ok := input[fieldName]; ok {
			if value != nil || value != "" {
				output[fieldName] = value
			}
		}
	}

	return output
}

func jsonObjectToTaskTag(jsonObj any) *TaskTag {
	var taskTag *TaskTag

	if _, ok := jsonObj.(map[string]any); ok {
		recordMap, _ := jsonObj.(map[string]any)
		taskTag, _ = NewTaskTag(recordMap["createdUUID"].(string), recordMap["taskTag"].(string))
	}

	return taskTag
}

func tagsListToCommaString(tags []string) string {
	tagsHyp := ""
	for _, tag := range tags {
		tagsHyp += tag + ","
	}

	tagsHyp = tagsHyp[:len(tagsHyp)-1] + ""

	return tagsHyp
}
