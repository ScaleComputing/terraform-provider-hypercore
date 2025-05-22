// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreVMResourceClone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccHypercoreVMResourceCloneConfig("testtf-vm"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.0", "testtf-tag-1"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.1", "testtf-tag-2"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "clone.meta_data", ""),
					resource.TestCheckResourceAttr("hypercore_vm.test", "clone.user_data", ""),
					resource.TestCheckResourceAttr("hypercore_vm.test", "clone.source_vm_uuid", source_vm_uuid),
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					// resource.TestCheckResourceAttr("hypercore_vm.test", "power_state", requested_power_state),
				),
			},
			// Update and Read testing
			{
				Config: testAccHypercoreVMResourceCloneConfig("testtf-vm"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.0", "testtf-tag-1"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.1", "testtf-tag-2"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					// resource.TestCheckResourceAttr("hypercore_vm.test", "power_state", requested_power_state),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

/*
func checkDiskSize(s *terraform.State) error {
	vm_name := "testtf-vm"
	expected_disk_size_b := 1.2 * 1000 * 1000 * 1000

	restClient := SetupRestClient()
	// query := map[string]any{}
	// restClient.ListRecords("/rest/v1/VirDomain", query, 60.0, false)
	query := map[string]any{"name": vm_name}
	all_vms := utils.GetVM(query, *restClient)
	if len(all_vms) != 1 {
		return fmt.Errorf("Expected exactly one VM with name %s, got %d VMs", vm_name, len(all_vms))
	}
	vm := all_vms[0]
	disks := utils.AnyToListOfMap(vm["blockDevs"])
	if len(disks) != 4 {
		return fmt.Errorf("Expected exactly four disk, VM name %s, got %d disks", vm_name, len(disks))
	}
	disk_size_b := utils.AnyToFloat64(disks[0]["capacity"])
	if disk_size_b != expected_disk_size_b {
		return fmt.Errorf("Expected disk size %f, VM name %s, got %f size", expected_disk_size_b, vm_name, disk_size_b)
	}
	return nil
}
*/

func testAccHypercoreVMResourceCloneConfig(vm_name string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name = %[1]q
  tags = [
    "testtf-tag-1",
    "testtf-tag-2",
  ]
  vcpu = 4
  memory = 4096
  description = "testtf-vm-description"
  snapshot_schedule_uuid = ""
  // power_state = %[3]q
  clone = {
	source_vm_uuid = %[2]q
	user_data = ""
	meta_data = ""
	preserve_mac_address = false
  }
  affinity_strategy = {}
}
`, vm_name, source_vm_uuid, requested_power_state)
}
