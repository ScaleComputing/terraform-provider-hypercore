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
	macAddress string,
	ctx context.Context,
) (string, map[string]any) {
	payload := map[string]any{
		"virDomainUUID": vmUUID,
		"type":          nic_type,
		"vlan":          vlan,
	}
	if macAddress != "" {
		payload["macAddress"] = macAddress
	}
	taskTag, _, _ := restClient.CreateRecord(
		"/rest/v1/VirDomainNetDevice",
		payload,
		-1,
	)
	taskTag.WaitTask(restClient, ctx)
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

// Checks that source VM UUID wasn't altered during update.
func ValidateNICSourceVMUUIDUnchanged(nicUUID string, oldVMUUID string, newVMUUID string) diag.Diagnostic {
	if oldVMUUID != newVMUUID {
		return diag.NewErrorDiagnostic(
			"Invalid NIC source virtual machine UUID",
			fmt.Sprintf(
				" virtual machine and NIC relationship is established at creation and cannot be changed, source UUID: %s, new VM UUID: %s, NIC UUID: %s",
				oldVMUUID, newVMUUID, nicUUID,
			),
		)
	}
	return nil
}
