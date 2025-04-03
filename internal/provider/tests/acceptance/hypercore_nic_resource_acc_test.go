// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var source_vm_uuid = "97904009-1878-4881-b6df-83c85ab7dc1a"
var test_vm_name = "integration-test-vm-nic"

//var source_vm_name = "integration-test-vm"

func TestAccHypercoreNicResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Clone VM and create NIC
			{
				Config: testAccHypercoreSourceVMRConfig(source_vm_uuid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", test_vm_name),
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "integration-vm-description"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "group", "Xlabintegrationtest"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					resource.TestCheckResourceAttr("hypercore_nic.test", "vlan", "11"),
					resource.TestCheckResourceAttr("hypercore_nic.test", "type", "VIRTIO"),
				),
			},
			{
				Config: testAccHypercoreNicResourceConfig(source_vm_uuid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_nic.testa", "vlan", "12"),
					resource.TestCheckResourceAttr("hypercore_nic.testa", "type", "VIRTIO"),
				),
			},
		},
	})
}

func testAccHypercoreSourceVMRConfig(source_vm_uuid string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name = %[2]q
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
data "hypercore_vm" "test" {
  name = %[2]q
}

resource "hypercore_nic" "test" {
  vm_uuid = hypercore_vm.test.vms.0.uuid
}
  vlan    = 11
  type    = "VIRTIO"
}

output "vm_id" {
  value = hypercore_vm.test.vms.0.uuid
}
}

`, source_vm_uuid, test_vm_name)
}

func testAccHypercoreNicResourceConfig(source_vm_uuid string) string {
	return `
resource "hypercore_nic" "testa" {
  vm_uuid = "${output.vm_id}"
  vlan    = 12
  type    = "VIRTIO"
}
`
}
