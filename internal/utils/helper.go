// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
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

func isSupersetRecursive(superset map[string]any, candidate map[string]any) bool {
	if candidate == nil {
		return true
	}

	for key, value := range candidate {
		supValue, ok := superset[key]
		if !ok {
			return false
		}

		switch v := value.(type) {
		case map[string]any:
			// recursive check if map
			if subMap, ok := supValue.(map[string]any); ok {
				if !isSupersetRecursive(subMap, v) {
					return false
				}
			} else {
				return false
			}
		default:
			// do normal check if not a map
			if supValue != v {
				return false
			}
		}
	}
	return true
}

func filterResultsRecursive(results []map[string]any, filterData map[string]any) []map[string]any {
	filtered := []map[string]any{}

	for _, element := range results {
		if isSupersetRecursive(element, filterData) {
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

func AnyToString(str any) string {
	stringifiedAny, ok := str.(string)
	if !ok {
		panic(fmt.Sprintf("Unexpected variable where a string was expected: %v", str))
	}
	return stringifiedAny
}

func AnyToBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	}

	panic(fmt.Sprintf("Unexpected variable where an bool was expected: %v (type %T)", value, value))
}

func AnyToInteger64(integer any) int64 {
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

func AnyToFloat64(floateger any) float64 {
	switch v := floateger.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return parsed
		}
	case json.Number: // handle json.Number type
		parsed, err := v.Float64()
		if err == nil {
			return parsed
		}
	}

	panic(fmt.Sprintf("Unexpected variable where an float64 was expected: %v (type %T)", floateger, floateger))
}

func AnyToMap(_map any) map[string]any {
	anyMap, ok := _map.(map[string]any)
	if !ok {
		panic(fmt.Sprintf("Unexpected variable where a map[string]any was expected: %v", anyMap))
	}
	return anyMap
}

func AnyToListOfMap(list any) []map[string]any {
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

func AnyToList(list any) []any {
	anyList, ok := list.([]any)
	if !ok {
		panic(fmt.Sprintf("Unexpected variable where a []any was expected: %v", list))
	}

	return anyList
}

func AnyToListOfStrings(list any) []string {
	if strList, ok := list.([]string); ok {
		return strList
	}

	anyList, ok := list.([]any)
	if !ok {
		panic(fmt.Sprintf("Unexpected variable where a []any was expected: %v", list))
	}

	strList := make([]string, len(anyList))
	for i, val := range anyList {
		strList[i] = AnyToString(val)
	}

	return strList
}

func ReadLocalFileBinary(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error opening file '%s': %s", filePath, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := make([]byte, 4096) // 4KiB buffer

	var binaryData []byte
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			binaryData = append(binaryData, buffer[:n]...)
		}
		if err != nil {
			break // EOF
		}
	}

	return binaryData, nil
}

func FetchFileBinaryFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var binaryData []byte
	buffer := make([]byte, 4096) // 4 KiB buffer
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			binaryData = append(binaryData, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return binaryData, nil
}

func GetFileSize(sourceFilePath string) int64 {
	fileInfo, err := os.Stat(sourceFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			panic(fmt.Errorf("ISO file %s not found", sourceFilePath))
		}
		panic(fmt.Errorf("unable to get file info for %s: %v", sourceFilePath, err))
	}
	return fileInfo.Size()
}
