// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var source_vm_uuid = "97904009-1878-4881-b6df-83c85ab7dc1a"

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
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", "integration-test-vm-nic"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "integration-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "group", "Xlabintegrationtest"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
				),
			},
			// Create NIC
			/*
				{
					Config: testAccHypercoreNicResourceConfig(source_vm_uuid),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("hypercore_vm.test", "name", "integration-test-vm-nic"),
						resource.TestCheckResourceAttr("hypercore_vm.test", "description", "integration-vm-description"),
						resource.TestCheckResourceAttr("hypercore_vm.test", "group", "Xlabintegrationtest"),
						resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
						resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					),
				},
			*/
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

/*
func testAccHypercoreNicResourceConfig(source_vm_uuid string) string {
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
*/
