// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	FromHypercoreToTerraformPowerState = map[string]string{
		"RUNNING":  "started",
		"SHUTOFF":  "stopped",
		"BLOCKED":  "blocked",
		"PAUSED":   "paused",
		"SHUTDOWN": "shutdown",
		"CRASHED":  "crashed",
	}

	FromTerraformToHypercorePowerAction = map[string]string{
		"start":    "START",
		"shutdown": "SHUTDOWN",
		"stop":     "STOP",
		"reboot":   "REBOOT",
		"reset":    "RESET",
		"started":  "START",
	}

	FromTerraformPowerActionToTerraformPowerState = map[string]string{
		"start":    "started",
		"shutdown": "stopped",
		"stop":     "stopped",
		"reboot":   "started",
		"reset":    "started",
		"started":  "started",
	}

	RebootLookup = map[string]bool{
		"description": false,
		"tags":        false,
		"memory":      true,
		"vcpu":        true,
		"powerState":  false,
		"diskSize":    true,
	}
)

const (
	SHUTDOWN_TIMEOUT_SECONDS = 300
)

type VM struct {
	UUID                 string
	VMName               string
	sourceVMUUID         string
	cloudInit            map[string]any
	preserveMacAddress   bool
	description          *string
	tags                 *[]string
	vcpu                 *int32
	memory               *int64
	snapshotScheduleUUID *string
	powerState           *string
	strictAffinity       bool
	preferredNodeUUID    string
	backupNodeUUID       string

	_wasNiceShutdownTried  bool
	_didNiceShutdownWork   bool
	_wasForceShutdownTried bool
	_wasStartTried         bool
	_wasRebootTried        bool
	_wasResetTried         bool
}

func NewVM(
	_VMName string,
	_sourceVMUUID string,
	userData string,
	metaData string,
	_description *string,
	_tags *[]string,
	_vcpu *int32,
	_memory *int64,
	_snapshotScheduleUUID *string,
	_powerState *string,
	_strictAffinity bool,
	_preferredNodeUUID string,
	_backupNodeUUID string,
) (*VM, error) {
	userDataB64 := base64.StdEncoding.EncodeToString([]byte(userData))
	metaDataB64 := base64.StdEncoding.EncodeToString([]byte(metaData))

	vmClone := &VM{
		UUID:               "",
		VMName:             _VMName,
		sourceVMUUID:       _sourceVMUUID,
		preserveMacAddress: false,
		cloudInit: map[string]any{
			"userData": userDataB64,
			"metaData": metaDataB64,
		},
		description:          _description,
		tags:                 _tags,
		vcpu:                 _vcpu,
		memory:               _memory,
		snapshotScheduleUUID: _snapshotScheduleUUID,
		powerState:           _powerState,
		strictAffinity:       _strictAffinity,
		preferredNodeUUID:    _preferredNodeUUID,
		backupNodeUUID:       _backupNodeUUID,

		// helpers
		_wasNiceShutdownTried:  false,
		_didNiceShutdownWork:   false,
		_wasForceShutdownTried: false,
		_wasStartTried:         false,
		_wasRebootTried:        false,
		_wasResetTried:         false,
	}

	return vmClone, nil
}

func (vc *VM) Clone(restClient RestClient, sourceVM map[string]any) *TaskTag {
	// Clone payload
	clonePayload := map[string]any{
		"template": map[string]any{
			"name":          vc.VMName,
			"cloudInitData": vc.cloudInit,
		},
	}

	taskTag, _, _ := restClient.CreateRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s/clone", sourceVM["uuid"]),
		clonePayload,
		-1,
	)

	return taskTag
}

func (vc *VM) Create(restClient RestClient, ctx context.Context) (bool, string) {
	vm := GetVM(map[string]any{"name": vc.VMName}, restClient)

	if len(vm) > 0 {
		vc.UUID = AnyToString(vm[0]["uuid"])
		return false, fmt.Sprintf("Virtual machine %s already exists.", vc.VMName)
	}

	sourceVM := GetOneVM(
		vc.sourceVMUUID,
		restClient,
	)
	sourceVMName, _ := sourceVM["name"].(string)

	// Clone payload
	task := vc.Clone(restClient, sourceVM)
	task.WaitTask(restClient, ctx)
	taskStatus := task.GetStatus(restClient)

	if taskStatus != nil {
		if state, ok := (*taskStatus)["state"]; ok && state == "COMPLETE" {
			vc.UUID = task.CreatedUUID
			return true, fmt.Sprintf("Virtual machine - %s %s - cloning complete to - %s.", sourceVMName, vc.sourceVMUUID, vc.VMName)
		}
	}

	panic(fmt.Sprintf("There was a problem during cloning of %s %s, cloning failed.", sourceVMName, vc.sourceVMUUID))
}

func (vc *VM) SetVMParams(restClient RestClient, ctx context.Context) (bool, bool, map[string]any) {
	vm := GetVMByName(vc.VMName, restClient, true)
	changed, changedParams := vc.GetChangedParams(ctx, *vm)

	if changed {
		updatePayload := vc.BuildUpdatePayload(changedParams)
		taskTag, _ := restClient.UpdateRecord(
			fmt.Sprintf("/rest/v1/VirDomain/%s", (*vm)["uuid"]),
			updatePayload,
			-1,
			ctx,
		)
		taskTag.WaitTask(restClient, ctx)

		vmMap := (*vm)
		if vc.NeedsReboot(changedParams) && (vmMap["state"] != "STOP" && vmMap["state"] != "SHUTOFF" && vmMap["state"] != "SHUTDOWN") {
			vmUUID, ok := vmMap["uuid"].(string)
			if ok {
				vc.DoShutdownSteps(vmUUID, SHUTDOWN_TIMEOUT_SECONDS, restClient, ctx)
			} else {
				panic(fmt.Sprintf("Unexpected value found for UUID: %v", vmMap["uuid"]))
			}
		}
	}

	if vc.powerState != nil {
		if *vc.powerState != "shutdown" && *vc.powerState != "stop" {
			vc.PowerUp(*vm, restClient, ctx)
		}

		if powerState, ok := changedParams["powerState"]; ok && powerState {
			ignoreRepeatedRequest := true
			vc.UpdatePowerState(
				*vm,
				restClient,
				*vc.powerState,
				ignoreRepeatedRequest,
				ctx,
			)
		}
	}

	afterVM := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", (*vm)["uuid"]),
		map[string]any{},
		true,
		-1,
	)

	diff := map[string]any{
		"before": vm,
		"after":  afterVM,
	}

	return changed, vc.WasRebooted(), diff
}

func (vc *VM) UpdatePowerState(
	vm map[string]any,
	restClient RestClient,
	requestedPowerAction string,
	ignoreRepeatedRequest bool,
	ctx context.Context,
) {
	panicOrIgnoreRepeatedRequest := func(msg string) {
		if ignoreRepeatedRequest {
			return
		}
		panic(msg)
	}

	if _, ok := vm["state"]; !ok {
		panic("No information about VM's power state.")
	}

	tflog.Debug(ctx, fmt.Sprintf("Requested power action: %s\n", requestedPowerAction))

	switch requestedPowerAction {
	case "start":
		if vc._wasStartTried {
			panicOrIgnoreRepeatedRequest("VM _wasStartTried already set")
			return
		}
		vc._wasStartTried = true
	case "shutdown":
		if vc._wasNiceShutdownTried {
			panicOrIgnoreRepeatedRequest("VM _wasNiceShutdownTried already set")
			return
		}
		vc._wasNiceShutdownTried = true
	case "stop":
		if vc._wasForceShutdownTried {
			panicOrIgnoreRepeatedRequest("VM _wasForceShutdownTried already set")
			return
		}
		vc._wasForceShutdownTried = true
	case "reboot":
		if vc._wasRebootTried {
			panicOrIgnoreRepeatedRequest("VM _wasRebootTried already set")
			return
		}
		vc._wasRebootTried = true
	case "reset":
		if vc._wasResetTried {
			panicOrIgnoreRepeatedRequest("VM _wasResetTried already set")
			return
		}
		vc._wasResetTried = true
	}

	taskTag, responseStatus, err := restClient.CreateRecordWithList(
		"/rest/v1/VirDomain/action",
		[]map[string]any{
			{
				"virDomainUUID": vm["uuid"],
				"actionType":    FromTerraformToHypercorePowerAction[requestedPowerAction],
				"cause":         "INTERNAL",
			},
		},
		-1,
	)

	if err != nil {
		if requestedPowerAction != "reset" {
			return
		}
		if responseStatus != 500 {
			return
		}
		tflog.Warn(ctx, "Ignoring failed VM RESET")
		return
	}
	taskTag.WaitTask(restClient, ctx)
}

func (vc *VM) PowerUp(vm map[string]any, restClient RestClient, ctx context.Context) {
	if vc.WasShutdown() && vm["state"] == "RUNNING" {
		vc.UpdatePowerState(vm, restClient, "start", false, ctx)
		return
	}

	if vc.powerState != nil && *vc.powerState == "start" {
		vc.UpdatePowerState(vm, restClient, *vc.powerState, false, ctx)
	}
}

func (vc *VM) WasShutdown() bool {
	return vc._didNiceShutdownWork || vc._wasForceShutdownTried
}

func (vc *VM) WasRebooted() bool {
	return vc.WasShutdown() && vc._wasStartTried
}

func (vc *VM) DoShutdownSteps(vmUUID string, shutdownTimeout int, restClient RestClient, ctx context.Context) {
	if !vc.WaitShutdown(vmUUID, shutdownTimeout, restClient, ctx) {
		if !vc.ShutdownForced(vmUUID, restClient, ctx) {
			panic(fmt.Sprintf("VM - %s - needs to be powered off and is not responding to a shutdown request.", vc.VMName))
		}
	}
}

func (vc *VM) WaitShutdown(vmUUID string, shutdownTimeout int, restClient RestClient, ctx context.Context) bool {
	vmFreshData := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vmUUID),
		map[string]any{},
		true,
		-1,
	)

	if (*vmFreshData)["state"] == "SHUTOFF" || (*vmFreshData)["state"] == "SHUTDOWN" {
		return true
	}

	if (*vmFreshData)["state"] == "RUNNING" && !vc._wasNiceShutdownTried {
		vc.UpdatePowerState(*vmFreshData, restClient, "shutdown", false, ctx)
		startTime := time.Now().Unix()
		for {
			vm := restClient.GetRecord(
				fmt.Sprintf("/rest/v1/VirDomain/%s", vmUUID),
				map[string]any{},
				true,
				-1,
			)
			duration := time.Now().Unix() - startTime
			if (*vm)["state"] == "SHUTDOWN" || (*vm)["state"] == "SHUTOFF" {
				vc._didNiceShutdownWork = true
				return true
			}
			if duration >= int64(shutdownTimeout) {
				return false
			}
			time.Sleep(10 * time.Second)
		}
	}

	return false
}

func (vc *VM) ShutdownForced(vmUUID string, restClient RestClient, ctx context.Context) bool {
	vmFreshData := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vmUUID),
		map[string]any{},
		true,
		-1,
	)

	if (*vmFreshData)["state"] == "SHUTOFF" || (*vmFreshData)["state"] == "SHUTDOWN" {
		return true
	}

	vc.UpdatePowerState(*vmFreshData, restClient, "stop", false, ctx)
	return true
}

func (vc *VM) NeedsReboot(changedParams map[string]bool) bool {
	for param, changed := range changedParams {
		if needsReboot, ok := RebootLookup[param]; ok && (needsReboot && changed) {
			return true
		}
	}
	return false
}

func (vc *VM) BuildUpdatePayload(changedParams map[string]bool) map[string]any {
	updatePayload := map[string]any{}

	if changed, ok := changedParams["description"]; ok && changed {
		updatePayload["description"] = *vc.description
	}
	if changed, ok := changedParams["tags"]; ok && changed {
		updatePayload["tags"] = tagsListToCommaString(*vc.tags)
	}
	if changed, ok := changedParams["memory"]; ok && changed {
		vcMemoryBytes := *vc.memory * 1024 * 1024 // MB to B
		updatePayload["mem"] = vcMemoryBytes
	}
	if changed, ok := changedParams["vcpu"]; ok && changed {
		updatePayload["numVCPU"] = *vc.vcpu
	}
	if changed, ok := changedParams["snapshotScheduleUUID"]; ok && changed {
		updatePayload["snapshotScheduleUUID"] = *vc.snapshotScheduleUUID
	}

	affinityStrategy := map[string]any{}
	if changed, ok := changedParams["strictAffinity"]; ok && changed {
		affinityStrategy["strictAffinity"] = vc.strictAffinity
	}
	if changed, ok := changedParams["preferredNodeUUID"]; ok && changed {
		affinityStrategy["preferredNodeUUID"] = vc.preferredNodeUUID
	}
	if changed, ok := changedParams["backupNodeUUID"]; ok && changed {
		affinityStrategy["backupNodeUUID"] = vc.backupNodeUUID
	}
	if len(affinityStrategy) > 0 {
		updatePayload["affinityStrategy"] = affinityStrategy
	}

	return updatePayload
}

func (vc *VM) GetChangedParams(ctx context.Context, vmFromClient map[string]any) (bool, map[string]bool) {
	changedParams := map[string]bool{}

	if vc.description != nil {
		changedParams["description"] = *vc.description != vmFromClient["description"]
	}
	if vc.tags != nil {
		changedParams["tags"] = !reflect.DeepEqual(*vc.tags, vmFromClient["tags"])
	}
	if vc.memory != nil {
		vcMemoryBytes := *vc.memory * 1024 * 1024 // MB to B
		changedParams["memory"] = vcMemoryBytes != vmFromClient["mem"]
	}
	if vc.vcpu != nil {
		changedParams["vcpu"] = *vc.memory != vmFromClient["numVCPU"]
	}
	if vc.powerState != nil {
		requestedPowerAction := *vc.powerState
		if requestedPowerAction == "reset" || requestedPowerAction == "reboot" {
			changedParams["powerState"] = true
		} else {
			desiredPowerState := FromTerraformPowerActionToTerraformPowerState[requestedPowerAction]
			changedParams["powerState"] = desiredPowerState != vmFromClient["state"]
		}
	}
	if vc.snapshotScheduleUUID != nil {
		changedParams["snapshotScheduleUUID"] = *vc.snapshotScheduleUUID != vmFromClient["snapshotScheduleUUID"]
	}

	hc3AffinityStrategy := AnyToMap(vmFromClient["affinityStrategy"])
	changedParams["strictAffinity"] = vc.strictAffinity != hc3AffinityStrategy["strictAffinity"]
	changedParams["preferredNodeUUID"] = vc.preferredNodeUUID != hc3AffinityStrategy["preferredNodeUUID"]
	changedParams["backupNodeUUID"] = vc.backupNodeUUID != hc3AffinityStrategy["backupNodeUUID"]

	for _, changed := range changedParams {
		if changed {
			return true, changedParams
		}
	}
	return false, changedParams
}

func GetOneVM(uuid string, restClient RestClient) map[string]any {
	url := "/rest/v1/VirDomain/" + uuid
	records := restClient.ListRecords(
		url,
		map[string]any{},
		-1.0,
	)

	if len(records) == 0 {
		panic(fmt.Errorf("VM not found: uuid=%v", uuid))
	}

	return records[0]
}

func GetOneVMWithError(uuid string, restClient RestClient) (*map[string]any, error) {
	record := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", uuid),
		nil,
		false,
		-1.0,
	)

	if record == nil {
		return nil, fmt.Errorf("VM not found - vmUUID=%s.\n", uuid)
	}

	return record, nil
}

func GetVMOrFail(query map[string]any, restClient RestClient) []map[string]any {
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

func GetVM(query map[string]any, restClient RestClient) []map[string]any {
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

func GetVMByName(name string, restClient RestClient, mustExist bool) *map[string]any {
	record := restClient.GetRecord(
		"/rest/v1/VirDomain",
		map[string]any{
			"name": name,
		},
		mustExist,
		-1,
	)

	return record
}

func GetVMByOldOrNewName(name string, newName string, restClient RestClient, mustExist bool) *map[string]any {
	oldVM := GetVMByName(name, restClient, mustExist)
	newVM := GetVMByName(newName, restClient, mustExist)

	if oldVM != nil && newVM != nil {
		panic(fmt.Sprintf("More than one VM matches requirement name==%s or newName==%s", name, newName))
	}

	var vm *map[string]any
	if oldVM == nil {
		vm = newVM
	} else if newVM == nil {
		vm = oldVM
	}

	if mustExist && vm == nil {
		panic(fmt.Sprintf("No VM found: name=%s or newName=%s", name, newName))
	}

	return vm
}
