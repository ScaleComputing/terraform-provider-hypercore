// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-hypercore/internal/provider"
)

var source_vm_name = os.Getenv("SOURCE_VM_NAME")
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

}
