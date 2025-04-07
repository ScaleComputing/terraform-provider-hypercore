// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercorePowerStateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercorePowerStateResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm_power_state.power_state_test", "state", "RUNNING"),
					resource.TestCheckResourceAttr("hypercore_vm_power_state.power_state_test_cleanup", "state", "SHUTOFF"),
				),
			},
		},
	})
}

func testAccHypercorePowerStateResourceConfig() string {
	return fmt.Sprintf(`
data "hypercore_vm" "integrationvm" {
  name = %[1]q
}

resource "hypercore_vm_power_state" "power_state_test" {
  vm_uuid = data.hypercore_vm.integrationvm.vms.0.uuid
  state   = "RUNNING"
}

resource "null_resource" "wait_before" {
  provisioner "local-exec" {
    command = "sleep 15"
  }
}

resource "hypercore_vm_power_state" "power_state_test_cleanup" {
  vm_uuid = data.hypercore_vm.integrationvm.vms.0.uuid
  state   = "SHUTOFF"
}

resource "null_resource" "wait_after" {
  provisioner "local-exec" {
    command = "sleep 15"
  }
}

`, source_vm_name)
}
