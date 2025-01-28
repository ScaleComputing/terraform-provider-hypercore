package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type VMDisk struct {
	Slot int32
	Type string
	Size *int64
}

func NewVMDiskForClone(size *int64) (*VMDisk, error) {
	var byteSize *int64

	if size != nil {
		byteSize = new(int64)
		*byteSize = *size * 1000 * 1000 * 1000 // GB to B
	} else {
		byteSize = nil
	}

	vmDiskForClone := &VMDisk{
		Slot: 1,
		Type: "VIRTIO_DISK",
		Size: byteSize,
	}

	return vmDiskForClone, nil
}

func (vd *VMDisk) SetSize(
	vc *VMClone,
	restClient RestClient,
	ctx context.Context,
) (bool, bool, map[string]any) {
	if vd.Size == nil {
		return false, false, map[string]any{}
	}

	changed := false
	vm := GetByName(vc.VMName, restClient, true)
	vmUUID := anyToString((*vm)["uuid"])
	vmDisks := anyToListOfMap((*vm)["blockDevs"])

	if vd.Size != nil {
		existingDisk := vd.GetSpecificDisk(vmDisks)
		desiredDisk := vd.BuildDiskPayload(vmUUID)

		tflog.Debug(ctx, fmt.Sprintf("Desired disk: %v", desiredDisk))
		tflog.Debug(ctx, fmt.Sprintf("Existing disk: %v", existingDisk))

		if existingDisk != nil && vd.Size != nil {
			existingDiskSize := anyToInteger64((*existingDisk)["capacity"])
			desiredDiskSize := anyToInteger64(desiredDisk["capacity"])
			if existingDiskSize > desiredDiskSize {
				panic(fmt.Sprintf("Disk size can only be enlarged, never downsized: %v > %v", existingDiskSize, desiredDiskSize))
			}
		}

		if existingDisk != nil {
			if isSuperset(*existingDisk, desiredDisk) {
				return false, vc.WasRebooted(), map[string]any{}
			}

			tflog.Debug(ctx, "Updating existing disk")
			vd.UpdateBlockDevice(vc, vmUUID, restClient, desiredDisk, *existingDisk, ctx)
			changed = true
		} else {
			tflog.Debug(ctx, "Creating new disk")
			vd.CreateBlockDevice(restClient, desiredDisk, ctx)
			changed = true
		}
	}

	vmAfter := GetByName(vc.VMName, restClient, true)
	vmDisksAfter := anyToListOfMap((*vmAfter)["blockDevs"])
	var diskDiff map[string]any

	if changed {
		diskDiff = map[string]any{
			"before": vmDisks,
			"after":  vmDisksAfter,
		}
	}

	return changed, vc.WasRebooted(), diskDiff
}

func (vd *VMDisk) UpdateBlockDevice(
	vc *VMClone,
	vmUUID string,
	restClient RestClient,
	desiredDisk map[string]any,
	existingDisk map[string]any,
	ctx context.Context,
) {
	vc.DoShutdownSteps(vmUUID, SHUTDOWN_TIMEOUT_SECONDS, restClient, ctx)

	existingDiskUUID := anyToString(existingDisk["uuid"])
	taskTag := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", existingDiskUUID),
		desiredDisk,
		-1,
		ctx,
	)

	taskTag.WaitTask(restClient, ctx)
}

func (vd *VMDisk) CreateBlockDevice(
	restClient RestClient,
	desiredDisk map[string]any,
	ctx context.Context,
) {
	taskTag, _, _ := restClient.CreateRecord(
		"/rest/v1/VirDomainBlockDevice",
		desiredDisk,
		-1,
	)
	taskTag.WaitTask(restClient, ctx)
}

// Will remove this function if it's not needed after further development
// nolint:unused
func (vd *VMDisk) EnsureAbsend(
	vc *VMClone,
	changedParams map[string]bool,
	restClient RestClient,
	ctx context.Context,
) (bool, bool, map[string]any) {
	// TODO: return changed, wasVMRebooted, VMDiffs (before and after disk changes)

	vm := GetByName(vc.VMName, restClient, true)
	vmDisks := anyToListOfMap((*vm)["blockDevs"])

	if vd.Size != nil {
		existingDisk := vd.GetSpecificDisk(vmDisks)
		if existingDisk == nil {
			return true, false, map[string]any{} // no disk - absent is already ensured
		}

		diskUUID := anyToString((*existingDisk)["uuid"])

		// Remove the disk to ensure it's absence
		vmUUID := anyToString((*vm)["uuid"])
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

func (vd *VMDisk) GetSpecificDisk(vmDisks []map[string]any) *map[string]any {
	for _, vmDisk := range vmDisks {
		if vmDisk["slot"] == vd.Slot || vmDisk["type"] == vd.Type {
			return &vmDisk
		}
	}
	return nil
}
