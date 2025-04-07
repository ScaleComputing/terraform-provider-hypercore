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

`, source_vm_name)
}
