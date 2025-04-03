// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var source_vm_uuid = "97904009-1878-4881-b6df-83c85ab7dc1a"
var test_vm_name = "integration-test-vm-nic"
var test_vm_uuid = ""

//var source_vm_name = "integration-test-vm"

func TestAccHypercoreNicResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Prepare VM for test (clone from source VM)
			{
				Config: testAccHypercoreSourceVMRConfig(source_vm_uuid),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Store cloned VM uuid for later use
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["hypercore_vm.test"]
						if !ok {
							return fmt.Errorf("not found: %s", test_vm_name)
						}
						test_vm_uuid = rs.Primary.Attributes["id"]
						fmt.Printf("Captured Resource ID: %s\n", test_vm_uuid)
						return nil
					},
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", test_vm_name),
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "integration-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "group", "Xlabintegrationtest"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
				),
			},
			// Create new NIC
			{
				Config: func() string {
					if test_vm_uuid == "" {
						panic("test_vm_uuid is empty! Ensure VM is created successfully before using it.")
					}
					return testAccHypercoreNicResourceConfig(test_vm_uuid)
				}(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_nic.test", "vlan", "11"),
					resource.TestCheckResourceAttr("hypercore_nic.test", "type", "VIRTIO"),
				),
			},
		},
	})
}

func testAccHypercoreSourceVMRConfig(source_vm_uuid string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name = "integration-test-vm-nic"
  group = "Xlabintegrationtest"
  vcpu = 4
  memory = 4096
  description = "integration-vm-description"
  clone = {
	source_vm_uuid = %[1]q
	user_data = ""
	meta_data = ""
  }
}
`, source_vm_uuid)
}

func testAccHypercoreNicResourceConfig(test_vm_uuid string) string {
	return fmt.Sprintf(`
resource "hypercore_nic" "test" {
  vm_uuid = %[1]q
  vlan    = 11
  type    = "VIRTIO"
}
`, test_vm_uuid)
}
