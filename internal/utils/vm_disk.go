// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ALLOWED_DISK_TYPES = map[string]bool{
	"IDE_DISK":    true,
	"SCSI_DISK":   true,
	"VIRTIO_DISK": true,
	"IDE_FLOPPY":  true,
	"NVRAM":       true,
	"VTPM":        true,
	"IDE_CDROM":   true,
}

var FROM_HUMAN_PRIORITY_FACTOR = map[int64]int64{
	0:  0,
	1:  1,
	2:  2,
	3:  4,
	4:  8,
	5:  16,
	6:  32,
	7:  64,
	8:  128,
	9:  256,
	10: 1024,
	11: 10240,
}

var TO_HUMAN_PRIORITY_FACTOR = map[int64]int64{
	0:     0,
	1:     1,
	2:     2,
	4:     3,
	8:     4,
	16:    5,
	32:    6,
	64:    7,
	128:   8,
	256:   9,
	1024:  10,
	10240: 11,
}

type VMDisk struct {
	Label string
	UUID  string // known after creation
	Slot  int64
	Type  string
	Size  *float64
}

func NewVMDisk(
	_label string,
	_slot int64,
	_type string,
	_size *float64,
) (*VMDisk, error) {
	if !ALLOWED_DISK_TYPES[_type] {
		return nil, fmt.Errorf("disk type '%s' not allowed. Allowed types are: IDE_DISK, SCSI_DISK, VIRTIO_DISK, IDE_FLOPPY, NVRAM, VTPM", _type)
	}

	var byteSize *float64
	if _size != nil {
		byteSize = new(float64)
		*byteSize = *_size * 1000 * 1000 * 1000 // GB to B
	} else {
		byteSize = nil
	}

	vmDisk := &VMDisk{
		Label: _label,
		Slot:  _slot,
		Type:  _type,
		Size:  byteSize,
	}

	return vmDisk, nil
}

func UpdateVMDisk(
	_uuid string,
	_label string,
	_slot int64,
	_type string,
	_size *float64,
) (*VMDisk, error) {

	var byteSize *float64
	if _size != nil {
		byteSize = new(float64)
		*byteSize = *_size * 1000 * 1000 * 1000 // GB to B
	} else {
		byteSize = nil
	}

	vmDisk := &VMDisk{
		UUID:  _uuid,
		Label: _label,
		Slot:  _slot,
		Type:  _type,
		Size:  byteSize,
	}

	return vmDisk, nil
}

func (vd *VMDisk) CreateOrUpdate(
	vc *VM,
	restClient RestClient,
	ctx context.Context,
) (bool, bool, string, error) {
	changed := false
	vm := GetVMByName(vc.VMName, restClient, true)
	vmUUID := AnyToString((*vm)["uuid"])
	vmDisks := AnyToListOfMap((*vm)["blockDevs"])

	if vd.Size != nil {
		existingDisk := vd.Get(vmDisks, ctx) // from HC3
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
					"disk of type '%s' on slot %d can only be expanded. Use a different slot or use a larger size. %v GB > %v GB",
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
	vc *VM,
	vmUUID string,
	restClient RestClient,
	desiredDisk map[string]any,
	existingDisk map[string]any,
	ctx context.Context,
) string {
	// TODO: this will be a new resource in the future, for now we act like the VMs are always shut down
	// vc.DoShutdownSteps(vmUUID, SHUTDOWN_TIMEOUT_SECONDS, restClient, ctx)

	existingDiskUUID := AnyToString(existingDisk["uuid"])
	taskTag, _ := restClient.UpdateRecord(
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

// TODO: this function might be useful when dealing with IDE_CDROM type disks: so for the future
// nolint:unused
func (vd *VMDisk) EnsureAbsend(
	vc *VM,
	changedParams map[string]bool,
	restClient RestClient,
	ctx context.Context,
) (bool, bool, map[string]any) {
	vm := GetVMByName(vc.VMName, restClient, true)
	vmDisks := AnyToListOfMap((*vm)["blockDevs"])

	if vd.Size != nil {
		existingDisk := vd.Get(vmDisks, ctx)
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

func (vd *VMDisk) Get(vmDisks []map[string]any, ctx context.Context) *map[string]any {
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

func GetDiskByTypeAndSlot(hc3Disks []map[string]any, diskSlot int64, diskType string, ctx context.Context) (string, float64) {
	for _, hc3Disk := range hc3Disks {
		hc3DiskUUID := AnyToString(hc3Disk["uuid"])
		hc3DiskSlot := AnyToInteger64(hc3Disk["slot"])
		hc3DiskType := AnyToString(hc3Disk["type"])
		hc3DiskSize := AnyToFloat64(hc3Disk["capacity"]) / 1000 / 1000 / 1000 // B -> GB

		if hc3DiskSlot == diskSlot && hc3DiskType == diskType {
			tflog.Debug(ctx, fmt.Sprintf("Got disk by slot and type: %v", hc3Disk))
			return hc3DiskUUID, hc3DiskSize
		}
	}
	return "", -2
}

func GetDiskByUUID(restClient RestClient, diskUUID string) *map[string]any {
	disk := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", diskUUID),
		nil,
		false,
		-1,
	)
	return disk
}

func BuildDiskPayload(vmUUID string, diskType string, diskSlot int64, diskSizeGB float64) map[string]any {
	return map[string]any{
		"virDomainUUID": vmUUID,
		"type":          diskType,
		"slot":          diskSlot,
		"capacity":      diskSizeGB * 1000 * 1000 * 1000, // GB to B
	}
}

func UpdateDisk(
	restClient RestClient,
	diskUUID string,
	payload map[string]any,
	ctx context.Context,
) diag.Diagnostic {
	taskTag, err := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", diskUUID),
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

	return nil
}

func CreateDisk(
	restClient RestClient,
	payload map[string]any,
	ctx context.Context,
) (string, map[string]any) {
	taskTag, _, _ := restClient.CreateRecord(
		"/rest/v1/VirDomainBlockDevice/",
		payload,
		-1,
	)

	taskTag.WaitTask(restClient, ctx)

	diskUUID := taskTag.CreatedUUID
	disk := GetDiskByUUID(restClient, diskUUID)
	return diskUUID, *disk
}

func ValidateDiskFlashPriority(diskFlashPriority int64) diag.Diagnostic {
	if diskFlashPriority < 0 || diskFlashPriority > 11 {
		return diag.NewErrorDiagnostic(
			"Invalid disk flash priority",
			fmt.Sprintf("Disk flash priority '%v' is invalid. Flash priority must be a positive number between (including) 0 and 11.", diskFlashPriority),
		)
	}

	return nil
}

func ValidateDiskType(diskType string, isoUUID string) diag.Diagnostic {
	if !ALLOWED_DISK_TYPES[diskType] {
		return diag.NewErrorDiagnostic(
			"Invalid disk type",
			fmt.Sprintf("Disk type '%s' not allowed. Allowed types are: IDE_DISK, IDE_CDROM, SCSI_DISK, VIRTIO_DISK, IDE_FLOPPY, NVRAM, VTPM", diskType),
		)
	}
	if isoUUID != "" && diskType != "IDE_CDROM" {
		return diag.NewErrorDiagnostic(
			"Invalid disk type",
			fmt.Sprintf("Disk type '%s' is not compatible with ISO, for ISO attach action type of IDE_CDROM is needed.", diskType),
		)
	}
	return nil
}

func ValidateISOAttach(restClient RestClient, isoUUID string, isAttachingISO bool) (diag.Diagnostic, *map[string]any) {
	if isAttachingISO {
		iso := GetISOByUUID(restClient, isoUUID)
		if iso == nil {
			return diag.NewErrorDiagnostic(
					"Invalid ISO UUID",
					fmt.Sprintf("ISO with UUID '%s' not found.", isoUUID),
				),
				nil
		}
		return nil, iso
	}
	return nil, nil
}

func ValidateDiskSize(diskUUID string, oldSize float64, newSize float64) diag.Diagnostic {
	if newSize < oldSize {
		return diag.NewErrorDiagnostic(
			"Invalid disk size",
			fmt.Sprintf(
				" can only be expanded. Use a larger size. %v GB > %v GB: diskUUID=%s",
				newSize, oldSize, diskUUID,
			),
		)
	}
	return nil
}
