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
					resource.TestCheckResourceAttr("hypercore_vm_boot_order.test", "boot_devices.#", "2"),
					resource.TestCheckResourceAttr("hypercore_vm_boot_order.test", "boot_devices.0", source_nic_uuid),
					resource.TestCheckResourceAttr("hypercore_vm_boot_order.test", "boot_devices.1", source_disk_uuid),
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

resource "hypercore_vm_boot_order" "test" {
  vm_uuid = data.hypercore_vm.bootvm.vms.0.uuid
  boot_devices = [
    %[2]q,
	%[3]q,
  ]
}
`, source_vm_name, source_nic_uuid, source_disk_uuid)
}
