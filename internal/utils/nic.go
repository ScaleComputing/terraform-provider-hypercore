// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func CreateNic(
	restClient RestClient,
	vmUUID string,
	nic_type string,
	vlan int64,
	ctx context.Context,
) (string, map[string]any) {
	payload := map[string]any{
		"virDomainUUID": vmUUID,
		"type":          nic_type,
		"vlan":          vlan,
	}
	taskTag, _, _ := restClient.CreateRecord(
		"/rest/v1/VirDomainNetDevice",
		payload,
		-1,
	)
	taskTag.WaitTask(restClient, ctx)
	fmt.Println(payload)
	nicUUID := taskTag.CreatedUUID
	nic := GetNic(restClient, nicUUID)
	return nicUUID, *nic
}

func GetNic(
	restClient RestClient,
	nicUUID string,
) *map[string]any {
	nic := restClient.GetRecord(
		strings.Join([]string{"/rest/v1/VirDomainNetDevice", nicUUID}, "/"),
		nil,
		false,
		-1,
	)
	return nic
}

func UpdateNic(
	restClient RestClient,
	nicUUID string,
	payload map[string]any,
	ctx context.Context,
) diag.Diagnostic {
	taskTag, err := restClient.UpdateRecord(
		strings.Join([]string{"/rest/v1/VirDomainNetDevice", nicUUID}, "/"),
		payload,
		-1,
		ctx,
	)

	if err != nil {
		return diag.NewWarningDiagnostic(
			"HC3 is receiving too many requests at the same time.",
			fmt.Sprintf("Please retry apply after Terraform finishes it's current operation. HC3 response message: %v", err.Error()),
		)
	}

	taskTag.WaitTask(restClient, ctx)
	tflog.Debug(ctx, fmt.Sprintf("TTRT Task Tag: %v\n", taskTag))

	return nil
}

/*
func (vd *VMDisk) CreateOrUpdate(
	vc *VMClone,
	restClient RestClient,
	ctx context.Context,
) (bool, bool, string, error) {
	changed := false
	vm := GetByName(vc.VMName, restClient, true)
	vmUUID := AnyToString((*vm)["uuid"])
	vmDisks := AnyToListOfMap((*vm)["blockDevs"])

	if vd.Size != nil {
		existingDisk := vd.GetSpecificDisk(vmDisks, ctx) // from HC3
		desiredDisk := vd.BuildDiskPayload(vmUUID)

		tflog.Debug(ctx, fmt.Sprintf("Desired disk: %v\n", desiredDisk))
		tflog.Debug(ctx, fmt.Sprintf("Existing disk: %v\n", existingDisk))

		if existingDisk != nil {
			existingDiskSize := AnyToFloat64((*existingDisk)["capacity"]) / 1000 / 1000 / 1000
			existingDiskSlot := AnyToInteger64((*existingDisk)["slot"])
			existingDiskType := AnyToString((*existingDisk)["type"])
			desiredDiskSize := AnyToFloat64(desiredDisk["capacity"]) / 1000 / 1000 / 1000
			if existingDiskSize > desiredDiskSize {
				return false, false, "", fmt.Errorf(
					"Disk of type '%s' on slot %d can only be expanded. Use a different slot or use a larger size. %v GB > %v GB\n",
					existingDiskType, existingDiskSlot, existingDiskSize, desiredDiskSize,
				)
			}
		}

		if existingDisk != nil {
			if isSuperset(*existingDisk, desiredDisk) {
				return false, vc.WasRebooted(), "", nil
			}

			tflog.Debug(ctx, "Updating existing disk\n")
			vd.UUID = vd.UpdateBlockDevice(vc, vmUUID, restClient, desiredDisk, *existingDisk, ctx)
			changed = true
		} else {
			tflog.Debug(ctx, "Creating new disk\n")
			vd.UUID = vd.CreateBlockDevice(restClient, desiredDisk, ctx)
			changed = true
		}
	}

	return changed, vc.WasRebooted(), vd.UUID, nil
}

func (vd *VMDisk) UpdateBlockDevice(
	vc *VMClone,
	vmUUID string,
	restClient RestClient,
	desiredDisk map[string]any,
	existingDisk map[string]any,
	ctx context.Context,
) string {
	vc.DoShutdownSteps(vmUUID, SHUTDOWN_TIMEOUT_SECONDS, restClient, ctx)

	existingDiskUUID := AnyToString(existingDisk["uuid"])
	taskTag := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", existingDiskUUID),
		desiredDisk,
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)

	return existingDiskUUID
}

func (vd *VMDisk) CreateBlockDevice(
	restClient RestClient,
	desiredDisk map[string]any,
	ctx context.Context,
) string {
	taskTag, _, _ := restClient.CreateRecord(
		"/rest/v1/VirDomainBlockDevice",
		desiredDisk,
		-1,
	)
	taskTag.WaitTask(restClient, ctx)

	return taskTag.CreatedUUID
}

// This function will be useful when dealing with IDE_CDROM type disks: so for the future
// nolint:unused
func (vd *VMDisk) EnsureAbsend(
	vc *VMClone,
	changedParams map[string]bool,
	restClient RestClient,
	ctx context.Context,
) (bool, bool, map[string]any) {
	vm := GetByName(vc.VMName, restClient, true)
	vmDisks := AnyToListOfMap((*vm)["blockDevs"])

	if vd.Size != nil {
		existingDisk := vd.GetSpecificDisk(vmDisks, ctx)
		if existingDisk == nil {
			return true, false, map[string]any{} // no disk - absent is already ensured
		}

		diskUUID := AnyToString((*existingDisk)["uuid"])

		// Remove the disk to ensure it's absence
		vmUUID := AnyToString((*vm)["uuid"])
		vc.DoShutdownSteps(vmUUID, SHUTDOWN_TIMEOUT_SECONDS, restClient, ctx)

		taskTag := restClient.DeleteRecord(
			fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", diskUUID),
			-1,
			ctx,
		)
		taskTag.WaitTask(restClient, ctx)

		vc.PowerUp(*vm, restClient, ctx)
		return true, true, map[string]any{}
	}

	return false, false, map[string]any{}
}

func (vd *VMDisk) BuildDiskPayload(vmUUID string) map[string]any {
	return map[string]any{
		"virDomainUUID": vmUUID,
		"type":          vd.Type,
		"slot":          vd.Slot,
		"capacity":      *vd.Size,
	}
}

func (vd *VMDisk) GetSpecificDisk(vmDisks []map[string]any, ctx context.Context) *map[string]any {
	for _, vmDisk := range vmDisks {
		vmDiskUUID := AnyToString(vmDisk["uuid"])
		vmDiskSlot := AnyToInteger64(vmDisk["slot"])
		vmDiskType := AnyToString(vmDisk["type"])
		if vmDiskUUID == vd.UUID {
			tflog.Debug(ctx, fmt.Sprintf("Got disk by UUID: %v", vmDisk))
			return &vmDisk
		}

		if vmDiskSlot == vd.Slot && vmDiskType == vd.Type {
			tflog.Debug(ctx, fmt.Sprintf("Got disk by slot and type: %v", vmDisk))
			return &vmDisk
		}
	}
	return nil
}
*/
