// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var ALLOWED_POWER_STATES = map[string]bool{
	"RUNNING": true,
	"PAUSED":  true,
	"SHUTOFF": true,
}

var NEEDED_ACTION_FOR_POWER_STATE = map[string]string{
	"RUNNING": "START",
	"SHUTOFF": "SHUTDOWN",
	"PAUSED":  "PAUSE",
}

func GetNeededActionForState(desiredState string, forceShutoff bool) string {
	if forceShutoff {
		return "STOP"
	}

	return NEEDED_ACTION_FOR_POWER_STATE[desiredState]
}

func ModifyVMPowerState(
	restClient RestClient,
	vmUUID string,
	payload []map[string]any,
	ctx context.Context,
) diag.Diagnostic {

	taskTag, _, err := restClient.CreateRecordWithList(
		"/rest/v1/VirDomain/action",
		payload,
		-1,
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

func GetVMPowerState(vmUUID string, restClient RestClient) (string, diag.Diagnostic) {
	vm, err := GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		return "", diag.NewErrorDiagnostic(
			"VM not found",
			err.Error(),
		)
	}

	powerState := AnyToString((*vm)["state"])

	return powerState, nil
}

func GetVMDesiredState(vmUUID string, restClient RestClient) (*string, diag.Diagnostic) {
	vm, err := GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		return nil, diag.NewErrorDiagnostic(
			"VM not found",
			err.Error(),
		)
	}

	powerState := AnyToString((*vm)["desiredDisposition"])

	return &powerState, nil
}

func ValidatePowerState(desiredState string) diag.Diagnostic {
	if !ALLOWED_POWER_STATES[desiredState] {
		return diag.NewErrorDiagnostic(
			"Invalid power state",
			fmt.Sprintf("Power state '%s' not allowed. Allowed states are: RUNNING, PAUSED, SHUTOFF", desiredState),
		)
	}
	return nil
}
