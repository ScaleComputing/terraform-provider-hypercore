// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"encoding/base64"
	"fmt"
)

// type CloneVMPayload struct {
// 	VMName             string
// 	SourceVMName       string
// 	preserveMacAddress bool
// 	sourceNics         map[string]any
// 	sourceSnapshotUUID string
// }

type VMClone struct {
	VMName             string
	sourceVMName       string
	cloudInit          map[string]any
	preserveMacAddress bool
	// tags               []string
}

func NewVMClone(_VMName string, _sourceVMName string, userData string, metaData string) (*VMClone, error) {
	userDataB64 := base64.StdEncoding.EncodeToString([]byte(userData))
	metaDataB64 := base64.StdEncoding.EncodeToString([]byte(metaData))

	vmClone := &VMClone{
		VMName:             _VMName,
		sourceVMName:       _sourceVMName,
		preserveMacAddress: false,
		cloudInit: map[string]any{
			"userData": userDataB64,
			"metaData": metaDataB64,
		},
	}

	return vmClone, nil
}

func (vc *VMClone) Clone(restClient RestClient, sourceVM map[string]any) *TaskTag {
	// Clone payload
	clonePayload := map[string]any{
		"template": map[string]any{
			"name":          vc.VMName,
			"cloudInitData": vc.cloudInit,
		},
	}

	record := restClient.CreateRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s/clone", sourceVM["uuid"]),
		clonePayload,
		-1,
	)

	var taskTag *TaskTag

	if _, ok := record.(map[string]any); ok {
		recordMap, _ := record.(map[string]any)
		taskTag, _ = NewTaskTag(recordMap["createdUUID"].(string), recordMap["taskTag"].(string))
	}

	return taskTag
}

func (vc *VMClone) Create(restClient RestClient, ctx context.Context) (bool, string) {
	if len(Get(map[string]any{"name": vc.VMName}, restClient)) > 0 {
		return false, fmt.Sprintf("Virtual machine %s already exists.", vc.VMName)
	}

	sourceVM := GetOrFail(
		map[string]any{
			"name": vc.sourceVMName,
		},
		restClient,
	)[0]

	// Clone payload
	task := vc.Clone(restClient, sourceVM)
	task.WaitTask(restClient, ctx)
	taskStatus := task.GetStatus(restClient)

	if taskStatus != nil {
		if state, ok := (*taskStatus)["state"]; ok && state == "COMPLETE" {
			return true, fmt.Sprintf("Virtual machine - %s - cloning complete to - %s.", vc.sourceVMName, vc.VMName)
		}
	}

	panic(fmt.Sprintf("There was a problem during cloning of %s, cloning failed.", vc.sourceVMName))
}

func GetOrFail(query map[string]any, restClient RestClient) []map[string]any {
	records := restClient.ListRecords(
		"/rest/v1/VirDomain",
		query,
		-1.0,
	)

	if len(records) == 0 {
		panic(fmt.Errorf("No VM found: %v", query))
	}

	return records
}

func Get(query map[string]any, restClient RestClient) []map[string]any {
	records := restClient.ListRecords(
		"/rest/v1/VirDomain",
		query,
		-1.0,
	)

	if len(records) == 0 {
		return []map[string]any{}
	}

	return records
}
