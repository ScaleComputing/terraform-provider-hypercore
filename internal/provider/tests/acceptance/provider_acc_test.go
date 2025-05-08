// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-hypercore/internal/provider"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

var source_vm_name = os.Getenv("SOURCE_VM_NAME")
var source_vm_uuid = os.Getenv("SOURCE_VM_UUID")
var existing_vdisk_uuid = os.Getenv("EXISTING_VDISK_UUID")
var source_nic_uuid = os.Getenv("SOURCE_NIC_UUID")
var source_disk_uuid = os.Getenv("SOURCE_DISK_UUID")
var smb_server = os.Getenv("SMB_SERVER")
var smb_username = os.Getenv("SMB_USERNAME")
var smb_password = os.Getenv("SMB_PASSWORD")
var smb_path = os.Getenv("SMB_PATH")
var smb_filename = os.Getenv("SMB_FILENAME")

var requested_power_state string = "stop" // "started"

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"hypercore": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Prechecks
	// Don't use terraform CRUD operations here, this is ran prior to the test and will not cleanup

	// Check environ variables are loaded (env.txt file)
	mandatory_env_vars := []string{
		"SOURCE_VM_NAME",
		"SOURCE_VM_UUID",
		"EXISTING_VDISK_UUID",
		"SOURCE_NIC_UUID",
		"SOURCE_DISK_UUID",
		"SMB_SERVER",
		"SMB_USERNAME",
		"SMB_PASSWORD",
		"SMB_PATH",
		"SMB_FILENAME",
	}
	for _, key := range mandatory_env_vars {
		value := os.Getenv(key)
		if value == "" {
			t.Fatalf("Environ variable %s must be set for acceptance tests", key)
		}
	}
}

var testAccRestClient *utils.RestClient

func SetupRestClient() *utils.RestClient {
	scHost := os.Getenv("HC_HOST")
	scUsername := os.Getenv("HC_USERNAME")
	scPassword := os.Getenv("HC_PASSWORD")
	scAuthMethod := os.Getenv("HC_AUTH_METHOD")
	scTimeoutF := 60.0
	// scTimeout := os.Getenv("HC_TIMEOUT")

	if testAccRestClient == nil {
		testAccRestClient, _ = utils.NewRestClient(
			scHost,
			scUsername,
			scPassword,
			scAuthMethod,
			scTimeoutF,
		)
		testAccRestClient.Login()
		// tflog.Debug(ctx, fmt.Sprintf("Logged in with session ID: %s\n", restClient.AuthHeader["Cookie"]))
	}

	return testAccRestClient
}
