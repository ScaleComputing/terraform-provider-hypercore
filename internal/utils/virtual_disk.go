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

func ValidateVirtualDiskSourceURL(url string) diag.Diagnostic {
	if strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "file:///") {
		return nil
	}

	return diag.NewErrorDiagnostic(
		"Invalid source URL for the virtual disk used",
		fmt.Sprintf("Source URL '%s' is invalid. Virtual disk source URL must start with either 'http://', 'https://' or 'file:///'", url),
	)
}

func GetVirtualDiskByUUID(
	restClient RestClient,
	vdUUID string,
) *map[string]any {
	virtualDisk := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/VirtualDisk/%s", vdUUID),
		nil,
		false,
		-1,
	)
	return virtualDisk
}

func GetVirtualDiskByName(
	restClient RestClient,
	name string,
) *map[string]any {
	virtualDisk := restClient.GetRecord(
		"/rest/v1/VirtualDisk",
		map[string]any{
			"name": name,
		},
		false,
		-1,
	)

	return virtualDisk
}

func UploadVirtualDisk(
	restClient RestClient,
	name string,
	sourceURL string,
	ctx context.Context,
) (string, *map[string]any, diag.Diagnostic) {
	var binaryData []byte
	var err error

	if strings.Contains(sourceURL, "http") {
		binaryData, err = FetchFileBinaryFromURL(sourceURL)
	} else if strings.Contains(sourceURL, "file:///") {
		sourceURLParts := strings.Split(sourceURL, "file:///")
		localFilePath := sourceURLParts[1]
		binaryData, err = ReadLocalFileBinary(localFilePath)
	}

	if err != nil {
		return "", nil, diag.NewErrorDiagnostic(
			"Couldn't fetch virtual disk from source",
			fmt.Sprintf("Couldn't fetch virtual disk from source '%s': %s", sourceURL, err.Error()),
		)
	}

	fileSize := len(binaryData)

	tflog.Debug(ctx, fmt.Sprintf("TTRT Virtual Disk Upload: source_url=%s, file_size=%d", sourceURL, fileSize))

	taskTag, err := restClient.PutBinaryRecord(
		fmt.Sprintf("/rest/v1/VirtualDisk/upload?filename=%s&filesize=%d", name, fileSize),
		binaryData,
		int64(fileSize),
		-1,
		ctx,
	)

	if err != nil {
		return "", nil, diag.NewWarningDiagnostic(
			"HC3 is receiving too many requests at the same time.",
			fmt.Sprintf("Please retry apply after Terraform finishes it's current operation or consider using the `-parallelism=1` terraform option. HC3 response message: %v", err.Error()),
		)
	}

	taskTag.WaitTask(restClient, ctx)
	vdUUID := taskTag.CreatedUUID
	vd := GetVirtualDiskByUUID(restClient, vdUUID)
	return vdUUID, vd, nil
}

func AttachVirtualDisk(
	restClient RestClient,
	payload map[string]any,
	sourceVirtualDiskUUID string,
	ctx context.Context,
) (string, map[string]any, error) {
	taskTag, _, _ := restClient.CreateRecord(
		fmt.Sprintf("/rest/v1/VirtualDisk/%s/attach", sourceVirtualDiskUUID),
		payload,
		-1,
	)

	taskTag.WaitTask(restClient, ctx)
	if taskTag == nil {
		return "", nil, fmt.Errorf("there was a problem attaching the virtual disk to the VM, check input parameters")
	}
	diskUUID := taskTag.CreatedUUID
	disk := GetDiskByUUID(restClient, diskUUID)
	return diskUUID, *disk, nil
}
