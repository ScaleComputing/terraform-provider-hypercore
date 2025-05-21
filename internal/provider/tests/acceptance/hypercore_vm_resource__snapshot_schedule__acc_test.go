// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreVMResourceSnapshotSchedule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testConfig_NoSnapshotScheduleUUID("testtf-vm"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.0", "testtf"),
					resource.TestCheckNoResourceAttr("hypercore_vm.test", "clone"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "snapshot_schedule_uuid", ""),
					// resource.TestCheckResourceAttr("hypercore_vm.test", "power_state", requested_power_state),
				),
			},
			// TODO make ImportState test pass again.
			/*
				// ImportState testing
				{
					ResourceName:      "hypercore_vm.test",
					ImportState:       true,
					ImportStateVerify: true,
					// This is not normally necessary, but is here because this
					// example code does not have an actual upstream service.
					// Once the Read method is able to refresh information from
					// the upstream service, this can be removed.
					ImportStateVerifyIgnore: []string{
						"id",
						// TODO do not ignore below attributes
						"name",
						"description",
						"group",
						"vcpu",
						"memory",
						"disk_size",
						"clone.source_vm_uuid",
						"clone.user_data",
						"clone.meta_data",
						"power_state",
					},
				},
			*/
			// Update and Read testing
			{
				Config: testConfig_NoSnapshotScheduleUUID("testtf-vm"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.0", "testtf"),
					resource.TestCheckNoResourceAttr("hypercore_vm.test", "clone"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "snapshot_schedule_uuid", ""),
					// resource.TestCheckResourceAttr("hypercore_vm.test", "power_state", requested_power_state),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// func testConfig_EmptySnapshotScheduleUUID(vm_name string) string {
// 	return fmt.Sprintf(`
// resource "hypercore_vm" "test" {
//   name = %[1]q
//   group = "testtf"
//   vcpu = 4
//   memory = 4096
//   description = "testtf-vm-description"
//   snapshot_schedule_uuid = ""
// }
// `, vm_name)
// }

func testConfig_NoSnapshotScheduleUUID(vm_name string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name = %[1]q
  tags = ["testtf"]
  vcpu = 4
  memory = 4096
  description = "testtf-vm-description"
  // snapshot_schedule_uuid = ""
  affinity_strategy = {}
}
`, vm_name)
}
