// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

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
		taskTagUUID, ok2 := recordMap["taskTag"].(string)
		if !ok2 {
			return taskTag // nil
		}
		createdUUID, ok3 := recordMap["createdUUID"].(string)
		if !ok3 {
			createdUUID = ""
		}
		taskTag, _ = NewTaskTag(createdUUID, taskTagUUID)
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

func anyToString(str any) string {
	stringifiedAny, ok := str.(string)
	if !ok {
		panic(fmt.Sprintf("Unexpected variable where a string was expected: %v", str))
	}
	return stringifiedAny
}

func anyToInteger64(integer any) int64 {
	switch v := integer.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(math.Round(v)) // Handles scientific notation correctly
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64) // Convert string to int64
		if err == nil {
			return parsed
		}
	case json.Number: // handle json.Number type
		parsed, err := v.Int64()
		if err == nil {
			return parsed
		}
	}

	panic(fmt.Sprintf("Unexpected variable where an int64 was expected: %v (type %T)", integer, integer))
}

func anyToListOfMap(list any) []map[string]any {
	anyList, ok := list.([]any)
	if !ok {
		panic(fmt.Sprintf("Unexpected variable where a []any was expected: %v", list))
	}

	result := make([]map[string]any, len(anyList))
	for i, item := range anyList {
		mapItem, ok := item.(map[string]any)
		if !ok {
			panic(fmt.Sprintf("Unexpected variable where a map[string]any was expected: %v", item))
		}
		result[i] = mapItem
	}

	return result
}
