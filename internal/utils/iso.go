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

func ValidateISOName(name string) diag.Diagnostic {
	if strings.HasSuffix(name, ".iso") {
		return nil
	}

	return diag.NewErrorDiagnostic(
		"Invalid ISO name for the ISO used",
		fmt.Sprintf("ISO name '%s' is invalid. ISO name must end with '.iso'", name),
	)
}

func ValidateISOSourceURL(url string) diag.Diagnostic {
	if strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "file:///") {
		return nil
	}

	return diag.NewErrorDiagnostic(
		"Invalid source URL for the ISO used",
		fmt.Sprintf("Source URL '%s' is invalid. ISO source URL must start with either 'http://', 'https://' or 'file:///'", url),
	)
}

func ReadISOBinary(sourceURL string) ([]byte, diag.Diagnostic) {
	var binaryData []byte
	var err error

	if strings.Contains(sourceURL, "http") {
		binaryData, err = FetchFileBinaryFromURL(sourceURL)
	} else if strings.Contains(sourceURL, "file:///") {
		sourceURLParts := strings.Split(sourceURL, "file:///")
		localFilePath := fmt.Sprintf("/%s", sourceURLParts[1]) // Add another 'slash' so it's an absolute path - that's because SMB has 3 slashes
		binaryData, err = ReadLocalFileBinary(localFilePath)
	}

	if err != nil {
		return nil, diag.NewErrorDiagnostic(
			"Couldn't fetch ISO from source",
			fmt.Sprintf("Couldn't fetch ISO from source '%s': %s", sourceURL, err.Error()),
		)
	}

	return binaryData, nil
}

func CreateISO(
	restClient RestClient,
	name string,
	readyForInsert bool,
	binaryData []byte,
	ctx context.Context,
) (string, map[string]any) {
	payload := map[string]any{
		"name":           name,
		"size":           len(binaryData),
		"readyForInsert": readyForInsert,
	}
	taskTag, _, _ := restClient.CreateRecord(
		"/rest/v1/ISO",
		payload,
		-1,
	)
	taskTag.WaitTask(restClient, ctx)
	isoUUID := taskTag.CreatedUUID
	iso := GetISOByUUID(restClient, isoUUID)
	return isoUUID, *iso
}

func GetISOByUUID(
	restClient RestClient,
	isoUUID string,
) *map[string]any {
	iso := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/ISO/%s", isoUUID),
		nil,
		false,
		-1,
	)
	return iso
}

func UpdateISO(
	restClient RestClient,
	isoUUID string,
	payload map[string]any,
	ctx context.Context,
) diag.Diagnostic {
	taskTag, err := restClient.UpdateRecord(
		fmt.Sprintf("/rest/v1/ISO/%s", isoUUID),
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

func UploadISO(
	restClient RestClient,
	isoUUID string,
	binaryData []byte,
	ctx context.Context,
) (*map[string]any, diag.Diagnostic) {
	fileSize := len(binaryData)

	_, err := restClient.PutBinaryRecordWithoutTaskTag(
		fmt.Sprintf("/rest/v1/ISO/%s/data/", isoUUID),
		binaryData,
		int64(fileSize),
		-1,
		ctx,
	)

	if err != nil {
		return nil, diag.NewWarningDiagnostic(
			"HC3 is receiving too many requests at the same time.",
			fmt.Sprintf("Please retry apply after Terraform finishes it's current operation or consider using the `-parallelism=1` terraform option. HC3 response message: %v", err.Error()),
		)
	}

	iso := GetISOByUUID(restClient, isoUUID)
	return iso, nil
}
