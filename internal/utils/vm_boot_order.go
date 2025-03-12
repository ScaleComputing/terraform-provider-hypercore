// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func ModifyVMBootOrder(
	restClient RestClient,
	vmUUID string,
	bootOrder []string,
	ctx context.Context,
) diag.Diagnostic {
	payload := map[string]any{
		"bootDevices": bootOrder,
	}

	taskTag, _, err := restClient.CreateRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vmUUID),
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

func GetVMBootOrder(vmUUID string, restClient RestClient) ([]string, diag.Diagnostic) {
	vm, err := GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		return nil, diag.NewErrorDiagnostic(
			"VM not found",
			err.Error(),
		)
	}

	bootOrder := AnyToListOfStrings((*vm)["bootDevices"])

	return bootOrder, nil
}
