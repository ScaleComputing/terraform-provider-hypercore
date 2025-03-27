// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func GetVmUUIDsBySnapshotScheduleUUID(
	restClient RestClient,
	scheduleUUID string,
	ctx context.Context,
) []string {
	vms := restClient.ListRecords(
		"/rest/v1/VirDomain",
		map[string]any{
			"snapshotScheduleUUID": scheduleUUID,
		},
		-1,
	)

	if len(vms) == 0 {
		return []string{}
	}

	vmUUIDs := []string{}
	for _, vm := range vms {
		vmUUIDs = append(vmUUIDs, AnyToString(vm["uuid"]))
	}
	vmUUIDs = SortUUIDs(vmUUIDs)

	return vmUUIDs
}

func GetVMSnapshotScheduleByUUIDFromVM(
	restClient RestClient,
	vmUUID string,
	ctx context.Context,
) (string, diag.Diagnostic) {
	vm, err := GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		return "", diag.NewErrorDiagnostic(
			"VM not found",
			err.Error(),
		)
	}

	scheduleUUID := AnyToString((*vm)["snapshotScheduleUUID"])

	return scheduleUUID, nil
}

func GetVMSnapshotScheduleByUUID(
	restClient RestClient,
	scheduleUUID string,
) *map[string]any {
	schedule := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomainSnapshotSchedule/%s", scheduleUUID),
		nil,
		false,
		-1,
	)

	return schedule
}

func GetVMSnapshotByUUID(
	restClient RestClient,
	snapUUID string,
) *map[string]any {
	snapshot := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomainSnapshot/%s", snapUUID),
		nil,
		false,
		-1,
	)

	return snapshot
}

func CreateVMSnapshot(
	restClient RestClient,
	vmUUID string,
	payload map[string]any,
	ctx context.Context,
) (string, map[string]any, diag.Diagnostic) {

	taskTag, _, err := restClient.CreateRecord(
		"/rest/v1/VirDomainSnapshot",
		payload,
		-1,
	)

	if err != nil {
		return "", nil, diag.NewWarningDiagnostic(
			"HC3 is receiving too many requests at the same time.",
			fmt.Sprintf("Please retry apply after Terraform finishes it's current operation. HC3 response message: %v", err.Error()),
		)
	}

	taskTag.WaitTask(restClient, ctx)
	snapUUID := taskTag.CreatedUUID
	snapshot := GetVMSnapshotByUUID(restClient, snapUUID)

	return snapUUID, *snapshot, nil
}

func CreateVMSnapshotSchedule(
	restClient RestClient,
	payload map[string]any,
	ctx context.Context,
) (string, map[string]any, diag.Diagnostic) {

	taskTag, status, err := restClient.CreateRecord(
		"/rest/v1/VirDomainSnapshotSchedule",
		payload,
		-1,
	)

	tflog.Debug(ctx, fmt.Sprintf("TTRT Snapshot Create Status: %d\n", status))

	if err != nil {
		return "", nil, diag.NewWarningDiagnostic(
			"HC3 is receiving too many requests at the same time.",
			fmt.Sprintf("Please retry apply after Terraform finishes it's current operation. HC3 response message: %v", err.Error()),
		)
	}

	taskTag.WaitTask(restClient, ctx)
	scheduleUUID := taskTag.CreatedUUID
	schedule := GetVMSnapshotScheduleByUUID(restClient, scheduleUUID)

	return scheduleUUID, *schedule, nil
}

func UpdateVMSnapshotSchedule(
	restClient RestClient,
	scheduleUUID string,
	payload map[string]any,
	ctx context.Context,
) diag.Diagnostic {

	taskTag, err := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomainSnapshotSchedule/%s", scheduleUUID),
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

func ApplyVMSnapshotSchedule(
	restClient RestClient,
	scheduleUUID string,
	vmUUID string,
	ctx context.Context,
) diag.Diagnostic {
	payload := map[string]any{
		"snapshotScheduleUUID": scheduleUUID,
	}

	taskTag, err := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vmUUID),
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

func RemoveVMSnapshotSchedule(
	restClient RestClient,
	vmUUID string,
	ctx context.Context,
) diag.Diagnostic {
	payload := map[string]any{
		"snapshotScheduleUUID": "",
	}

	taskTag, err := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vmUUID),
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
