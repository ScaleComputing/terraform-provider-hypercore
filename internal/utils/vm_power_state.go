// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	actionType string,
	ctx context.Context,
) diag.Diagnostic {

	payload := []map[string]any{
		{
			"virDomainUUID": vmUUID,
			"actionType":    actionType,
			"cause":         "INTERNAL",
		},
	}
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

	// corner case. If actionType=SHUTDOWN, the taskTag is empty, and we need to manuall wait on state transition to happen.
	// Say at most 300 seconds.
	if actionType == "SHUTDOWN" {
		waitVMPowerState(300, "SHUTOFF", vmUUID, restClient, ctx)
	}

	return nil
}

func waitVMPowerState(waitTimeout int32, desiredPowerState string, vmUUID string, restClient RestClient, ctx context.Context) bool {
	startTime := time.Now().Unix()
	for {
		vmPowerState, _ := GetVMPowerState(vmUUID, restClient)
		if vmPowerState == desiredPowerState {
			return true
		}
		tflog.Info(ctx, fmt.Sprintf("TTRT waitVMPowerState %v != %v", vmPowerState, desiredPowerState))

		duration := time.Now().Unix() - startTime
		if duration >= int64(waitTimeout) {
			return false
		}
		time.Sleep(10 * time.Second)
	}
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

func GetVMDesiredState(vmUUID string, restClient RestClient) (string, diag.Diagnostic) {
	vm, err := GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		return "", diag.NewErrorDiagnostic(
			"VM not found",
			err.Error(),
		)
	}

	powerState := AnyToString((*vm)["desiredDisposition"])

	return powerState, nil
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
