// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// state = "RUNNING" with force_shutoff = true must START the VM; force_shutoff only
// changes how a VM is turned off (issue #113). Self-contained: creates its own VM.
func TestAccHypercorePowerStateResource_ForceShutoffStillStarts(t *testing.T) {
	vmName := "testtf-vm-force-shutoff-running"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercorePowerStateForceShutoffConfig(vmName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm_power_state.test", "state", "RUNNING"),
					checkVMPowerStateInHyperCore("hypercore_vm.test", "RUNNING"),
				),
			},
		},
	})
}

// Reads the VM's state from HyperCore itself, not from Terraform state.
func checkVMPowerStateInHyperCore(vmResource string, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[vmResource]
		if !ok {
			return fmt.Errorf("resource %s not found in state", vmResource)
		}
		got, diag := utils.GetVMPowerState(rs.Primary.ID, *SetupRestClient())
		if diag != nil {
			return fmt.Errorf("reading power state: %s", diag.Detail())
		}
		if got != want {
			return fmt.Errorf("HyperCore reports VM %s as %s, want %s", rs.Primary.ID, got, want)
		}
		return nil
	}
}

func testAccHypercorePowerStateForceShutoffConfig(vmName string) string {
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
  state         = "RUNNING"
  force_shutoff = true
}
`, vmName)
}
