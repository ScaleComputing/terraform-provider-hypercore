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

func GetVMReplicationByUUID(
	restClient RestClient,
	replicationUUID string,
) *map[string]any {
	replication := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirDomainReplication/%s", replicationUUID),
		nil,
		false,
		-1,
	)
	return replication
}

func CreateVMReplication(
	restClient RestClient,
	sourceVmUUID string,
	connectionUUID string,
	label string,
	enable bool,
	ctx context.Context,
) (string, map[string]any, diag.Diagnostic) {
	payload := map[string]any{
		"sourceDomainUUID": sourceVmUUID,
		"connectionUUID":   connectionUUID,
		"label":            label,
		"enable":           enable,
	}
	taskTag, status, err := restClient.CreateRecord(
		"/rest/v1/VirDomainReplication",
		payload,
		-1,
	)

	if status == 400 && err != nil {
		isReplicationError := strings.Contains(err.Error(), "Failed to create replication")
		if isReplicationError {
			return "", nil, diag.NewWarningDiagnostic(
				"Couldn't create a VM replication",
				fmt.Sprintf("VM replication failed. Source VM '%s' might already have configured replication. Response message: %s", sourceVmUUID, err.Error()),
			)
		}
		return "", nil, diag.NewErrorDiagnostic(
			"Couldn't create a VM replication",
			fmt.Sprintf("Something unexpected happened while attempting to create replication for VM '%s'. Response message: %s", sourceVmUUID, err.Error()),
		)
	}

	taskTag.WaitTask(restClient, ctx)
	replicationUUID := taskTag.CreatedUUID
	replication := GetVMReplicationByUUID(restClient, replicationUUID)
	return replicationUUID, *replication, nil
}

func UpdateVMReplication(
	restClient RestClient,
	replicationUUID string,
	sourceVmUUID string,
	connectionUUID string,
	label string,
	enable bool,
	ctx context.Context,
) diag.Diagnostic {
	payload := map[string]any{
		"sourceDomainUUID": sourceVmUUID,
		"connectionUUID":   connectionUUID,
		"label":            label,
		"enable":           enable,
	}

	taskTag, err := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/VirDomainReplication/%s", replicationUUID),
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
