// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreBootOrderResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercoreBootOrderResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_disk.test", "size", "3"),
					resource.TestCheckResourceAttr("hypercore_disk.test", "type", "IDE_DISK"),
					resource.TestCheckResourceAttr("hypercore_nic.test", "vlan", "11"),
					resource.TestCheckResourceAttr("hypercore_nic.test", "type", "VIRTIO"),
					resource.TestCheckResourceAttr("hypercore_vm_boot_order.test", "boot_devices", "[hypercore_nic.test.id, hypercore_disk.test.id]"),
				),
			},
		},
	})
}

func testAccHypercoreBootOrderResourceConfig() string {
	return fmt.Sprintf(`
data "hypercore_vm" "bootvm" {
  name = %[1]q
}

resource "hypercore_disk" "test" {
  vm_uuid = data.hypercore_vm.bootvm.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3
}

resource "hypercore_nic" "test" {
  vm_uuid = data.hypercore_vm.nicvm.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"
}

resource "hypercore_vm_boot_order" "test" {
  vm_uuid = data.hypercore_vm.bootvm.vms.0.uuid
  boot_devices = [
    hypercore_disk.test.id,
    hypercore_nic.test.id,
  ]
}
`, source_vm_name)
}
