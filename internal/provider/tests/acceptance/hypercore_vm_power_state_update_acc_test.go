// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Changing state in place must leave the computed vm known and current. Before the fix,
// Update never set vm, and Terraform rejected every in-place change with "provider still
// indicated an unknown value for hypercore_vm_power_state.test.vm" (issue #118).
// Self-contained: creates its own VM, needs no pre-existing fixtures. The VM has no disk or
// OS, so SHUTOFF uses force_shutoff. It is set only for SHUTOFF: on RUNNING it would also
// power-cycle the VM (issue #113).
func TestAccHypercorePowerStateResource_UpdateRefreshesVM(t *testing.T) {
	vmName := "testtf-vm-power-state-update"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercorePowerStateUpdateConfig(vmName, "RUNNING"),
				Check:  resource.TestCheckResourceAttr("hypercore_vm_power_state.test", "vm.power_state", "RUNNING"),
			},
			{
				Config: testAccHypercorePowerStateUpdateConfig(vmName, "SHUTOFF"),
				Check:  resource.TestCheckResourceAttr("hypercore_vm_power_state.test", "vm.power_state", "SHUTOFF"),
			},
			{
				Config: testAccHypercorePowerStateUpdateConfig(vmName, "RUNNING"),
				Check:  resource.TestCheckResourceAttr("hypercore_vm_power_state.test", "vm.power_state", "RUNNING"),
			},
		},
	})
}

func testAccHypercorePowerStateUpdateConfig(vmName, state string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name              = %[1]q
  description       = "testtf acceptance test VM, safe to delete"
  tags              = ["testtf"] # with no tags, refresh reads back [""] and the plan never settles
  vcpu              = 1
  memory            = 512
  affinity_strategy = {}
}

resource "hypercore_vm_power_state" "test" {
  vm_uuid       = hypercore_vm.test.id
  state         = %[2]q
  force_shutoff = %[3]t
}
`, vmName, state, state == "SHUTOFF")
}
