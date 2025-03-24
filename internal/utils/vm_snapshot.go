// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var ALLOWED_TYPES = map[string]bool{
	"USER":      true,
	"AUTOMATED": false,
	"SUPPORT":   true,
}

func ValidateSnapshotType(desiredType string) diag.Diagnostic {
	if !ALLOWED_TYPES[desiredType] {
		return diag.NewErrorDiagnostic(
			"Invalid Snapshot type",
			fmt.Sprintf("Snapshot type '%s' not allowed. Allowed states are: USER, SUPPORT", desiredType),
		)
	}
	return nil
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
	vmUUID string,
	payload []map[string]any,
) (string, map[string]any, diag.Diagnostic) {
	// TODO
	return "", nil, nil
}
